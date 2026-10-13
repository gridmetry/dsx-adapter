package bridge

import (
	"crypto/ecdsa"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"gridmetry/dsx-adapter/internal/flex"
	"gridmetry/dsx-adapter/internal/lab"
	"gridmetry/dsx-adapter/internal/presentation"
)

type fakeHub struct {
	mu        sync.Mutex
	published [][]byte
}

func (f *fakeHub) Publish(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, data)
}

func (f *fakeHub) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.published)
}

func buildCodecAndEnvelope(t *testing.T) (*flex.Codec, []byte) {
	t.Helper()
	f := newBridgeTestFixture(t)
	now := time.Now().UTC()
	payload := map[string]any{
		"targets": []map[string]any{
			{"interval": map[string]any{}, "load_constraint": nil},
		},
	}
	return f.codec, f.envelope(now, "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e", payload)
}

// bridgeTestFixture builds a codec plus a reusable signing identity so
// tests can produce several distinct signed envelopes (e.g. varying
// interval.end_time) without repeating the trust/policy/signing setup.
type bridgeTestFixture struct {
	t     *testing.T
	priv  *ecdsa.PrivateKey
	kid   string
	codec *flex.Codec
}

func newBridgeTestFixture(t *testing.T) *bridgeTestFixture {
	t.Helper()
	priv, err := flex.GenerateSigningKey()
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}
	kid := "lab-signer-1"
	trust := flex.NewTrust(flex.TrustedSigner{KeyID: kid, PublicKey: &priv.PublicKey, Source: "//grid/v1/isv/gridmetry"})
	policy := flex.Policy{FeedTag: "main-a", MaxWatts: 1_000_000, MaxEventAge: 5 * time.Minute, MaxClockSkew: 30 * time.Second}
	codec := &flex.Codec{Trust: trust, Policy: policy, Dedup: flex.NewDedup(1024, time.Hour)}
	return &bridgeTestFixture{t: t, priv: priv, kid: kid, codec: codec}
}

func (f *bridgeTestFixture) envelope(now time.Time, id string, payload map[string]any) []byte {
	f.t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		f.t.Fatalf("marshal payload: %v", err)
	}
	compact, err := flex.SignES256(f.priv, f.kid, payloadBytes)
	if err != nil {
		f.t.Fatalf("SignES256: %v", err)
	}
	ev := flex.CloudEvent{
		SpecVersion: flex.SpecVersion, ID: id,
		Source: "//grid/v1/isv/gridmetry", Type: flex.LoadTargetSetType,
		Time: now.Format(time.RFC3339), DataContentType: "application/json",
		Subject: "loadtarget", Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data: compact,
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		f.t.Fatalf("marshal envelope: %v", err)
	}
	return raw
}

// buildCodecAndEnvelopeWithEndTime signs an envelope carrying an
// interval.end_time, using ingress as the CloudEvents "time" so the
// command is fresh at decode time regardless of where endTime falls.
func buildCodecAndEnvelopeWithEndTime(t *testing.T, ingress, endTime time.Time) (*flex.Codec, []byte) {
	t.Helper()
	f := newBridgeTestFixture(t)
	payload := map[string]any{
		"targets": []map[string]any{
			{
				"interval":        map[string]any{"end_time": endTime.Format(time.RFC3339)},
				"load_constraint": map[string]any{"value": 100, "unit": "watt"},
			},
		},
	}
	return f.codec, f.envelope(ingress, "c4f6a1d2-3e5b-4a7c-8d9e-0f1a2b3c4d5e", payload)
}

// sequenceClock returns a func() time.Time that yields each of times in
// order, repeating the last value once exhausted, so tests can simulate a
// fresh clock read advancing between ingress decode and a later
// pre-delivery expiry check.
func sequenceClock(times ...time.Time) func() time.Time {
	i := 0
	return func() time.Time {
		t := times[i]
		if i < len(times)-1 {
			i++
		}
		return t
	}
}

