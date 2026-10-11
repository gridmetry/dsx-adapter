package flex

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gridmetry/dsx-adapter/internal/flex/schema"
)

// TrustedSigner binds a JWS "kid" to the ES256 public key authorized to use
// it and, optionally, the required CloudEvents "source" for that key. The
// outer envelope source/id/time are not covered by the JWS signature, so
// authorization is anchored on the signed kid, not on unsigned fields.
type TrustedSigner struct {
	KeyID     string
	PublicKey *ecdsa.PublicKey
	Source    string
}

// Trust is a small allowlist of trusted signing keys for this lab.
type Trust struct {
	signers map[string]TrustedSigner
}

// NewTrust builds a Trust allowlist keyed by KeyID.
func NewTrust(signers ...TrustedSigner) *Trust {
	m := make(map[string]TrustedSigner, len(signers))
	for _, s := range signers {
		m[s.KeyID] = s
	}
	return &Trust{signers: m}
}

// Lookup returns the trusted signer for kid, if any.
func (t *Trust) Lookup(kid string) (TrustedSigner, bool) {
	if t == nil {
		return TrustedSigner{}, false
	}
	s, ok := t.signers[kid]
	return s, ok
}

// ErrDuplicate is returned when Decode drops an event because it is an
// exact repeat of a previously accepted (id, signed content) pair. It is
// not a processing error; callers should count it separately from
// rejections.
var ErrDuplicate = errors.New("flex: duplicate event dropped")

// Codec validates and normalizes one raw grid.loadtarget.set.v1 wire
// message end to end: size bounds, strict JSON, schema, signature,
// signer/source trust, lab policy, and bounded dedup.
type Codec struct {
	Trust  *Trust
	Policy Policy
	Dedup  *Dedup
}

// Decode runs the full pipeline against one raw MQTT payload. now is the
// ingress clock reading used for freshness/interval checks.
func (c *Codec) Decode(raw []byte, now time.Time) (*NormalizedEvent, error) {
	if len(raw) == 0 {
		return nil, errors.New("flex: empty message")
	}
	if len(raw) > MaxRawEnvelopeBytes {
		return nil, fmt.Errorf("flex: raw message exceeds %d bytes", MaxRawEnvelopeBytes)
	}

	var ev CloudEvent
	if err := DecodeStrict(raw, &ev); err != nil {
		return nil, fmt.Errorf("flex: decoding envelope: %w", err)
	}

	instance, err := schema.ToInstance(raw)
	if err != nil {
		return nil, fmt.Errorf("flex: parsing envelope for schema validation: %w", err)
	}
	if err := schema.EnvelopeSchema.Validate(instance); err != nil {
		return nil, fmt.Errorf("flex: envelope schema validation failed: %w", err)
	}
	if err := ValidateEnvelope(&ev); err != nil {
		return nil, err
	}

	kid, err := PeekKid(ev.Data)
	if err != nil {
		return nil, err
	}
	signer, ok := c.Trust.Lookup(kid)
	if !ok {
		return nil, errors.New("flex: unknown or untrusted signing key id")
	}
	if signer.Source != "" && ev.Source != signer.Source {
		return nil, errors.New("flex: envelope source does not match trusted signer binding")
	}

	payloadBytes, err := VerifyES256(ev.Data, signer.PublicKey, kid)
	if err != nil {
		return nil, err
	}
	if len(payloadBytes) > MaxDecodedPayloadBytes {
		return nil, fmt.Errorf("flex: decoded payload exceeds %d bytes", MaxDecodedPayloadBytes)
	}

	var data SetLoadTargetData
	if err := DecodeStrict(payloadBytes, &data); err != nil {
		return nil, fmt.Errorf("flex: decoding payload: %w", err)
	}
	payloadInstance, err := schema.ToInstance(payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("flex: parsing payload for schema validation: %w", err)
	}
	if err := schema.PayloadSchema.Validate(payloadInstance); err != nil {
		return nil, fmt.Errorf("flex: payload schema validation failed: %w", err)
	}

	// Dedup runs after policy acceptance, not before: recording an id/
	// fingerprint pair here for a message the policy goes on to reject
	// (expired, unknown feed, best_effort, ...) would poison the cache and
	// cause a later, valid resubmission of that same id to be misread as a
	// duplicate/conflict even though nothing was ever accepted.
	normalized, err := c.Policy.Normalize(&ev, &data, now)
	if err != nil {
		return nil, err
	}

	if c.Dedup != nil {
		fingerprint := sha256Hex(ev.Data)
		switch c.Dedup.Check(kid, ev.ID, fingerprint, now) {
		case DedupDuplicate:
			return nil, ErrDuplicate
		case DedupConflict:
			return nil, errors.New("flex: event id reused with a different signed payload")
		}
	}

	return normalized, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
