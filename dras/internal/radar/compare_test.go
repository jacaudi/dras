package radar

import (
	"strings"
	"testing"
)

func TestCompareData(t *testing.T) {
	oldData := &Data{
		Name:              "KATX",
		VCP:               "R31",
		Mode:              "Clear Air",
		Status:            "Online",
		OperabilityStatus: "Normal",
		PowerSource:       "Utility",
		GenState:          "Off",
	}

	t.Run("no changes", func(t *testing.T) {
		newData := &Data{
			Name:              "KATX",
			VCP:               "R31",
			Mode:              "Clear Air",
			Status:            "Online",
			OperabilityStatus: "Normal",
			PowerSource:       "Utility",
			GenState:          "Off",
		}

		alertConfig := AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		}

		changed, message := CompareData(oldData, newData, alertConfig)
		if changed {
			t.Error("Expected no changes, but changes were detected")
		}
		if message != "" {
			t.Errorf("Expected empty message, got %q", message)
		}
	})

	t.Run("VCP change to precipitation", func(t *testing.T) {
		newData := &Data{
			Name:              "KATX",
			VCP:               "R12",
			Mode:              "Precipitation",
			Status:            "Online",
			OperabilityStatus: "Normal",
			PowerSource:       "Utility",
			GenState:          "Off",
		}

		alertConfig := AlertConfig{VCP: true}

		changed, message := CompareData(oldData, newData, alertConfig)
		if !changed {
			t.Error("Expected changes to be detected")
		}
		if !strings.Contains(message, "Precipitation Mode Active") {
			t.Errorf("Expected precipitation message, got %q", message)
		}
	})

	t.Run("VCP change to clear air", func(t *testing.T) {
		precipData := &Data{
			Name:              "KATX",
			VCP:               "R12",
			Mode:              "Precipitation",
			Status:            "Online",
			OperabilityStatus: "Normal",
			PowerSource:       "Utility",
			GenState:          "Off",
		}

		newData := &Data{
			Name:              "KATX",
			VCP:               "R35",
			Mode:              "Clear Air",
			Status:            "Online",
			OperabilityStatus: "Normal",
			PowerSource:       "Utility",
			GenState:          "Off",
		}

		alertConfig := AlertConfig{VCP: true}

		changed, message := CompareData(precipData, newData, alertConfig)
		if !changed {
			t.Error("Expected changes to be detected")
		}
		if !strings.Contains(message, "Clear Air Mode Active") {
			t.Errorf("Expected clear air message, got %q", message)
		}
	})

	t.Run("status change", func(t *testing.T) {
		newData := &Data{
			Name:              "KATX",
			VCP:               "R31",
			Mode:              "Clear Air",
			Status:            "Offline",
			OperabilityStatus: "Normal",
			PowerSource:       "Utility",
			GenState:          "Off",
		}

		alertConfig := AlertConfig{Status: true}

		changed, message := CompareData(oldData, newData, alertConfig)
		if !changed {
			t.Error("Expected changes to be detected")
		}
		if !strings.Contains(message, "status changed from Online to Offline") {
			t.Errorf("Expected status change message, got %q", message)
		}
	})

	t.Run("multiple changes", func(t *testing.T) {
		newData := &Data{
			Name:              "KATX",
			VCP:               "R12",
			Mode:              "Precipitation",
			Status:            "Offline",
			OperabilityStatus: "Maintenance",
			PowerSource:       "Generator",
			GenState:          "On",
		}

		alertConfig := AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		}

		changed, message := CompareData(oldData, newData, alertConfig)
		if !changed {
			t.Error("Expected changes to be detected")
		}

		// Check that all changes are reported
		expectedStrings := []string{
			"Precipitation Mode Active",
			"Radar status changed from Online to Offline",
			"Radar operability changed from Normal to Maintenance",
			"Power source changed from Utility to Generator",
			"Generator state changed from Off to On",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(message, expected) {
				t.Errorf("Expected message to contain %q, but got %q", expected, message)
			}
		}
	})

	// Skip-Unknown behavior (issue #129). A degraded NWS payload produces
	// <Type>Unknown values for the affected fields; the comparator must not
	// fire a change notification for a flip TO an Unknown sentinel, even
	// though the strings differ. Cache update is gated on `changed` in the
	// caller, so this leaves the cache holding the last known-good value
	// and a subsequent recovery to that same value produces a clean
	// no-change.
	t.Run("skips flip to Unknown on every field", func(t *testing.T) {
		degraded := &Data{
			Name:              "KATX",
			VCP:               VCPUnknown,
			Mode:              ModeUnknown,
			Status:            StatusUnknown,
			OperabilityStatus: OpStatusUnknown,
			PowerSource:       PowerSourceUnknown,
			GenState:          GenStateUnknown,
		}

		alertConfig := AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		}

		changed, message := CompareData(oldData, degraded, alertConfig)
		if changed {
			t.Errorf("Expected no changes for OK→Unknown flip on every field, got changed=true, message=%q", message)
		}
		if message != "" {
			t.Errorf("Expected empty message for OK→Unknown flip, got %q", message)
		}
	})

	t.Run("flip FROM Unknown is reported (recovery)", func(t *testing.T) {
		// First-run / post-recovery: previous cached value was Unknown
		// (e.g. cache seeded during a degraded poll). A subsequent known
		// value IS a genuine state-clarification and should fire.
		degraded := &Data{
			Name:              "KATX",
			VCP:               VCPUnknown,
			Mode:              ModeUnknown,
			Status:            StatusUnknown,
			OperabilityStatus: OpStatusUnknown,
			PowerSource:       PowerSourceUnknown,
			GenState:          GenStateUnknown,
		}

		recovered := &Data{
			Name:              "KATX",
			VCP:               VCPR12,
			Mode:              ModePrecipitation,
			Status:            "Operate",
			OperabilityStatus: "RDA - On-line",
			PowerSource:       PowerSourceUtility,
			GenState:          GenStateOff,
		}

		alertConfig := AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		}

		changed, message := CompareData(degraded, recovered, alertConfig)
		if !changed {
			t.Error("Expected change to be reported for Unknown→OK recovery")
		}
		// VCP path: oldData=Unknown, newData=R12 known → "Precipitation Mode Active".
		if !strings.Contains(message, "Precipitation Mode Active") {
			t.Errorf("Expected precipitation message on recovery, got %q", message)
		}
		// Status: Unknown → Operate is reported as a normal flip.
		if !strings.Contains(message, "status changed from Unknown to Operate") {
			t.Errorf("Expected status change message on recovery, got %q", message)
		}
	})

	t.Run("partial degradation: change on known field only", func(t *testing.T) {
		// Only VCP flips to Unknown; other fields stay known and unchanged.
		// Expect: no change reported (VCP path skips Unknown, others equal).
		partial := &Data{
			Name:              "KATX",
			VCP:               VCPUnknown,
			Mode:              ModeUnknown,
			Status:            "Online",
			OperabilityStatus: "Normal",
			PowerSource:       PowerSourceUtility,
			GenState:          GenStateOff,
		}

		alertConfig := AlertConfig{
			VCP:         true,
			Status:      true,
			Operability: true,
			PowerSource: true,
			GenState:    true,
		}

		changed, message := CompareData(oldData, partial, alertConfig)
		if changed {
			t.Errorf("Expected no change for partial degradation (VCP→Unknown, others stable), got changed=true message=%q", message)
		}
	})

	// Functional coverage for the alarmSummary field (issue #128). Same shape
	// as the existing five comparators: fire on a real flip, stay silent on a
	// flip to the Unknown sentinel, and honor the enable toggle.
	t.Run("alarm summary change", func(t *testing.T) {
		newData := &Data{
			Name:              "KATX",
			VCP:               "R31",
			Mode:              "Clear Air",
			Status:            "Online",
			OperabilityStatus: "Normal",
			AlarmSummary:      AlarmSummaryCommunication,
			PowerSource:       "Utility",
			GenState:          "Off",
		}

		alertConfig := AlertConfig{AlarmSummary: true}

		changed, message := CompareData(oldData, newData, alertConfig)
		if !changed {
			t.Error("Expected alarm-summary change to be detected")
		}
		if !strings.Contains(message, "Alarm summary changed from  to Communication") {
			t.Errorf("Expected alarm-summary change message, got %q", message)
		}
	})

	t.Run("alarm summary skips flip to Unknown", func(t *testing.T) {
		base := &Data{Name: "KATX", AlarmSummary: AlarmSummaryNone}
		degraded := &Data{Name: "KATX", AlarmSummary: AlarmSummaryUnknown}

		changed, message := CompareData(base, degraded, AlertConfig{AlarmSummary: true})
		if changed {
			t.Errorf("Expected no change for alarm-summary flip to Unknown, got message=%q", message)
		}
	})

	t.Run("alarm summary respects disabled toggle", func(t *testing.T) {
		base := &Data{Name: "KATX", AlarmSummary: AlarmSummaryNone}
		alarming := &Data{Name: "KATX", AlarmSummary: AlarmSummaryCommunication}

		changed, _ := CompareData(base, alarming, AlertConfig{AlarmSummary: false})
		if changed {
			t.Error("Expected no change when ALERT_ALARM_SUMMARY is off")
		}
	})

	t.Run("ignores disabled alerts", func(t *testing.T) {
		newData := &Data{
			Name:              "KATX",
			VCP:               "R12",
			Mode:              "Precipitation",
			Status:            "Offline",
			OperabilityStatus: "Maintenance",
			PowerSource:       "Generator",
			GenState:          "On",
		}

		alertConfig := AlertConfig{} // All alerts disabled

		changed, message := CompareData(oldData, newData, alertConfig)
		if changed {
			t.Error("Expected no changes when all alerts are disabled")
		}
		if message != "" {
			t.Errorf("Expected empty message when all alerts disabled, got %q", message)
		}
	})
}
