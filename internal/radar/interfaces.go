package radar

import (
	"errors"
)

// DataFetcher interface for abstracting radar data fetching
type DataFetcher interface {
	FetchData(stationID string) (*Data, error)
}

// MockDataFetcher provides a mock implementation for testing
type MockDataFetcher struct {
	responses map[string]*Data
	errors    map[string]error
	callCount int
}

// NewMockDataFetcher creates a new mock data fetcher
func NewMockDataFetcher() *MockDataFetcher {
	return &MockDataFetcher{
		responses: make(map[string]*Data),
		errors:    make(map[string]error),
	}
}

// SetResponse sets a mock response for a station ID
func (m *MockDataFetcher) SetResponse(stationID string, response *Data) {
	m.responses[stationID] = response
}

// SetError sets a mock error for a station ID
func (m *MockDataFetcher) SetError(stationID string, err error) {
	m.errors[stationID] = err
}

// FetchData returns the mock response or error
func (m *MockDataFetcher) FetchData(stationID string) (*Data, error) {
	m.callCount++

	if err, exists := m.errors[stationID]; exists {
		return nil, err
	}

	if response, exists := m.responses[stationID]; exists {
		return response, nil
	}

	// Default response if none set
	return &Data{
		Name:              stationID,
		VCP:               "R31",
		Mode:              "Clear Air",
		Status:            "Online",
		OperabilityStatus: "Normal",
		PowerSource:       "Utility",
		GenState:          "Off",
	}, nil
}

// GetCallCount returns the number of API calls made
func (m *MockDataFetcher) GetCallCount() int {
	return m.callCount
}

// ResetCallCount resets the call counter
func (m *MockDataFetcher) ResetCallCount() {
	m.callCount = 0
}

// SimulateError creates a mock error for testing
func SimulateError(message string) error {
	return errors.New(message)
}
