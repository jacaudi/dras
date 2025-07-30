package radar

import (
	"testing"
)

func TestGetMode(t *testing.T) {
	tests := []struct {
		vcp      string
		expected string
		hasError bool
	}{
		{"R31", "Clear Air", false},
		{"R35", "Clear Air", false},
		{"R12", "Precipitation", false},
		{"R112", "Precipitation", false},
		{"R212", "Precipitation", false},
		{"R215", "Precipitation", false},
		{"R99", "", true}, // Unknown VCP should return error
	}

	for _, tt := range tests {
		t.Run(tt.vcp, func(t *testing.T) {
			mode, err := GetMode(tt.vcp)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for VCP %s, but got none", tt.vcp)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for VCP %s: %v", tt.vcp, err)
				return
			}

			if mode != tt.expected {
				t.Errorf("For VCP %s, expected mode %q, got %q", tt.vcp, tt.expected, mode)
			}
		})
	}
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
		{"katx krax", []string{"KATX", "KRAX"}}, // test lowercase conversion
		{"KATX, invalid, KRAX", []string{"KATX", "KRAX"}}, // test filtering invalid
		{"K1TX", []string{}}, // test invalid - contains number
		{"ATXX", []string{"ATXX"}}, // test valid non-K station
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
