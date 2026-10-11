//go:build integration

package integration

import (
	"strings"
	"testing"
	"time"
)

// TestBridgeReadinessDropsAndReturnsAcrossBrokerRestart stops and
// restarts the NATS MQTT broker container under its own isolated Compose
// project and asserts the bridge's /readyz (checked directly from inside
// its own container, not inferred from Compose's health-string polling
// interval) honestly drops during the outage and only returns once a
// fresh SUBACK confirms the resubscription, then that a signed command
// published after the restart still flows through end to end. It is
// authored to document the expected reconnect/readiness contract; like
// the rest of this file's suite it requires a working Docker runtime and
// is not claimed to have run live in every environment (see README's
// "Observed results").
func TestBridgeReadinessDropsAndReturnsAcrossBrokerRestart(t *testing.T) {
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
		t.Fatalf("bridge did not become ready before the broker restart; logs:\n%s", out)
	}

	if out, err := runCompose(t, project, "stop", "broker"); err != nil {
		t.Fatalf("compose stop broker failed: %v\n%s", out, err)
	}

	if !waitForBridgeNotReady(t, project, "bridge", 30*time.Second) {
		t.Fatal("expected the bridge's /readyz to drop while the broker is stopped")
	}

	if out, err := runCompose(t, project, "start", "broker"); err != nil {
		t.Fatalf("compose start broker failed: %v\n%s", out, err)
	}
	if !waitForBridgeReady(t, project, 60*time.Second) {
		out, _ := runCompose(t, project, "logs", "bridge")
		t.Fatalf("bridge did not become ready again after resubscribing post-restart; logs:\n%s", out)
	}

	out, err := runCompose(t, project, "run", "--rm", "scenario")
	if err != nil {
		t.Fatalf("post-reconnect scenario failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "received matching normalized event") {
		t.Fatalf("post-reconnect scenario did not report a matching received event; output:\n%s", out)
	}
}

// waitForBridgeNotReady is the inverse of waitForBridgeReady: it returns
// once the bridge's /readyz check starts failing (broker down / not yet
// resubscribed), observed directly rather than inferred from a Docker
// healthcheck interval.
func waitForBridgeNotReady(t *testing.T, project, service string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := runCompose(t, project, "exec", "-T", service,
			"/usr/local/bin/dsx-lab", "check", "--url", "http://localhost:8090/readyz"); err != nil {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
