package radar

import (
	"fmt"
	"strings"
)

// AlertConfig holds configuration for which events to alert on.
type AlertConfig struct {
	VCP         bool
	Status      bool
	Operability bool
	PowerSource bool
	GenState    bool
}

// CompareData compares the old and new radar data and returns whether there are any changes and the details of the changes.
// It takes two pointers to Data structs as input and returns a boolean indicating if there are any changes and a string containing the details of the changes.
func CompareData(oldData, newData *Data, alertConfig AlertConfig) (bool, string) {
	var changes []string

	if alertConfig.VCP && oldData.VCP != newData.VCP {
		// Look up the new VCP in the catalog. Known VCPs surface their
		// AlertText (preferred) or fall back to "<Mode> Mode Active".
		// Unknown VCPs report the raw transition so users still see what
		// changed.
		if info, err := GetVCPInfo(newData.VCP); err == nil {
			switch {
			case info.AlertText != "":
				changes = append(changes, info.AlertText)
			default:
				changes = append(changes, fmt.Sprintf("%s Mode Active", info.Mode))
			}
		} else {
			changes = append(changes, fmt.Sprintf("Radar mode changed from %s to %s", oldData.VCP, newData.VCP))
		}
	}

	if alertConfig.Status && oldData.Status != newData.Status {
		changes = append(changes, fmt.Sprintf("Radar status changed from %s to %s", oldData.Status, newData.Status))
	}

	if alertConfig.Operability && oldData.OperabilityStatus != newData.OperabilityStatus {
		changes = append(changes, fmt.Sprintf("Radar operability changed from %s to %s", oldData.OperabilityStatus, newData.OperabilityStatus))
	}

	if alertConfig.PowerSource && oldData.PowerSource != newData.PowerSource {
		changes = append(changes, fmt.Sprintf("Power source changed from %s to %s", oldData.PowerSource, newData.PowerSource))
	}

	if alertConfig.GenState && oldData.GenState != newData.GenState {
		changes = append(changes, fmt.Sprintf("Generator state changed from %s to %s", oldData.GenState, newData.GenState))
	}

	if len(changes) > 0 {
		return true, strings.Join(changes, "\n")
	}

	return false, ""
}

