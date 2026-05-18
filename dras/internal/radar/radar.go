package radar

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jacaudi/nws/cmd/nws"
)

// -----------------------------------------------------------------------------
// Typed status fields
//
// Per go-standards.md §4.1 / §4.3, each discrete-value field on radar.Data is
// a named string type. Constants declare the values we currently know; the
// underlying string type still accepts arbitrary NWS-supplied values, so a
// surprise input from the API is preserved verbatim for forensics rather than
// being squashed into a generic "Unknown" bucket.
//
// The <Type>Unknown sentinel (always the literal string "Unknown") is the
// canonical "missing / unparseable" value. It pairs with the comparator's
// skip-Unknown logic in compare.go so a degraded poll never fires a spurious
// change notification.
// -----------------------------------------------------------------------------

// VCP is a NEXRAD WSR-88D Volume Coverage Pattern code as reported by the NWS
// radar metadata API.
type VCP string

const (
	VCPR12     VCP = "R12"
	VCPR31     VCP = "R31"
	VCPR35     VCP = "R35"
	VCPR112    VCP = "R112"
	VCPR212    VCP = "R212"
	VCPR215    VCP = "R215"
	VCPUnknown VCP = "Unknown"
)

// String implements fmt.Stringer for log/notification interpolation.
func (v VCP) String() string { return string(v) }

// RadarMode is the coarse scan-strategy category derived from a VCP code.
type RadarMode string

const (
	ModeClearAir      RadarMode = "Clear Air"
	ModePrecipitation RadarMode = "Precipitation"
	ModeUnknown       RadarMode = "Unknown"
)

func (m RadarMode) String() string { return string(m) }

// RadarStatus is the RDA operational mode reported by NWS. Observed values
// in production include "Operate" (the dominant value) and the four below.
type RadarStatus string

const (
	StatusOperate        RadarStatus = "Operate"
	StatusStandby        RadarStatus = "Standby"
	StatusTest           RadarStatus = "Test"
	StatusOfflineOperate RadarStatus = "Offline Operate"
	StatusUnknown        RadarStatus = "Unknown"
)

func (s RadarStatus) String() string { return string(s) }

// OperabilityStatus is the RDA health-summary string reported by NWS. The
// "RDA - On-line" value indicates a fully healthy station; everything else
// signals a degradation of some kind.
type OperabilityStatus string

const (
	OpStatusOnline      OperabilityStatus = "RDA - On-line"
	OpStatusMaintenance OperabilityStatus = "RDA - Maintenance Action Mandatory"
	OpStatusOffline     OperabilityStatus = "RDA - Off-line"
	OpStatusStandby     OperabilityStatus = "RDA - Standby"
	OpStatusCoasting    OperabilityStatus = "RDA - On-line / Coasting"
	OpStatusUnknown     OperabilityStatus = "Unknown"
)

func (o OperabilityStatus) String() string { return string(o) }

// PowerSource indicates whether the radar is running on utility power or its
// backup generator.
type PowerSource string

const (
	PowerSourceUtility   PowerSource = "Utility"
	PowerSourceGenerator PowerSource = "Generator"
	PowerSourceUnknown   PowerSource = "Unknown"
)

func (p PowerSource) String() string { return string(p) }

// GeneratorState is the simplified On/Off interpretation of the NWS
// pipe-separated generatorState field. See ParseGeneratorState for the
// semantic rules.
type GeneratorState string

const (
	GenStateOn      GeneratorState = "On"
	GenStateOff     GeneratorState = "Off"
	GenStateUnknown GeneratorState = "Unknown"
)

func (g GeneratorState) String() string { return string(g) }

// -----------------------------------------------------------------------------
// Sentinel errors — soft-fail signals for the typed parsers
// -----------------------------------------------------------------------------

// ErrUnknownVCP is returned by GetMode / GetVCPInfo when the VCP code is empty
// or not in the catalog. Callers can use errors.Is to detect this case and
// treat it as a soft condition: use the returned fallback value, log a
// warning, continue. Aborting station processing on a single bad field would
// stall the whole 5-minute poll cycle (issue #121, PR #123).
var ErrUnknownVCP = errors.New("unknown VCP")

// ErrUnknownGeneratorState is returned by ParseGeneratorState when the input
// cannot be classified as On or Off. Caller treatment mirrors ErrUnknownVCP
// (issue #129).
var ErrUnknownGeneratorState = errors.New("unknown generator state")

// -----------------------------------------------------------------------------
// VCP catalog + lookups
// -----------------------------------------------------------------------------

// VCPInfo describes a NEXRAD WSR-88D Volume Coverage Pattern.
type VCPInfo struct {
	Mode        RadarMode // Coarse category: ModeClearAir or ModePrecipitation
	Description string    // Human-readable detail about the scan pattern
	AlertText   string    // User-facing message used in change notifications
}

