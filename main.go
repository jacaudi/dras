package main

import (
	"context"
	"fmt"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/logger"
	"github.com/jacaudi/dras/internal/monitor"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
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
		logger.Fatal("Error loading configuration: %v", err)
	}

	// Set up logging level from configuration
	logLevel := logger.ParseLevel(cfg.LogLevel)
	logger.SetDefaultLevel(logLevel)

	// Display version information
	versionInfo := version.Get()
	logger.Info("Starting %s", versionInfo.String())

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatal("Configuration validation failed: %v", err)
	}

	// Display runtime configuration (but mask sensitive values)
	logger.Info("Configuration loaded successfully")
	logger.WithFields(map[string]string{
		"dry_run":        fmt.Sprintf("%t", cfg.DryRun),
		"check_interval": cfg.CheckInterval.String(),
		"log_level":      cfg.LogLevel,
	}).Debug("Runtime configuration")

	// Set NWS UserAgent
	logger.Info("Setting NWS UserAgent to https://github.com/jacaudi/dras")
	nwsConfig := nws.Config{}
	nwsConfig.SetUserAgent("dras/1.0 (+https://github.com/jacaudi/dras)")

	// Initialize services
	radarService := radar.New()
	var notifyService *notify.Service
	if !cfg.DryRun {
		logger.Debug("Initializing notification service")
		notifyService = notify.New(cfg.PushoverAPIToken, cfg.PushoverUserKey)
		
		// Validate Pushover credentials
		if err := notifyService.ValidateCredentials(); err != nil {
			logger.Fatal("Pushover credentials validation failed: %v", err)
		}
		logger.Info("Pushover credentials validated successfully")
	} else {
		logger.Info("Running in dry-run mode, notifications disabled")
	}

	// Initialize monitor
	monitorService := monitor.New(radarService, notifyService, cfg)

	// Start monitoring
	logger.Info("Starting radar monitoring service")
	ctx := context.Background()
	if err := monitorService.Start(ctx); err != nil {
		logger.Fatal("Error starting monitor: %v", err)
	}
}
