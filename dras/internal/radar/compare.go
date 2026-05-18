package radar

import (
	"errors"
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

// CompareData compares old and new radar data and returns whether any
// alert-enabled field changed, plus a multi-line message describing the
// changes.
//
// Per-field rule: a change is reported only when (a) the alert is enabled,
// (b) the new value is not the field's Unknown sentinel, and (c) old and new
// differ. Condition (b) is the noise-floor — when NWS returns a degraded
// payload that parses to <Type>Unknown for some field, the monitor would
// otherwise fire spurious change notifications on the OK→Unknown flip and
// again on the Unknown→OK recovery. The caller (monitor.processStation) only
// updates the cache when `changed` is true, so skipping the flip leaves the
// cache holding the last known-good value; recovery to the same value
// produces a true no-change.
//
// The skip is one-sided (new == Unknown only). If a previous cached value
// was somehow Unknown (e.g. first-run with degraded data), a subsequent
// known value is a genuine state-clarification and should fire.
func CompareData(oldData, newData *Data, alertConfig AlertConfig) (bool, string) {
	var changes []string

	if alertConfig.VCP && newData.VCP != VCPUnknown && oldData.VCP != newData.VCP {
		// Known VCPs surface their AlertText (preferred) or fall back to
		// "<Mode> Mode Active". An unknown VCP reaching this branch means
		// oldData.VCP was Unknown (we only skip when newData is Unknown) —
		// report the raw transition so users still see what changed.
		info, err := GetVCPInfo(newData.VCP)
		switch {
		case err == nil && info.AlertText != "":
			changes = append(changes, info.AlertText)
		case err == nil:
			changes = append(changes, fmt.Sprintf("%s Mode Active", info.Mode))
		case errors.Is(err, ErrUnknownVCP):
			changes = append(changes, fmt.Sprintf("Radar mode changed from %s to %s", oldData.VCP, newData.VCP))
		}
	}

	if alertConfig.Status && newData.Status != StatusUnknown && oldData.Status != newData.Status {
		changes = append(changes, fmt.Sprintf("Radar status changed from %s to %s", oldData.Status, newData.Status))
	}

	if alertConfig.Operability && newData.OperabilityStatus != OpStatusUnknown && oldData.OperabilityStatus != newData.OperabilityStatus {
		changes = append(changes, fmt.Sprintf("Radar operability changed from %s to %s", oldData.OperabilityStatus, newData.OperabilityStatus))
	}

	if alertConfig.PowerSource && newData.PowerSource != PowerSourceUnknown && oldData.PowerSource != newData.PowerSource {
		changes = append(changes, fmt.Sprintf("Power source changed from %s to %s", oldData.PowerSource, newData.PowerSource))
	}

	if alertConfig.GenState && newData.GenState != GenStateUnknown && oldData.GenState != newData.GenState {
		changes = append(changes, fmt.Sprintf("Generator state changed from %s to %s", oldData.GenState, newData.GenState))
	}

	if len(changes) > 0 {
		return true, strings.Join(changes, "\n")
	}

	return false, ""
}
