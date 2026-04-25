package config

import (
	"strings"
	"testing"
	"time"

	"github.com/jacaudi/dras/internal/radar"
)

func TestLoad(t *testing.T) {
	// Keys that may be inherited from CI/dev environment and need explicit clearing.
	// config.Load() uses os.Getenv() (not os.LookupEnv), so empty string == unset.
	envKeys := []string{
		"STATION_IDS",
		"PUSHOVER_API_TOKEN",
		"PUSHOVER_USER_KEY",
		"DRYRUN",
		"INTERVAL",
		"ALERT_VCP",
		"ALERT_STATUS",
		"ALERT_OPERABILITY",
		"ALERT_POWER_SOURCE",
		"ALERT_GEN_STATE",
		"RADAR_IMAGE_ENABLED",
		"RADAR_IMAGE_URL_TEMPLATE",
		"RADAR_IMAGE_RETENTION",
	}

	clearEnv := func(t *testing.T) {
		t.Helper()
		for _, key := range envKeys {
			t.Setenv(key, "")
		}
	}

	t.Run("loads default configuration", func(t *testing.T) {
		clearEnv(t)

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
		if !cfg.RadarImageEnabled {
			t.Errorf("Expected RadarImageEnabled to be true by default")
		}
		if cfg.RadarImageURLTmpl != "" {
			t.Errorf("Expected RadarImageURLTmpl to default to empty (template applied at New), got %q", cfg.RadarImageURLTmpl)
		}
		if cfg.RadarImageRetention != time.Hour {
			t.Errorf("Expected RadarImageRetention to default to 1h, got %v", cfg.RadarImageRetention)
		}
	})

	t.Run("loads custom radar image configuration", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DRYRUN", "true")
		t.Setenv("RADAR_IMAGE_ENABLED", "false")
		t.Setenv("RADAR_IMAGE_URL_TEMPLATE", "https://example.com/{station}.png")
		t.Setenv("RADAR_IMAGE_RETENTION", "30m")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		if cfg.RadarImageEnabled {
			t.Errorf("Expected RadarImageEnabled=false")
		}
		if cfg.RadarImageURLTmpl != "https://example.com/{station}.png" {
			t.Errorf("Expected custom URL template, got %q", cfg.RadarImageURLTmpl)
		}
		if cfg.RadarImageRetention != 30*time.Minute {
			t.Errorf("Expected RadarImageRetention=30m, got %v", cfg.RadarImageRetention)
		}
	})

	t.Run("handles invalid RADAR_IMAGE_RETENTION value", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("RADAR_IMAGE_RETENTION", "not-a-duration")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid RADAR_IMAGE_RETENTION value")
		}
	})

	t.Run("handles invalid RADAR_IMAGE_ENABLED value", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("RADAR_IMAGE_ENABLED", "maybe")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid RADAR_IMAGE_ENABLED value")
		}
	})

	t.Run("loads custom configuration", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("STATION_IDS", "KATX,KRAX")
		t.Setenv("PUSHOVER_API_TOKEN", "test-token")
		t.Setenv("PUSHOVER_USER_KEY", "test-key")
		t.Setenv("DRYRUN", "true")
		t.Setenv("INTERVAL", "5")
		t.Setenv("ALERT_VCP", "false")
		t.Setenv("ALERT_STATUS", "true")

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
		clearEnv(t)
		t.Setenv("DRYRUN", "invalid")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid DRYRUN value")
		}
	})

	t.Run("handles invalid INTERVAL value", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("INTERVAL", "invalid")

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
			CheckInterval:    5 * time.Minute,                  // Ensure minimum interval is met
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

	t.Run("fails when image enabled with non-positive retention", func(t *testing.T) {
		cfg := &Config{
			DryRun:              true,
			CheckInterval:       5 * time.Minute,
			RadarImageEnabled:   true,
			RadarImageRetention: 0,
		}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("Validation should fail when retention is 0 with image polling enabled")
		}
		if !strings.Contains(err.Error(), "RADAR_IMAGE_RETENTION") {
			t.Errorf("error %q should mention RADAR_IMAGE_RETENTION", err)
		}
	})

	t.Run("passes when image disabled regardless of retention", func(t *testing.T) {
		cfg := &Config{
			DryRun:              true,
			CheckInterval:       5 * time.Minute,
			RadarImageEnabled:   false,
			RadarImageRetention: 0,
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validation should pass when image polling is disabled, got %v", err)
		}
	})
}

func TestMaskString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		show     int
		expected string
	}{
		{"empty string", "", 6, ""},
		{"shorter than show", "abc", 6, "***"},
		{"equal to show", "abcdef", 6, "******"},
		{"longer than show", "abcdefghij", 6, "abcdef****"},
		{"show zero", "secret", 0, "******"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := maskString(tc.input, tc.show)
			if got != tc.expected {
				t.Errorf("maskString(%q, %d) = %q, want %q", tc.input, tc.show, got, tc.expected)
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	t.Run("dry run mode", func(t *testing.T) {
		cfg := &Config{
			DryRun:        true,
			CheckInterval: 10 * time.Minute,
			LogLevel:      "INFO",
			AlertConfig:   radar.AlertConfig{VCP: true},
		}
		out := cfg.String()

		if !strings.Contains(out, "Dry Run: true") {
			t.Error("expected 'Dry Run: true' in output")
		}
		if !strings.Contains(out, "Pushover: disabled (dry run)") {
			t.Error("expected dry-run pushover notice")
		}
		if !strings.Contains(out, "KATX,KRAX (test mode)") {
			t.Error("expected test-mode station IDs")
		}
		if !strings.Contains(out, "Alert Types: VCP") {
			t.Error("expected VCP in alert types")
		}
	})

	t.Run("production mode masks credentials", func(t *testing.T) {
		cfg := &Config{
			DryRun:           false,
			StationInput:     "KATX",
			PushoverAPIToken: "abcdefghijklmnop",
			PushoverUserKey:  "ABCDEFGHIJKLMNOP",
			CheckInterval:    5 * time.Minute,
			LogLevel:         "DEBUG",
			AlertConfig: radar.AlertConfig{
				VCP:         true,
				Status:      true,
				Operability: true,
				PowerSource: true,
				GenState:    true,
			},
		}
		out := cfg.String()

		if !strings.Contains(out, "Station IDs: KATX") {
			t.Error("expected station IDs in output")
		}
		if strings.Contains(out, "abcdefghijklmnop") {
			t.Error("full API token should not appear in output")
		}
		if !strings.Contains(out, "abcdef**********") {
			t.Errorf("expected masked API token, got: %s", out)
		}
		if !strings.Contains(out, "VCP, Status, Operability, PowerSource, GenState") {
			t.Errorf("expected all alert types in output, got: %s", out)
		}
	})

	t.Run("no alert types", func(t *testing.T) {
		cfg := &Config{
			DryRun:      true,
			AlertConfig: radar.AlertConfig{},
		}
		out := cfg.String()

		if !strings.Contains(out, "Alert Types: none") {
			t.Error("expected 'Alert Types: none' when no alerts enabled")
		}
	})
}
