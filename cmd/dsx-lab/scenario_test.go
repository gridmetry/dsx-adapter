package main

import (
	"bufio"
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"gridmetry/dsx-adapter/internal/flex"
)

func TestBuildTargetRelease(t *testing.T) {
	got := buildTarget(true, 800_000)
	if got["load_constraint"] != nil {
		t.Fatalf("expected a nil load_constraint for a release command, got %v", got["load_constraint"])
	}
}

func TestBuildTargetConstraint(t *testing.T) {
	got := buildTarget(false, 800_000)
	lc, ok := got["load_constraint"].(map[string]any)
	if !ok {
		t.Fatalf("expected load_constraint to be a map, got %T", got["load_constraint"])
	}
	if lc["value"] != 800_000.0 || lc["unit"] != "watt" {
		t.Fatalf("unexpected load_constraint: %+v", lc)
	}
}

func TestWaitForConnectedFindsConnectedSignal(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(": heartbeat\n\n: connected\n\ndata: {}\n\n"))
	if err := waitForConnected(reader); err != nil {
		t.Fatalf("waitForConnected: %v", err)
	}
}

func TestWaitForConnectedErrorsOnClosedStreamWithoutSignal(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(": heartbeat\n\n"))
	if err := waitForConnected(reader); err == nil {
		t.Fatal("expected an error when the stream ends before a connected signal")
	}
}

func TestWaitForEventSkipsNonMatchingAndHeartbeatLines(t *testing.T) {
	other, _ := json.Marshal(flex.NormalizedEvent{ID: "other-id"})
	match, _ := json.Marshal(flex.NormalizedEvent{ID: "wanted-id", Release: true})
	stream := ": heartbeat\n\ndata: " + string(other) + "\n\ndata: " + string(match) + "\n\n"
	reader := bufio.NewReader(strings.NewReader(stream))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	event, err := waitForEvent(ctx, reader, "wanted-id")
	if err != nil {
		t.Fatalf("waitForEvent: %v", err)
	}
	if event.ID != "wanted-id" || !event.Release {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestWaitForEventReturnsErrorWhenContextExpiresFirst(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(": heartbeat\n\n"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := waitForEvent(ctx, reader, "never-arrives"); err == nil {
		t.Fatal("expected an error when the context expires before a matching event arrives")
	}
}

func TestNewUUIDv4LooksLikeAUUID(t *testing.T) {
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if got := newUUIDv4(); !uuidPattern.MatchString(got) {
		t.Fatalf("newUUIDv4 produced a non-UUIDv4-looking value: %q", got)
	}
}

func TestNewTraceparentLooksWellFormed(t *testing.T) {
	traceparentPattern := regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-01$`)
	if got := newTraceparent(); !traceparentPattern.MatchString(got) {
		t.Fatalf("newTraceparent produced an unexpected shape: %q", got)
	}
}
