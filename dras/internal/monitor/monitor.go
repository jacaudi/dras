package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/image"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

// Monitor handles the monitoring logic for radar stations.
type Monitor struct {
	radarService  radar.DataFetcher
	notifyService notify.Notifier
	imageService  image.Source
	config        *config.Config
	radarDataMap  map[string]map[string]interface{}
	mu            sync.Mutex
}

// New creates a new monitor instance. imageService may be nil to disable
// fetching and attaching radar images. notifyService may also be nil when
// running in dry-run mode.
func New(radarService radar.DataFetcher, notifyService notify.Notifier, imageService image.Source, cfg *config.Config) *Monitor {
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

	slog.Info("Starting monitoring service")
	if m.config.DryRun {
		stationIDs = []string{"KATX", "KRAX"} // Test with Seattle, WA & Raleigh, NC Radar Sites
		slog.Info(fmt.Sprintf("Running in dry-run mode with test stations: %v", stationIDs))
	} else {
		stationIDs = radar.SanitizeStationIDs(m.config.StationInput)
		slog.Info(fmt.Sprintf("Monitoring %d stations: %v", len(stationIDs), stationIDs))
	}

	// Initial fetch
	slog.Info("Performing initial radar data fetch")
	m.fetchAndReportRadarData(ctx, stationIDs)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	slog.Info(fmt.Sprintf("Monitoring started, checking every %v", m.config.CheckInterval))
	for {
		select {
		case <-ctx.Done():
			slog.Info(fmt.Sprintf("Monitoring stopped: %v", ctx.Err()))
			return ctx.Err()
		case <-ticker.C:
			slog.Debug("Performing periodic radar data update")
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
				slog.Error(fmt.Sprintf("Failed to process station: %v", err), "station", stationID)
			}
		}(stationID)
	}

	wg.Wait()
}

// processStation handles the processing of a single radar station.
//
// The image source is invoked lazily — only when a notification will
// carry the freshly-rendered image (first run, or a VCP change). The
// previous "fetch every poll, attach on change" pattern made the
// renderer absorb a request per station per CheckInterval (~12/hr per
// station with the 5 min default) just to discard most of them. Only
// poll the renderer when the result will reach a user.
func (m *Monitor) processStation(ctx context.Context, stationID string) error {
	stationLogger := slog.Default().With("station", stationID)
	stationLogger.Debug("Fetching radar data")
	newRadarData, err := m.radarService.FetchData(stationID)
	if err != nil {
		return fmt.Errorf("error fetching radar data for station %s: %w", stationID, err)
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
		initialMessage := fmt.Sprintf("%s %s - %s Mode", stationID, newRadarData.Name, newRadarData.Mode)
		stationLogger.Info(fmt.Sprintf("Initial radar data stored - %s", initialMessage))
		if m.config.DryRun {
			stationLogger.Debug(fmt.Sprintf("Would send startup notification: %s", initialMessage))
		} else {
			// First run always carries the freshly-rendered image so
			// the user has visual context the moment the monitor
			// comes online.
			radarImage := m.fetchRadarImage(ctx, stationID, stationLogger)
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
	if !changed {
		stationLogger.Debug("No changes detected in radar data")
		return nil
	}

	slog.Info("Radar data changed",
		"station", stationID,
		"station_name", newRadarData.Name,
		"change", changeMessage,
	)

	vcpChanged := lastData.VCP != newRadarData.VCP

	if m.config.DryRun {
		stationLogger.Debug(fmt.Sprintf("Would send change notification: %s", changeMessage))
	} else {
		// Only invoke the image source when the notification will
		// actually carry an attachment (currently: VCP changes only).
		// Other changes — power source, mode without VCP shift, etc.
		// — reach the user as text-only and don't justify a render.
		var radarImage *image.Image
		if vcpChanged {
			radarImage = m.fetchRadarImage(ctx, stationID, stationLogger)
		}
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

	return nil
}

// fetchRadarImage downloads and caches the latest radar image for the given
// station. Returns nil if image fetching is disabled or the download fails.
func (m *Monitor) fetchRadarImage(ctx context.Context, stationID string, stationLogger *slog.Logger) *image.Image {
	if m.imageService == nil {
		return nil
	}

	img, err := m.imageService.Fetch(ctx, stationID)
	if err != nil {
		stationLogger.Warn(fmt.Sprintf("Failed to fetch radar image: %v", err))
		return nil
	}
	return img
}

// attachmentForChange returns the radar image to attach to a change
// notification, or nil when no attachment should be sent. Images are only
// attached when the VCP changed, matching the user-facing feature scope.
func (m *Monitor) attachmentForChange(stationID string, vcpChanged bool, justFetched *image.Image, stationLogger *slog.Logger) *notify.Attachment {
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
