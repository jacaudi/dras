package notify

import (
	"context"
	"testing"
)

func TestMockNotifier(t *testing.T) {
	mock := NewMockNotifier()
	ctx := context.Background()

	t.Run("successful notification", func(t *testing.T) {
		err := mock.SendNotification(ctx, "Test Title", "Test message")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		notifications := mock.GetNotifications()
		if len(notifications) != 1 {
			t.Errorf("Expected 1 notification, got %d", len(notifications))
		}

		if notifications[0].Title != "Test Title" {
			t.Errorf("Expected title 'Test Title', got %s", notifications[0].Title)
		}

		if notifications[0].Message != "Test message" {
			t.Errorf("Expected message 'Test message', got %s", notifications[0].Message)
		}
	})

	t.Run("multiple notifications", func(t *testing.T) {
		mock.ClearNotifications()
		mock.ResetCallCount()

		_ = mock.SendNotification(ctx, "Title 1", "Message 1")
		_ = mock.SendNotification(ctx, "Title 2", "Message 2")

		if mock.GetCallCount() != 2 {
			t.Errorf("Expected 2 calls, got %d", mock.GetCallCount())
		}

		notifications := mock.GetNotifications()
		if len(notifications) != 2 {
			t.Errorf("Expected 2 notifications, got %d", len(notifications))
		}
	})

	t.Run("error simulation", func(t *testing.T) {
		mock.ClearNotifications()
		mock.SetShouldError(true)

		err := mock.SendNotification(ctx, "Error Title", "Error message")
		if err == nil {
			t.Error("Expected error, got nil")
		}

		notifications := mock.GetNotifications()
		if len(notifications) != 0 {
			t.Errorf("Expected 0 notifications on error, got %d", len(notifications))
		}
	})

	t.Run("specific title error", func(t *testing.T) {
		mock.ClearNotifications()
		mock.SetShouldError(false)
		mock.SetError("Fail Title", SimulateNotificationError("specific error"))

		_ = mock.SendNotification(ctx, "Success Title", "Success message")
		err := mock.SendNotification(ctx, "Fail Title", "Fail message")

		if err == nil {
			t.Error("Expected error for 'Fail Title', got nil")
		}

		notifications := mock.GetNotifications()
		if len(notifications) != 1 {
			t.Errorf("Expected 1 successful notification, got %d", len(notifications))
		}
	})

	t.Run("has notification check", func(t *testing.T) {
		mock.ClearNotifications()
		mock.SetShouldError(false)

		_ = mock.SendNotification(ctx, "DRAS Startup", "KATX Test - Clear Air Mode")

		if !mock.HasNotification("DRAS Startup") {
			t.Error("Expected to find 'DRAS Startup' notification")
		}

		if mock.HasNotification("Non-existent") {
			t.Error("Expected NOT to find 'Non-existent' notification")
		}
	})

	t.Run("get last notification", func(t *testing.T) {
		mock.ClearNotifications()

		lastNotif := mock.GetLastNotification()
		if lastNotif != nil {
			t.Error("Expected nil for empty notifications")
		}

		_ = mock.SendNotification(ctx, "First", "First message")
		_ = mock.SendNotification(ctx, "Last", "Last message")

		lastNotif = mock.GetLastNotification()
		if lastNotif == nil {
			t.Error("Expected last notification, got nil")
			return
		}

		if lastNotif.Title != "Last" {
			t.Errorf("Expected title 'Last', got %s", lastNotif.Title)
		}
	})
}

func TestNotifierInterface(t *testing.T) {
	// Test that Service implements Notifier interface
	var _ Notifier = &Service{}

	// Test that MockNotifier implements Notifier interface
	var _ Notifier = &MockNotifier{}
}

// SimulateNotificationError creates a mock notification error
func SimulateNotificationError(message string) error {
	return &NotificationError{Message: message}
}

// NotificationError represents a notification error
type NotificationError struct {
	Message string
}

func (e *NotificationError) Error() string {
	return e.Message
}
