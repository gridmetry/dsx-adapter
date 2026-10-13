package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"gridmetry/dsx-adapter/internal/flex"
	"gridmetry/dsx-adapter/internal/lab"
	"gridmetry/dsx-adapter/internal/presentation"
)

type scenarioReceipt struct {
	flex.NormalizedEvent
	Visualization *presentation.Command `json:"visualization,omitempty"`
}

type scenarioInput struct {
	Action string
	Watts  float64
}

func scenarioMaximum(configured []float64) (float64, error) {
	maximum := float64(1_000_000)
	if len(configured) > 1 {
		return 0, errors.New("invalid publisher maximum")
	}
	if len(configured) == 1 {
		maximum = configured[0]
	}
	if maximum <= 0 || math.IsNaN(maximum) || math.IsInf(maximum, 0) {
		return 0, errors.New("invalid publisher maximum")
	}
	return maximum, nil
}

func parseScenarioInput(line string, configured ...float64) (scenarioInput, error) {
	maximum, err := scenarioMaximum(configured)
	if err != nil {
		return scenarioInput{}, err
	}
	line = strings.TrimSpace(line)
	switch strings.ToLower(line) {
	case "":
		return scenarioInput{Action: "empty"}, nil
	case "release", "help":
		return scenarioInput{Action: strings.ToLower(line)}, nil
	case "quit", "exit":
		return scenarioInput{Action: "quit"}, nil
	}
	watts, err := strconv.ParseFloat(line, 64)
	if err != nil || math.IsNaN(watts) || math.IsInf(watts, 0) || watts <= 0 || watts > maximum {
		return scenarioInput{}, errors.New("expected a supported positive watt target, release, help or quit")
	}
	return scenarioInput{Action: "target", Watts: watts}, nil
}

func interactiveScenario(ctx context.Context, input io.Reader, output io.Writer, commandTimeout time.Duration, send func(context.Context, scenarioInput) (*scenarioReceipt, error), configured ...float64) error {
	maximum, err := scenarioMaximum(configured)
	if err != nil {
		return err
	}
	if commandTimeout <= 0 || send == nil {
		return errors.New("invalid interactive publisher configuration")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	type inputLine struct {
		text string
		err  error
	}
	readCtx, cancelReader := context.WithCancel(ctx)
	defer cancelReader()
	lines := make(chan inputLine)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 4096), 8192)
		for scanner.Scan() {
			select {
			case lines <- inputLine{text: scanner.Text()}:
			case <-readCtx.Done():
				return
			}
		}
		if scanner.Err() != nil {
			select {
			case lines <- inputLine{err: errors.New("publisher input limit or read failure")}:
			case <-readCtx.Done():
			}
		}
	}()
	for {
		fmt.Fprint(output, "flex> ")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, open := <-lines:
			if !open {
				return nil
			}
			if line.err != nil {
				return line.err
			}
			command, err := parseScenarioInput(line.text, maximum)
			if err != nil {
				fmt.Fprintf(output, "Expected positive watts up to %.0f, release, help or quit.\n", maximum)
				continue
			}
			switch command.Action {
			case "quit":
				return nil
			case "empty":
				continue
			case "help":
				fmt.Fprintf(output, "Enter positive watts up to %.0f, release, or quit. Receipt confirms adapter delivery, not browser execution. Release is explicit; quit does not release.\n", maximum)
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			commandCtx, cancel := context.WithTimeout(ctx, commandTimeout)
			receipt, err := send(commandCtx, command)
			cancel()
			if err := ctx.Err(); err != nil {
				return err
			}
			if err != nil || receipt == nil {
				return errors.New("publish or matching receipt failed; interactive session stopped")
			}
			encoded, err := json.Marshal(receipt)
			if err != nil {
				return errors.New("receipt encoding failed")
			}
			fmt.Fprintf(output, "dsx-lab scenario: received matching normalized event: %s\n", encoded)
		}
	}
}

