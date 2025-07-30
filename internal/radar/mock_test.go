package radar

import (
	"testing"
)

func TestMockDataFetcher(t *testing.T) {
	mock := NewMockDataFetcher()

	t.Run("default response", func(t *testing.T) {
		data, err := mock.FetchData("KATX")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if data == nil {
			t.Error("Expected data, got nil")
		}
		if data.Name != "KATX" {
			t.Errorf("Expected Name=KATX, got %s", data.Name)
		}
		if data.VCP != "R31" {
			t.Errorf("Expected VCP=R31, got %s", data.VCP)
		}
	})

	t.Run("custom response", func(t *testing.T) {
		customData := &Data{
			Name:              "KRAX",
			VCP:               "R12",
			Mode:              "Precipitation",
			Status:            "Online",
			OperabilityStatus: "Normal",
			PowerSource:       "Generator",
			GenState:          "On",
		}
		mock.SetResponse("KRAX", customData)

		data, err := mock.FetchData("KRAX")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if data.VCP != "R12" {
			t.Errorf("Expected VCP=R12, got %s", data.VCP)
		}
		if data.PowerSource != "Generator" {
			t.Errorf("Expected PowerSource=Generator, got %s", data.PowerSource)
		}
	})

	t.Run("error response", func(t *testing.T) {
		testError := SimulateError("API connection failed")
		mock.SetError("KBGM", testError)

		data, err := mock.FetchData("KBGM")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if data != nil {
			t.Error("Expected nil data on error")
		}
		if err.Error() != "API connection failed" {
			t.Errorf("Expected 'API connection failed', got %s", err.Error())
		}
	})

	t.Run("call counting", func(t *testing.T) {
		mock.ResetCallCount()
		if mock.GetCallCount() != 0 {
			t.Errorf("Expected call count to be 0 after reset, got %d", mock.GetCallCount())
		}

		_, _ = mock.FetchData("KATX")
		_, _ = mock.FetchData("KRAX")

		if mock.GetCallCount() != 2 {
			t.Errorf("Expected call count to be 2, got %d", mock.GetCallCount())
		}
	})
}

func TestServiceImplementsDataFetcher(t *testing.T) {
	// Test that Service implements DataFetcher interface
	var _ DataFetcher = &Service{}
	
	// Test that MockDataFetcher implements DataFetcher interface
	var _ DataFetcher = &MockDataFetcher{}
}