//go:build integration

// Native (Docker-free) variant of the Compose-backed suite in
// compose_test.go: it drives a real external nats-server binary (MQTT
// 3.1.1 + JetStream) and a real dsx-lab (built fresh from this repo) over
// loopback TCP on randomly reserved ports, with no container runtime
// involved at all. It is gated behind the same "integration" build tag,
// plus an explicit DSX_NATS_SERVER opt-in (skipped, not failed, when
// unset), so neither the default `go test ./...` suite nor
// `go test -tags integration ./integration/...` without that variable set
// requires an external nats-server binary:
//
//	DSX_NATS_SERVER=/path/to/nats-server go test -tags integration ./integration -run '^TestNative'
//
// Every subprocess this file starts is test-owned, bounded, and stopped in
// t.Cleanup regardless of test outcome; nothing here is a daemon the
// operator relies on, and no secret file content is ever read into the
// test process or logged, only file paths and HTTP status/counts.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// nativeProcessTimeout bounds the lifetime of every subprocess started
// here (nats-server, dsx-lab serve/scenario/init): nothing is allowed to
// run unbounded even if an assertion fails partway through the test.
const nativeProcessTimeout = 240 * time.Second

// nativeNATSServerPath returns the external nats-server binary configured
// via DSX_NATS_SERVER, skipping (not failing) the test when it is unset.
func nativeNATSServerPath(t *testing.T) string {
	t.Helper()
	path := os.Getenv("DSX_NATS_SERVER")
	if path == "" {
		t.Skip("DSX_NATS_SERVER not set; skipping native (non-Docker) broker tests")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("DSX_NATS_SERVER %q: %v", path, err)
	}
	return path
}

// buildNativeDsxLab builds a fresh dsx-lab binary from this repo's
// cmd/dsx-lab into t.TempDir(), so the test exercises the code under test
// rather than whatever binary happens to be on PATH.
func buildNativeDsxLab(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "dsx-lab")
	ctx, cancel := context.WithTimeout(context.Background(), nativeProcessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./cmd/dsx-lab")
	cmd.Dir = ".."
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building dsx-lab: %v\n%s", err, combined)
	}
	return out
}

// reservePort asks the OS for an unused loopback TCP port by briefly
// listening then closing it immediately. A race with another process
// grabbing the same port before our subprocess binds it is an accepted,
// bounded risk here: this is a local, single-purpose test lab, not a
// public-facing allocator.
func reservePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// filteredBaseEnv returns the current process environment with any
// existing DSX_-prefixed variable stripped out, so a leftover DSX_*
// export in the invoking shell can never leak into these test-owned
// subprocesses; only the values explicitly set below are ever used.
func filteredBaseEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if strings.HasPrefix(kv, "DSX_") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// nativeProcess wraps a started subprocess whose stdout/stderr are
// captured to its own on-disk log file (never an in-memory buffer read
// concurrently with the writing side), readable after the process stops
// for failure diagnostics. Logs are dynamic process output, not secret
// file content, and are never dumped except on test failure.
type nativeProcess struct {
	cmd     *exec.Cmd
	logPath string
}

func startNativeProcess(t *testing.T, ctx context.Context, name, path string, args, env []string) *nativeProcess {
	t.Helper()
	logFile, err := os.CreateTemp(t.TempDir(), name+"-*.log")
	if err != nil {
		t.Fatalf("creating log file for %s: %v", name, err)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = env
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		t.Fatalf("starting %s: %v", name, err)
	}
	return &nativeProcess{cmd: cmd, logPath: logFile.Name()}
}

// stop sends SIGTERM and waits briefly, falling back to Kill so a
// slow/hung subprocess never leaks past the test. It is safe to call more
// than once.
func (p *nativeProcess) stop() {
	if p == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_ = p.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		<-done
	}
}

func (p *nativeProcess) log() string {
	data, err := os.ReadFile(p.logPath)
	if err != nil {
		return fmt.Sprintf("(could not read log: %v)", err)
	}
	return string(data)
}

// waitForHTTPStatus polls url (via GET, ignoring connection errors, which
// are expected while a server is still starting or down) until it returns
// want or the deadline passes. It never busy-spins: each retry waits for
// a 100ms ticker, not a tight loop.
func waitForHTTPStatus(url string, want int, deadline time.Time) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if resp, err := client.Get(url); err == nil {
			resp.Body.Close()
			if resp.StatusCode == want {
				return true
			}
		}
		if time.Now().After(deadline) {
			return false
		}
		<-ticker.C
	}
}

// nativeHealthzCounts is the minimal shape this file reads from the
// bridge's /healthz response.
type nativeHealthzCounts struct {
	Accepted     int64 `json:"accepted"`
	Rejected     int64 `json:"rejected"`
	Duplicate    int64 `json:"duplicate"`
	QueueDropped int64 `json:"queue_dropped"`
	Expired      int64 `json:"expired"`
}

