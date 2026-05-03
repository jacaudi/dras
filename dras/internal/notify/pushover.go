package notify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/gregdel/pushover"
	"github.com/nikoksr/notify"
	pushoverNotify "github.com/nikoksr/notify/service/pushover"
)

// Attachment is an optional image to include with a Pushover notification.
type Attachment struct {
	Data        []byte
	ContentType string
	Filename    string
}

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

// ValidateCredentials validates Pushover API credentials format
func (s *Service) ValidateCredentials() error {
	if err := ValidateAPIToken(s.apiToken); err != nil {
		return fmt.Errorf("invalid API token: %w", err)
	}

	if err := ValidateUserKey(s.userKey); err != nil {
		return fmt.Errorf("invalid user key: %w", err)
	}

	return nil
}

// ValidateAPIToken validates the format of a Pushover API token
func ValidateAPIToken(token string) error {
	if token == "" {
		return errors.New("API token cannot be empty")
	}

	// Pushover API tokens are 30 characters long, alphanumeric
	if len(token) != 30 {
		return errors.New("API token must be exactly 30 characters long")
	}

	// Check for alphanumeric characters only
	matched, err := regexp.MatchString(`^[a-zA-Z0-9]+$`, token)
	if err != nil {
		return fmt.Errorf("error validating API token format: %w", err)
	}

	if !matched {
		return errors.New("API token must contain only alphanumeric characters")
	}

	return nil
}

// ValidateUserKey validates the format of a Pushover user key
func ValidateUserKey(userKey string) error {
	if userKey == "" {
		return errors.New("user key cannot be empty")
	}

	// Pushover user keys are 30 characters long, alphanumeric
	if len(userKey) != 30 {
		return errors.New("user key must be exactly 30 characters long")
	}

	// Check for alphanumeric characters only
	matched, err := regexp.MatchString(`^[a-zA-Z0-9]+$`, userKey)
	if err != nil {
		return fmt.Errorf("error validating user key format: %w", err)
	}

	if !matched {
		return errors.New("user key must contain only alphanumeric characters")
	}

	return nil
}

// SendNotification sends a Pushover notification with the specified title and message.
// It uses the Pushover service to send the notification.
// The function returns an error if the notification fails to send, otherwise it returns nil.
func (s *Service) SendNotification(ctx context.Context, title, message string) error {
	// Create a new Pushover service
	pushoverService := pushoverNotify.New(s.apiToken)

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

	slog.Debug("Pushover notification sent successfully")
	return nil
}

// SendNotificationWithAttachment sends a Pushover notification that includes
// an image attachment. When attachment is nil this is equivalent to
// SendNotification. The Pushover REST API is invoked directly because the
// nikoksr/notify wrapper does not expose attachment support.
func (s *Service) SendNotificationWithAttachment(ctx context.Context, title, message string, attachment *Attachment) error {
	if attachment == nil || len(attachment.Data) == 0 {
		return s.SendNotification(ctx, title, message)
	}

	app := pushover.New(s.apiToken)
	recipient := pushover.NewRecipient(s.userKey)

	msg := pushover.NewMessageWithTitle(message, title)
	if err := msg.AddAttachment(bytes.NewReader(attachment.Data)); err != nil {
		return fmt.Errorf("failed to attach image to notification: %w", err)
	}

	type result struct {
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		_, err := app.SendMessage(msg, recipient)
		resCh <- result{err: err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-resCh:
		if r.err != nil {
			return r.err
		}
	}

	slog.Debug("Pushover notification with attachment sent successfully", "bytes", fmt.Sprintf("%d", len(attachment.Data)))
	return nil
}
