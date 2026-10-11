package transport

import (
	"testing"
	"time"
)

func TestHubDeliversToSubscriber(t *testing.T) {
	h := NewHub(4)
	ch, cancel := h.Subscribe()
	defer cancel()

	h.Publish([]byte("hello"))

	select {
	case got := <-ch:
		if string(got) != "hello" {
			t.Fatalf("got %q, want hello", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestHubFanOutToMultipleSubscribers(t *testing.T) {
	h := NewHub(4)
	ch1, cancel1 := h.Subscribe()
	defer cancel1()
	ch2, cancel2 := h.Subscribe()
	defer cancel2()

	h.Publish([]byte("x"))

	for _, ch := range []<-chan []byte{ch1, ch2} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for fan-out delivery")
		}
	}
}

func TestHubDropsSlowSubscriberInsteadOfBlocking(t *testing.T) {
	h := NewHub(1)
	ch, cancel := h.Subscribe()
	defer cancel()

	// Fill the bounded buffer, then publish again without ever draining ch:
	// Publish must not block, and the slow subscriber must be disconnected.
	h.Publish([]byte("first"))
	done := make(chan struct{})
	go func() {
		h.Publish([]byte("second"))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}

	if _, open := <-ch; open {
		// Drain the one buffered message; channel should then be closed.
		if _, open := <-ch; open {
			t.Fatal("expected slow subscriber channel to be closed after drop")
		}
	}
}

func TestHubCancelRemovesSubscriber(t *testing.T) {
	h := NewHub(4)
	_, cancel := h.Subscribe()
	if h.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber, got %d", h.SubscriberCount())
	}
	cancel()
	if h.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after cancel, got %d", h.SubscriberCount())
	}
}
