package monitor

import (
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

// BenchmarkMonitorMemoryUsage tests memory usage patterns for long-running monitoring
func BenchmarkMonitorMemoryUsage(b *testing.B) {
	// Create test configuration
	cfg := &config.Config{
		DryRun:        true,
		CheckInterval: 1 * time.Millisecond, // Fast for benchmarking
		AlertConfig: config.AlertConfig{
			VCP:         true,
			Status:      false,
			Operability: false,
			PowerSource: false,
			GenState:    false,
		},
	}

	b.Run("MonitorCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			radarService := radar.New()
			var notifyService *notify.Service // nil for dry run
			_ = New(radarService, notifyService, cfg)
		}
	})

	b.Run("DataMapGrowth", func(b *testing.B) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		stations := []string{"KATX", "KRAX", "KBGM", "KTLX", "KFFC"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate data storage
			stationID := stations[i%len(stations)]
			monitor.mu.Lock()
			if _, exists := monitor.radarDataMap[stationID]; !exists {
				monitor.radarDataMap[stationID] = make(map[string]interface{})
			}
			monitor.radarDataMap[stationID]["last"] = &radar.Data{
				Name:              stationID,
				VCP:               "R31",
				Mode:              "Clear Air",
				Status:            "Online",
				OperabilityStatus: "Normal",
				PowerSource:       "Utility",
				GenState:          "Off",
			}
			monitor.mu.Unlock()
		}
	})
}

// BenchmarkConcurrentProcessing tests concurrent station processing performance
func BenchmarkConcurrentProcessing(b *testing.B) {
	cfg := &config.Config{
		DryRun: true,
		AlertConfig: config.AlertConfig{
			VCP: true,
		},
	}

	b.Run("SingleStation", func(b *testing.B) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate processing without external API calls
			monitor.mu.Lock()
			monitor.radarDataMap["KATX"] = make(map[string]interface{})
			monitor.radarDataMap["KATX"]["last"] = &radar.Data{
				Name:              "KATX",
				VCP:               "R31",
				Mode:              "Clear Air",
				Status:            "Online",
				OperabilityStatus: "Normal",
				PowerSource:       "Utility",
				GenState:          "Off",
			}
			monitor.mu.Unlock()
		}
	})

	b.Run("MultipleStations", func(b *testing.B) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		stations := []string{"KATX", "KRAX", "KBGM", "KTLX", "KFFC"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, stationID := range stations {
				monitor.mu.Lock()
				if _, exists := monitor.radarDataMap[stationID]; !exists {
					monitor.radarDataMap[stationID] = make(map[string]interface{})
				}
				monitor.radarDataMap[stationID]["last"] = &radar.Data{
					Name:              stationID,
					VCP:               "R31",
					Mode:              "Clear Air",
					Status:            "Online",
					OperabilityStatus: "Normal",
					PowerSource:       "Utility",
					GenState:          "Off",
				}
				monitor.mu.Unlock()
			}
		}
	})
}

// BenchmarkConfigValidation tests configuration validation performance
func BenchmarkConfigValidation(b *testing.B) {
	cfg := &config.Config{
		StationInput:     "KATX,KRAX,KBGM",
		PushoverAPIToken: "test-token",
		PushoverUserKey:  "test-key",
		DryRun:           false,
		CheckInterval:    10 * time.Minute,
		AlertConfig: config.AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: false,
			PowerSource: false,
			GenState:    false,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}