func TestPipelineProcessesAcceptedMessageAndPublishes(t *testing.T) {
	codec, raw := buildCodecAndEnvelope(t)
	hub := &fakeHub{}
	stats := &lab.Stats{}
	p := NewPipeline(codec, hub, stats, 8)

	done := make(chan struct{})
	go p.Run(done)
	defer close(done)

	if !p.Enqueue(InboundMessage{Topic: "grid/v1/isv/gridmetry/loadtarget/set", Payload: raw, Received: time.Now()}) {
		t.Fatal("expected Enqueue to succeed")
	}

	waitFor(t, func() bool { return hub.count() == 1 })
	if stats.Snapshot().Accepted != 1 {
		t.Fatalf("expected 1 accepted, got %+v", stats.Snapshot())
	}
}

func TestPipelineCountsRejected(t *testing.T) {
	codec, _ := buildCodecAndEnvelope(t)
	hub := &fakeHub{}
	stats := &lab.Stats{}
	p := NewPipeline(codec, hub, stats, 8)

	done := make(chan struct{})
	go p.Run(done)
	defer close(done)

	p.Enqueue(InboundMessage{Topic: "x", Payload: []byte("not json"), Received: time.Now()})

	waitFor(t, func() bool { return stats.Snapshot().Rejected == 1 })
	if got := hub.count(); got != 0 {
		t.Fatalf("expected no published events for a rejected message, got %d", got)
	}
}

func TestPipelineCountsDuplicate(t *testing.T) {
	codec, raw := buildCodecAndEnvelope(t)
	hub := &fakeHub{}
	stats := &lab.Stats{}
	p := NewPipeline(codec, hub, stats, 8)

	done := make(chan struct{})
	go p.Run(done)
	defer close(done)

	p.Enqueue(InboundMessage{Topic: "t", Payload: raw, Received: time.Now()})
	waitFor(t, func() bool { return stats.Snapshot().Accepted == 1 })
	p.Enqueue(InboundMessage{Topic: "t", Payload: raw, Received: time.Now()})
	waitFor(t, func() bool { return stats.Snapshot().Duplicate == 1 })

	if got := hub.count(); got != 1 {
		t.Fatalf("expected exactly 1 publish despite the duplicate, got %d", got)
	}
}

func TestPipelineDropsAcceptedCommandWhoseEndTimeExpiresBeforeHubPublish(t *testing.T) {
	ingress := time.Now().UTC()
	endTime := ingress.Add(2 * time.Second)
	codec, raw := buildCodecAndEnvelopeWithEndTime(t, ingress, endTime)
	hub := &fakeHub{}
	stats := &lab.Stats{}
	p := NewPipeline(codec, hub, stats, 8)
	// Simulate the queue delaying processing past end_time: decode reads
	// ingress (still fresh), but the clock read immediately before the hub
	// publish is already past end_time.
	p.nowFn = sequenceClock(ingress, endTime.Add(time.Second))

	done := make(chan struct{})
	go p.Run(done)
	defer close(done)

	if !p.Enqueue(InboundMessage{Topic: "t", Payload: raw, Received: ingress}) {
		t.Fatal("expected Enqueue to succeed")
	}

	waitFor(t, func() bool { return stats.Snapshot().Expired == 1 })
	if got := hub.count(); got != 0 {
		t.Fatalf("expected no publish for a command whose end_time expired before hub publish, got %d", got)
	}
	if stats.Snapshot().Accepted != 1 {
		t.Fatalf("expected the command to still be counted as policy-accepted before the delivery-time expiry check, got %+v", stats.Snapshot())
	}
}

