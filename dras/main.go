package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/image"
	"github.com/jacaudi/dras/internal/monitor"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
	"github.com/jacaudi/dras/internal/renderer"
	"github.com/jacaudi/dras/internal/version"
	"github.com/jacaudi/nws/cmd/nws"
)

// The program checks environment variables, initializes services, and starts the monitoring service.
// If the minuteInterval is not set, it defaults to 10 minutes.
// If dryrun is enabled, it uses test radar sites for monitoring.
// Otherwise, it sanitizes the station IDs provided by the user.
// It sets the UserAgent to the DRAS GitHub repository and fetches and reports radar data.
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fatal("Error loading configuration: %v", err)
	}

	// Configure structured logging from $LOG_LEVEL (debug|info|warn|error) and
	// $LOG_FORMAT (text|json), both case-insensitive. Defaults: level=info,
	// format=text. Routes through slog's stdlib handlers — no custom code.
	slog.SetDefault(newLogger(cfg.LogLevel, os.Getenv("LOG_FORMAT"), os.Stdout))

	// Display version information
	versionInfo := version.Get()
	slog.Info(fmt.Sprintf("Starting %s", versionInfo.String()))

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fatal("Configuration validation failed: %v", err)
	}

	// Display runtime configuration (but mask sensitive values)
	slog.Info("Configuration loaded successfully")
	slog.Debug("Runtime configuration",
		"dry_run", fmt.Sprintf("%t", cfg.DryRun),
		"check_interval", cfg.CheckInterval.String(),
		"log_level", cfg.LogLevel,
	)

	// Set NWS UserAgent
	userAgent := fmt.Sprintf("dras/%s (+https://github.com/jacaudi/dras)", versionInfo.Version)
	slog.Info(fmt.Sprintf("Setting NWS UserAgent to %s", userAgent))
	nwsConfig := nws.Config{}
	nwsConfig.SetUserAgent(userAgent)

	// Initialize services
	radarService := radar.New()
	var notifyService *notify.Service
	if !cfg.DryRun {
		slog.Debug("Initializing notification service")
		notifyService = notify.New(cfg.PushoverAPIToken, cfg.PushoverUserKey)

		// Validate Pushover credentials
		if err := notifyService.ValidateCredentials(); err != nil {
			fatal("Pushover credentials validation failed: %v", err)
		}
		slog.Info("Pushover credentials validated successfully")
	} else {
		slog.Info("Running in dry-run mode, notifications disabled")
	}

	// Initialize image source. Three mutually-exclusive modes:
	//   - Advanced (RENDERER_URL set): HTTP renderer service.
	//   - Basic   (RENDERER_URL empty, RadarImageEnabled true): legacy ridge GIF fetcher.
	//   - Disabled (neither): no image attached to notifications.
	var imageSource image.Source
	switch {
	case cfg.RendererURL != "":
		// Modes are mutually exclusive. If basic-mode settings were also
		// configured, log a warning so operators don't silently rely on
		// values that have no effect in advanced mode.
		if cfg.RadarImageURLTmpl != "" || cfg.RadarImageRetention != 0 {
			slog.Info("RENDERER_URL is set; basic-mode RADAR_IMAGE_* settings are ignored",
				"renderer_url", cfg.RendererURL,
				"radar_image_url_template", cfg.RadarImageURLTmpl,
				"radar_image_retention", cfg.RadarImageRetention.String(),
			)
		}
		imageSource = renderer.New(renderer.Config{
			BaseURL:   cfg.RendererURL,
			Timeout:   cfg.RendererTimeout,
			UserAgent: userAgent,
		})
		slog.Info("Radar image source enabled",
			"mode", "advanced",
			"renderer_url", cfg.RendererURL,
			"renderer_timeout", cfg.RendererTimeout.String(),
		)

	case cfg.RadarImageEnabled:
		svc := image.New(image.Config{
			URLTemplate: cfg.RadarImageURLTmpl,
			Retention:   cfg.RadarImageRetention,
			UserAgent:   userAgent,
		})
		imageSource = svc

		var pollStations []string
		if cfg.DryRun {
			pollStations = []string{"KATX", "KRAX"}
		} else {
			pollStations = radar.SanitizeStationIDs(cfg.StationInput)
		}
		pollURLs := make([]string, len(pollStations))
		for i, s := range pollStations {
			pollURLs[i] = svc.URLFor(s)
		}
		slog.Info("Radar image source enabled",
			"mode", "basic",
			"stations", strings.Join(pollStations, ","),
			"urls", strings.Join(pollURLs, ","),
			"retention", cfg.RadarImageRetention.String(),
		)

	default:
		slog.Info("Radar image source disabled")
	}

	// Initialize monitor
	monitorService := monitor.New(radarService, notifyService, imageSource, cfg)

	// Start monitoring
	slog.Info("Starting radar monitoring service")
	ctx := context.Background()
	if err := monitorService.Start(ctx); err != nil {
		fatal("Error starting monitor: %v", err)
	}
}
