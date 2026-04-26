package notify

import (
	"testing"
)

func TestValidateAPIToken(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		shouldErr bool
	}{
		{
			name:      "valid token",
			token:     "abcdef1234567890123456789012ab",
			shouldErr: false,
		},
		{
			name:      "empty token",
			token:     "",
			shouldErr: true,
		},
		{
			name:      "contains special characters",
			token:     "abcdef123456789012345678901@ab",
			shouldErr: true,
		},
		{
			name:      "contains spaces",
			token:     "abcdef123456789012345678901 ab",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIToken(tt.token)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateAPIToken() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

func TestValidateUserKey(t *testing.T) {
	tests := []struct {
		name      string
		userKey   string
		shouldErr bool
	}{
		{
			name:      "valid user key",
			userKey:   "uvwxyz1234567890123456789012uv",
			shouldErr: false,
		},
		{
			name:      "empty user key",
			userKey:   "",
			shouldErr: true,
		},
		{
			name:      "contains special characters",
			userKey:   "uvwxyz123456789012345678901#uv",
			shouldErr: true,
		},
		{
			name:      "contains hyphen",
			userKey:   "uvwxyz123456789012345678901-uv",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserKey(tt.userKey)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateUserKey() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

func TestService_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name      string
		apiToken  string
		userKey   string
		shouldErr bool
	}{
		{
			name:      "valid credentials",
			apiToken:  "abcdef1234567890123456789012ab",
			userKey:   "uvwxyz1234567890123456789012uv",
			shouldErr: false,
		},
		{
			name:      "invalid api token",
			apiToken:  "invalid",
			userKey:   "uvwxyz1234567890123456789012uv",
			shouldErr: true,
		},
		{
			name:      "invalid user key",
			apiToken:  "abcdef1234567890123456789012ab",
			userKey:   "invalid",
			shouldErr: true,
		},
		{
			name:      "both invalid",
			apiToken:  "invalid",
			userKey:   "invalid",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New(tt.apiToken, tt.userKey)
			err := service.ValidateCredentials()
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateCredentials() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}
