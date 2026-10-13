package main

import "testing"

func TestVisualizationFlags(t *testing.T) {
	mapper, err := visualizationFlags(nil, "main-a")
	if err != nil || mapper != nil {
		t.Fatal("visualization must be disabled by default")
	}
	mapper, err = visualizationFlags([]string{"--visual-edp", "1.5", "--visual-baseline-load", "80"}, "main-a")
	if err != nil || mapper == nil {
		t.Fatal("explicit visualization configuration was rejected")
	}
	for _, args := range [][]string{
		{"--visual-edp", "other"},
		{"--visual-edp", "1.5", "--visual-baseline-load", "101"},
		{"--visual-baseline-load", "80"},
		{"--unknown"},
		{"unexpected"},
	} {
		if _, err := visualizationFlags(args, "main-a"); err == nil {
			t.Fatalf("invalid flags accepted: %v", args)
		}
	}
}
