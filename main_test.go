package main

import (
	"os"
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
		// Save original environment
		origEnv := map[string]string{
			"STATION_IDS":        os.Getenv("STATION_IDS"),
			"PUSHOVER_API_TOKEN": os.Getenv("PUSHOVER_API_TOKEN"),
			"PUSHOVER_USER_KEY":  os.Getenv("PUSHOVER_USER_KEY"),
			"DRYRUN":             os.Getenv("DRYRUN"),
		}

		// Clean environment
		defer func() {
			for key, value := range origEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		}()

		// Set up test environment for dry run
		os.Setenv("DRYRUN", "true")
		os.Setenv("INTERVAL", "1") // 1 minute for testing

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
		// Save original environment
		origEnv := map[string]string{
			"DRYRUN":             os.Getenv("DRYRUN"),
			"PUSHOVER_API_TOKEN": os.Getenv("PUSHOVER_API_TOKEN"),
			"PUSHOVER_USER_KEY":  os.Getenv("PUSHOVER_USER_KEY"),
		}

		// Clean environment
		defer func() {
			for key, value := range origEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		}()

		t.Run("dry run mode - no notification service", func(t *testing.T) {
			os.Setenv("DRYRUN", "true")

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
			os.Setenv("DRYRUN", "false")
			os.Setenv("PUSHOVER_API_TOKEN", "abcdefghijklmnopqrstuvwxyz1234") // 30 alphanumeric chars
			os.Setenv("PUSHOVER_USER_KEY", "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234")  // 30 alphanumeric chars
			os.Setenv("STATION_IDS", "KATX")
			os.Setenv("INTERVAL", "1") // 1 minute for testing

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
		// Save original environment
		origEnv := map[string]string{
			"DRYRUN":             os.Getenv("DRYRUN"),
			"STATION_IDS":        os.Getenv("STATION_IDS"),
			"PUSHOVER_API_TOKEN": os.Getenv("PUSHOVER_API_TOKEN"),
			"PUSHOVER_USER_KEY":  os.Getenv("PUSHOVER_USER_KEY"),
		}

		// Clean environment
		defer func() {
			for key, value := range origEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		}()

		t.Run("missing required config in production mode", func(t *testing.T) {
			// Clear all environment variables
			for key := range origEnv {
				os.Unsetenv(key)
			}
			os.Setenv("DRYRUN", "false")
			os.Setenv("INTERVAL", "1") // 1 minute for testing

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
