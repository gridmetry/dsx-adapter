package flex

import (
	"crypto/ecdsa"
	"encoding/json"
	"testing"
	"time"
)

type codecFixture struct {
	t      *testing.T
	key    *ecdsa.PrivateKey
	kid    string
	trust  *Trust
	policy Policy
}

func newCodecFixture(t *testing.T) *codecFixture {
	t.Helper()
	key := mustGenerateKey(t)
	kid := "lab-signer-1"
	trust := NewTrust(TrustedSigner{KeyID: kid, PublicKey: &key.PublicKey, Source: "//grid/v1/isv/gridmetry"})
	policy := basePolicy()
	return &codecFixture{t: t, key: key, kid: kid, trust: trust, policy: policy}
}

func (f *codecFixture) codec() *Codec {
	return &Codec{Trust: f.trust, Policy: f.policy, Dedup: NewDedup(1024, time.Hour)}
}

func (f *codecFixture) signedEnvelope(now time.Time, id string, payload any) []byte {
	f.t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		f.t.Fatalf("marshal payload: %v", err)
	}
	compact, err := SignES256(f.key, f.kid, payloadBytes)
	if err != nil {
		f.t.Fatalf("SignES256: %v", err)
	}
	ev := CloudEvent{
		SpecVersion:     SpecVersion,
		ID:              id,
		Source:          "//grid/v1/isv/gridmetry",
		Type:            LoadTargetSetType,
		Time:            now.Format(time.RFC3339),
		DataContentType: "application/json",
		Subject:         "loadtarget",
		Traceparent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            compact,
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		f.t.Fatalf("marshal envelope: %v", err)
	}
	return raw
}

func releasePayload() map[string]any {
	return map[string]any{
		"targets": []map[string]any{
			{"interval": map[string]any{}, "load_constraint": nil},
		},
	}
}

func TestCodecAcceptsValidSignedReleaseCommand(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	raw := f.signedEnvelope(now, "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e", releasePayload())

	event, err := f.codec().Decode(raw, now)
	if err != nil {
		t.Fatalf("expected acceptance, got: %v", err)
	}
	if !event.Release || event.FeedTag != "main-a" {
		t.Fatalf("unexpected normalized event: %+v", event)
	}
}

func TestCodecRejectsUnknownSigningKey(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	otherKey := mustGenerateKey(t)
	payloadBytes, _ := json.Marshal(releasePayload())
	compact, err := SignES256(otherKey, "intruder-kid", payloadBytes)
	if err != nil {
		t.Fatalf("SignES256: %v", err)
	}
	ev := CloudEvent{
		SpecVersion: SpecVersion, ID: "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e",
		Source: "//grid/v1/isv/gridmetry", Type: LoadTargetSetType,
		Time: now.Format(time.RFC3339), DataContentType: "application/json",
		Subject: "loadtarget", Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data: compact,
	}
	raw, _ := json.Marshal(ev)
	if _, err := f.codec().Decode(raw, now); err == nil {
		t.Fatal("expected rejection of untrusted signing key")
	}
}

func TestCodecRejectsTamperedSignedData(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	raw := f.signedEnvelope(now, "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e", releasePayload())
	tampered := []byte(string(raw))
	// Corrupt a byte inside the quoted "data" JWS value without altering JSON structure.
	idx := indexOf(tampered, []byte(`"data":"`))
	if idx < 0 {
		t.Fatal("fixture missing data field")
	}
	flipPos := idx + len(`"data":"`) + 5
	tampered[flipPos] = flipByte(tampered[flipPos])
	if _, err := f.codec().Decode(tampered, now); err == nil {
		t.Fatal("expected rejection of tampered signed data")
	}
}

func TestCodecRejectsExpiredEnvelope(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	stale := now.Add(-10 * time.Minute)
	raw := f.signedEnvelope(stale, "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e", releasePayload())
	if _, err := f.codec().Decode(raw, now); err == nil {
		t.Fatal("expected rejection of expired envelope")
	}
}

func TestCodecDropsExactDuplicateAndRejectsConflictingReuse(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	id := "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e"
	raw := f.signedEnvelope(now, id, releasePayload())
	codec := f.codec()

	if _, err := codec.Decode(raw, now); err != nil {
		t.Fatalf("first delivery should be accepted: %v", err)
	}
	if _, err := codec.Decode(raw, now); err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate on exact repeat, got: %v", err)
	}

	otherPayload := map[string]any{
		"targets": []map[string]any{
			{"interval": map[string]any{}, "load_constraint": map[string]any{"value": 100, "unit": "watt"}},
		},
	}
	conflicting := f.signedEnvelope(now, id, otherPayload)
	if _, err := codec.Decode(conflicting, now); err == nil || err == ErrDuplicate {
		t.Fatalf("expected a conflict rejection (not duplicate, not success), got: %v", err)
	}
}

func TestCodecPolicyRejectionDoesNotPoisonDedupForLaterValidSameID(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	id := "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e"
	codec := f.codec()

	stale := f.signedEnvelope(now.Add(-10*time.Minute), id, releasePayload())
	if _, err := codec.Decode(stale, now); err == nil {
		t.Fatal("expected the stale envelope to be rejected by policy")
	}

	fresh := f.signedEnvelope(now, id, releasePayload())
	if _, err := codec.Decode(fresh, now); err != nil {
		t.Fatalf("expected the later valid resubmission with the same id to be accepted, got: %v", err)
	}
}

func TestCodecRejectsOversizedRawMessage(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	big := make([]byte, MaxRawEnvelopeBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	if _, err := f.codec().Decode(big, now); err == nil {
		t.Fatal("expected rejection of oversized raw message")
	}
}

func TestCodecRejectsDuplicateJSONFieldInEnvelope(t *testing.T) {
	f := newCodecFixture(t)
	now := time.Now().UTC()
	raw := []byte(`{"specversion":"1.0.2","specversion":"1.0.2","id":"b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e","source":"//grid/v1/isv/gridmetry","type":"grid.loadtarget.set.v1","time":"` + now.Format(time.RFC3339) + `","datacontenttype":"application/json","subject":"loadtarget","traceparent":"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01","data":"x.y.z"}`)
	if _, err := f.codec().Decode(raw, now); err == nil {
		t.Fatal("expected rejection of duplicate JSON field")
	}
}

func indexOf(haystack, needle []byte) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func flipByte(b byte) byte {
	if b == 'A' {
		return 'B'
	}
	return 'A'
}
