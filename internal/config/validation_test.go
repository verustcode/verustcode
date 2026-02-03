package config

import (
	"testing"

	"github.com/verustcode/verustcode/pkg/errors"
)

func TestValidatePassword(t *testing.T) {
	req := DefaultPasswordRequirements()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password with all requirements",
			password: "MyP@ssw0rd!",
			wantErr:  false,
		},
		{
			name:     "valid password with minimum length",
			password: "Ab1!abcd",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Ab1!abc",
			wantErr:  true,
		},
		{
			name:     "missing uppercase",
			password: "myp@ssw0rd!",
			wantErr:  true,
		},
		{
			name:     "missing lowercase",
			password: "MYP@SSW0RD!",
			wantErr:  true,
		},
		{
			name:     "missing digit",
			password: "MyP@ssword!",
			wantErr:  true,
		},
		{
			name:     "missing special character",
			password: "MyPassw0rd1",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "only lowercase",
			password: "abcdefghij",
			wantErr:  true,
		},
		{
			name:     "only uppercase",
			password: "ABCDEFGHIJ",
			wantErr:  true,
		},
		{
			name:     "only digits",
			password: "1234567890",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAdminConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *AdminConfig
		wantErr  bool
		wantCode errors.ErrorCode
	}{
		{
			name:    "nil config - should pass",
			cfg:     nil,
			wantErr: false,
		},
		{
			name: "disabled admin - should pass",
			cfg: &AdminConfig{
				Enabled:      false,
				Username:     "",
				PasswordHash: "",
			},
			wantErr: false,
		},
		{
			name: "valid admin config with bcrypt hash",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "$2a$10$YtJ6lCmNwS7g9IpuaR7nPOE/M/3.G6VdMBm7eJdLpSfnLdG/CvxMq", // valid bcrypt hash
				JWTSecret:    "this-is-a-32-character-secret!!!",                             // exactly 32 chars
			},
			wantErr: false,
		},
		{
			name: "empty username",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "",
				PasswordHash: "$2a$10$YtJ6lCmNwS7g9IpuaR7nPOE/M/3.G6VdMBm7eJdLpSfnLdG/CvxMq",
				JWTSecret:    "12345678901234567890123456789012",
			},
			wantErr:  true,
			wantCode: errors.ErrCodeAdminCredentialsEmpty,
		},
		{
			name: "empty password_hash - should pass (can be set via Web UI)",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "",
				JWTSecret:    "12345678901234567890123456789012",
			},
			wantErr: false, // password_hash is NOT validated anymore
		},
		{
			name: "whitespace only username",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "   ",
				PasswordHash: "$2a$10$YtJ6lCmNwS7g9IpuaR7nPOE/M/3.G6VdMBm7eJdLpSfnLdG/CvxMq",
				JWTSecret:    "12345678901234567890123456789012",
			},
			wantErr:  true,
			wantCode: errors.ErrCodeAdminCredentialsEmpty,
		},
		{
			name: "invalid password_hash - should pass (not validated anymore)",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "not-a-valid-bcrypt-hash",
				JWTSecret:    "12345678901234567890123456789012",
			},
			wantErr: false, // password_hash format is NOT validated anymore
		},
		{
			name: "empty jwt secret",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "$2a$10$YtJ6lCmNwS7g9IpuaR7nPOE/M/3.G6VdMBm7eJdLpSfnLdG/CvxMq",
				JWTSecret:    "",
			},
			wantErr:  true,
			wantCode: errors.ErrCodeJWTSecretInvalid,
		},
		{
			name: "jwt secret too short",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "$2a$10$YtJ6lCmNwS7g9IpuaR7nPOE/M/3.G6VdMBm7eJdLpSfnLdG/CvxMq",
				JWTSecret:    "short-secret", // less than 32 chars
			},
			wantErr:  true,
			wantCode: errors.ErrCodeJWTSecretInvalid,
		},
		{
			name: "jwt secret exactly 32 chars",
			cfg: &AdminConfig{
				Enabled:      true,
				Username:     "admin",
				PasswordHash: "$2a$10$YtJ6lCmNwS7g9IpuaR7nPOE/M/3.G6VdMBm7eJdLpSfnLdG/CvxMq",
				JWTSecret:    "12345678901234567890123456789012", // exactly 32 chars
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdminConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAdminConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantCode != "" && err.Code != tt.wantCode {
				t.Errorf("ValidateAdminConfig() code = %v, wantCode %v", err.Code, tt.wantCode)
			}
		})
	}
}

func TestFormatPasswordRequirements(t *testing.T) {
	result := FormatPasswordRequirements()

	// Should contain key requirements
	if result == "" {
		t.Error("FormatPasswordRequirements() returned empty string")
	}

	expectedParts := []string{
		"8 characters",
		"uppercase",
		"lowercase",
		"digit",
		"special character",
	}

	for _, part := range expectedParts {
		if !contains(result, part) {
			t.Errorf("FormatPasswordRequirements() should contain %q", part)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
