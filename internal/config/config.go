package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the DRAS application.
type Config struct {
	StationInput     string
	PushoverAPIToken string
	PushoverUserKey  string
	DryRun           bool
	CheckInterval    time.Duration
	AlertConfig      AlertConfig
}

// AlertConfig holds configuration for which events to alert on.
type AlertConfig struct {
	VCP         bool
	Status      bool
	Operability bool
	PowerSource bool
	GenState    bool
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

	// Parse alert configuration
	cfg.AlertConfig.VCP, err = strconv.ParseBool(getEnvDefault("ALERT_VCP", "true"))
	if err != nil {
		return nil, fmt.Errorf("invalid ALERT_VCP value: %w", err)
	}

	cfg.AlertConfig.Status, err = strconv.ParseBool(getEnvDefault("ALERT_STATUS", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid ALERT_STATUS value: %w", err)
	}

	cfg.AlertConfig.Operability, err = strconv.ParseBool(getEnvDefault("ALERT_OPERABILITY", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid ALERT_OPERABILITY value: %w", err)
	}

	cfg.AlertConfig.PowerSource, err = strconv.ParseBool(getEnvDefault("ALERT_POWER_SOURCE", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid ALERT_POWER_SOURCE value: %w", err)
	}

	cfg.AlertConfig.GenState, err = strconv.ParseBool(getEnvDefault("ALERT_GEN_STATE", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid ALERT_GEN_STATE value: %w", err)
	}

	return cfg, nil
}

// Validate checks if the required environment variables are set.
// If any of the required variables are missing, it returns an error.
func (c *Config) Validate() error {
	var missingVars []string
	if !c.DryRun {
		if c.StationInput == "" {
			missingVars = append(missingVars, "STATION_IDS")
		}
		if c.PushoverAPIToken == "" {
			missingVars = append(missingVars, "PUSHOVER_API_TOKEN")
		}
		if c.PushoverUserKey == "" {
			missingVars = append(missingVars, "PUSHOVER_USER_KEY")
		}

		if len(missingVars) > 0 {
			return fmt.Errorf("the following environment variables are not set: %s", strings.Join(missingVars, ", "))
		}
	}
	return nil
}

// getEnvDefault returns the value of the environment variable if set, otherwise returns the default value.
func getEnvDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
