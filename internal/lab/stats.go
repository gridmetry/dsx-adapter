// Package lab holds small, dependency-free building blocks shared by the
// bridge, transport, and scenario binaries: counters, configuration, and
// secret/key generation used by the init command. It intentionally does not
// import internal/flex, internal/bridge, or internal/transport.
package lab

import "sync/atomic"

// Stats are bounded, non-secret counters surfaced on /healthz so a scenario
// run or operator can see honest accepted/rejected/duplicate/dropped
// counts instead of inferring success from silence.
type Stats struct {
	accepted     atomic.Int64
	rejected     atomic.Int64
	duplicate    atomic.Int64
	queueDropped atomic.Int64
	expired      atomic.Int64
}

func (s *Stats) IncAccepted()     { s.accepted.Add(1) }
func (s *Stats) IncRejected()     { s.rejected.Add(1) }
func (s *Stats) IncDuplicate()    { s.duplicate.Add(1) }
func (s *Stats) IncQueueDropped() { s.queueDropped.Add(1) }

// IncExpired counts an accepted command dropped before delivery (hub
// publish or SSE write) because its end_time had already passed by the
// time delivery was attempted; it is not a decode/policy rejection.
func (s *Stats) IncExpired() { s.expired.Add(1) }

// Snapshot is a point-in-time, non-secret copy of the counters.
type Snapshot struct {
	Accepted     int64 `json:"accepted"`
	Rejected     int64 `json:"rejected"`
	Duplicate    int64 `json:"duplicate"`
	QueueDropped int64 `json:"queue_dropped"`
	Expired      int64 `json:"expired"`
}

func (s *Stats) Snapshot() Snapshot {
	return Snapshot{
		Accepted:     s.accepted.Load(),
		Rejected:     s.rejected.Load(),
		Duplicate:    s.duplicate.Load(),
		QueueDropped: s.queueDropped.Load(),
		Expired:      s.expired.Load(),
	}
}
