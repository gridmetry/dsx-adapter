package flex

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	josev4 "github.com/go-jose/go-jose/v4"
)

// GenerateSigningKey creates a fresh P-256 keypair for ES256 signing.
func GenerateSigningKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// EncodePrivateKeyPEM encodes an ECDSA private key as PKCS8 PEM.
func EncodePrivateKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

// EncodePublicKeyPEM encodes an ECDSA public key as PKIX PEM.
func EncodePublicKeyPEM(key *ecdsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// DecodePrivateKeyPEM parses a PKCS8 PEM-encoded ECDSA private key.
func DecodePrivateKeyPEM(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("flex: no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("flex: parsing private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("flex: private key is not ECDSA")
	}
	return ecKey, nil
}

// DecodePublicKeyPEM parses a PKIX PEM-encoded ECDSA public key.
func DecodePublicKeyPEM(data []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("flex: no PEM block found")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("flex: parsing public key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("flex: public key is not ECDSA")
	}
	return ecKey, nil
}

// SignES256 produces a JWS compact serialization for payload, signed with
// ES256 and carrying kid in the protected header.
func SignES256(key *ecdsa.PrivateKey, kid string, payload []byte) (string, error) {
	if kid == "" {
		return "", errors.New("flex: kid is required")
	}
	jwk := josev4.JSONWebKey{Key: key, KeyID: kid, Algorithm: "EC", Use: "sig"}
	signer, err := josev4.NewSigner(josev4.SigningKey{Algorithm: josev4.ES256, Key: &jwk}, nil)
	if err != nil {
		return "", fmt.Errorf("flex: creating signer: %w", err)
	}
	obj, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("flex: signing payload: %w", err)
	}
	return obj.CompactSerialize()
}

// PeekKid extracts the protected "kid" header from a JWS compact
// serialization without verifying its signature. Callers must still call
// VerifyES256 with the public key looked up for this kid before trusting
// the payload; PeekKid only identifies which key to try.
func PeekKid(compact string) (string, error) {
	obj, err := josev4.ParseSignedCompact(compact, []josev4.SignatureAlgorithm{josev4.ES256})
	if err != nil {
		return "", fmt.Errorf("flex: parsing JWS: %w", err)
	}
	if len(obj.Signatures) != 1 {
		return "", errors.New("flex: expected exactly one JWS signature")
	}
	kid := obj.Signatures[0].Header.KeyID
	if kid == "" {
		return "", errors.New("flex: JWS missing kid header")
	}
	return kid, nil
}

// VerifyES256 verifies a JWS compact serialization strictly: exactly one
// ES256 signature (never "none" or an unexpected algorithm), whose
// protected "kid" matches expectKid. It returns the verified plaintext
// payload, never trusting the unverified payload.
func VerifyES256(compact string, pub *ecdsa.PublicKey, expectKid string) ([]byte, error) {
	obj, err := josev4.ParseSignedCompact(compact, []josev4.SignatureAlgorithm{josev4.ES256})
	if err != nil {
		return nil, fmt.Errorf("flex: parsing JWS: %w", err)
	}
	if len(obj.Signatures) != 1 {
		return nil, errors.New("flex: expected exactly one JWS signature")
	}
	header := obj.Signatures[0].Header
	if header.Algorithm != string(josev4.ES256) {
		return nil, errors.New("flex: unexpected JWS algorithm")
	}
	if header.KeyID == "" || header.KeyID != expectKid {
		return nil, errors.New("flex: JWS kid does not match trusted signer")
	}
	payload, err := obj.Verify(pub)
	if err != nil {
		return nil, fmt.Errorf("flex: JWS signature verification failed: %w", err)
	}
	return payload, nil
}