// runScenario subscribes to the SSE event stream first, waits for its
// ready signal, then publishes one signed loadtarget command and verifies
// a matching normalized event is received, or repeats this interactively.
// It is a repeatable offline
// demonstration, not a claim of DSX execution or delivered power.
func runScenario(args []string) int {
	fs := flag.NewFlagSet("dsx-lab scenario", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	release := fs.Bool("release", false, "publish a null (release) command instead of a positive constraint")
	wattsFlag := fs.Float64("watts", 800_000, "load_constraint value in watts for a non-release command")
	timeoutFlag := fs.Duration("timeout", 10*time.Second, "bounded overall scenario timeout")
	interactive := fs.Bool("interactive", false, "keep one MQTT/SSE session for typed targets, release, help and quit")
	duration := fs.Duration("duration", 8*time.Minute, "interactive session lifetime, up to 10m")
	maximumFlag := fs.String("max-watts", envOr("DSX_MAX_WATTS", "1000000"), "publisher target ceiling, defaults to DSX_MAX_WATTS or 1000000")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	configuredMaximum, err := strconv.ParseFloat(*maximumFlag, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dsx-lab scenario: invalid publisher maximum")
		return 2
	}
	maximumWatts, err := scenarioMaximum([]float64{configuredMaximum})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dsx-lab scenario: invalid publisher maximum")
		return 2
	}
	if fs.NArg() != 0 || *timeoutFlag <= 0 || *timeoutFlag > time.Minute || *duration <= 0 || *duration > 10*time.Minute {
		fmt.Fprintln(os.Stderr, "dsx-lab scenario: timeout must be positive and at most 1m; duration must be positive and at most 10m")
		return 2
	}
	conflicting := false
	fs.Visit(func(option *flag.Flag) {
		if *interactive && (option.Name == "watts" || option.Name == "release") {
			conflicting = true
		}
		if !*interactive && option.Name == "duration" {
			conflicting = true
		}
	})
	if conflicting {
		fmt.Fprintln(os.Stderr, "dsx-lab scenario: interactive mode uses typed commands; duration requires interactive mode")
		return 2
	}
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	lifetime := *timeoutFlag
	if *interactive {
		lifetime = *duration
	}
	ctx, cancel := context.WithTimeout(signalCtx, lifetime)
	defer cancel()

	secretsDir := envOr("DSX_SECRETS_DIR", "/secrets")
	eventsURL := envOr("DSX_EVENTS_URL", "http://bridge:8090/events")
	brokerURL := envOr("DSX_MQTT_BROKER_URL", "tcp://broker:1883")
	topic := envOr("DSX_MQTT_TOPIC", "grid/v1/isv/gridmetry/loadtarget/set")
	kid := envOr("DSX_SIGNER_KID", "gridmetry-lab-signer-1")

	privKeyBytes, err := lab.ReadBoundedFile(filepath.Join(secretsDir, "signing-private.pem"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: reading signing private key: %v\n", err)
		return 1
	}
	privKey, err := flex.DecodePrivateKeyPEM(privKeyBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: decoding signing private key: %v\n", err)
		return 1
	}
	sseToken, err := lab.ReadBoundedFile(filepath.Join(secretsDir, "sse-token"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: reading SSE token: %v\n", err)
		return 1
	}
	credsBytes, err := lab.ReadBoundedFile(filepath.Join(secretsDir, "mqtt-publisher-credentials.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: reading MQTT publisher credentials: %v\n", err)
		return 1
	}
	var creds lab.MQTTCredentials
	if err := json.Unmarshal(credsBytes, &creds); err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: parsing MQTT publisher credentials: %v\n", err)
		return 1
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventsURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: building SSE request: %v\n", err)
		return 1
	}
	req.Header.Set("Authorization", "Bearer "+string(sseToken))
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") },
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: connecting to event stream: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: event stream returned status %d\n", resp.StatusCode)
		return 1
	}

	reader := bufio.NewReader(resp.Body)
	lines := readScenarioLines(ctx, reader, func() {
		if *interactive {
			cancel()
		}
	})
	startupCtx, cancelStartup := context.WithTimeout(ctx, *timeoutFlag)
	err = waitForScenarioConnected(startupCtx, lines)
	cancelStartup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: %v\n", err)
		return 1
	}
	fmt.Println("dsx-lab scenario: subscribed to event stream, ready")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID("dsx-scenario-" + newUUIDv4()[:8])
	opts.SetUsername(creds.Username)
	opts.SetPassword(creds.Password)
	opts.SetProtocolVersion(4)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(false)
	opts.SetConnectionLostHandler(func(mqtt.Client, error) { cancel() })
	client := mqtt.NewClient(opts)
	if token := client.Connect(); !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		if token.Error() != nil {
			fmt.Fprintf(os.Stderr, "dsx-lab scenario: connecting to MQTT broker: %v\n", token.Error())
		} else {
			fmt.Fprintln(os.Stderr, "dsx-lab scenario: timed out connecting to MQTT broker")
		}
		return 1
	}
	defer client.Disconnect(250)
	fmt.Println("dsx-lab scenario: MQTT connected")

	send := func(commandCtx context.Context, input scenarioInput) (*scenarioReceipt, error) {
		if commandCtx.Err() != nil || !client.IsConnected() {
			return nil, errors.New("publisher session unavailable")
		}
		commandID := newUUIDv4()
		payloadBytes, err := json.Marshal(map[string]any{
			"targets": []map[string]any{buildTarget(input.Action == "release", input.Watts)},
		})
		if err != nil {
			return nil, errors.New("payload encoding failed")
		}
		compact, err := flex.SignES256(privKey, kid, payloadBytes)
		if err != nil {
			return nil, errors.New("command signing failed")
		}
		envelope := flex.CloudEvent{
			SpecVersion: flex.SpecVersion, ID: commandID, Source: "//grid/v1/isv/gridmetry",
			Type: flex.LoadTargetSetType, Time: time.Now().UTC().Format(time.RFC3339),
			DataContentType: "application/json", Subject: "loadtarget", Traceparent: newTraceparent(), Data: compact,
		}
		envelopeBytes, err := json.Marshal(envelope)
		if err != nil {
			return nil, errors.New("envelope encoding failed")
		}
		deadline, _ := commandCtx.Deadline()
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, errors.New("command deadline elapsed")
		}
		pubToken := client.Publish(topic, 1, false, envelopeBytes)
		if !pubToken.WaitTimeout(remaining) || pubToken.Error() != nil {
			return nil, errors.New("MQTT publish failed or timed out")
		}
		fmt.Printf("dsx-lab scenario: published signed command id=%s release=%v\n", commandID, input.Action == "release")
		return waitForScenarioReceipt(commandCtx, lines, commandID)
	}
	if *interactive {
		fmt.Printf("Interactive Flex publisher ready for %s. Enter positive watts up to %.0f, release, help or quit.\n", duration.String(), maximumWatts)
		err := interactiveScenario(ctx, os.Stdin, os.Stdout, *timeoutFlag, send, maximumWatts)
		if err == nil || errors.Is(err, context.DeadlineExceeded) || signalCtx.Err() != nil {
			fmt.Fprintln(os.Stdout, "dsx-lab scenario: interactive session closed; release constraints explicitly before quit")
			return 0
		}
		fmt.Fprintln(os.Stderr, "dsx-lab scenario: interactive publish or receipt failed; session stopped")
		return 1
	}
	input := scenarioInput{Action: "target", Watts: *wattsFlag}
	if *release {
		input.Action = "release"
	}
	event, err := send(ctx, input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dsx-lab scenario: publish or matching receipt failed")
		return 1
	}

	result, _ := json.Marshal(event)
	fmt.Printf("dsx-lab scenario: received matching normalized event: %s\n", result)
	return 0
}

