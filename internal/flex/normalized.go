package flex

import "time"

// NormalizedEvent is what the bridge publishes to the local authenticated
// event stream after signature verification, schema validation, and lab
// policy acceptance. It reports receipt, not actuation: there is no DSX
// execution acknowledgement, modeled power claim, or Omniverse path here.
type NormalizedEvent struct {
	Type            string     `json:"type"`
	ID              string     `json:"id"`
	FeedTag         string     `json:"feed_tag"`
	Release         bool       `json:"release"`
	ConstraintWatts *float64   `json:"constraint_watts,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	ReceivedAt      time.Time  `json:"received_at"`
	Traceparent     string     `json:"traceparent"`
	CorrelationID   string     `json:"correlation_id,omitempty"`
}

// CommandReceivedType is the only normalized event type Phase 2 emits.
const CommandReceivedType = "command_received"
