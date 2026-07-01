package radar

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGetMode(t *testing.T) {
	tests := []struct {
		name          string
		vcp           VCP
		expectedMode  RadarMode
		expectUnknown bool
	}{
		{name: "R31", vcp: VCPR31, expectedMode: ModeClearAir},
		{name: "R35", vcp: VCPR35, expectedMode: ModeClearAir},
		{name: "R12", vcp: VCPR12, expectedMode: ModePrecipitation},
		{name: "R112", vcp: VCPR112, expectedMode: ModePrecipitation},
		{name: "R212", vcp: VCPR212, expectedMode: ModePrecipitation},
		{name: "R215", vcp: VCPR215, expectedMode: ModePrecipitation},
		{name: "unknown_R99", vcp: "R99", expectUnknown: true},
		{name: "empty", vcp: "", expectUnknown: true},
		{name: "whitespace", vcp: " ", expectUnknown: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := GetMode(tt.vcp)

			if tt.expectUnknown {
				if !errors.Is(err, ErrUnknownVCP) {
					t.Fatalf("expected ErrUnknownVCP for VCP %q, got %v", tt.vcp, err)
				}
				if mode != ModeUnknown {
					t.Errorf("expected ModeUnknown for VCP %q, got %q", tt.vcp, mode)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for VCP %s: %v", tt.vcp, err)
				return
			}

			if mode != tt.expectedMode {
				t.Errorf("For VCP %s, expected mode %q, got %q", tt.vcp, tt.expectedMode, mode)
			}
		})
	}
}

func TestGetVCPInfo(t *testing.T) {
	t.Run("known", func(t *testing.T) {
		info, err := GetVCPInfo(VCPR212)
		if err != nil {
			t.Fatalf("unexpected error for R212: %v", err)
		}
		if info.Mode != ModePrecipitation {
			t.Errorf("Mode = %q, want %q", info.Mode, ModePrecipitation)
		}
		if !strings.Contains(info.Description, "SAILS") {
			t.Errorf("Description = %q, expected to mention SAILS", info.Description)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		info, err := GetVCPInfo("R999")
		if !errors.Is(err, ErrUnknownVCP) {
			t.Fatalf("expected ErrUnknownVCP, got %v", err)
		}
		if info.Mode != ModeUnknown {
			t.Errorf("Mode = %q, expected ModeUnknown", info.Mode)
		}
		if !strings.Contains(info.Description, "R999") {
			t.Errorf("Description = %q, expected to contain raw VCP %q", info.Description, "R999")
		}
	})
}

func TestSanitizeStationIDs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"KATX KRAX", []string{"KATX", "KRAX"}},
		{"KATX,KRAX", []string{"KATX", "KRAX"}},
		{"KATX;KRAX", []string{"KATX", "KRAX"}},
		{"KATX, KRAX ; KBGM", []string{"KATX", "KRAX", "KBGM"}},
		{"KATX", []string{"KATX"}},
		{"", []string{}},
		{"katx krax", []string{"KATX", "KRAX"}},           // test lowercase conversion
		{"KATX, invalid, KRAX", []string{"KATX", "KRAX"}}, // test filtering invalid
		{"K1TX", []string{}},                              // test invalid - contains number
		{"ATXX", []string{"ATXX"}},                        // test valid non-K station
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeStationIDs(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d stations, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected station[%d] = %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestValidateStationID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		name     string
	}{
		{"KATX", true, "valid US station"},
		{"KRAX", true, "valid US station"},
		{"ATXX", true, "valid non-K station"},
		{"PGUA", true, "valid international station"},
		{"katx", false, "lowercase should fail"},
		{"K1TX", false, "contains number"},
		{"KAT", false, "too short"},
		{"KATXX", false, "too long"},
		{"", false, "empty string"},
		{"123A", false, "starts with number"},
		{"K@TX", false, "contains special character"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateStationID(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateStationID(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGeneratorState(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        GeneratorState
		expectError bool
	}{
		// Known good — all four canonical raw strings collapse to On / Off.
		{name: "all_three_tokens_with_generator_on", input: "Switched to Auxiliary Power|Utility PWR Available|Generator On", want: GenStateOn},
		{name: "aux_power_plus_generator_on", input: "Switched to Auxiliary Power|Generator On", want: GenStateOn},
		{name: "utility_plus_generator_on", input: "Utility PWR Available|Generator On", want: GenStateOn},
		{name: "utility_only", input: "Utility PWR Available", want: GenStateOff},

		// Token-set robustness — order shouldn't matter (Option 3 logic).
		{name: "reversed_order", input: "Generator On|Utility PWR Available", want: GenStateOn},

		// Soft-fail surface area — empty + novel + ambiguous inputs land on
		// GenStateUnknown with ErrUnknownGeneratorState wrapping. This is the
		// regression coverage for issue #129; the previous implementation
		// returned ("", errors.New("unknown input")) and the caller aborted.
		{name: "empty", input: "", want: GenStateUnknown, expectError: true},
		{name: "totally_novel", input: "Cold Fusion Reactor Active", want: GenStateUnknown, expectError: true},
		// Utility token present but with extra unknown tokens → not safe to
		// classify as Off (might mean "running on generator but also..."),
		// so soft-fail.
		{name: "utility_with_extra_tokens", input: "Utility PWR Available|Mystery State", want: GenStateUnknown, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGeneratorState(tt.input)

			if got != tt.want {
				t.Errorf("ParseGeneratorState(%q) state = %q, want %q", tt.input, got, tt.want)
			}

			switch {
			case tt.expectError && !errors.Is(err, ErrUnknownGeneratorState):
				t.Errorf("ParseGeneratorState(%q) err = %v, want errors.Is(err, ErrUnknownGeneratorState)", tt.input, err)
			case !tt.expectError && err != nil:
				t.Errorf("ParseGeneratorState(%q) unexpected err = %v", tt.input, err)
			}
		})
	}
}

// TestParseAlarmSummary is the unit coverage for the alarmSummary parser
// (issue #128). Empty input maps to AlarmSummaryUnknown (the skip-Unknown
// sentinel); every non-empty value passes through verbatim so an
// unenumerated NWS value survives for forensic logging.
func TestParseAlarmSummary(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  AlarmSummary
	}{
		{name: "empty_maps_to_unknown", input: "", want: AlarmSummaryUnknown},
		{name: "healthy", input: "No Alarms", want: AlarmSummaryNone},
		{name: "communication", input: "Communication", want: AlarmSummaryCommunication},
		{name: "rda_control", input: "RDA Control", want: AlarmSummaryRDAControl},
		{name: "tower_utilities", input: "Tower / Utilities", want: AlarmSummaryTowerUtil},
		// Unenumerated value passes through verbatim (not squashed to Unknown).
		{name: "novel_value_passthrough", input: "Pedestal", want: AlarmSummary("Pedestal")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseAlarmSummary(tt.input); got != tt.want {
				t.Errorf("ParseAlarmSummary(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ExampleParseGeneratorState documents the typical happy-path call shape and
// is exercised by `go test` so the example never rots (§8.3 of
// go-standards.md: "A broken example is a broken release.").
func ExampleParseGeneratorState() {
	state, _ := ParseGeneratorState("Utility PWR Available|Generator On")
	// state implements fmt.Stringer; %s prints the underlying string.
	fmt.Println(state)
	// Output: On
}
