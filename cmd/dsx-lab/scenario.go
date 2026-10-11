package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"gridmetry/dsx-adapter/internal/flex"
	"gridmetry/dsx-adapter/internal/lab"
)

// runScenario subscribes to the SSE event stream first, waits for its
// ready signal, then publishes one signed loadtarget command and verifies
// a matching normalized event is received. It is a repeatable offline
// demonstration, not a claim of DSX execution or delivered power.
func runScenario(args []string) int {
	fs := flag.NewFlagSet("dsx-lab scenario", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	release := fs.Bool("release", false, "publish a null (release) command instead of a positive constraint")
	wattsFlag := fs.Float64("watts", 800_000, "load_constraint value in watts for a non-release command")
	timeoutFlag := fs.Duration("timeout", 10*time.Second, "bounded overall scenario timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeoutFlag)
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
	resp, err := http.DefaultClient.Do(req)
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
	if err := waitForConnected(reader); err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: %v\n", err)
		return 1
	}
	fmt.Println("dsx-lab scenario: subscribed to event stream, ready")

	commandID := newUUIDv4()
	payload := map[string]any{
		"targets": []map[string]any{buildTarget(*release, *wattsFlag)},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: marshaling payload: %v\n", err)
		return 1
	}
	compact, err := flex.SignES256(privKey, kid, payloadBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: signing payload: %v\n", err)
		return 1
	}
	envelope := flex.CloudEvent{
		SpecVersion:     flex.SpecVersion,
		ID:              commandID,
		Source:          "//grid/v1/isv/gridmetry",
		Type:            flex.LoadTargetSetType,
		Time:            time.Now().UTC().Format(time.RFC3339),
		DataContentType: "application/json",
		Subject:         "loadtarget",
		Traceparent:     newTraceparent(),
		Data:            compact,
	}
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: marshaling envelope: %v\n", err)
		return 1
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID("dsx-scenario-" + commandID[:8])
	opts.SetUsername(creds.Username)
	opts.SetPassword(creds.Password)
	opts.SetProtocolVersion(4)
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

	pubToken := client.Publish(topic, 1, false, envelopeBytes)
	if !pubToken.WaitTimeout(5*time.Second) || pubToken.Error() != nil {
		if pubToken.Error() != nil {
			fmt.Fprintf(os.Stderr, "dsx-lab scenario: publishing command: %v\n", pubToken.Error())
		} else {
			fmt.Fprintln(os.Stderr, "dsx-lab scenario: timed out publishing command")
		}
		return 1
	}
	fmt.Printf("dsx-lab scenario: published signed command id=%s release=%v\n", commandID, *release)

	event, err := waitForEvent(ctx, reader, commandID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab scenario: %v\n", err)
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

func waitForEvent(ctx context.Context, reader *bufio.Reader, expectID string) (*flex.NormalizedEvent, error) {
	type lineResult struct {
		line string
		err  error
	}
	lines := make(chan lineResult, 1)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			lines <- lineResult{line, err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for the matching event: %w", ctx.Err())
		case lr := <-lines:
			if lr.err != nil {
				if lr.err == io.EOF {
					return nil, fmt.Errorf("event stream closed before the matching event arrived")
				}
				return nil, fmt.Errorf("reading event stream: %w", lr.err)
			}
			if !strings.HasPrefix(lr.line, "data: ") {
				continue
			}
			var event flex.NormalizedEvent
			if err := json.Unmarshal([]byte(strings.TrimPrefix(lr.line, "data: ")), &event); err != nil {
				continue
			}
			if event.ID == expectID {
				return &event, nil
			}
		}
	}
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
