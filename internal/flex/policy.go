package flex

import (
	"fmt"
	"math"
	"time"
)

// Policy is the lab's narrower acceptance rules on top of the DSX Flex
// schema: exactly one immediate target, one configured feed, a positive
// bounded absolute constraint or an explicit null release.
type Policy struct {
	// FeedTag is the single feed this lab configuration serves (e.g.
	// "main-a"). An omitted/empty feed_tags list means "all feeds", which
	// in this single-feed lab is exactly this feed. Any explicit feed_tags
	// entry that is not this feed is rejected as unknown, not ignored.
	FeedTag string
	// MaxWatts bounds load_constraint.value after unit conversion. This is
	// an illustrative lab policy bound, not a claim about NVIDIA GPU/rack
	// capacity.
	MaxWatts float64
	// MaxEventAge bounds how old a CloudEvents "time" may be at ingress.
	MaxEventAge time.Duration
	// MaxClockSkew bounds how far in the future "time" may be, to catch
	// clearly wrong clocks without requiring perfectly synced clocks.
	MaxClockSkew time.Duration
}

func unitToWatts(value float64, unit string) (float64, error) {
	switch unit {
	case "watt":
		return value, nil
	case "kilowatt":
		return value * 1_000, nil
	case "megawatt":
		return value * 1_000_000, nil
	default:
		return 0, fmt.Errorf("flex: unsupported power unit %q", unit)
	}
}

// Normalize applies this lab's policy to an already schema-validated,
// signature-verified command and produces the event to fan out, or an error
// identifying the first unsupported/rejected aspect. now is the ingress
// clock reading; tests use a fixed value for determinism.
func (p Policy) Normalize(ev *CloudEvent, data *SetLoadTargetData, now time.Time) (*NormalizedEvent, error) {
	evTime, err := time.Parse(time.RFC3339, ev.Time)
	if err != nil {
		return nil, fmt.Errorf("flex: invalid envelope time: %w", err)
	}
	age := now.Sub(evTime)
	if age > p.MaxEventAge {
		return nil, fmt.Errorf("flex: envelope expired (age %s exceeds max %s)", age, p.MaxEventAge)
	}
	if age < -p.MaxClockSkew {
		return nil, fmt.Errorf("flex: envelope time is in the future beyond allowed clock skew")
	}

	if len(data.Targets) != 1 {
		return nil, fmt.Errorf("flex: exactly one load target is supported, got %d", len(data.Targets))
	}
	target := data.Targets[0]

	if target.Strategy.BestEffort {
		return nil, fmt.Errorf("flex: best_effort strategy is not supported")
	}

	if len(target.FeedTags) > 0 {
		for _, tag := range target.FeedTags {
			if tag != p.FeedTag {
				return nil, fmt.Errorf("flex: unknown feed %q, only %q is configured", tag, p.FeedTag)
			}
		}
	}

	var startTime time.Time
	if target.Interval.StartTime != "" {
		startTime, err = time.Parse(time.RFC3339, target.Interval.StartTime)
		if err != nil {
			return nil, fmt.Errorf("flex: invalid interval start_time: %w", err)
		}
		if startTime.After(now) {
			return nil, fmt.Errorf("flex: future-scheduled start_time is not supported")
		}
	} else {
		startTime = now
	}

	var endTimePtr *time.Time
	if target.Interval.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, target.Interval.EndTime)
		if err != nil {
			return nil, fmt.Errorf("flex: invalid interval end_time: %w", err)
		}
		if !endTime.After(startTime) {
			return nil, fmt.Errorf("flex: interval end_time must be after start_time/now")
		}
		if !endTime.After(now) {
			return nil, fmt.Errorf("flex: interval end_time has already expired")
		}
		endTimePtr = &endTime
	}

	var release bool
	var constraintWatts *float64
	if target.LoadConstraint == nil {
		release = true
	} else {
		value := target.LoadConstraint.Value
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("flex: load_constraint value must be finite")
		}
		if value <= 0 {
			return nil, fmt.Errorf("flex: load_constraint value must be greater than zero (use null to release)")
		}
		watts, err := unitToWatts(value, target.LoadConstraint.Unit)
		if err != nil {
			return nil, err
		}
		if watts > p.MaxWatts {
			return nil, fmt.Errorf("flex: load_constraint %g W exceeds configured bound %g W", watts, p.MaxWatts)
		}
		constraintWatts = &watts
	}

	return &NormalizedEvent{
		Type:            CommandReceivedType,
		ID:              ev.ID,
		FeedTag:         p.FeedTag,
		Release:         release,
		ConstraintWatts: constraintWatts,
		EndTime:         endTimePtr,
		ReceivedAt:      now,
		Traceparent:     ev.Traceparent,
		CorrelationID:   ev.CorrelationID,
	}, nil
}
