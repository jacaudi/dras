package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeStationIDs(t *testing.T) {
	// Set up environment variables
	t.Setenv("STATION_IDS", "KATX,KRAX,KTLX")
	t.Setenv("PUSHOVER_API_TOKEN", "your_api_token")
	t.Setenv("PUSHOVER_USER_KEY", "your_user_key")

	input := "KATX, KRAX; KTLX"
	expected := []string{"KATX", "KRAX", "KTLX"}
	result := sanitizeStationIDs(input)
	assert.Equal(t, expected, result)
}

func TestRadarMode(t *testing.T) {
	tests := []struct {
		vcp      string
		expected string
		err      bool
	}{
		{"R35", "Clear Air", false},
		{"R215", "Precipitation", false},
		{"R999", "", true},
	}

	for _, test := range tests {
		result, err := radarMode(test.vcp)
		if test.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}

func TestCompareRadarData(t *testing.T) {
	oldData := &RadarData{
		VCP:               "R35",
		Status:            "Operational",
		PowerSource:       "Commercial",
		GenState:          "Running",
		OperabilityStatus: "RDA - Inoperable",
	}

	newData := &RadarData{
		VCP:               "R215",
		Status:            "Operational",
		PowerSource:       "Backup",
		GenState:          "Stopped",
		OperabilityStatus: "Functioning",
	}

	changed, message := compareRadarData(oldData, newData)
	assert.True(t, changed)
	assert.Contains(t, message, "The Radar is in Precipitation Mode (Vertical Scanning Emphasis) -- Precipitation Detected")
	assert.Contains(t, message, "Power source changed from Commercial to Backup")
	assert.Contains(t, message, "Generator state changed from Running to Stopped")
	assert.Contains(t, message, "Radar operability changed from RDA - Inoperable to Functioning")
}

func TestGenStateSimp(t *testing.T) {
	tests := []struct {
		genInput  string
		expected  string
		expectErr bool
	}{
		{"Switched to Auxiliary Power|Utility PWR Available|Generator On", "On", false},
		{"Switched to Auxiliary Power|Generator On", "On", false},
		{"Utility PWR Available|Generator On", "On", false},
		{"Utility PWR Available", "Off", false},
		{"Unknown Input", "", true},
	}

	for _, test := range tests {
		result, err := genStateSimp(test.genInput)
		if test.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}
