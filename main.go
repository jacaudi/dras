package main

import (
	"context"
	"log"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/monitor"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
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
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Set NWS UserAgent
	log.Println("Set UserAgent to https://github.com/jacaudi/dras")
	nwsConfig := nws.Config{}
	nwsConfig.SetUserAgent("dras/1.0 (+https://github.com/jacaudi/dras)")

	// Initialize services
	radarService := radar.New()
	var notifyService *notify.Service
	if !cfg.DryRun {
		notifyService = notify.New(cfg.PushoverAPIToken, cfg.PushoverUserKey)
	}

	// Initialize monitor
	monitorService := monitor.New(radarService, notifyService, cfg)

	// Start monitoring
	ctx := context.Background()
	if err := monitorService.Start(ctx); err != nil {
		log.Fatalf("Error starting monitor: %v", err)
	}
}