func fetchNativeHealthzCounts(t *testing.T, url string) nativeHealthzCounts {
	t.Helper()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var body struct {
		Counts nativeHealthzCounts `json:"counts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decoding healthz response from %s: %v", url, err)
	}
	return body.Counts
}

// runNativeScenario runs one bounded `dsx-lab scenario` invocation and
// returns its combined stdout/stderr and exit error.
func runNativeScenario(dsxLabPath string, env []string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, dsxLabPath, append([]string{"scenario"}, args...)...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestNativeBrokerBridgeScenario brings up a real nats-server and a
// freshly built dsx-lab bridge directly as test-owned subprocesses (no
// Docker), confirms the bridge only reports /readyz once a real SUBACK
// has been observed, that signed accept/release commands flow end to end
// over SSE, that an unauthenticated subscriber is rejected, that a
// policy-rejected command is counted as rejected (never accepted, never
// delivered), and that a SIGTERM broker restart against the same
// JetStream store drops then restores readiness and resumed delivery.
func TestNativeBrokerBridgeScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping native broker/bridge integration test in -short mode")
	}
	natsServerPath := nativeNATSServerPath(t)
	dsxLabPath := buildNativeDsxLab(t)
	baseEnv := filteredBaseEnv()

	root := t.TempDir()
	secretsDir := filepath.Join(root, "secrets")
	dataDir := filepath.Join(root, "data")
	natsConfPath := filepath.Join(secretsDir, "nats.conf")

	mqttPort := reservePort(t)
	monitorPort := reservePort(t)
	clientPort := reservePort(t)
	bridgePort := reservePort(t)

	const (
		signerKID = "dsx-native-test-signer"
		topic     = "grid/v1/isv/gridmetry/loadtarget/set"
	)

	initCtx, initCancel := context.WithTimeout(context.Background(), nativeProcessTimeout)
	defer initCancel()
	initCmd := exec.CommandContext(initCtx, dsxLabPath, "init")
	initCmd.Env = append(append([]string{}, baseEnv...),
		"DSX_SECRETS_DIR="+secretsDir,
		"DSX_SIGNER_KID="+signerKID,
		"DSX_MQTT_TOPIC="+topic,
		"DSX_PUBLISHER_USER=gridmetry",
		"DSX_BRIDGE_USER=bridge",
		"DSX_MQTT_LISTEN_ADDR=127.0.0.1:"+strconv.Itoa(mqttPort),
		"DSX_HTTP_MONITOR_ADDR=127.0.0.1:"+strconv.Itoa(monitorPort),
		"DSX_JETSTREAM_DIR="+dataDir,
		"DSX_OWNER_UID="+strconv.Itoa(os.Getuid()),
		"DSX_OWNER_GID="+strconv.Itoa(os.Getgid()),
	)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("dsx-lab init failed: %v\n%s", err, out)
	}

	brokerCtx, brokerCancel := context.WithTimeout(context.Background(), nativeProcessTimeout)
	defer brokerCancel()
	broker := startNativeProcess(t, brokerCtx, "broker", natsServerPath,
		[]string{"-c", natsConfPath, "-a", "127.0.0.1", "-p", strconv.Itoa(clientPort)}, baseEnv)
	brokerStopped := false
	t.Cleanup(func() {
		if !brokerStopped {
			broker.stop()
		}
	})

	monitorHealthzURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", monitorPort)
	if !waitForHTTPStatus(monitorHealthzURL, http.StatusOK, time.Now().Add(20*time.Second)) {
		t.Fatalf("broker monitor endpoint did not become healthy in time; broker log:\n%s", broker.log())
	}

	bridgeEnv := append(append([]string{}, baseEnv...),
		"DSX_MQTT_BROKER_URL=tcp://127.0.0.1:"+strconv.Itoa(mqttPort),
		"DSX_MQTT_CLIENT_ID=dsx-bridge-native-test",
		"DSX_MQTT_TOPIC="+topic,
		"DSX_MQTT_CREDENTIALS_PATH="+filepath.Join(secretsDir, "mqtt-bridge-credentials.json"),
		"DSX_FEED_TAG=main-a",
		"DSX_MAX_WATTS=1000000",
		"DSX_MAX_EVENT_AGE=5m",
		"DSX_MAX_CLOCK_SKEW=30s",
		"DSX_SIGNER_KID="+signerKID,
		"DSX_SIGNER_SOURCE=//grid/v1/isv/gridmetry",
		"DSX_SIGNING_PUBLIC_KEY_PATH="+filepath.Join(secretsDir, "signing-public.pem"),
		"DSX_SSE_TOKEN_PATH="+filepath.Join(secretsDir, "sse-token"),
		"DSX_HTTP_ADDR=127.0.0.1:"+strconv.Itoa(bridgePort),
		"DSX_CORS_ALLOW_ORIGIN=",
	)

	bridgeCtx, bridgeCancel := context.WithTimeout(context.Background(), nativeProcessTimeout)
	defer bridgeCancel()
	bridge := startNativeProcess(t, bridgeCtx, "bridge", dsxLabPath, []string{"serve"}, bridgeEnv)
	t.Cleanup(bridge.stop)

	readyzURL := fmt.Sprintf("http://127.0.0.1:%d/readyz", bridgePort)
	healthzURL := fmt.Sprintf("http://127.0.0.1:%d/healthz", bridgePort)
	eventsURL := fmt.Sprintf("http://127.0.0.1:%d/events", bridgePort)

	if !waitForHTTPStatus(readyzURL, http.StatusOK, time.Now().Add(20*time.Second)) {
		t.Fatalf("bridge did not become ready in time; bridge log:\n%s\nbroker log:\n%s", bridge.log(), broker.log())
	}

	scenarioEnv := append(append([]string{}, baseEnv...),
		"DSX_EVENTS_URL="+eventsURL,
		"DSX_MQTT_BROKER_URL=tcp://127.0.0.1:"+strconv.Itoa(mqttPort),
		"DSX_MQTT_TOPIC="+topic,
		"DSX_SIGNER_KID="+signerKID,
		"DSX_SECRETS_DIR="+secretsDir,
	)

	if out, err := runNativeScenario(dsxLabPath, scenarioEnv, 15*time.Second); err != nil {
		t.Fatalf("cap scenario failed: %v\n%s", err, out)
	} else if !strings.Contains(out, "received matching normalized event") {
		t.Fatalf("cap scenario did not report a matching received event; output:\n%s", out)
	}

	if out, err := runNativeScenario(dsxLabPath, scenarioEnv, 15*time.Second, "--release"); err != nil {
		t.Fatalf("release scenario failed: %v\n%s", err, out)
	} else if !strings.Contains(out, "received matching normalized event") {
		t.Fatalf("release scenario did not report a matching received event; output:\n%s", out)
	}

	// An unauthenticated subscriber must be rejected, never silently served.
	resp, err := http.Get(eventsURL)
	if err != nil {
		t.Fatalf("unauthenticated GET %s: %v", eventsURL, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET %s = %d, want %d", eventsURL, resp.StatusCode, http.StatusUnauthorized)
	}

	// A zero-watt constraint is neither a positive bounded value nor an
	// explicit release, so lab policy must reject it: it is counted as
	// rejected, never accepted, and never surfaces as a matching SSE
	// event (the short --timeout keeps this bounded rather than waiting
	// out the scenario's full default timeout for an event that will
	// never arrive).
	before := fetchNativeHealthzCounts(t, healthzURL)
	out, err := runNativeScenario(dsxLabPath, scenarioEnv, 10*time.Second, "--watts", "0", "--timeout", "1s")
	if err == nil {
		t.Fatalf("expected the zero-watts scenario to fail (rejected, never delivered); output:\n%s", out)
	}
	after := fetchNativeHealthzCounts(t, healthzURL)
	if after.Rejected != before.Rejected+1 {
		t.Fatalf("rejected count = %d, want %d (was %d); scenario output:\n%s", after.Rejected, before.Rejected+1, before.Rejected, out)
	}
	if after.Accepted != before.Accepted {
		t.Fatalf("accepted count changed from %d to %d on a rejected command; scenario output:\n%s", before.Accepted, after.Accepted, out)
	}

	// Stop the broker (SIGTERM, as an operator restart would) and confirm
	// readiness honestly drops within a bounded poll.
	broker.stop()
	brokerStopped = true
	if !waitForHTTPStatus(readyzURL, http.StatusServiceUnavailable, time.Now().Add(10*time.Second)) {
		t.Fatalf("bridge /readyz did not drop to 503 after the broker stopped; bridge log:\n%s", bridge.log())
	}

	// Restart the broker against the same config/JetStream store and
	// confirm the bridge's bounded reconnect/resubscribe brings readiness
	// back without restarting the bridge itself.
	broker2Ctx, broker2Cancel := context.WithTimeout(context.Background(), nativeProcessTimeout)
	defer broker2Cancel()
	broker2 := startNativeProcess(t, broker2Ctx, "broker-restart", natsServerPath,
		[]string{"-c", natsConfPath, "-a", "127.0.0.1", "-p", strconv.Itoa(clientPort)}, baseEnv)
	t.Cleanup(broker2.stop)

	if !waitForHTTPStatus(readyzURL, http.StatusOK, time.Now().Add(20*time.Second)) {
		t.Fatalf("bridge /readyz did not return to 200 after the broker restarted; bridge log:\n%s\nbroker log:\n%s",
			bridge.log(), broker2.log())
	}

	if out, err := runNativeScenario(dsxLabPath, scenarioEnv, 15*time.Second); err != nil {
		t.Fatalf("post-restart cap scenario failed: %v\n%s", err, out)
	} else if !strings.Contains(out, "received matching normalized event") {
		t.Fatalf("post-restart cap scenario did not report a matching received event; output:\n%s", out)
	}
	if out, err := runNativeScenario(dsxLabPath, scenarioEnv, 15*time.Second, "--release"); err != nil {
		t.Fatalf("post-restart release scenario failed: %v\n%s", err, out)
	} else if !strings.Contains(out, "received matching normalized event") {
		t.Fatalf("post-restart release scenario did not report a matching received event; output:\n%s", out)
	}
}
