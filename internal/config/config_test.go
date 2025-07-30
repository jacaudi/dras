package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment
	origEnv := map[string]string{
		"STATION_IDS":        os.Getenv("STATION_IDS"),
		"PUSHOVER_API_TOKEN": os.Getenv("PUSHOVER_API_TOKEN"),
		"PUSHOVER_USER_KEY":  os.Getenv("PUSHOVER_USER_KEY"),
		"DRYRUN":             os.Getenv("DRYRUN"),
		"INTERVAL":           os.Getenv("INTERVAL"),
		"ALERT_VCP":          os.Getenv("ALERT_VCP"),
		"ALERT_STATUS":       os.Getenv("ALERT_STATUS"),
		"ALERT_OPERABILITY":  os.Getenv("ALERT_OPERABILITY"),
		"ALERT_POWER_SOURCE": os.Getenv("ALERT_POWER_SOURCE"),
		"ALERT_GEN_STATE":    os.Getenv("ALERT_GEN_STATE"),
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

	t.Run("loads default configuration", func(t *testing.T) {
		// Clear all env vars
		for key := range origEnv {
			os.Unsetenv(key)
		}

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		if cfg.CheckInterval != 10*time.Minute {
			t.Errorf("Expected default interval of 10m, got %v", cfg.CheckInterval)
		}
		if !cfg.AlertConfig.VCP {
			t.Errorf("Expected VCP alerts to be true by default")
		}
		if cfg.AlertConfig.Status {
			t.Errorf("Expected Status alerts to be false by default")
		}
	})

	t.Run("loads custom configuration", func(t *testing.T) {
		os.Setenv("STATION_IDS", "KATX,KRAX")
		os.Setenv("PUSHOVER_API_TOKEN", "test-token")
		os.Setenv("PUSHOVER_USER_KEY", "test-key")
		os.Setenv("DRYRUN", "true")
		os.Setenv("INTERVAL", "5")
		os.Setenv("ALERT_VCP", "false")
		os.Setenv("ALERT_STATUS", "true")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		if cfg.StationInput != "KATX,KRAX" {
			t.Errorf("Expected StationInput='KATX,KRAX', got %q", cfg.StationInput)
		}
		if !cfg.DryRun {
			t.Errorf("Expected DryRun=true")
		}
		if cfg.CheckInterval != 5*time.Minute {
			t.Errorf("Expected CheckInterval=5m, got %v", cfg.CheckInterval)
		}
		if cfg.AlertConfig.VCP {
			t.Errorf("Expected VCP alerts to be false")
		}
		if !cfg.AlertConfig.Status {
			t.Errorf("Expected Status alerts to be true")
		}
	})

	t.Run("handles invalid DRYRUN value", func(t *testing.T) {
		os.Setenv("DRYRUN", "invalid")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid DRYRUN value")
		}
	})

	t.Run("handles invalid INTERVAL value", func(t *testing.T) {
		os.Setenv("INTERVAL", "invalid")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid INTERVAL value")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("validates dry run mode", func(t *testing.T) {
		cfg := &Config{
			DryRun:        true,
			CheckInterval: 5 * time.Minute, // Ensure minimum interval is met
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validation should pass for dry run mode: %v", err)
		}
	})

	t.Run("validates required fields for production mode", func(t *testing.T) {
		cfg := &Config{
			DryRun:           false,
			StationInput:     "KATX",
			PushoverAPIToken: "abcdefghijklmnopqrstuvwxyz1234", // 30 alphanumeric chars
			PushoverUserKey:  "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234", // 30 alphanumeric chars  
			CheckInterval:    5 * time.Minute,                   // Ensure minimum interval is met
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validation should pass with all required fields: %v", err)
		}
	})

	t.Run("fails validation when missing required fields", func(t *testing.T) {
		cfg := &Config{
			DryRun:        false,
			CheckInterval: 5 * time.Minute, // Ensure minimum interval is met
		}
		if err := cfg.Validate(); err == nil {
			t.Error("Validation should fail when missing required fields")
		}
	})
}
