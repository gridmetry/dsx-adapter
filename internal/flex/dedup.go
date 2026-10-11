package flex

import (
	"sync"
	"time"
)

// DedupOutcome classifies one dedup lookup/insert.
type DedupOutcome int

const (
	// DedupNew means this (scope, id) has not been seen within the
	// retention window; the caller should process the event normally.
	DedupNew DedupOutcome = iota
	// DedupDuplicate means the identical fingerprint was already recorded
	// for this (scope, id); the caller should drop it silently (but count
	// it) rather than reprocess it.
	DedupDuplicate
	// DedupConflict means this id was already recorded for this scope with
	// a *different* fingerprint. The event is rejected, not silently
	// applied over the earlier one.
	DedupConflict
)

type dedupRecord struct {
	scope       string
	id          string
	fingerprint string
	expiresAt   time.Time
}

// Dedup is a bounded, in-memory, best-effort duplicate suppression ring.
// It is NOT a cryptographic replay-protection mechanism: the CloudEvents
// "id" and "time" fields are outside the JWS signature, so an attacker
// controlling only those fields (not the signed payload) could in
// principle vary them to defeat id-based dedup; defense against that
// requires a durably signed freshness/nonce contract, which is out of
// scope for this lab profile. State is memory-only and does not survive a
// process restart.
type Dedup struct {
	mu      sync.Mutex
	cap     int
	ttl     time.Duration
	entries map[string]*dedupRecord // key: scope+"\x00"+id
	ring    []string                // bounded FIFO of keys, for eviction
	next    int
	filled  bool
}

// NewDedup creates a dedup cache bounded to capacity entries, each expiring
// after ttl.
func NewDedup(capacity int, ttl time.Duration) *Dedup {
	if capacity < 1 {
		capacity = 1
	}
	return &Dedup{
		cap:     capacity,
		ttl:     ttl,
		entries: make(map[string]*dedupRecord, capacity),
		ring:    make([]string, capacity),
	}
}

func dedupKey(scope, id string) string {
	return scope + "\x00" + id
}

// Check records (scope, id, fingerprint) at time now if new, and reports
// whether it was new, an exact duplicate, or a conflicting reuse of id with
// a different fingerprint. Expired entries are treated as new (and
// overwritten), documenting that this is a transient suppression window,
// not durable replay protection.
func (d *Dedup) Check(scope, id, fingerprint string, now time.Time) DedupOutcome {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := dedupKey(scope, id)
	if rec, ok := d.entries[key]; ok && now.Before(rec.expiresAt) {
		if rec.fingerprint == fingerprint {
			return DedupDuplicate
		}
		return DedupConflict
	}

	d.insertLocked(key, scope, id, fingerprint, now)
	return DedupNew
}

func (d *Dedup) insertLocked(key, scope, id, fingerprint string, now time.Time) {
	if d.filled {
		evictKey := d.ring[d.next]
		delete(d.entries, evictKey)
	}
	d.ring[d.next] = key
	d.entries[key] = &dedupRecord{
		scope:       scope,
		id:          id,
		fingerprint: fingerprint,
		expiresAt:   now.Add(d.ttl),
	}
	d.next++
	if d.next >= d.cap {
		d.next = 0
		d.filled = true
	}
}
