//go:build integration

// Package integration brings up the Compose stack under an isolated
// project name/volumes and drives it through the real NATS MQTT 3.1.1
// broker and the real HTTP SSE endpoint. These tests are gated behind the
// "integration" build tag so `go test ./...` (the default, Docker-free
// suite) never requires Docker:
//
//	go test -tags integration ./integration/...
//
// Each test uses its own `-p` project name, applies override.yaml (which
// removes the demo's fixed host port bindings), and tears its stack down
// (`down -v`) on completion. The scenario/healthcheck driver only needs
// the project's internal Docker network (service names "bridge"/
// "broker"), so this never collides with a human's own
// `docker compose up -d` demo stack on 127.0.0.1:8090/1883.
package integration

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// composeTimeout bounds a single docker compose invocation. It is 3
// minutes rather than the usual short test bound because "up --build"
// can exceed 120s on a cold image build; composeCleanupTimeout is used
// instead for teardown, which never builds.
const (
	composeTimeout        = 3 * time.Minute
	composeCleanupTimeout = 15 * time.Second
)

func randomProjectName(t *testing.T) string {
	t.Helper()
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("generating random project suffix: %v", err)
	}
	return "dsxlab-it-" + hex.EncodeToString(b[:])
}

func runCompose(t *testing.T, project string, args ...string) (string, error) {
	t.Helper()
	return runComposeBounded(t, composeTimeout, project, args...)
}

// runComposeCleanup is the bounded variant used from t.Cleanup, where a
// hung teardown must not be allowed to stall the whole suite.
func runComposeCleanup(t *testing.T, project string, args ...string) (string, error) {
	t.Helper()
	return runComposeBounded(t, composeCleanupTimeout, project, args...)
}

func runComposeBounded(t *testing.T, timeout time.Duration, project string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	base := []string{"compose", "-p", project, "-f", "compose.yaml", "-f", "integration/override.yaml"}
	cmd := exec.CommandContext(ctx, "docker", append(base, args...)...)
	cmd.Dir = ".."
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// TestScenarioAcceptsOneSignedCommand brings up an isolated stack, runs the
// scenario container (which itself subscribes, publishes one signed
// command, and verifies receipt), and asserts the scenario reported
// success. This is the Phase 2 MVP acceptance check: a signed
// grid.loadtarget.set.v1 command flows from MQTT through the bridge to
// the authenticated SSE stream.
func TestScenarioAcceptsOneSignedCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker-backed integration test in -short mode")
	}
	project := randomProjectName(t)
	t.Cleanup(func() {
		if _, err := runComposeCleanup(t, project, "down", "-v", "--remove-orphans"); err != nil {
			t.Logf("cleanup: docker compose down for project %s: %v", project, err)
		}
	})

	if out, err := runCompose(t, project, "run", "--rm", "init"); err != nil {
		t.Fatalf("compose run init failed: %v\n%s", err, out)
	}

	if out, err := runCompose(t, project, "up", "-d", "--build", "broker", "bridge"); err != nil {
		t.Fatalf("compose up broker/bridge failed: %v\n%s", out, err)
	}

	if !waitForBridgeReady(t, project, 60*time.Second) {
		out, _ := runCompose(t, project, "logs", "bridge")
		t.Fatalf("bridge did not become ready in time; logs:\n%s", out)
	}

	out, err := runCompose(t, project, "run", "--rm", "scenario")
	if err != nil {
		t.Fatalf("scenario failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "received matching normalized event") {
		t.Fatalf("scenario did not report a matching received event; output:\n%s", out)
	}
	t.Logf("scenario output:\n%s", out)
}

func TestInteractiveScenarioAcceptsMultipleSignedCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker-backed integration test in -short mode")
	}
	project := randomProjectName(t)
	t.Cleanup(func() {
		if _, err := runComposeCleanup(t, project, "down", "-v", "--remove-orphans"); err != nil {
			t.Logf("cleanup for %s: %v", project, err)
		}
	})
	if output, err := runCompose(t, project, "run", "--rm", "--build", "init"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, output)
	}
	if output, err := runCompose(t, project, "up", "-d", "--build", "broker", "bridge"); err != nil {
		t.Fatalf("stack failed: %v\n%s", err, output)
	}
	if !waitForBridgeReady(t, project, time.Minute) {
		t.Fatal("test-owned bridge did not become ready")
	}
	ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
	defer cancel()
	args := []string{"compose", "-p", project, "-f", "compose.yaml", "-f", "integration/override.yaml", "run", "--rm", "-T", "--no-deps", "--build", "scenario", "--interactive", "--duration", "30s"}
	command := exec.CommandContext(ctx, "docker", args...)
	command.Dir = ".."
	command.Stdin = strings.NewReader("920000\nnot-a-command\n650000\n800000\nrelease\nquit\n")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		t.Fatalf("interactive scenario failed: %v\n%s", err, output.String())
	}
	text := output.String()
	if strings.Count(text, "MQTT connected") != 1 || strings.Count(text, "subscribed to event stream, ready") != 1 {
		t.Fatalf("connections were not reused:\n%s", text)
	}
	const prefix = "received matching normalized event: "
	ids := make(map[string]bool)
	var receipts []struct {
		ID      string   `json:"id"`
		Watts   *float64 `json:"constraint_watts"`
		Release bool     `json:"release"`
	}
	for _, line := range strings.Split(text, "\n") {
		index := strings.Index(line, prefix)
		if index < 0 {
			continue
		}
		var receipt struct {
			ID      string   `json:"id"`
			Watts   *float64 `json:"constraint_watts"`
			Release bool     `json:"release"`
		}
		if err := json.Unmarshal([]byte(line[index+len(prefix):]), &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.ID == "" || ids[receipt.ID] {
			t.Fatal("command identifiers were reused")
		}
		ids[receipt.ID] = true
		receipts = append(receipts, receipt)
	}
	if len(receipts) != 4 || !receipts[3].Release || receipts[3].Watts != nil {
		t.Fatalf("expected three targets and one release:\n%s", text)
	}
	for index, expected := range []float64{920000, 650000, 800000} {
		if receipts[index].Watts == nil || *receipts[index].Watts != expected || receipts[index].Release {
			t.Fatal("unexpected target receipt")
		}
	}
	if strings.Contains(text, "not-a-command") {
		t.Fatal("untrusted terminal input was echoed")
	}
	t.Log("three signed targets and one release verified with one MQTT/SSE session")
}

