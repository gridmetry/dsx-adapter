package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
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

func TestWaitForEventPreservesVisualization(t *testing.T) {
	stream := "data: {\"type\":\"command_received\",\"id\":\"wanted-id\",\"visualization\":{\"type\":\"visualization_target\",\"id\":\"wanted-id\",\"load_percent\":60,\"displayed_power_watts\":920000}}\n\n"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	event, err := waitForEvent(ctx, bufio.NewReader(strings.NewReader(stream)), "wanted-id")
	if err != nil {
		t.Fatal(err)
	}
	if event.Visualization == nil || event.Visualization.ID != event.ID || event.Visualization.LoadPercent != 60 {
		t.Fatal("visual target was lost from its correlated receipt")
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

func TestParseInteractiveScenarioInput(t *testing.T) {
	for _, test := range []struct {
		input  string
		action string
		watts  float64
	}{
		{"920000", "target", 920000},
		{" 650000.5 ", "target", 650000.5},
		{"release", "release", 0},
		{"help", "help", 0},
		{"quit", "quit", 0},
		{"exit", "quit", 0},
		{"", "empty", 0},
	} {
		input, err := parseScenarioInput(test.input)
		if err != nil || input.Action != test.action || input.Watts != test.watts {
			t.Fatalf("unexpected parsed input for %q: %+v, %v", test.input, input, err)
		}
	}
	for _, input := range []string{"0", "-1", "1000001", "NaN", "Inf", "-Inf", "not-a-command"} {
		if _, err := parseScenarioInput(input); err == nil {
			t.Fatalf("unsafe input accepted: %q", input)
		}
	}
}

func TestInteractiveScenarioConfiguredMaximum(t *testing.T) {
	command, err := parseScenarioInput("1316000", 1316000)
	if err != nil || command.Watts != 1316000 {
		t.Fatal("configured full-load input was rejected")
	}
	if _, err := parseScenarioInput("1316001", 1316000); err == nil {
		t.Fatal("configured ceiling was exceeded")
	}
	for _, maximum := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := parseScenarioInput("920000", maximum); err == nil {
			t.Fatal("invalid publisher ceiling accepted")
		}
	}
}

func TestInteractiveScenarioPublishesSeveralTargetsOnOneSession(t *testing.T) {
	reader := strings.NewReader("920000\ninvalid\n650000\nrelease\nquit\n800000\n")
	var output bytes.Buffer
	var commands []scenarioInput
	send := func(ctx context.Context, input scenarioInput) (*scenarioReceipt, error) {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			t.Fatal("command receipt wait is not bounded")
		}
		commands = append(commands, input)
		return &scenarioReceipt{NormalizedEvent: flex.NormalizedEvent{ID: "receipt-id"}}, nil
	}
	if err := interactiveScenario(context.Background(), reader, &output, time.Second, send); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 3 || commands[0].Watts != 920000 || commands[1].Watts != 650000 || commands[2].Action != "release" {
		t.Fatalf("unexpected interactive commands: %+v", commands)
	}
	if strings.Contains(output.String(), "invalid") {
		t.Fatal("untrusted input was echoed")
	}
}

func TestInteractiveScenarioStopsOnReceiptFailure(t *testing.T) {
	var calls int
	send := func(context.Context, scenarioInput) (*scenarioReceipt, error) {
		calls++
		return nil, errors.New("synthetic receipt failure")
	}
	if err := interactiveScenario(context.Background(), strings.NewReader("920000\n650000\n"), &bytes.Buffer{}, time.Second, send); err == nil || calls != 1 {
		t.Fatal("failed session continued publishing")
	}
}

func TestScenarioStreamReusesOneReaderForMultipleReceipts(t *testing.T) {
	stream := ": connected\n\ndata: {\"id\":\"first\"}\n\n: heartbeat\n\ndata: {\"id\":\"second\",\"release\":true}\n\n"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lines := readScenarioLines(ctx, bufio.NewReader(strings.NewReader(stream)))
	if err := waitForScenarioConnected(ctx, lines); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		event, err := waitForScenarioReceipt(ctx, lines, id)
		if err != nil || event.ID != id {
			t.Fatalf("lost sequential receipt %s: %v", id, err)
		}
	}
}

func TestInteractiveScenarioStopsWhenIdleDeadlineExpires(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	send := func(context.Context, scenarioInput) (*scenarioReceipt, error) {
		t.Fatal("idle session published a command")
		return nil, nil
	}
	if err := interactiveScenario(ctx, reader, &bytes.Buffer{}, time.Second, send); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("idle deadline was not enforced: %v", err)
	}
}

func TestScenarioStreamClosureCancelsIdlePublisher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closed := make(chan struct{})
	readScenarioLines(ctx, strings.NewReader(": connected\n\n"), func() {
		cancel()
		close(closed)
	})
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("stream closure was not reported")
	}
	var published int
	send := func(context.Context, scenarioInput) (*scenarioReceipt, error) {
		published++
		return nil, errors.New("should not publish")
	}
	if err := interactiveScenario(ctx, strings.NewReader("920000\n"), &bytes.Buffer{}, time.Second, send); !errors.Is(err, context.Canceled) || published != 0 {
		t.Fatal("closed stream allowed a command")
	}
}

func TestInteractiveScenarioPreservesSessionDeadlineDuringCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	send := func(commandCtx context.Context, input scenarioInput) (*scenarioReceipt, error) {
		<-commandCtx.Done()
		return nil, commandCtx.Err()
	}
	if err := interactiveScenario(ctx, strings.NewReader("920000\n"), &bytes.Buffer{}, time.Second, send); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("session deadline became a publish failure: %v", err)
	}
}
