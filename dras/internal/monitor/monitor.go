package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/image"
	"github.com/jacaudi/dras/internal/logger"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

// Monitor handles the monitoring logic for radar stations.
type Monitor struct {
	radarService  radar.DataFetcher
	notifyService notify.Notifier
	imageService  *image.Service
	config        *config.Config
	radarDataMap  map[string]map[string]interface{}
	mu            sync.Mutex
}

// New creates a new monitor instance. imageService may be nil to disable
// fetching and attaching radar images. notifyService may also be nil when
// running in dry-run mode.
func New(radarService radar.DataFetcher, notifyService notify.Notifier, imageService *image.Service, cfg *config.Config) *Monitor {
	return &Monitor{
		radarService:  radarService,
		notifyService: notifyService,
		imageService:  imageService,
		config:        cfg,
		radarDataMap:  make(map[string]map[string]interface{}),
	}
}

// Start begins the monitoring process with the specified context.
func (m *Monitor) Start(ctx context.Context) error {
	var stationIDs []string

	logger.Info("Starting monitoring service")
	if m.config.DryRun {
		stationIDs = []string{"KATX", "KRAX"} // Test with Seattle, WA & Raleigh, NC Radar Sites
		logger.Info("Running in dry-run mode with test stations: %v", stationIDs)
	} else {
		stationIDs = radar.SanitizeStationIDs(m.config.StationInput)
		logger.Info("Monitoring %d stations: %v", len(stationIDs), stationIDs)
	}

	// Initial fetch
	logger.Info("Performing initial radar data fetch")
	m.fetchAndReportRadarData(ctx, stationIDs)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	logger.Info("Monitoring started, checking every %v", m.config.CheckInterval)
	for {
		select {
		case <-ctx.Done():
			logger.Info("Monitoring stopped: %v", ctx.Err())
			return ctx.Err()
		case <-ticker.C:
			logger.Debug("Performing periodic radar data update")
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
				logger.WithField("station", stationID).Error("Failed to process station: %v", err)
			}
		}(stationID)
	}

	wg.Wait()
}

// processStation handles the processing of a single radar station.
func (m *Monitor) processStation(ctx context.Context, stationID string) error {
	stationLogger := logger.WithField("station", stationID)
	stationLogger.Debug("Fetching radar data")
	newRadarData, err := m.radarService.FetchData(stationID)
	if err != nil {
		return fmt.Errorf("error fetching radar data for station %s: %w", stationID, err)
	}

	mode, err := radar.GetMode(newRadarData.VCP)
	if err != nil {
		return fmt.Errorf("error determining radar mode for station %s: %w", stationID, err)
	}

	// Poll and store the latest radar image so it can be attached to the
	// next change notification. Image fetch failures are logged but do not
	// fail the whole poll.
	radarImage := m.fetchRadarImage(stationID, stationLogger)

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
		stationLogger.Info("Initial radar data stored - %s", initialMessage)
		if m.config.DryRun {
			stationLogger.Debug("Would send startup notification: %s", initialMessage)
		} else {
			attachment := m.attachmentForStation(stationID, radarImage)
			if err := m.notifyService.SendNotificationWithAttachment(ctx, "DRAS Startup", initialMessage, attachment); err != nil {
				return fmt.Errorf("failed to send startup notification for station %s: %w", stationID, err)
			}
			stationLogger.Info("Startup notification sent successfully")
		}
		return nil
	}

	// Compare with previous data
	lastData, ok := lastRadarData.(*radar.Data)
	if !ok {
		return fmt.Errorf("invalid radar data type in cache for station %s", stationID)
	}

	// Use alert configuration directly (no conversion needed since config uses radar.AlertConfig)
	alertConfig := m.config.AlertConfig

	changed, changeMessage := radar.CompareData(lastData, newRadarData, alertConfig)
	if changed {
		logger.WithFields(map[string]string{
			"station":      stationID,
			"station_name": newRadarData.Name,
			"change":       changeMessage,
		}).Info("Radar data changed")

		vcpChanged := lastData.VCP != newRadarData.VCP

		if m.config.DryRun {
			stationLogger.Debug("Would send change notification: %s", changeMessage)
		} else {
			title := fmt.Sprintf("%s Update", stationID)
			attachment := m.attachmentForChange(stationID, vcpChanged, radarImage, stationLogger)
			if err := m.notifyService.SendNotificationWithAttachment(ctx, title, changeMessage, attachment); err != nil {
				return fmt.Errorf("failed to send change notification for station %s: %w", stationID, err)
			}
			stationLogger.Info("Change notification sent successfully")
		}
		m.mu.Lock()
		m.radarDataMap[stationID]["last"] = newRadarData
		m.mu.Unlock()
	} else {
		stationLogger.Debug("No changes detected in radar data")
	}

	return nil
}

// fetchRadarImage downloads and caches the latest radar image for the given
// station. Returns nil if image fetching is disabled or the download fails.
func (m *Monitor) fetchRadarImage(stationID string, stationLogger *logger.FieldLogger) *image.Image {
	if m.imageService == nil {
		return nil
	}

	img, err := m.imageService.Fetch(stationID)
	if err != nil {
		stationLogger.Warn("Failed to fetch radar image: %v", err)
		return nil
	}
	return img
}

// attachmentForChange returns the radar image to attach to a change
// notification, or nil when no attachment should be sent. Images are only
// attached when the VCP changed, matching the user-facing feature scope.
func (m *Monitor) attachmentForChange(stationID string, vcpChanged bool, justFetched *image.Image, stationLogger *logger.FieldLogger) *notify.Attachment {
	if !vcpChanged {
		return nil
	}
	att := m.attachmentForStation(stationID, justFetched)
	if att == nil {
		stationLogger.Debug("VCP changed but no radar image available to attach")
	}
	return att
}

// attachmentForStation builds a notification attachment from the latest
// available radar image for the station. It falls back to the imageService
// cache if the just-fetched image is nil, and returns nil when no image is
// available or image polling is disabled.
func (m *Monitor) attachmentForStation(stationID string, justFetched *image.Image) *notify.Attachment {
	if m.imageService == nil {
		return nil
	}

	img := justFetched
	if img == nil {
		if cached, ok := m.imageService.Latest(stationID); ok {
			img = cached
		}
	}
	if img == nil {
		return nil
	}

	return &notify.Attachment{
		Data:        img.Data,
		ContentType: img.ContentType,
		Filename:    img.Filename,
	}
}
