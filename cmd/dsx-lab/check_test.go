package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunCheckSucceedsOn2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if got := runCheck([]string{"--url", srv.URL}); got != 0 {
		t.Fatalf("expected exit code 0 for a 2xx response, got %d", got)
	}
}

func TestRunCheckFailsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if got := runCheck([]string{"--url", srv.URL}); got != 1 {
		t.Fatalf("expected exit code 1 for a non-2xx response, got %d", got)
	}
}

func TestRunCheckFailsOnUnreachableURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // now guaranteed unreachable

	if got := runCheck([]string{"--url", url, "--timeout", "200ms"}); got != 1 {
		t.Fatalf("expected exit code 1 for an unreachable URL, got %d", got)
	}
}

func TestRunCheckRequiresURL(t *testing.T) {
	if got := runCheck(nil); got != 2 {
		t.Fatalf("expected exit code 2 when --url is missing, got %d", got)
	}
}