func TestPipelinePublishesWhenEndTimeStillValidAtHubPublish(t *testing.T) {
	ingress := time.Now().UTC()
	endTime := ingress.Add(time.Hour)
	codec, raw := buildCodecAndEnvelopeWithEndTime(t, ingress, endTime)
	hub := &fakeHub{}
	stats := &lab.Stats{}
	p := NewPipeline(codec, hub, stats, 8)

	done := make(chan struct{})
	go p.Run(done)
	defer close(done)

	p.Enqueue(InboundMessage{Topic: "t", Payload: raw, Received: ingress})

	waitFor(t, func() bool { return hub.count() == 1 })
	if stats.Snapshot().Expired != 0 {
		t.Fatalf("expected no expired count for a still-valid end_time, got %+v", stats.Snapshot())
	}
}

func TestPipelineEnqueueDropsWhenQueueFull(t *testing.T) {
	codec, _ := buildCodecAndEnvelope(t)
	hub := &fakeHub{}
	stats := &lab.Stats{}
	p := NewPipeline(codec, hub, stats, 1)
	// Do not start Run, so the single queue slot stays occupied.
	ok1 := p.Enqueue(InboundMessage{Topic: "t", Payload: []byte("a"), Received: time.Now()})
	ok2 := p.Enqueue(InboundMessage{Topic: "t", Payload: []byte("b"), Received: time.Now()})
	if !ok1 {
		t.Fatal("expected first enqueue to succeed")
	}
	if ok2 {
		t.Fatal("expected second enqueue to be dropped when queue is full")
	}
	if stats.Snapshot().QueueDropped != 1 {
		t.Fatalf("expected 1 queue_dropped, got %+v", stats.Snapshot())
	}
}

func TestPipelineAddsVisualizationToVerifiedCommand(t *testing.T) {
	fixture := newBridgeTestFixture(t)
	now := time.Now().UTC()
	hub := &fakeHub{}
	pipeline := NewPipeline(fixture.codec, hub, &lab.Stats{}, 8)
	mapper, err := presentation.NewMapper(presentation.Config{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: 80})
	if err != nil {
		t.Fatal(err)
	}
	pipeline.Visualization = mapper
	payload := map[string]any{
		"targets": []map[string]any{{
			"interval":        map[string]any{},
			"load_constraint": map[string]any{"value": 920000, "unit": "watt"},
		}},
	}
	raw := fixture.envelope(now, "d4f6a1d2-3e5b-4a7c-8d9e-0f1a2b3c4d5e", payload)
	pipeline.process(InboundMessage{Topic: "grid/v1/isv/gridmetry/loadtarget/set", Payload: raw, Received: now})
	if hub.count() != 1 {
		t.Fatal("verified command was not published")
	}
	var received struct {
		flex.NormalizedEvent
		Visualization presentation.Command `json:"visualization"`
	}
	if err := json.Unmarshal(hub.published[0], &received); err != nil {
		t.Fatal(err)
	}
	if received.Type != flex.CommandReceivedType || received.Visualization.ID != received.ID || received.Visualization.LoadPercent != 60 || received.Visualization.DisplayedPowerWatts != 920000 {
		t.Fatalf("unexpected verified visualization: %+v", received)
	}
	pipeline.process(InboundMessage{Payload: []byte("unsigned invalid input"), Received: now})
	if hub.count() != 1 {
		t.Fatal("rejected input reached visualization output")
	}
}

func TestPipelineMapsReleaseToConfiguredBaseline(t *testing.T) {
	codec, raw := buildCodecAndEnvelope(t)
	hub := &fakeHub{}
	pipeline := NewPipeline(codec, hub, &lab.Stats{}, 8)
	pipeline.Visualization, _ = presentation.NewMapper(presentation.Config{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: 80})
	pipeline.process(InboundMessage{Payload: raw, Received: time.Now()})
	if hub.count() != 1 {
		t.Fatal("release receipt missing")
	}
	var received struct {
		Visualization presentation.Command `json:"visualization"`
	}
	if err := json.Unmarshal(hub.published[0], &received); err != nil {
		t.Fatal(err)
	}
	if !received.Visualization.Release || received.Visualization.LoadPercent != 80 {
		t.Fatal("release did not restore the configured baseline")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
