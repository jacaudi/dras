// +build integration

package radar

import (
	"testing"
	"time"
)

// These tests are designed to test against the real NWS API
// Run with: go test -tags=integration ./internal/radar
// Note: These tests require internet connectivity and depend on external services

func TestRealNWSIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	service := New()

	t.Run("fetch real radar data", func(t *testing.T) {
		// Test with known radar stations
		testStations := []string{"KATX", "KRAX"} // Seattle, WA and Raleigh, NC
		
		for _, stationID := range testStations {
			t.Run("station_"+stationID, func(t *testing.T) {
				data, err := service.FetchData(stationID)
				if err != nil {
					t.Errorf("Failed to fetch data for %s: %v", stationID, err)
					return
				}

				// Validate the returned data structure
				if data == nil {
					t.Errorf("Expected data for station %s, got nil", stationID)
					return
				}

				if data.Name == "" {
					t.Errorf("Expected station name for %s, got empty string", stationID)
				}

				if data.VCP == "" {
					t.Errorf("Expected VCP for %s, got empty string", stationID)
				}

				if data.Status == "" {
					t.Errorf("Expected status for %s, got empty string", stationID)
				}

				// Test that VCP can be converted to mode
				mode, err := GetMode(data.VCP)
				if err != nil {
					t.Errorf("Failed to get mode for VCP %s: %v", data.VCP, err)
				} else {
					if mode != "Clear Air" && mode != "Precipitation" {
						t.Errorf("Expected mode to be 'Clear Air' or 'Precipitation', got %s", mode)
					}
				}

				t.Logf("Station %s: Name=%s, VCP=%s, Mode=%s, Status=%s, PowerSource=%s", 
					stationID, data.Name, data.VCP, data.Mode, data.Status, data.PowerSource)
			})
		}
	})

	t.Run("test error handling with invalid station", func(t *testing.T) {
		invalidStation := "INVALID"
		_, err := service.FetchData(invalidStation)
		if err == nil {
			t.Errorf("Expected error for invalid station %s, got nil", invalidStation)
		}
		t.Logf("Got expected error for invalid station: %v", err)
	})

	t.Run("performance test with multiple stations", func(t *testing.T) {
		stations := []string{"KATX", "KRAX", "KBGM"}
		start := time.Now()

		for _, stationID := range stations {
			_, err := service.FetchData(stationID)
			if err != nil {
				t.Logf("Warning: Failed to fetch %s: %v", stationID, err)
			}
		}

		duration := time.Since(start)
		t.Logf("Fetched %d stations in %v (avg: %v per station)", 
			len(stations), duration, duration/time.Duration(len(stations)))

		// Reasonable performance expectation (adjust based on network)
		maxExpectedDuration := 30 * time.Second
		if duration > maxExpectedDuration {
			t.Errorf("Performance test took too long: %v (max expected: %v)", duration, maxExpectedDuration)
		}
	})
}

func TestRealDataComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	service := New()
	stationID := "KATX" // Seattle radar

	t.Run("compare real data changes over time", func(t *testing.T) {
		// Fetch initial data
		initialData, err := service.FetchData(stationID)
		if err != nil {
			t.Skipf("Skipping comparison test due to fetch error: %v", err)
		}

		// Wait a short time
		time.Sleep(2 * time.Second)

		// Fetch data again
		laterData, err := service.FetchData(stationID)
		if err != nil {
			t.Errorf("Failed to fetch later data: %v", err)
			return
		}

		// Compare the data (most likely will be the same in short time)
		alertConfig := AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		}

		changed, message := CompareData(initialData, laterData, alertConfig)
		
		if changed {
			t.Logf("Data changed for %s: %s", stationID, message)
		} else {
			t.Logf("No changes detected for %s over short time period", stationID)
		}

		// Verify the comparison logic works correctly
		if initialData.VCP == laterData.VCP && changed {
			// If VCP is the same but other fields changed, check the message
			if message == "" {
				t.Error("Expected change message when data changed")
			}
		}
	})
}

func TestSanitizeStationIDsWithRealData(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{"KATX,KRAX", []string{"KATX", "KRAX"}},
		{"KATX KRAX KBGM", []string{"KATX", "KRAX", "KBGM"}},
		{"KATX; KRAX, KBGM", []string{"KATX", "KRAX", "KBGM"}},
	}

	for _, tc := range testCases {
		t.Run("input_"+tc.input, func(t *testing.T) {
			result := SanitizeStationIDs(tc.input)
			
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d stations, got %d", len(tc.expected), len(result))
				return
			}

			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("Expected station[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

// Helper function to test connection to NWS API
func TestNWSConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connectivity test in short mode")
	}

	service := New()
	
	// Quick connectivity test
	_, err := service.FetchData("KATX")
	if err != nil {
		t.Logf("NWS API connectivity issue: %v", err)
		t.Skip("Skipping further integration tests due to connectivity issues")
	} else {
		t.Log("NWS API connectivity confirmed")
	}
}