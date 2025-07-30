package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

// Monitor handles the monitoring logic for radar stations.
type Monitor struct {
	radarService  *radar.Service
	notifyService *notify.Service
	config        *config.Config
	radarDataMap  map[string]map[string]interface{}
	mu            sync.Mutex
}

// New creates a new monitor instance.
func New(radarService *radar.Service, notifyService *notify.Service, cfg *config.Config) *Monitor {
	return &Monitor{
		radarService:  radarService,
		notifyService: notifyService,
		config:        cfg,
		radarDataMap:  make(map[string]map[string]interface{}),
	}
}

// Start begins the monitoring process with the specified context.
func (m *Monitor) Start(ctx context.Context) error {
	var stationIDs []string

	log.Println("DRAS -- Start Monitoring Service")
	if m.config.DryRun {
		stationIDs = []string{"KATX", "KRAX"} // Test with Seattle, WA & Raleigh, NC Radar Sites
	} else {
		stationIDs = radar.SanitizeStationIDs(m.config.StationInput)
	}

	// Initial fetch
	m.fetchAndReportRadarData(ctx, stationIDs)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			log.Println("DRAS -- Updating Radar Data")
			m.fetchAndReportRadarData(ctx, stationIDs)
		}
	}
}

// fetchAndReportRadarData fetches radar data for a list of station IDs and reports any changes in the data.
// The fetched data is compared with the last stored data for each station ID, and if there are changes a
// push notification is sent using the notification service.
// The radar data is stored in the radarDataMap in memory.
// Goroutines are used to perform the api call and data processing per station ID
func (m *Monitor) fetchAndReportRadarData(ctx context.Context, stationIDs []string) {
	var wg sync.WaitGroup

	for _, stationID := range stationIDs {
		wg.Add(1)
		go func(stationID string) {
			defer wg.Done()
			if err := m.processStation(ctx, stationID); err != nil {
				log.Printf("Error processing station %s: %v\n", stationID, err)
			}
		}(stationID)
	}

	wg.Wait()
}

// processStation handles the processing of a single radar station.
func (m *Monitor) processStation(ctx context.Context, stationID string) error {
	log.Printf("Fetching radar data for station: %s\n", stationID)
	newRadarData, err := m.radarService.FetchData(stationID)
	if err != nil {
		return fmt.Errorf("error fetching radar data for station %s: %w", stationID, err)
	}

	mode, err := radar.GetMode(newRadarData.VCP)
	if err != nil {
		return fmt.Errorf("error determining radar mode for station %s: %w", stationID, err)
	}

	// Check if we need to initialize or if this is first run
	m.mu.Lock()
	if _, exists := m.radarDataMap[stationID]; !exists {
		m.radarDataMap[stationID] = make(map[string]interface{})
	}
	lastRadarData, exists := m.radarDataMap[stationID]["last"]
	isFirstRun := !exists || lastRadarData == nil
	if isFirstRun {
		m.radarDataMap[stationID]["last"] = newRadarData
	}
	m.mu.Unlock()

	// Handle first run outside of mutex
	if isFirstRun {
		initialMessage := fmt.Sprintf("%s %s - %s Mode", stationID, newRadarData.Name, mode)
		log.Printf("Initial radar data stored for station %s.", stationID)
		if m.config.DryRun {
			log.Printf("Debug Pushover -- Title: DRAS Startup - Msg: %s\n", initialMessage)
		} else {
			if err := m.notifyService.SendNotification(ctx, "DRAS Startup", initialMessage); err != nil {
				return fmt.Errorf("error sending Pushover alert for station %s: %w", stationID, err)
			}
		}
		return nil
	}

	// Compare with previous data
	lastData, ok := lastRadarData.(*radar.Data)
	if !ok {
		return fmt.Errorf("invalid radar data type in cache for station %s", stationID)
	}

	// Convert config.AlertConfig to radar.AlertConfig
	alertConfig := radar.AlertConfig{
		VCP:         m.config.AlertConfig.VCP,
		Status:      m.config.AlertConfig.Status,
		Operability: m.config.AlertConfig.Operability,
		PowerSource: m.config.AlertConfig.PowerSource,
		GenState:    m.config.AlertConfig.GenState,
	}

	changed, changeMessage := radar.CompareData(lastData, newRadarData, alertConfig)
	if changed {
		log.Printf("Radar data changed for station %s %s: %s\n", stationID, newRadarData.Name, changeMessage)
		if m.config.DryRun {
			log.Printf("Debug Pushover -- Title: %s - Msg: %s\n", stationID, changeMessage)
		} else {
			if err := m.notifyService.SendNotification(ctx, fmt.Sprintf("%s Update", stationID), changeMessage); err != nil {
				return fmt.Errorf("error sending Pushover alert for station %s: %w", stationID, err)
			}
		}
		m.mu.Lock()
		m.radarDataMap[stationID]["last"] = newRadarData
		m.mu.Unlock()
	} else {
		log.Printf("No changes in radar data for station %s\n", stationID)
	}

	return nil
}