// vcpCatalog maps VCP codes (as reported by the NWS radar metadata API) to
// their coarse mode, a short description, and the alert text shown to users
// when this VCP becomes active. Source: NOAA/NWS WSR-88D operations
// documentation.
var vcpCatalog = map[VCP]VCPInfo{
	VCPR31:  {Mode: ModeClearAir, Description: "Clear Air, long pulse (~10 min cycle, stratiform/biological targets)", AlertText: "Clear Air Mode Active"},
	VCPR35:  {Mode: ModeClearAir, Description: "Clear Air, short pulse with clutter mitigation (~7 min cycle)", AlertText: "Clear Air Mode Active"},
	VCPR12:  {Mode: ModePrecipitation, Description: "Precipitation, rapid evolution (~4.2 min cycle, 14 elevations)", AlertText: "Precipitation Mode Active"},
	VCPR112: {Mode: ModePrecipitation, Description: "Precipitation with MRLE (multi-PRF range-folding mitigation)", AlertText: "Precipitation Mode (Velocity Scanning Emphasis) Active"},
	VCPR212: {Mode: ModePrecipitation, Description: "Precipitation with SAILS (~4.5 min cycle, common severe-weather VCP)", AlertText: "Precipitation Mode Active"},
	VCPR215: {Mode: ModePrecipitation, Description: "Precipitation (~6 min cycle, 15 elevations, tropical/widespread)", AlertText: "Precipitation Mode (Vertical Scanning Emphasis) Active"},
}

// GetMode returns the radar mode for a given VCP code, looked up from
// vcpCatalog.
//
// Unknown VCPs return ModeUnknown along with an error wrapping ErrUnknownVCP.
// Callers should treat this as a soft condition: errors.Is(err, ErrUnknownVCP)
// → use ModeUnknown and continue.
func GetMode(vcp VCP) (RadarMode, error) {
	if info, ok := vcpCatalog[vcp]; ok {
		return info.Mode, nil
	}
	return ModeUnknown, fmt.Errorf("%w: %q", ErrUnknownVCP, string(vcp))
}

// GetVCPInfo returns the full catalog entry (mode + description + alert text)
// for a VCP code. Unknown VCPs return a fallback VCPInfo and an error wrapping
// ErrUnknownVCP, with the same soft-handle semantics as GetMode.
func GetVCPInfo(vcp VCP) (VCPInfo, error) {
	if info, ok := vcpCatalog[vcp]; ok {
		return info, nil
	}
	fallback := VCPInfo{
		Mode:        ModeUnknown,
		Description: fmt.Sprintf("Unrecognized VCP code %q", string(vcp)),
	}
	return fallback, fmt.Errorf("%w: %q", ErrUnknownVCP, string(vcp))
}

// -----------------------------------------------------------------------------
// Parse<X> constructors
//
// Each ParseX takes the raw NWS string and returns the typed value. For
// Status, OperabilityStatus, and PowerSource the parser is a thin guard
// against empty input — non-empty values pass through verbatim as the typed
// string (preserving forensic detail when NWS surfaces a value we haven't
// enumerated as a constant yet).
//
// VCP and Mode use catalog-based lookups (GetMode / GetVCPInfo).
// GeneratorState requires semantic parsing — see ParseGeneratorState.
// -----------------------------------------------------------------------------

// ParseVCP wraps a raw VCP string in the VCP type, mapping the empty string
// to VCPUnknown. Unknown non-empty codes pass through verbatim so the raw
// value survives for forensic logging.
func ParseVCP(s string) VCP {
	if s == "" {
		return VCPUnknown
	}
	return VCP(s)
}

// ParseRadarStatus wraps a raw status string in RadarStatus, mapping empty
// input to StatusUnknown.
func ParseRadarStatus(s string) RadarStatus {
	if s == "" {
		return StatusUnknown
	}
	return RadarStatus(s)
}

// ParseOperabilityStatus wraps a raw operability-status string in
// OperabilityStatus, mapping empty input to OpStatusUnknown.
func ParseOperabilityStatus(s string) OperabilityStatus {
	if s == "" {
		return OpStatusUnknown
	}
	return OperabilityStatus(s)
}

// ParsePowerSource wraps a raw power-source string in PowerSource, mapping
// empty input to PowerSourceUnknown.
func ParsePowerSource(s string) PowerSource {
	if s == "" {
		return PowerSourceUnknown
	}
	return PowerSource(s)
}

