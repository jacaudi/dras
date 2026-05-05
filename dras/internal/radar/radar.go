package radar

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jacaudi/nws/cmd/nws"
)

// ErrUnknownVCP is returned by GetMode/GetVCPInfo when the VCP code is empty
// or not recognized. Callers can use errors.Is to detect this case and treat
// it as a soft condition (use the returned fallback label, log a warning,
// continue) rather than aborting station processing.
var ErrUnknownVCP = errors.New("unknown VCP")

// VCPInfo describes a NEXRAD WSR-88D Volume Coverage Pattern.
type VCPInfo struct {
	Mode        string // Coarse category: "Clear Air" or "Precipitation"
	Description string // Human-readable detail about the scan pattern
	AlertText   string // User-facing message used in change notifications
}

// vcpCatalog maps VCP codes (as reported by the NWS radar metadata API) to
// their coarse mode, a short description, and the alert text shown to users
// when this VCP becomes active. Source: NOAA/NWS WSR-88D operations
// documentation.
var vcpCatalog = map[string]VCPInfo{
	"R31":  {Mode: "Clear Air", Description: "Clear Air, long pulse (~10 min cycle, stratiform/biological targets)", AlertText: "Clear Air Mode Active"},
	"R35":  {Mode: "Clear Air", Description: "Clear Air, short pulse with clutter mitigation (~7 min cycle)", AlertText: "Clear Air Mode Active"},
	"R12":  {Mode: "Precipitation", Description: "Precipitation, rapid evolution (~4.2 min cycle, 14 elevations)", AlertText: "Precipitation Mode Active"},
	"R112": {Mode: "Precipitation", Description: "Precipitation with MRLE (multi-PRF range-folding mitigation)", AlertText: "Precipitation Mode (Velocity Scanning Emphasis) Active"},
	"R212": {Mode: "Precipitation", Description: "Precipitation with SAILS (~4.5 min cycle, common severe-weather VCP)", AlertText: "Precipitation Mode Active"},
	"R215": {Mode: "Precipitation", Description: "Precipitation (~6 min cycle, 15 elevations, tropical/widespread)", AlertText: "Precipitation Mode (Vertical Scanning Emphasis) Active"},
}

// Data represents the data for a radar.
type Data struct {
	Name              string // Name of the radar.
	VCP               string // Volume Coverage Pattern of the radar.
	Mode              string // Scanning mode of the radar.
	Status            string // Status of the radar.
	OperabilityStatus string // Operability Status of the radar.
	PowerSource       string // Power source of the radar.
	GenState          string // General state of the radar.
}

// Service handles radar data operations.
type Service struct {
	// Add HTTP client interface here if needed for testing
}

// New creates a new radar service.
func New() *Service {
	return &Service{}
}

// FetchData retrieves radar data for a given station ID.
// It returns a pointer to a Data struct and an error if any.
func (s *Service) FetchData(stationID string) (*Data, error) {
	radarResponse, err := nws.RadarStation(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RADAR data for station %q: %w", stationID, err)
	}

	// Fetching radar VCP and determine mode. An empty or unrecognized VCP is
	// not fatal: GetMode returns a "Unknown (VCP ...)" fallback label along
	// with ErrUnknownVCP. We log a warning and continue so the rest of the
	// station data (status, generator state, etc.) still gets processed.
	radarVCPCode := radarResponse.RDA.Properties.VolumeCoveragePattern
	radarMode, err := GetMode(radarVCPCode)
	if err != nil {
		if !errors.Is(err, ErrUnknownVCP) {
			return nil, err
		}
		slog.Warn("Unrecognized VCP, using fallback mode label", "station", stationID, "vcp", radarVCPCode, "mode", radarMode)
	}

	// Fetching radar VCP and determine mode
	genStateResponse := radarResponse.RDA.Properties.GeneratorState
	genStateStatement, err := simplifyGeneratorState(genStateResponse) // Converting generator state response to understandable text
	if err != nil {
		return nil, err
	}

	// Constructing the Data structure with both VCP and human-readable translation
	radarData := &Data{
		Name:              radarResponse.Name,
		VCP:               radarVCPCode,
		Mode:              radarMode,
		Status:            radarResponse.RDA.Properties.Status,
		OperabilityStatus: radarResponse.RDA.Properties.OperabilityStatus,
		PowerSource:       radarResponse.Performance.Properties.PowerSource,
		GenState:          genStateStatement,
	}

	return radarData, nil
}

