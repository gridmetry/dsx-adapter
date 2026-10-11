package flex

import (
	"testing"
	"time"
)

func TestDedupFirstSeenIsNew(t *testing.T) {
	d := NewDedup(1024, time.Minute)
	if got := d.Check("signer-1", "id-1", "fp-1", time.Now()); got != DedupNew {
		t.Fatalf("expected DedupNew, got %v", got)
	}
}

func TestDedupExactRepeatIsDuplicate(t *testing.T) {
	d := NewDedup(1024, time.Minute)
	now := time.Now()
	d.Check("signer-1", "id-1", "fp-1", now)
	if got := d.Check("signer-1", "id-1", "fp-1", now); got != DedupDuplicate {
		t.Fatalf("expected DedupDuplicate, got %v", got)
	}
}

func TestDedupSameIDDifferentFingerprintIsConflict(t *testing.T) {
	d := NewDedup(1024, time.Minute)
	now := time.Now()
	d.Check("signer-1", "id-1", "fp-1", now)
	if got := d.Check("signer-1", "id-1", "fp-2", now); got != DedupConflict {
		t.Fatalf("expected DedupConflict, got %v", got)
	}
}

func TestDedupScopedPerPublisher(t *testing.T) {
	d := NewDedup(1024, time.Minute)
	now := time.Now()
	d.Check("signer-1", "id-1", "fp-1", now)
	if got := d.Check("signer-2", "id-1", "fp-1", now); got != DedupNew {
		t.Fatalf("expected DedupNew for a different publisher scope, got %v", got)
	}
}

func TestDedupExpiresAfterTTL(t *testing.T) {
	d := NewDedup(1024, time.Minute)
	now := time.Now()
	d.Check("signer-1", "id-1", "fp-1", now)
	later := now.Add(2 * time.Minute)
	if got := d.Check("signer-1", "id-1", "fp-1", later); got != DedupNew {
		t.Fatalf("expected DedupNew after TTL expiry, got %v", got)
	}
}

func TestDedupBoundedCapacityEvictsOldest(t *testing.T) {
	d := NewDedup(2, time.Hour)
	now := time.Now()
	d.Check("s", "id-1", "fp-1", now)
	d.Check("s", "id-2", "fp-2", now)
	d.Check("s", "id-3", "fp-3", now) // evicts id-1
	if got := d.Check("s", "id-1", "fp-1", now); got != DedupNew {
		t.Fatalf("expected id-1 to have been evicted and treated as new, got %v", got)
	}
}
