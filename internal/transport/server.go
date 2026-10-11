package transport

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"gridmetry/dsx-adapter/internal/lab"
)

// Server is the authenticated local HTTP surface: GET /events (SSE), plus
// /healthz and /readyz for Compose/operator probing. It never accepts a
// token via query string and never logs the Authorization header or token
// value.
type Server struct {
	Hub               *Hub
	Token             string
	AllowOrigin       string
	Ready             func() bool
	Stats             *lab.Stats
	HeartbeatInterval time.Duration
	// Now, if set, is used instead of time.Now for the pre-write end_time
	// expiry check, so tests can inject a deterministic clock.
	Now func() time.Time
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Mux builds the HTTP handler. HeartbeatInterval defaults to 15s if zero.
func (s *Server) Mux() http.Handler {
	heartbeat := s.HeartbeatInterval
	if heartbeat <= 0 {
		heartbeat = 15 * time.Second
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", s.handleEvents(heartbeat))
	mux.HandleFunc("OPTIONS /events", s.handleEventsPreflight)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	return mux
}

// handleEventsPreflight answers a browser CORS preflight for GET /events
// with Authorization. It never requires/accepts a bearer token itself
// (preflights are credential-less); only an exact match against the one
// configured AllowOrigin is granted, never a wildcard.
func (s *Server) handleEventsPreflight(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" || s.AllowOrigin == "" || origin != s.AllowOrigin {
		http.Error(w, `{"error":"origin not allowed"}`, http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) applyCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" || s.AllowOrigin == "" || origin != s.AllowOrigin {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
}

// eventEndTime is the minimal shape transport needs from a normalized
// event's JSON encoding; it intentionally does not depend on internal/flex.
type eventEndTime struct {
	EndTime *time.Time `json:"end_time"`
}

// eventExpired reports whether data's end_time (if any) is not after now.
// A missing/unparseable end_time never expires.
func eventExpired(data []byte, now time.Time) bool {
	var ev eventEndTime
	if err := json.Unmarshal(data, &ev); err != nil || ev.EndTime == nil {
		return false
	}
	return !ev.EndTime.After(now)
}

func constantTimeTokenEqual(provided, expected string) bool {
	a := sha256.Sum256([]byte(provided))
	b := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

const bearerPrefix = "Bearer "

func (s *Server) authorized(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	if len(h) <= len(bearerPrefix) || h[:len(bearerPrefix)] != bearerPrefix {
		return false
	}
	return constantTimeTokenEqual(h[len(bearerPrefix):], s.Token)
}

func (s *Server) handleEvents(heartbeat time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.applyCORS(w, r)
		if !s.authorized(r) {
			// Never echo the attempted token or the request URL/headers.
			log.Printf("transport: /events unauthorized request from %s", r.RemoteAddr)
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		rc := http.NewResponseController(w)
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)

		ch, cancel := s.Hub.Subscribe()
		defer cancel()

		if _, err := w.Write([]byte(": connected\n\n")); err != nil {
			return
		}
		if err := rc.Flush(); err != nil {
			return
		}

		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case data, open := <-ch:
				if !open {
					return
				}
				// Re-check end_time immediately before writing: a slow
				// subscriber's buffered backlog can hold an event past its
				// end_time even though it was still valid when published.
				if eventExpired(data, s.now()) {
					if s.Stats != nil {
						s.Stats.IncExpired()
					}
					continue
				}
				if _, err := w.Write([]byte("data: ")); err != nil {
					return
				}
				if _, err := w.Write(data); err != nil {
					return
				}
				if _, err := w.Write([]byte("\n\n")); err != nil {
					return
				}
				if err := rc.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
					return
				}
				if err := rc.Flush(); err != nil {
					return
				}
			}
		}
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	s.applyCORS(w, r)
	w.Header().Set("Content-Type", "application/json")
	body := map[string]any{"status": "ok"}
	if s.Stats != nil {
		body["counts"] = s.Stats.Snapshot()
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	s.applyCORS(w, r)
	w.Header().Set("Content-Type", "application/json")
	ready := s.Ready != nil && s.Ready()
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "not_ready"})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ready"})
}
