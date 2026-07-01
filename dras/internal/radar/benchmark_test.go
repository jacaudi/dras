package radar

import (
	"sync"
	"testing"
)

// BenchmarkGetMode tests performance of VCP to mode conversion
func BenchmarkGetMode(b *testing.B) {
	vcps := []VCP{VCPR31, VCPR35, VCPR12, VCPR112, VCPR212, VCPR215}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vcp := vcps[i%len(vcps)]
		_, _ = GetMode(vcp)
	}
}

// BenchmarkSanitizeStationIDs tests performance of station ID parsing
func BenchmarkSanitizeStationIDs(b *testing.B) {
	testInputs := []string{
		"KATX,KRAX,KBGM",
		"KATX KRAX KBGM KTLX KFFC",
		"KATX; KRAX, KBGM ; KTLX   KFFC",
		"KATX,KRAX,KBGM,KTLX,KFFC,KJAX,KMLB,KTBW,KMHX,KAKQ",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		_ = SanitizeStationIDs(input)
	}
}

// BenchmarkCompareData tests performance of radar data comparison
func BenchmarkCompareData(b *testing.B) {
	oldData := &Data{
		Name:              "KATX",
		VCP:               "R31",
		Mode:              "Clear Air",
		Status:            "Online",
		OperabilityStatus: "Normal",
		PowerSource:       "Utility",
		GenState:          "Off",
	}

	newData := &Data{
		Name:              "KATX",
		VCP:               "R12",
		Mode:              "Precipitation",
		Status:            "Online",
		OperabilityStatus: "Normal",
		PowerSource:       "Generator",
		GenState:          "On",
	}

	alertConfig := AlertConfig{
		VCP:         true,
		Status:      true,
		Operability: true,
		PowerSource: true,
		GenState:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CompareData(oldData, newData, alertConfig)
	}
}

// BenchmarkParseAlarmSummary is the performance coverage for the alarmSummary
// parser (issue #128). It should be allocation-free for the pass-through path
// since the parser only converts a string to a named string type.
func BenchmarkParseAlarmSummary(b *testing.B) {
	inputs := []string{"No Alarms", "Communication", "RDA Control", "", "Pedestal"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseAlarmSummary(inputs[i%len(inputs)])
	}
}

// BenchmarkConcurrentRadarProcessing simulates concurrent radar data processing
func BenchmarkConcurrentRadarProcessing(b *testing.B) {
	stationIDs := []string{"KATX", "KRAX", "KBGM", "KTLX", "KFFC"}

	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, stationID := range stationIDs {
				// Simulate radar data processing
				data := &Data{
					Name:              stationID,
					VCP:               "R31",
					Mode:              "Clear Air",
					Status:            "Online",
					OperabilityStatus: "Normal",
					PowerSource:       "Utility",
					GenState:          "Off",
				}
				_, _ = GetMode(data.VCP)
			}
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup
			for _, stationID := range stationIDs {
				wg.Add(1)
				go func(id string) {
					defer wg.Done()
					// Simulate radar data processing
					data := &Data{
						Name:              id,
						VCP:               "R31",
						Mode:              "Clear Air",
						Status:            "Online",
						OperabilityStatus: "Normal",
						PowerSource:       "Utility",
						GenState:          "Off",
					}
					_, _ = GetMode(data.VCP)
				}(stationID)
			}
			wg.Wait()
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("DataStructCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = &Data{
				Name:              "KATX",
				VCP:               "R31",
				Mode:              "Clear Air",
				Status:            "Online",
				OperabilityStatus: "Normal",
				PowerSource:       "Utility",
				GenState:          "Off",
			}
		}
	})

	b.Run("AlertConfigCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = AlertConfig{
				VCP:         true,
				Status:      true,
				Operability: true,
				PowerSource: true,
				GenState:    true,
			}
		}
	})
}