// GetMode returns the radar mode for a given VCP code, looked up from
// vcpCatalog.
//
// If the VCP code is empty or not in the catalog, GetMode returns a fallback
// label of the form `Unknown (VCP %q)` along with an error wrapping
// ErrUnknownVCP. Callers should treat this as a soft condition:
// errors.Is(err, ErrUnknownVCP) → use the fallback label and continue.
func GetMode(vcp string) (string, error) {
	if info, ok := vcpCatalog[vcp]; ok {
		return info.Mode, nil
	}
	fallback := fmt.Sprintf("Unknown (VCP %q)", vcp)
	return fallback, fmt.Errorf("%w: %q", ErrUnknownVCP, vcp)
}

// GetVCPInfo returns the full catalog entry (mode + description) for a VCP
// code. Unknown VCPs return a fallback VCPInfo and an error wrapping
// ErrUnknownVCP, with the same soft-handle semantics as GetMode.
func GetVCPInfo(vcp string) (VCPInfo, error) {
	if info, ok := vcpCatalog[vcp]; ok {
		return info, nil
	}
	fallback := VCPInfo{
		Mode:        fmt.Sprintf("Unknown (VCP %q)", vcp),
		Description: fmt.Sprintf("Unrecognized VCP code %q", vcp),
	}
	return fallback, fmt.Errorf("%w: %q", ErrUnknownVCP, vcp)
}

// simplifyGeneratorState generates the simplified generator state based on the given input.
// It takes a genInput string as input and returns the corresponding genState string and an error (if any).
func simplifyGeneratorState(input string) (string, error) {
	replacements := map[string]string{
		"Switched to Auxiliary Power|Utility PWR Available|Generator On": "On",
		"Switched to Auxiliary Power|Generator On":                       "On",
		"Utility PWR Available|Generator On":                             "On",
		"Utility PWR Available":                                          "Off",
	}

	for pattern, replacement := range replacements {
		if input == pattern {
			return replacement, nil
		}
	}

	return "", errors.New("unknown input")
}

// SanitizeStationIDs splits a string of station IDs by space, comma, or semicolon
// and returns a slice of sanitized and validated station IDs.
func SanitizeStationIDs(stationInput string) []string {
	// Define a regular expression to split by space, comma, or semicolon
	re := regexp.MustCompile(`[ ,;]+`)
	stationIDs := re.Split(stationInput, -1)

	var validStations []string
	for _, stationID := range stationIDs {
		trimmed := strings.TrimSpace(stationID)
		if trimmed == "" {
			continue
		}
		// Convert to uppercase and validate format
		trimmed = strings.ToUpper(trimmed)
		if ValidateStationID(trimmed) {
			validStations = append(validStations, trimmed)
		}
	}
	return validStations
}

// ValidateStationID validates a radar station ID format
// Station IDs should be 4 characters, starting with 'K' for US stations
func ValidateStationID(stationID string) bool {
	if len(stationID) != 4 {
		return false
	}

	// US radar stations typically start with 'K'
	// Some exceptions exist but this covers 99% of cases
	if !strings.HasPrefix(stationID, "K") {
		// Allow some international stations or special cases
		// But validate they are all uppercase letters
		for _, char := range stationID {
			if char < 'A' || char > 'Z' {
				return false
			}
		}
	}

	// Validate all characters are uppercase letters
	for _, char := range stationID {
		if char < 'A' || char > 'Z' {
			return false
		}
	}

	return true
}
