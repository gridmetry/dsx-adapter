// Package transport provides the authenticated local HTTP event stream
// (Server-Sent Events) that Phase 2 exposes instead of a browser SDK
// client. It reports message receipt, not DSX execution or Omniverse
// state.
package transport

import (
	"sync"
)

// Hub fans out normalized event bytes to bounded per-subscriber buffers. A
// slow subscriber that cannot keep up is disconnected rather than allowed
// to block publishers or grow memory unboundedly.
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan []byte]struct{}
	bufferSize  int
}

// NewHub creates a Hub whose subscriber channels each hold up to
// bufferSize pending events before the subscriber is dropped.
func NewHub(bufferSize int) *Hub {
	if bufferSize < 1 {
		bufferSize = 1
	}
	return &Hub{subscribers: make(map[chan []byte]struct{}), bufferSize: bufferSize}
}

// Subscribe registers a new subscriber and returns its channel plus a
// cancel function the caller must call exactly once when done.
func (h *Hub) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, h.bufferSize)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish sends data to every current subscriber without blocking. A
// subscriber whose buffer is full is dropped (its channel closed and
// removed) rather than stalling delivery to others; this trades guaranteed
// delivery for bounded memory and liveness, and is documented as such.
func (h *Hub) Publish(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- data:
		default:
			delete(h.subscribers, ch)
			close(ch)
		}
	}
}

// SubscriberCount reports the current number of connected subscribers, for
// health/diagnostic reporting only.
func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}
