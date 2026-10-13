package presentation

import (
	"math"
	"strings"
	"testing"
	"time"

	"gridmetry/dsx-adapter/internal/flex"
)

func acceptedEvent(now time.Time, watts float64) flex.NormalizedEvent {
	return flex.NormalizedEvent{
		Type: flex.CommandReceivedType, ID: "presentation-test", FeedTag: "main-a",
		ConstraintWatts: &watts, ReceivedAt: now,
	}
}

func TestMapperConservativeDisplayedPower(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	mapper, err := NewMapper(Config{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: 80})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		watts   float64
		percent int
		power   int
	}{
		{920000, 60, 920000},
		{918600, 59, 904000},
		{800000, 52, 796000},
		{650000, 42, 644000},
		{2000000, 100, 1532000},
		{1, 0, 0},
	} {
		command, err := mapper.Map(acceptedEvent(now, test.watts), now)
		if err != nil {
			t.Fatal(err)
		}
		if command.LoadPercent != test.percent || command.DisplayedPowerWatts != test.power {
			t.Fatalf("target %.0f: got %d%% and %d W", test.watts, command.LoadPercent, command.DisplayedPowerWatts)
		}
		if float64(command.DisplayedPowerWatts) > test.watts {
			t.Fatal("displayed load exceeds target")
		}
	}
}

func TestMapperReleaseAndEDP(t *testing.T) {
	now := time.Now()
	mapper, err := NewMapper(Config{FeedTag: "main-a", EDPSetting: "1.2", BaselineLoadPercent: 80})
	if err != nil {
		t.Fatal(err)
	}
	event := acceptedEvent(now, 1000)
	event.Release = true
	event.ConstraintWatts = nil
	command, err := mapper.Map(event, now)
	if err != nil {
		t.Fatal(err)
	}
	if !command.Release || command.LoadPercent != 80 || command.DisplayedPowerWatts != 1052000 {
		t.Fatalf("unexpected baseline restoration: %+v", command)
	}
	if !strings.Contains(mapper.BrowserClient(), `"edpSetting":"1.2"`) || !strings.Contains(mapper.BrowserClient(), `"baselineLoadPercent":80`) {
		t.Fatal("configured browser profile was not served")
	}
}

func TestMapperFullLoadIndependentOfStartingBaseline(t *testing.T) {
	now := time.Now()
	mapper, err := NewMapper(Config{FeedTag: "main-a", EDPSetting: "1.2", BaselineLoadPercent: 80})
	if err != nil {
		t.Fatal(err)
	}
	command, err := mapper.Map(acceptedEvent(now, 1316000), now)
	if err != nil || command.LoadPercent != 100 || command.DisplayedPowerWatts != 1316000 {
		t.Fatalf("full load was clipped by the startup baseline: %+v, %v", command, err)
	}
	mapper, _ = NewMapper(Config{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: 60})
	if !strings.Contains(mapper.BrowserClient(), `"baselineLoadPercent":60`) {
		t.Fatal("browser profile does not use the configured startup load")
	}
}

func TestMapperRejectsInvalidOrExpiredEvents(t *testing.T) {
	now := time.Now()
	mapper, _ := NewMapper(Config{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: 80})
	mutations := []func(*flex.NormalizedEvent){
		func(event *flex.NormalizedEvent) { event.Type = "other" },
		func(event *flex.NormalizedEvent) { event.ID = "" },
		func(event *flex.NormalizedEvent) { event.FeedTag = "other" },
		func(event *flex.NormalizedEvent) { event.ConstraintWatts = nil },
		func(event *flex.NormalizedEvent) { event.Release = true },
		func(event *flex.NormalizedEvent) { event.ReceivedAt = time.Time{} },
		func(event *flex.NormalizedEvent) { event.ReceivedAt = now.Add(-6 * time.Minute) },
		func(event *flex.NormalizedEvent) { event.ReceivedAt = now.Add(time.Minute) },
		func(event *flex.NormalizedEvent) { expired := now; event.EndTime = &expired },
		func(event *flex.NormalizedEvent) { invalid := math.NaN(); event.ConstraintWatts = &invalid },
		func(event *flex.NormalizedEvent) { invalid := math.Inf(1); event.ConstraintWatts = &invalid },
		func(event *flex.NormalizedEvent) { invalid := 0.0; event.ConstraintWatts = &invalid },
	}
	for index, mutate := range mutations {
		event := acceptedEvent(now, 920000)
		mutate(&event)
		if _, err := mapper.Map(event, now); err == nil {
			t.Fatalf("mutation %d was accepted", index)
		}
	}
}

func TestMapperRejectsInvalidConfiguration(t *testing.T) {
	for _, config := range []Config{
		{EDPSetting: "1.5", BaselineLoadPercent: 80},
		{FeedTag: "main-a", EDPSetting: "other", BaselineLoadPercent: 80},
		{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: -1},
		{FeedTag: "main-a", EDPSetting: "1.5", BaselineLoadPercent: 101},
	} {
		if _, err := NewMapper(config); err == nil {
			t.Fatal("invalid mapper configuration accepted")
		}
	}
}
