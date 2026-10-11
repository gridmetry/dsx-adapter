package flex

import (
	"testing"
	"time"
)

func basePolicy() Policy {
	return Policy{FeedTag: "main-a", MaxWatts: 1_000_000, MaxEventAge: 5 * time.Minute, MaxClockSkew: 30 * time.Second}
}

func baseEvent(now time.Time) *CloudEvent {
	return &CloudEvent{
		SpecVersion:     SpecVersion,
		ID:              "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e",
		Source:          "//grid/v1/isv/gridmetry",
		Type:            LoadTargetSetType,
		Time:            now.Format(time.RFC3339),
		DataContentType: "application/json",
		Subject:         "loadtarget",
		Traceparent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}
}

func TestPolicyAcceptsPositiveConstraint(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{},
		LoadConstraint: &LoadConstraint{Value: 500, Unit: "kilowatt"},
	}}}
	got, err := basePolicy().Normalize(baseEvent(now), data, now)
	if err != nil {
		t.Fatalf("expected acceptance, got: %v", err)
	}
	if got.Release {
		t.Fatal("expected release=false")
	}
	if got.ConstraintWatts == nil || *got.ConstraintWatts != 500_000 {
		t.Fatalf("expected 500000 W, got %v", got.ConstraintWatts)
	}
	if got.FeedTag != "main-a" {
		t.Fatalf("unexpected feed tag %q", got.FeedTag)
	}
}

func TestPolicyAcceptsNullRelease(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{Interval: Interval{}, LoadConstraint: nil}}}
	got, err := basePolicy().Normalize(baseEvent(now), data, now)
	if err != nil {
		t.Fatalf("expected acceptance, got: %v", err)
	}
	if !got.Release || got.ConstraintWatts != nil {
		t.Fatalf("expected release with nil constraint, got %+v", got)
	}
}

func TestPolicyRejectsZeroValue(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{},
		LoadConstraint: &LoadConstraint{Value: 0, Unit: "watt"},
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection of zero value")
	}
}

func TestPolicyRejectsExceedingMaxBound(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{},
		LoadConstraint: &LoadConstraint{Value: 2, Unit: "megawatt"},
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection above configured max bound")
	}
}

func TestPolicyRejectsMultipleTargets(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{
		{Interval: Interval{}, LoadConstraint: nil},
		{Interval: Interval{}, LoadConstraint: nil},
	}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection of multiple targets")
	}
}

func TestPolicyRejectsBestEffort(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{},
		LoadConstraint: nil,
		Strategy:       Strategy{BestEffort: true},
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection of best_effort")
	}
}

func TestPolicyRejectsUnknownFeed(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{},
		FeedTags:       []string{"main-b"},
		LoadConstraint: nil,
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection of unknown feed")
	}
}

func TestPolicyAcceptsConfiguredFeedExplicitly(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{},
		FeedTags:       []string{"main-a"},
		LoadConstraint: nil,
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err != nil {
		t.Fatalf("expected acceptance of explicit configured feed, got: %v", err)
	}
}

func TestPolicyRejectsFutureStart(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour).Format(time.RFC3339)
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{StartTime: future},
		LoadConstraint: nil,
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection of future start_time")
	}
}

func TestPolicyAcceptsOmittedStartAsNow(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{Interval: Interval{}, LoadConstraint: nil}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err != nil {
		t.Fatalf("expected acceptance with omitted start, got: %v", err)
	}
}

func TestPolicyAcceptsOpenEndedInterval(t *testing.T) {
	now := time.Now().UTC()
	data := &SetLoadTargetData{Targets: []LoadTarget{{Interval: Interval{}, LoadConstraint: nil}}}
	got, err := basePolicy().Normalize(baseEvent(now), data, now)
	if err != nil {
		t.Fatalf("expected acceptance, got: %v", err)
	}
	if got.EndTime != nil {
		t.Fatal("expected nil end time for open-ended interval")
	}
}

func TestPolicyRejectsAlreadyExpiredEnd(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour).Format(time.RFC3339)
	data := &SetLoadTargetData{Targets: []LoadTarget{{
		Interval:       Interval{EndTime: past},
		LoadConstraint: nil,
	}}}
	if _, err := basePolicy().Normalize(baseEvent(now), data, now); err == nil {
		t.Fatal("expected rejection of already-expired end_time")
	}
}

func TestPolicyRejectsExpiredEnvelope(t *testing.T) {
	now := time.Now().UTC()
	stale := now.Add(-10 * time.Minute)
	data := &SetLoadTargetData{Targets: []LoadTarget{{Interval: Interval{}, LoadConstraint: nil}}}
	if _, err := basePolicy().Normalize(baseEvent(stale), data, now); err == nil {
		t.Fatal("expected rejection of stale envelope time")
	}
}

func TestPolicyRejectsEnvelopeTimeTooFarInFuture(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(5 * time.Minute)
	data := &SetLoadTargetData{Targets: []LoadTarget{{Interval: Interval{}, LoadConstraint: nil}}}
	if _, err := basePolicy().Normalize(baseEvent(future), data, now); err == nil {
		t.Fatal("expected rejection of envelope time far in the future")
	}
}
