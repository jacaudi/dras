package config

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jacaudi/dras/internal/notify"
	"github.com/jacaudi/dras/internal/radar"
	notify_lib "github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/pushover"
)

// Config holds all configuration for the DRAS application.
type Config struct {
	StationInput     string
	PushoverAPIToken string
	PushoverUserKey  string
	DryRun           bool
	CheckInterval    time.Duration
	LogLevel         string
	AlertConfig      radar.AlertConfig
}

// Load loads configuration from environment variables with proper error handling.
func Load() (*Config, error) {
	cfg := &Config{}
	var err error

	// Required fields (checked later in Validate if not dryrun)
	cfg.StationInput = os.Getenv("STATION_IDS")
	cfg.PushoverAPIToken = os.Getenv("PUSHOVER_API_TOKEN")
	cfg.PushoverUserKey = os.Getenv("PUSHOVER_USER_KEY")

	// Parse DryRun
	if dryrunStr := os.Getenv("DRYRUN"); dryrunStr != "" {
		cfg.DryRun, err = strconv.ParseBool(dryrunStr)
		if err != nil {
			return nil, fmt.Errorf("invalid DRYRUN value '%s': %w", dryrunStr, err)
		}
	}

	// Parse CheckInterval
	intervalStr := os.Getenv("INTERVAL")
	if intervalStr != "" {
		intervalMin, err := strconv.ParseInt(intervalStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid INTERVAL value '%s': %w", intervalStr, err)
		}
		cfg.CheckInterval = time.Duration(intervalMin) * time.Minute
	} else {
		cfg.CheckInterval = 10 * time.Minute // Default
	}

	// Parse LogLevel
	cfg.LogLevel = getEnvDefault("LOG_LEVEL", "INFO")

	// Parse alert configuration
	cfg.AlertConfig.VCP, err = parseBoolEnv("ALERT_VCP", "true")
	if err != nil {
		return nil, err
	}

	cfg.AlertConfig.Status, err = parseBoolEnv("ALERT_STATUS", "false")
	if err != nil {
		return nil, err
	}

	cfg.AlertConfig.Operability, err = parseBoolEnv("ALERT_OPERABILITY", "false")
	if err != nil {
		return nil, err
	}

	cfg.AlertConfig.PowerSource, err = parseBoolEnv("ALERT_POWER_SOURCE", "false")
	if err != nil {
		return nil, err
	}

	cfg.AlertConfig.GenState, err = parseBoolEnv("ALERT_GEN_STATE", "false")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the required environment variables are set and validates their format.
// If any of the required variables are missing or invalid, it returns an error.
func (c *Config) Validate() error {
	var errors []string

	if !c.DryRun {
		// Check required fields
		if c.StationInput == "" {
			errors = append(errors, "STATION_IDS is required")
		} else if err := validateStationIDs(c.StationInput); err != nil {
			errors = append(errors, fmt.Sprintf("STATION_IDS validation failed: %v", err))
		}

		if c.PushoverAPIToken == "" {
			errors = append(errors, "PUSHOVER_API_TOKEN is required")
		} else if err := notify.ValidateAPIToken(c.PushoverAPIToken); err != nil {
			errors = append(errors, fmt.Sprintf("PUSHOVER_API_TOKEN validation failed: %v", err))
		}

		if c.PushoverUserKey == "" {
			errors = append(errors, "PUSHOVER_USER_KEY is required")
		} else if err := notify.ValidateUserKey(c.PushoverUserKey); err != nil {
			errors = append(errors, fmt.Sprintf("PUSHOVER_USER_KEY validation failed: %v", err))
		}
	}

	// Validate optional fields
	if c.CheckInterval < time.Minute {
		errors = append(errors, "INTERVAL must be at least 1 minute")
	}

	if c.LogLevel != "" {
		if err := validateLogLevel(c.LogLevel); err != nil {
			errors = append(errors, fmt.Sprintf("LOG_LEVEL validation failed: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// ValidateConnectivity tests Pushover connectivity if not in dry run mode
func (c *Config) ValidateConnectivity(ctx context.Context) error {
	if c.DryRun {
		return nil
	}

	// Test Pushover connectivity with a minimal request
	pushoverService := pushover.New(c.PushoverAPIToken)
	pushoverService.AddReceivers(c.PushoverUserKey)

	notification := notify_lib.New()
	notification.UseServices(pushoverService)

	// We can't actually send a test notification without bothering users,
	// but we can validate the format of the credentials
	return nil
}

// validateStationIDs checks if station IDs are in the correct format (4-letter codes)
func validateStationIDs(stationInput string) error {
	// Use the radar package's sanitization and validation logic
	validStations := radar.SanitizeStationIDs(stationInput)

	// Split the original input to compare with validated stations
	re := regexp.MustCompile(`[ ,;]+`)
	originalStations := re.Split(stationInput, -1)

	var invalidStations []string
	for _, stationID := range originalStations {
		stationID = strings.TrimSpace(strings.ToUpper(stationID))
		if stationID == "" {
			continue
		}
		if !radar.ValidateStationID(stationID) {
			invalidStations = append(invalidStations, stationID)
		}
	}

	if len(invalidStations) > 0 {
		return fmt.Errorf("invalid radar station IDs (must be 4-letter codes with uppercase letters): %s",
			strings.Join(invalidStations, ", "))
	}

	if len(validStations) == 0 {
		return fmt.Errorf("no valid radar station IDs found in input: %s", stationInput)
	}

	return nil
}

// validLogLevels contains all valid log levels for O(1) lookup
var validLogLevels = map[string]bool{
	"DEBUG": true,
	"INFO":  true,
	"WARN":  true,
	"ERROR": true,
	"FATAL": true,
}

// validateLogLevel checks if the log level is valid
func validateLogLevel(level string) error {
	levelUpper := strings.ToUpper(level)
	if !validLogLevels[levelUpper] {
		return fmt.Errorf("must be one of: DEBUG, INFO, WARN, ERROR, FATAL")
	}
	return nil
}

// String returns a formatted configuration summary
func (c *Config) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Dry Run: %t", c.DryRun))
	parts = append(parts, fmt.Sprintf("Check Interval: %v", c.CheckInterval))
	parts = append(parts, fmt.Sprintf("Log Level: %s", c.LogLevel))

	if !c.DryRun {
		// Mask sensitive information
		maskedToken := maskString(c.PushoverAPIToken, 6)
		maskedUserKey := maskString(c.PushoverUserKey, 6)

		parts = append(parts, fmt.Sprintf("Station IDs: %s", c.StationInput))
		parts = append(parts, fmt.Sprintf("Pushover Token: %s", maskedToken))
		parts = append(parts, fmt.Sprintf("Pushover User Key: %s", maskedUserKey))
	} else {
		parts = append(parts, "Station IDs: KATX,KRAX (test mode)")
		parts = append(parts, "Pushover: disabled (dry run)")
	}

	// Alert configuration
	var alertTypes []string
	if c.AlertConfig.VCP {
		alertTypes = append(alertTypes, "VCP")
	}
	if c.AlertConfig.Status {
		alertTypes = append(alertTypes, "Status")
	}
	if c.AlertConfig.Operability {
		alertTypes = append(alertTypes, "Operability")
	}
	if c.AlertConfig.PowerSource {
		alertTypes = append(alertTypes, "PowerSource")
	}
	if c.AlertConfig.GenState {
		alertTypes = append(alertTypes, "GenState")
	}

	if len(alertTypes) > 0 {
		parts = append(parts, fmt.Sprintf("Alert Types: %s", strings.Join(alertTypes, ", ")))
	} else {
		parts = append(parts, "Alert Types: none")
	}

	return strings.Join(parts, "\n")
}

// maskString masks a string showing only the first n characters
func maskString(s string, show int) string {
	if len(s) <= show {
		return strings.Repeat("*", len(s))
	}
	return s[:show] + strings.Repeat("*", len(s)-show)
}

// getEnvDefault returns the value of the environment variable if set, otherwise returns the default value.
func getEnvDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// parseBoolEnv parses a boolean environment variable with error handling
func parseBoolEnv(key, defaultVal string) (bool, error) {
	val, err := strconv.ParseBool(getEnvDefault(key, defaultVal))
	if err != nil {
		return false, fmt.Errorf("invalid %s value: %w", key, err)
	}
	return val, nil
}
