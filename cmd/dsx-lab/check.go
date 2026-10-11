package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

// runCheck performs one bounded HTTP GET against --url and exits 0 only on
// a 2xx response within --timeout. It is used as the Compose healthcheck
// command for both the NATS broker (its /healthz monitoring endpoint) and
// this bridge (its own /readyz), since the minimal images here do not ship
// curl.
func runCheck(args []string) int {
	fs := flag.NewFlagSet("dsx-lab check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	url := fs.String("url", "", "URL to GET; exit 0 only on a 2xx response (required)")
	timeout := fs.Duration("timeout", 3*time.Second, "bounded request timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *url == "" {
		fmt.Fprintln(os.Stderr, "dsx-lab check: --url is required")
		return 2
	}

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Get(*url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab check: request failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "dsx-lab check: unhealthy status %d\n", resp.StatusCode)
		return 1
	}
	return 0
}
