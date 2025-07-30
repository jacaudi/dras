// +build race

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

// These tests are designed to be run with the race detector enabled
// Run with: go test -race ./internal/monitor

func TestRaceConditions(t *testing.T) {
	cfg := &config.Config{
		DryRun:        true,
		CheckInterval: 1 * time.Millisecond,
		AlertConfig: config.AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		},
	}

	t.Run("concurrent map access race detection", func(t *testing.T) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		stations := []string{"KATX", "KRAX", "KBGM", "KTLX", "KFFC"}
		var wg sync.WaitGroup

		// This test will detect races if the mutex isn't used properly
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				stationID := stations[iteration%len(stations)]
				
				// Simulate the operations that happen in processStation
				monitor.mu.Lock()
				if _, exists := monitor.radarDataMap[stationID]; !exists {
					monitor.radarDataMap[stationID] = make(map[string]interface{})
				}
				lastRadarData, exists := monitor.radarDataMap[stationID]["last"]
				isFirstRun := !exists || lastRadarData == nil
				
				newData := &radar.Data{
					Name:              stationID,
					VCP:               "R31",
					Mode:              "Clear Air",
					Status:            "Online",
					OperabilityStatus: "Normal",
					PowerSource:       "Utility",
					GenState:          "Off",
				}
				
				if isFirstRun {
					monitor.radarDataMap[stationID]["last"] = newData
				} else {
					// Simulate comparison and update
					if lastData, ok := lastRadarData.(*radar.Data); ok {
						if lastData.VCP != newData.VCP {
							monitor.radarDataMap[stationID]["last"] = newData
						}
					}
				}
				monitor.mu.Unlock()
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent read/write race detection", func(t *testing.T) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		// Initialize some data
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

		// Start multiple readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					monitor.mu.Lock()
					if data, exists := monitor.radarDataMap["KATX"]["last"]; exists {
						if radarData, ok := data.(*radar.Data); ok {
							_ = radarData.VCP // Read access
							_ = radarData.Status // Read access
						}
					}
					monitor.mu.Unlock()
				}
			}()
		}

		// Start multiple writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					vcps := []string{"R31", "R35", "R12", "R112", "R212", "R215"}
					statuses := []string{"Online", "Offline", "Maintenance"}
					
					newData := &radar.Data{
						Name:              "KATX",
						VCP:               vcps[j%len(vcps)],
						Mode:              "Clear Air",
						Status:            statuses[j%len(statuses)],
						OperabilityStatus: "Normal",
						PowerSource:       "Utility",
						GenState:          "Off",
					}
					
					monitor.mu.Lock()
					monitor.radarDataMap["KATX"]["last"] = newData // Write access
					monitor.mu.Unlock()
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("stress test with high concurrency", func(t *testing.T) {
		radarService := radar.New()
		var notifyService *notify.Service
		monitor := New(radarService, notifyService, cfg)

		stations := []string{"KATX", "KRAX", "KBGM", "KTLX", "KFFC", "KJAX", "KMLB", "KTBW", "KMHX", "KAKQ"}
		var wg sync.WaitGroup

		// High concurrency stress test
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				stationID := stations[iteration%len(stations)]
				
				monitor.mu.Lock()
				if _, exists := monitor.radarDataMap[stationID]; !exists {
					monitor.radarDataMap[stationID] = make(map[string]interface{})
				}
				
				// Simulate rapid updates
				for k := 0; k < 10; k++ {
					monitor.radarDataMap[stationID]["last"] = &radar.Data{
						Name:              stationID,
						VCP:               "R31",
						Mode:              "Clear Air",
						Status:            "Online",
						OperabilityStatus: "Normal",
						PowerSource:       "Utility",
						GenState:          "Off",
					}
				}
				monitor.mu.Unlock()
			}(i)
		}

		wg.Wait()
	})
}

// TestRaceWithRealMonitoring tests race conditions during actual monitoring
func TestRaceWithRealMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	cfg := &config.Config{
		DryRun:        true,
		CheckInterval: 1 * time.Millisecond, // Very fast to trigger races
		AlertConfig: config.AlertConfig{
			VCP: true,
		},
	}

	radarService := radar.New()
	var notifyService *notify.Service
	monitor := New(radarService, notifyService, cfg)

	// Start the monitor in a goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- monitor.Start(ctx)
	}()

	// While the monitor is running, try to access the data concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				monitor.mu.Lock()
				for stationID := range monitor.radarDataMap {
					if data, exists := monitor.radarDataMap[stationID]["last"]; exists {
						if radarData, ok := data.(*radar.Data); ok {
							_ = radarData.Name
						}
					}
				}
				monitor.mu.Unlock()
				time.Sleep(100 * time.Microsecond)
			}
		}()
	}

	wg.Wait()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Unexpected error from monitor: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Monitor did not return within expected time")
	}
}