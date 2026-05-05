package radar

import (
	"errors"
	"strings"
	"testing"
)

func TestGetMode(t *testing.T) {
	tests := []struct {
		name           string
		vcp            string
		expectedMode   string
		expectUnknown  bool
		fallbackSubstr string
	}{
		{name: "R31", vcp: "R31", expectedMode: "Clear Air"},
		{name: "R35", vcp: "R35", expectedMode: "Clear Air"},
		{name: "R12", vcp: "R12", expectedMode: "Precipitation"},
		{name: "R112", vcp: "R112", expectedMode: "Precipitation"},
		{name: "R212", vcp: "R212", expectedMode: "Precipitation"},
		{name: "R215", vcp: "R215", expectedMode: "Precipitation"},
		{name: "unknown_R99", vcp: "R99", expectUnknown: true, fallbackSubstr: `"R99"`},
		{name: "empty", vcp: "", expectUnknown: true, fallbackSubstr: `""`},
		{name: "whitespace", vcp: " ", expectUnknown: true, fallbackSubstr: `" "`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := GetMode(tt.vcp)

			if tt.expectUnknown {
				if !errors.Is(err, ErrUnknownVCP) {
					t.Fatalf("expected ErrUnknownVCP for VCP %q, got %v", tt.vcp, err)
				}
				if !strings.Contains(mode, "Unknown") {
					t.Errorf("expected fallback mode label to contain \"Unknown\" for VCP %q, got %q", tt.vcp, mode)
				}
				if !strings.Contains(mode, tt.fallbackSubstr) {
					t.Errorf("expected fallback mode label to contain %s for VCP %q, got %q", tt.fallbackSubstr, tt.vcp, mode)
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
		info, err := GetVCPInfo("R212")
		if err != nil {
			t.Fatalf("unexpected error for R212: %v", err)
		}
		if info.Mode != "Precipitation" {
			t.Errorf("Mode = %q, want %q", info.Mode, "Precipitation")
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
		if !strings.Contains(info.Mode, "Unknown") {
			t.Errorf("Mode = %q, expected to contain \"Unknown\"", info.Mode)
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

func TestSimplifyGeneratorState(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasError bool
	}{
		{"Switched to Auxiliary Power|Utility PWR Available|Generator On", "On", false},
		{"Switched to Auxiliary Power|Generator On", "On", false},
		{"Utility PWR Available|Generator On", "On", false},
		{"Utility PWR Available", "Off", false},
		{"Unknown State", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := simplifyGeneratorState(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("For input %q, expected %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}