func TestStagedInteractiveScenarioAcceptsFullLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker-backed integration test in -short mode")
	}
	project := randomProjectName(t)
	execute := func(input string, args ...string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), composeTimeout)
		defer cancel()
		base := []string{"compose", "-p", project, "-f", "compose.yaml", "-f", "compose.stage.yaml", "-f", "integration/override.yaml"}
		command := exec.CommandContext(ctx, "docker", append(base, args...)...)
		command.Dir = ".."
		command.Stdin = strings.NewReader(input)
		output, err := command.CombinedOutput()
		return string(output), err
	}
	t.Cleanup(func() {
		if _, err := runComposeCleanup(t, project, "down", "-v", "--remove-orphans"); err != nil {
			t.Logf("cleanup for %s: %v", project, err)
		}
	})
	if output, err := execute("", "run", "--rm", "--build", "init"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, output)
	}
	if output, err := execute("", "up", "-d", "--build", "broker", "bridge"); err != nil {
		t.Fatalf("stack failed: %v\n%s", err, output)
	}
	if !waitForBridgeReady(t, project, time.Minute) {
		t.Fatal("test-owned staged bridge did not become ready")
	}
	output, err := execute("1316000\nrelease\nquit\n", "run", "--rm", "-T", "--no-deps", "--build", "scenario", "--interactive", "--duration", "30s")
	if err != nil {
		t.Fatalf("full-load publisher failed: %v\n%s", err, output)
	}
	var loads []int
	var powers []int
	for _, line := range strings.Split(output, "\n") {
		const prefix = "received matching normalized event: "
		index := strings.Index(line, prefix)
		if index < 0 {
			continue
		}
		var receipt struct {
			Visualization struct {
				LoadPercent int    `json:"load_percent"`
				PowerWatts  int    `json:"displayed_power_watts"`
				EDPSetting  string `json:"edp_setting"`
			} `json:"visualization"`
		}
		if err := json.Unmarshal([]byte(line[index+len(prefix):]), &receipt); err != nil || receipt.Visualization.EDPSetting != "1.2" {
			t.Fatal("incorrect staged profile receipt")
		}
		loads = append(loads, receipt.Visualization.LoadPercent)
		powers = append(powers, receipt.Visualization.PowerWatts)
	}
	if len(loads) != 2 || loads[0] != 100 || powers[0] != 1316000 || loads[1] != 80 || powers[1] != 1052000 {
		t.Fatalf("full-load and release mapping incorrect:\n%s", output)
	}
	t.Log("staged EDP 1.2 full-load target and baseline release verified")
}

// composePort mirrors the subset of `docker compose config --format
// json`'s per-service port entry this test needs; unused fields are
// omitted.
type composePort struct {
	Target    int    `json:"target"`
	Published string `json:"published"`
}

type composeConfigOutput struct {
	Services map[string]struct {
		Ports []composePort `json:"ports"`
	} `json:"services"`
}

// TestComposeOverrideClearsPublishedPorts guards against override.yaml's
// `ports: []` silently failing to clear the base compose file's fixed
// host bindings: Compose's default list-merge behavior appends rather
// than replaces, so an empty override list is a no-op and the base
// 127.0.0.1:1883/8090 bindings leak through. This only renders the
// merged config (no containers start), so it never touches a human's own
// `docker compose up -d` demo stack.
func TestComposeOverrideClearsPublishedPorts(t *testing.T) {
	project := randomProjectName(t)
	out, err := runCompose(t, project, "config", "--format", "json")
	if err != nil {
		t.Fatalf("compose config failed: %v\n%s", err, out)
	}

	var cfg composeConfigOutput
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("parsing compose config json: %v", err)
	}

	for _, name := range []string{"broker", "bridge"} {
		svc, ok := cfg.Services[name]
		if !ok {
			t.Fatalf("service %q missing from compose config output", name)
		}
		if len(svc.Ports) != 0 {
			t.Fatalf("service %q still has %d published port(s) after override, want 0", name, len(svc.Ports))
		}
	}
}

// waitForBridgeReady and waitForBridgeNotReady poll the bridge's actual
// /readyz endpoint from inside its own container (via the same bounded
// check binary its Compose healthcheck uses), rather than parsing docker
// compose ps's reported health string: that string's "healthy"/"unhealthy"
// values both contain the substring "healthy", so a naive
// strings.Contains(out, "healthy") check is satisfied by either state and
// also lags behind the healthcheck's own interval/retries. Polling
// /readyz directly observes the real state immediately.
func waitForBridgeReady(t *testing.T, project string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := runCompose(t, project, "exec", "-T", "bridge",
			"/usr/local/bin/dsx-lab", "check", "--url", "http://localhost:8090/readyz"); err == nil {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func init() {
	// Fail fast with a clear message if docker is not on PATH, rather than
	// a confusing exec error deep in the first test.
	if _, err := exec.LookPath("docker"); err != nil {
		panic(fmt.Sprintf("integration tests require docker on PATH: %v", err))
	}
}
