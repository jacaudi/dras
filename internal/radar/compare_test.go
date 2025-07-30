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
			"Precipitation Mode -- Precipitation Detected",
			"status changed from Online to Offline",
			"operability changed from Normal to Maintenance",
			"Power source changed from Utility to Generator",
			"Generator state changed from Off to On",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(message, expected) {
				t.Errorf("Expected message to contain %q, but got %q", expected, message)
			}
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
