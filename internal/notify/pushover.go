package notify

import (
	"context"
	"log"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/pushover"
)

// Service handles notification operations and implements the Notifier interface.
type Service struct {
	apiToken string
	userKey  string
}

// New creates a new notification service.
func New(apiToken, userKey string) *Service {
	return &Service{
		apiToken: apiToken,
		userKey:  userKey,
	}
}

// SendNotification sends a Pushover notification with the specified title and message.
// It uses the Pushover service to send the notification.
// The function returns an error if the notification fails to send, otherwise it returns nil.
func (s *Service) SendNotification(ctx context.Context, title, message string) error {
	// Create a new Pushover service
	pushoverService := pushover.New(s.apiToken)

	// Add a recipient
	pushoverService.AddReceivers(s.userKey)

	// Create a new notification
	notification := notify.New()
	notification.UseServices(pushoverService)

	// Send the notification
	err := notification.Send(ctx, title, message)
	if err != nil {
		return err
	}

	log.Println("Pushover notification sent successfully!")
	return nil
}