func buildTarget(release bool, watts float64) map[string]any {
	if release {
		return map[string]any{"interval": map[string]any{}, "load_constraint": nil}
	}
	return map[string]any{
		"interval":        map[string]any{},
		"load_constraint": map[string]any{"value": watts, "unit": "watt"},
	}
}

func waitForConnected(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("waiting for connected signal: %w", err)
		}
		if strings.HasPrefix(line, ": connected") {
			return nil
		}
	}
}

type scenarioLine struct {
	text string
	err  error
}

func readScenarioLines(ctx context.Context, reader io.Reader, onClosed ...func()) <-chan scenarioLine {
	lines := make(chan scenarioLine, 64)
	go func() {
		defer close(lines)
		defer func() {
			for _, notify := range onClosed {
				if notify != nil {
					notify()
				}
			}
		}()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 4096), 65536)
		for scanner.Scan() {
			select {
			case lines <- scenarioLine{text: scanner.Text()}:
			case <-ctx.Done():
				return
			}
		}
		if scanner.Err() != nil {
			select {
			case lines <- scenarioLine{err: errors.New("event stream limit or read failure")}:
			case <-ctx.Done():
			}
		}
	}()
	return lines
}

func waitForScenarioConnected(ctx context.Context, lines <-chan scenarioLine) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line, open := <-lines:
			if !open {
				return errors.New("event stream closed before connected marker")
			}
			if line.err != nil {
				return line.err
			}
			if line.text == ": connected" {
				return nil
			}
		}
	}
}

func waitForScenarioReceipt(ctx context.Context, lines <-chan scenarioLine, expectID string) (*scenarioReceipt, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case line, open := <-lines:
			if !open {
				return nil, errors.New("event stream closed before the matching event arrived")
			}
			if line.err != nil {
				return nil, line.err
			}
			if !strings.HasPrefix(line.text, "data: ") {
				continue
			}
			var event scenarioReceipt
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line.text, "data: ")), &event); err == nil && event.ID == expectID {
				return &event, nil
			}
		}
	}
}

func waitForEvent(ctx context.Context, reader *bufio.Reader, expectID string) (*scenarioReceipt, error) {
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	return waitForScenarioReceipt(ctx, readScenarioLines(readCtx, reader), expectID)
}

func newUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]), hex.EncodeToString(b[10:16]))
}

func newTraceparent() string {
	var traceID [16]byte
	var parentID [8]byte
	_, _ = rand.Read(traceID[:])
	_, _ = rand.Read(parentID[:])
	return fmt.Sprintf("00-%s-%s-01", hex.EncodeToString(traceID[:]), hex.EncodeToString(parentID[:]))
}
