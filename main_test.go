package main

import (
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/config"
	"github.com/jacaudi/dras/internal/monitor"
	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
)

func TestMainIntegration(t *testing.T) {
	// Since main() calls log.Fatalf on errors and runs indefinitely,
	// we can't directly test the main function. Instead, we'll test
	// the integration components that main() uses.

	t.Run("config loading and validation", func(t *testing.T) {
		// Set up test environment for dry run
		t.Setenv("DRYRUN", "true")
		t.Setenv("INTERVAL", "1") // 1 minute for testing

		// Test that config loads successfully
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Config load failed: %v", err)
		}

		// Test that validation passes for dry run
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Config validation failed: %v", err)
		}

		// Verify dry run settings
		if !cfg.DryRun {
			t.Error("Expected DryRun to be true")
		}

		if cfg.CheckInterval != 1*time.Minute {
			t.Errorf("Expected CheckInterval to be 1m, got %v", cfg.CheckInterval)
		}
	})

	t.Run("service initialization", func(t *testing.T) {
		t.Run("dry run mode - no notification service", func(t *testing.T) {
			t.Setenv("DRYRUN", "true")

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Config load failed: %v", err)
			}

			// Initialize services like main() does
			radarService := radar.New()
			if radarService == nil {
				t.Error("Expected radar service to be initialized")
			}

			var notifyService *notify.Service
			if !cfg.DryRun {
				notifyService = notify.New(cfg.PushoverAPIToken, cfg.PushoverUserKey)
			}

			// In dry run mode, notify service should be nil
			if notifyService != nil {
				t.Error("Expected notify service to be nil in dry run mode")
			}

			// Initialize monitor
			monitorService := monitor.New(radarService, notifyService, cfg)
			if monitorService == nil {
				t.Error("Expected monitor service to be initialized")
			}
		})

		t.Run("production mode - with notification service", func(t *testing.T) {
			t.Setenv("DRYRUN", "false")
			t.Setenv("PUSHOVER_API_TOKEN", "abcdefghijklmnopqrstuvwxyz1234") // 30 alphanumeric chars
			t.Setenv("PUSHOVER_USER_KEY", "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234")  // 30 alphanumeric chars
			t.Setenv("STATION_IDS", "KATX")
			t.Setenv("INTERVAL", "1") // 1 minute for testing

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Config load failed: %v", err)
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("Config validation failed: %v", err)
			}

			// Initialize services like main() does
			radarService := radar.New()
			if radarService == nil {
				t.Error("Expected radar service to be initialized")
			}

			var notifyService *notify.Service
			if !cfg.DryRun {
				notifyService = notify.New(cfg.PushoverAPIToken, cfg.PushoverUserKey)
			}

			// In production mode, notify service should be initialized
			if notifyService == nil {
				t.Error("Expected notify service to be initialized in production mode")
			}

			// Initialize monitor
			monitorService := monitor.New(radarService, notifyService, cfg)
			if monitorService == nil {
				t.Error("Expected monitor service to be initialized")
			}
		})
	})

	t.Run("error conditions", func(t *testing.T) {
		t.Run("missing required config in production mode", func(t *testing.T) {
			// Clear any inherited env vars by setting to empty (config uses os.Getenv)
			t.Setenv("STATION_IDS", "")
			t.Setenv("PUSHOVER_API_TOKEN", "")
			t.Setenv("PUSHOVER_USER_KEY", "")
			t.Setenv("DRYRUN", "false")
			t.Setenv("INTERVAL", "1") // 1 minute for testing

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Config load failed: %v", err)
			}

			// Validation should fail
			if err := cfg.Validate(); err == nil {
				t.Error("Expected validation to fail when missing required config")
			}
		})
	})
}
