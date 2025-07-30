package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

func TestConcurrentStationProcessing(t *testing.T) {
	cfg := &config.Config{
		DryRun:        true,
		CheckInterval: 10 * time.Millisecond,
		AlertConfig: config.AlertConfig{
			VCP: true,
		},
	}

	t.Run("concurrent data access", func(t *testing.T) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		stations := []string{"KATX", "KRAX", "KBGM", "KTLX", "KFFC"}
		var wg sync.WaitGroup

		// Simulate concurrent access to radarDataMap
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				for _, stationID := range stations {
					// Simulate the map operations that happen in processStation
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
			}(i)
		}

		wg.Wait()

		// Verify all stations were processed
		if len(monitor.radarDataMap) != len(stations) {
			t.Errorf("Expected %d stations in map, got %d", len(stations), len(monitor.radarDataMap))
		}
	})

	t.Run("concurrent read/write operations", func(t *testing.T) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		// Pre-populate the data map
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

		var wg sync.WaitGroup
		readCount := 0
		writeCount := 0
		var countMu sync.Mutex

		// Start readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					monitor.mu.Lock()
					if data, exists := monitor.radarDataMap["KATX"]["last"]; exists {
						if radarData, ok := data.(*radar.Data); ok && radarData.Name == "KATX" {
							countMu.Lock()
							readCount++
							countMu.Unlock()
						}
					}
					monitor.mu.Unlock()
					time.Sleep(1 * time.Millisecond)
				}
			}()
		}

		// Start writers
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					newData := &radar.Data{
						Name:              "KATX",
						VCP:               "R12",
						Mode:              "Precipitation",
						Status:            "Online",
						OperabilityStatus: "Normal",
						PowerSource:       "Generator",
						GenState:          "On",
					}
					monitor.mu.Lock()
					monitor.radarDataMap["KATX"]["last"] = newData
					monitor.mu.Unlock()

					countMu.Lock()
					writeCount++
					countMu.Unlock()

					time.Sleep(1 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		countMu.Lock()
		if readCount == 0 {
			t.Error("Expected some reads to occur")
		}
		if writeCount == 0 {
			t.Error("Expected some writes to occur")
		}
		t.Logf("Completed %d reads and %d writes", readCount, writeCount)
		countMu.Unlock()
	})
}

func TestGoroutineErrorHandling(t *testing.T) {
	cfg := &config.Config{
		DryRun: true,
		AlertConfig: config.AlertConfig{
			VCP: true,
		},
	}

	t.Run("goroutine panic recovery", func(t *testing.T) {
		// This test simulates what would happen if a goroutine panicked
		// In our current implementation, goroutines return errors instead of panicking
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		var wg sync.WaitGroup
		errorCount := 0
		var errorMu sync.Mutex

		// Simulate error conditions in goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Simulate the error handling that occurs in processStation
				defer func() {
					if r := recover(); r != nil {
						errorMu.Lock()
						errorCount++
						errorMu.Unlock()
					}
				}()

				// This should not panic in our implementation
				monitor.mu.Lock()
				monitor.radarDataMap["TEST"] = make(map[string]interface{})
				monitor.mu.Unlock()
			}()
		}

		wg.Wait()

		errorMu.Lock()
		if errorCount > 0 {
			t.Errorf("Expected no panics, got %d", errorCount)
		}
		errorMu.Unlock()
	})
}

func TestMonitorStartStop(t *testing.T) {
	cfg := &config.Config{
		DryRun:        true,
		CheckInterval: 100 * time.Millisecond, // Reasonable interval for testing
		AlertConfig: config.AlertConfig{
			VCP: true,
		},
	}

	t.Run("start and cancel", func(t *testing.T) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() {
			done <- monitor.Start(ctx)
		}()

		// Let it run long enough to complete initial processing then cancel
		time.Sleep(2 * time.Second) // Give time for API calls to complete
		cancel()

		select {
		case err := <-done:
			if err != context.Canceled {
				t.Errorf("Expected context.Canceled, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Monitor.Start() did not return within expected time")
		}
	})
}
