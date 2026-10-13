package presentation

import (
	_ "embed"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"gridmetry/dsx-adapter/internal/flex"
)

//go:embed browser-receiver.js
var BrowserScript string

type Config struct {
	FeedTag             string
	EDPSetting          string
	BaselineLoadPercent int
}

type Command struct {
	Type                string     `json:"type"`
	ID                  string     `json:"id"`
	FeedTag             string     `json:"feed_tag"`
	LoadPercent         int        `json:"load_percent"`
	EDPSetting          string     `json:"edp_setting"`
	FailedRPPs          int        `json:"failed_rpps"`
	DisplayedPowerWatts int        `json:"displayed_power_watts"`
	RequestedWatts      *float64   `json:"requested_watts,omitempty"`
	Release             bool       `json:"release"`
	ApplyBefore         time.Time  `json:"apply_before"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Evidence            string     `json:"evidence"`
}

type Mapper struct {
	config    Config
	maximumKW int
}

func NewMapper(config Config) (*Mapper, error) {
	maximumKW := map[string]int{"1.2": 1316, "1.5": 1531}[config.EDPSetting]
	if maximumKW == 0 || config.FeedTag == "" || len(config.FeedTag) > 64 || strings.TrimSpace(config.FeedTag) != config.FeedTag || config.BaselineLoadPercent < 0 || config.BaselineLoadPercent > 100 {
		return nil, errors.New("invalid visualization configuration")
	}
	return &Mapper{config: config, maximumKW: maximumKW}, nil
}

func (mapper *Mapper) displayedWatts(percent int) int {
	perRPPKW := (mapper.maximumKW*percent + 200) / 400
	return perRPPKW * 4000
}

func (mapper *Mapper) BrowserClient() string {
	source := strings.TrimSpace(BrowserScript)
	const invocation = "})(globalThis);"
	if !strings.HasSuffix(source, invocation) {
		return ""
	}
	profile, err := json.Marshal(struct {
		FeedTag             string `json:"feedTag"`
		EDPSetting          string `json:"edpSetting"`
		BaselineLoadPercent int    `json:"baselineLoadPercent"`
		MaximumKW           int    `json:"maximumKW"`
	}{mapper.config.FeedTag, mapper.config.EDPSetting, mapper.config.BaselineLoadPercent, mapper.maximumKW})
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(source, invocation) + "})(globalThis," + string(profile) + ");\n"
}

func (mapper *Mapper) Map(event flex.NormalizedEvent, now time.Time) (Command, error) {
	if event.Type != flex.CommandReceivedType || event.ID == "" || event.FeedTag != mapper.config.FeedTag || event.ReceivedAt.IsZero() || now.IsZero() || event.ReceivedAt.After(now.Add(30*time.Second)) || !event.ReceivedAt.Add(5*time.Minute).After(now) {
		return Command{}, errors.New("invalid or stale received event")
	}
	deadline := event.ReceivedAt.Add(5 * time.Minute)
	if event.EndTime != nil {
		if !event.EndTime.After(now) {
			return Command{}, errors.New("event has expired")
		}
		if event.EndTime.Before(deadline) {
			deadline = *event.EndTime
		}
	}
	percent := 100
	if event.Release {
		if event.ConstraintWatts != nil {
			return Command{}, errors.New("release cannot contain a power constraint")
		}
		percent = mapper.config.BaselineLoadPercent
	} else {
		if event.ConstraintWatts == nil || math.IsNaN(*event.ConstraintWatts) || math.IsInf(*event.ConstraintWatts, 0) || *event.ConstraintWatts <= 0 {
			return Command{}, errors.New("invalid visualization power constraint")
		}
		for percent > 0 && float64(mapper.displayedWatts(percent)) > *event.ConstraintWatts {
			percent--
		}
	}
	return Command{
		Type: "visualization_target", ID: event.ID, FeedTag: event.FeedTag,
		LoadPercent: percent, EDPSetting: mapper.config.EDPSetting, FailedRPPs: 0,
		DisplayedPowerWatts: mapper.displayedWatts(percent), RequestedWatts: event.ConstraintWatts,
		Release: event.Release, ApplyBefore: deadline, EndTime: event.EndTime,
		Evidence: "visualization_input",
	}, nil
}
