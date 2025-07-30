package notify

import (
	"context"
	"errors"
)

// Notifier interface for abstracting notification sending
type Notifier interface {
	SendNotification(ctx context.Context, title, message string) error
}

// MockNotifier provides a mock implementation for testing
type MockNotifier struct {
	notifications []Notification
	errors        map[string]error
	callCount     int
	shouldError   bool
}

// Notification represents a sent notification for testing
type Notification struct {
	Title   string
	Message string
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		notifications: make([]Notification, 0),
		errors:        make(map[string]error),
	}
}

// SendNotification simulates sending a notification
func (m *MockNotifier) SendNotification(ctx context.Context, title, message string) error {
	m.callCount++
	
	if m.shouldError {
		return errors.New("mock notification error")
	}
	
	if err, exists := m.errors[title]; exists {
		return err
	}
	
	m.notifications = append(m.notifications, Notification{
		Title:   title,
		Message: message,
	})
	
	return nil
}

// GetNotifications returns all sent notifications
func (m *MockNotifier) GetNotifications() []Notification {
	return m.notifications
}

// GetCallCount returns the number of notification attempts
func (m *MockNotifier) GetCallCount() int {
	return m.callCount
}

// ResetCallCount resets the call counter
func (m *MockNotifier) ResetCallCount() {
	m.callCount = 0
}

// ClearNotifications clears all recorded notifications
func (m *MockNotifier) ClearNotifications() {
	m.notifications = make([]Notification, 0)
}

// SetError sets a mock error for a specific title
func (m *MockNotifier) SetError(title string, err error) {
	m.errors[title] = err
}

// SetShouldError makes all notifications fail
func (m *MockNotifier) SetShouldError(shouldError bool) {
	m.shouldError = shouldError
}

// GetLastNotification returns the most recent notification
func (m *MockNotifier) GetLastNotification() *Notification {
	if len(m.notifications) == 0 {
		return nil
	}
	return &m.notifications[len(m.notifications)-1]
}

// HasNotification checks if a notification with the given title was sent
func (m *MockNotifier) HasNotification(title string) bool {
	for _, notification := range m.notifications {
		if notification.Title == title {
			return true
		}
	}
	return false
}