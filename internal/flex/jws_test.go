package flex

import (
	"crypto/ecdsa"
	"testing"
)

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	return key
}

func TestSignVerifyRoundTrip(t *testing.T) {
	key := mustGenerateKey(t)
	payload := []byte(`{"targets":[{"interval":{},"load_constraint":null}]}`)

	compact, err := SignES256(key, "signer-1", payload)
	if err != nil {
		t.Fatalf("SignES256: %v", err)
	}

	got, err := VerifyES256(compact, &key.PublicKey, "signer-1")
	if err != nil {
		t.Fatalf("VerifyES256: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch: got %s want %s", got, payload)
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	key := mustGenerateKey(t)
	other := mustGenerateKey(t)
	compact, err := SignES256(key, "signer-1", []byte(`{}`))
	if err != nil {
		t.Fatalf("SignES256: %v", err)
	}
	if _, err := VerifyES256(compact, &other.PublicKey, "signer-1"); err == nil {
		t.Fatal("expected verification failure with wrong key")
	}
}

func TestVerifyRejectsKidMismatch(t *testing.T) {
	key := mustGenerateKey(t)
	compact, err := SignES256(key, "signer-1", []byte(`{}`))
	if err != nil {
		t.Fatalf("SignES256: %v", err)
	}
	if _, err := VerifyES256(compact, &key.PublicKey, "signer-2"); err == nil {
		t.Fatal("expected verification failure on kid mismatch")
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	key := mustGenerateKey(t)
	compact, err := SignES256(key, "signer-1", []byte(`{"value":1}`))
	if err != nil {
		t.Fatalf("SignES256: %v", err)
	}
	// Flip a character in the payload segment without re-signing.
	tampered := compact[:len(compact)-20] + "A" + compact[len(compact)-19:]
	if _, err := VerifyES256(tampered, &key.PublicKey, "signer-1"); err == nil {
		t.Fatal("expected verification failure on tampered payload")
	}
}

func TestVerifyRejectsUnsignedNoneAlgorithm(t *testing.T) {
	// alg "none" compact JWS: header {"alg":"none"}, empty object payload, empty signature.
	noneJWS := "eyJhbGciOiJub25lIn0.e30."
	_, err := VerifyES256(noneJWS, nil, "signer-1")
	if err == nil {
		t.Fatal("expected rejection of alg=none JWS")
	}
}

func TestPeekKidWithoutVerifying(t *testing.T) {
	key := mustGenerateKey(t)
	compact, err := SignES256(key, "signer-42", []byte(`{}`))
	if err != nil {
		t.Fatalf("SignES256: %v", err)
	}
	kid, err := PeekKid(compact)
	if err != nil {
		t.Fatalf("PeekKid: %v", err)
	}
	if kid != "signer-42" {
		t.Fatalf("got kid %q, want signer-42", kid)
	}
}

func TestPEMRoundTrip(t *testing.T) {
	key := mustGenerateKey(t)
	privPEM, err := EncodePrivateKeyPEM(key)
	if err != nil {
		t.Fatalf("EncodePrivateKeyPEM: %v", err)
	}
	pubPEM, err := EncodePublicKeyPEM(&key.PublicKey)
	if err != nil {
		t.Fatalf("EncodePublicKeyPEM: %v", err)
	}
	decodedPriv, err := DecodePrivateKeyPEM(privPEM)
	if err != nil {
		t.Fatalf("DecodePrivateKeyPEM: %v", err)
	}
	decodedPub, err := DecodePublicKeyPEM(pubPEM)
	if err != nil {
		t.Fatalf("DecodePublicKeyPEM: %v", err)
	}
	if decodedPriv.D.Cmp(key.D) != 0 {
		t.Fatal("private key did not round-trip")
	}
	if decodedPub.X.Cmp(key.PublicKey.X) != 0 {
		t.Fatal("public key did not round-trip")
	}
}
