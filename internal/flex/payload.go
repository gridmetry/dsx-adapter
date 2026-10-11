package flex

// Interval is start/end UTC bounds for a load target. Both are optional per
// the contract: omitted start means now; omitted end means indefinite.
type Interval struct {
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

// Strategy carries the upstream best_effort flag. This lab profile rejects
// best_effort: true rather than attempting to simulate DPM-aware shedding.
type Strategy struct {
	BestEffort bool `json:"best_effort,omitempty"`
}

// LoadConstraint is an absolute power limit. A nil *LoadConstraint in
// LoadTarget.LoadConstraint represents the schema's explicit null release,
// which is distinct from the field being omitted (the schema requires it).
type LoadConstraint struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// LoadTarget is one requested target within a grid.loadtarget.set.v1
// payload. LoadConstraint is a pointer so JSON null decodes to nil,
// distinguishing release-to-default from a positive bound.
type LoadTarget struct {
	Interval       Interval        `json:"interval"`
	FeedTags       []string        `json:"feed_tags,omitempty"`
	LoadConstraint *LoadConstraint `json:"load_constraint"`
	Strategy       Strategy        `json:"strategy,omitempty"`
}

// SetLoadTargetData is the decoded (post-JWS-verification) plaintext payload
// for grid.loadtarget.set.v1.
type SetLoadTargetData struct {
	Targets []LoadTarget `json:"targets"`
}
