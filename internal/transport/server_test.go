package transport

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"gridmetry/dsx-adapter/internal/lab"
)

func newTestServer(t *testing.T, ready bool) (*Server, *httptest.Server) {
	t.Helper()
	hub := NewHub(4)
	s := &Server{
		Hub:               hub,
		Token:             "test-token-value",
		AllowOrigin:       "https://example.test",
		Ready:             func() bool { return ready },
		HeartbeatInterval: time.Hour, // disabled in practice for these tests
	}
	srv := httptest.NewServer(s.Mux())
	t.Cleanup(srv.Close)
	return s, srv
}

func TestVisualizationAssetIsOptionalAndDoesNotGrantEventAccess(t *testing.T) {
	server := &Server{Hub: NewHub(1), Token: "private-test-token", AllowOrigin: "https://build.nvidia.com"}
	request := httptest.NewRequest(http.MethodGet, "/visualization-client.js", nil)
	missing := httptest.NewRecorder()
	server.Mux().ServeHTTP(missing, request)
	if missing.Code != http.StatusNotFound {
		t.Fatal("default server exposed a visual client")
	}
	server.VisualizationClient = "token-free-client-source"
	request.Header.Set("Origin", server.AllowOrigin)
	response := httptest.NewRecorder()
	server.Mux().ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != server.VisualizationClient ||
		response.Header().Get("Content-Type") != "text/javascript; charset=utf-8" ||
		response.Header().Get("Cache-Control") != "no-store" ||
		response.Header().Get("X-Content-Type-Options") != "nosniff" ||
		response.Header().Get("Access-Control-Allow-Origin") != server.AllowOrigin ||
		strings.Contains(response.Body.String(), server.Token) {
		t.Fatal("unsafe visualization asset response")
	}
	denied := httptest.NewRecorder()
	server.Mux().ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/events", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatal("public script granted authenticated event access")
	}
}

func TestHealthzAlwaysOK(t *testing.T) {
	_, srv := newTestServer(t, false)
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestReadyzReflectsReadiness(t *testing.T) {
	_, srv := newTestServer(t, false)
	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when not ready, got %d", resp.StatusCode)
	}
}

func TestReadyzOKWhenReady(t *testing.T) {
	_, srv := newTestServer(t, true)
	resp, err := http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when ready, got %d", resp.StatusCode)
	}
}

func TestEventsRejectsMissingAuth(t *testing.T) {
	_, srv := newTestServer(t, true)
	resp, err := http.Get(srv.URL + "/events")
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestEventsRejectsWrongToken(t *testing.T) {
	_, srv := newTestServer(t, true)
	req, _ := http.NewRequest("GET", srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestEventsRejectsQueryStringToken(t *testing.T) {
	_, srv := newTestServer(t, true)
	u, _ := url.Parse(srv.URL + "/events")
	q := u.Query()
	q.Set("token", "test-token-value")
	u.RawQuery = q.Encode()
	resp, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for query-string token, got %d", resp.StatusCode)
	}
}

func TestEventsStreamsConnectedThenPublishedEvent(t *testing.T) {
	s, srv := newTestServer(t, true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer test-token-value")
	req.Header.Set("Origin", "https://example.test")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://example.test" {
		t.Fatalf("expected CORS header to echo allowed origin, got %q", got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("unexpected content-type %q", ct)
	}

	reader := bufio.NewReader(resp.Body)
	connectedLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("reading connected comment: %v", err)
	}
	if !strings.HasPrefix(connectedLine, ": connected") {
		t.Fatalf("expected connected comment first, got %q", connectedLine)
	}

	// Wait for the subscription to register, then publish.
	deadline := time.Now().Add(2 * time.Second)
	for s.Hub.SubscriberCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	s.Hub.Publish([]byte(`{"type":"command_received"}`))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading event line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			if !strings.Contains(line, "command_received") {
				t.Fatalf("unexpected event payload: %q", line)
			}
			break
		}
	}
}

func TestEventsDeniesCORSForUnknownOrigin(t *testing.T) {
	_, srv := newTestServer(t, true)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer test-token-value")
	req.Header.Set("Origin", "https://evil.test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for unknown origin, got %q", got)
	}
}

func TestEventsDropsExpiredEventBeforeSSEWrite(t *testing.T) {
	hub := NewHub(4)
	fixedNow := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	stats := &lab.Stats{}
	s := &Server{
		Hub:               hub,
		Token:             "test-token-value",
		AllowOrigin:       "https://example.test",
		Ready:             func() bool { return true },
		Stats:             stats,
		HeartbeatInterval: time.Hour,
		Now:               func() time.Time { return fixedNow },
	}
	srv := httptest.NewServer(s.Mux())
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer test-token-value")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("reading connected comment: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for s.Hub.SubscriberCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Already expired relative to the injected clock: the buffered hub
	// delivery must not write it to the SSE stream.
	hub.Publish([]byte(`{"type":"command_received","id":"expired-1","end_time":"2024-01-01T00:00:00Z"}`))
	// A command with no end_time, or one still in the future, must still
	// be delivered.
	hub.Publish([]byte(`{"type":"command_received","id":"valid-1"}`))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading event line: %v", err)
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		if strings.Contains(line, "expired-1") {
			t.Fatalf("expired event must not be written to the SSE stream: %q", line)
		}
		if strings.Contains(line, "valid-1") {
			break
		}
	}

	if stats.Snapshot().Expired != 1 {
		t.Fatalf("expected 1 expired count, got %+v", stats.Snapshot())
	}
}

func TestEventsPreflightAllowsConfiguredOriginWithoutAuth(t *testing.T) {
	s, srv := newTestServer(t, true)
	req, _ := http.NewRequest("OPTIONS", srv.URL+"/events", nil)
	req.Header.Set("Origin", "https://example.test")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://example.test" {
		t.Fatalf("expected exact-origin echo, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "GET" {
		t.Fatalf("expected Allow-Methods GET, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("expected Allow-Headers to include Authorization, got %q", got)
	}
	if got := resp.Header.Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary: Origin, got %q", got)
	}
	if s.Hub.SubscriberCount() != 0 {
		t.Fatal("preflight must not open an SSE subscription")
	}
}

func TestEventsPreflightRejectsUnknownOrigin(t *testing.T) {
	_, srv := newTestServer(t, true)
	req, _ := http.NewRequest("OPTIONS", srv.URL+"/events", nil)
	req.Header.Set("Origin", "https://evil.test")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /events: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for unknown origin, got %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("must never use a permissive wildcard origin")
	}
	if resp.StatusCode == http.StatusNoContent {
		t.Fatal("expected preflight for an unconfigured origin to be rejected, not 204")
	}
}

func TestEventsPreflightRejectsMissingOrigin(t *testing.T) {
	_, srv := newTestServer(t, true)
	req, _ := http.NewRequest("OPTIONS", srv.URL+"/events", nil)
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /events: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header without an Origin header, got %q", got)
	}
	if resp.StatusCode == http.StatusNoContent {
		t.Fatal("expected preflight without an Origin header to be rejected, not 204")
	}
}

func TestEventsRejectsNonGETWithMethodNotAllowed(t *testing.T) {
	_, srv := newTestServer(t, true)
	req, _ := http.NewRequest("POST", srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer test-token-value")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /events: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for a non-GET/OPTIONS request, got %d", resp.StatusCode)
	}
}
