package radar

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jacaudi/nws/cmd/nws"
)

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

	// Fetching radar VCP and determine mode
	radarVCPCode := radarResponse.RDA.Properties.VolumeCoveragePattern
	radarMode, err := GetMode(radarVCPCode) // Converting VCP to readable mode
	if err != nil {
		return nil, err
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

// GetMode returns the radar mode based on the given VCP (Volume Coverage Pattern) code.
// It maps specific VCP codes to corresponding radar modes.
// If the VCP code is not recognized, it returns an error.
func GetMode(vcp string) (string, error) {
	var radarMode string

	switch vcp {
	case "R31":
		radarMode = "Clear Air"
	case "R35":
		radarMode = "Clear Air"
	case "R12":
		radarMode = "Precipitation"
	case "R112":
		radarMode = "Precipitation"
	case "R212":
		radarMode = "Precipitation"
	case "R215":
		radarMode = "Precipitation"
	default:
		return "", fmt.Errorf("unknown mode for VCP %s -- please update code", vcp)
	}

	return radarMode, nil
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
// and returns a slice of sanitized station IDs.
func SanitizeStationIDs(stationInput string) []string {
	// Define a regular expression to split by space, comma, or semicolon
	re := regexp.MustCompile(`[ ,;]+`)
	stationIDs := re.Split(stationInput, -1)
	for i := range stationIDs {
		stationIDs[i] = strings.TrimSpace(stationIDs[i])
	}
	return stationIDs
}