// ParseGeneratorState interprets the NWS pipe-separated generatorState field
// into a simplified GeneratorState.
//
// Strategy: split on `|`, decide by token-set membership.
//   - `Generator On` ∈ tokens  → GenStateOn
//   - only token == `Utility PWR Available` → GenStateOff
//   - anything else (including empty input) → GenStateUnknown plus an error
//     wrapping ErrUnknownGeneratorState
//
// Returning an error on Unknown is deliberate so callers can choose between
// soft-fail and abort via errors.Is — but the typed return is always usable.
// The previous implementation returned ("", "unknown input") and the caller
// aborted the whole 5-minute poll cycle (issue #129).
func ParseGeneratorState(input string) (GeneratorState, error) {
	if input == "" {
		return GenStateUnknown, fmt.Errorf("%w: %q", ErrUnknownGeneratorState, input)
	}
	tokens := map[string]bool{}
	for _, t := range strings.Split(input, "|") {
		tokens[strings.TrimSpace(t)] = true
	}
	switch {
	case tokens["Generator On"]:
		return GenStateOn, nil
	case tokens["Utility PWR Available"] && len(tokens) == 1:
		return GenStateOff, nil
	default:
		return GenStateUnknown, fmt.Errorf("%w: %q", ErrUnknownGeneratorState, input)
	}
}

// -----------------------------------------------------------------------------
// Data
// -----------------------------------------------------------------------------

// Data represents the per-station radar metadata sampled from the NWS API.
// Status fields are named string types so the compiler catches accidental
// untyped-string usage and the const blocks above document the value space.
type Data struct {
	Name              string // Identifying metadata, not compared
	VCP               VCP
	Mode              RadarMode
	Status            RadarStatus
	OperabilityStatus OperabilityStatus
	PowerSource       PowerSource
	GenState          GeneratorState
}

// -----------------------------------------------------------------------------
// Service
// -----------------------------------------------------------------------------

// Service handles radar data operations.
type Service struct {
	// Add HTTP client interface here if needed for testing
}

// New creates a new radar service.
func New() *Service {
	return &Service{}
}

// FetchData retrieves radar data for a given station ID.
//
// Per-field soft-fail policy:
//   - VCP / Mode: an empty or unrecognized VCP yields ModeUnknown and logs a
//     warning; processing continues. (issue #121 / PR #123)
//   - GeneratorState: an empty or unrecognized generatorState yields
//     GenStateUnknown and logs a warning; processing continues. (issue #129)
//   - Status / OperabilityStatus / PowerSource: empty input yields the
//     XUnknown sentinel; non-empty values pass through as typed strings.
//
// This means a degraded NWS payload (missing fields) returns a populated
// *Data with sentinels in the affected slots, rather than aborting the whole
// poll. The comparator (CompareData) skips change notifications when the new
// value of a tracked field equals its Unknown sentinel so degraded polls
// produce silence, not spurious flips.
func (s *Service) FetchData(stationID string) (*Data, error) {
	radarResponse, err := nws.RadarStation(stationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get RADAR data for station %q: %w", stationID, err)
	}

	vcp := ParseVCP(radarResponse.RDA.Properties.VolumeCoveragePattern)
	mode, err := GetMode(vcp)
	if err != nil {
		if !errors.Is(err, ErrUnknownVCP) {
			return nil, err
		}
		slog.Warn("Unrecognized VCP, using fallback mode label",
			"station", stationID, "vcp", string(vcp), "mode", string(mode))
	}

	genState, err := ParseGeneratorState(radarResponse.RDA.Properties.GeneratorState)
	if err != nil {
		if !errors.Is(err, ErrUnknownGeneratorState) {
			return nil, err
		}
		slog.Warn("Unrecognized generator state, using fallback label",
			"station", stationID,
			"input", radarResponse.RDA.Properties.GeneratorState,
			"state", string(genState))
	}

	return &Data{
		Name:              radarResponse.Name,
		VCP:               vcp,
		Mode:              mode,
		Status:            ParseRadarStatus(radarResponse.RDA.Properties.Status),
		OperabilityStatus: ParseOperabilityStatus(radarResponse.RDA.Properties.OperabilityStatus),
		PowerSource:       ParsePowerSource(radarResponse.Performance.Properties.PowerSource),
		GenState:          genState,
	}, nil
}

// SanitizeStationIDs splits a string of station IDs by space, comma, or
// semicolon and returns a slice of sanitized and validated station IDs.
func SanitizeStationIDs(stationInput string) []string {
	re := regexp.MustCompile(`[ ,;]+`)
	stationIDs := re.Split(stationInput, -1)

	var validStations []string
	for _, stationID := range stationIDs {
		trimmed := strings.TrimSpace(stationID)
		if trimmed == "" {
			continue
		}
		trimmed = strings.ToUpper(trimmed)
		if ValidateStationID(trimmed) {
			validStations = append(validStations, trimmed)
		}
	}
	return validStations
}

// ValidateStationID validates a radar station ID format. Station IDs should
// be 4 characters, starting with 'K' for US stations; some non-K
// international stations are also valid.
func ValidateStationID(stationID string) bool {
	if len(stationID) != 4 {
		return false
	}

	if !strings.HasPrefix(stationID, "K") {
		for _, char := range stationID {
			if char < 'A' || char > 'Z' {
				return false
			}
		}
	}

	for _, char := range stationID {
		if char < 'A' || char > 'Z' {
			return false
		}
	}

	return true
}
