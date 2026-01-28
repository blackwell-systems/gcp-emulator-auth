package emulatorauth

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected Config
	}{
		{
			name: "all defaults",
			env:  map[string]string{},
			expected: Config{
				Mode:  AuthModeOff,
				Host:  "localhost:8080",
				Trace: false,
			},
		},
		{
			name: "permissive mode",
			env: map[string]string{
				"IAM_MODE": "permissive",
			},
			expected: Config{
				Mode:  AuthModePermissive,
				Host:  "localhost:8080",
				Trace: false,
			},
		},
		{
			name: "strict mode with custom host",
			env: map[string]string{
				"IAM_MODE":          "strict",
				"IAM_EMULATOR_HOST": "iam-emulator:9000",
			},
			expected: Config{
				Mode:  AuthModeStrict,
				Host:  "iam-emulator:9000",
				Trace: false,
			},
		},
		{
			name: "trace enabled",
			env: map[string]string{
				"IAM_TRACE": "true",
			},
			expected: Config{
				Mode:  AuthModeOff,
				Host:  "localhost:8080",
				Trace: true,
			},
		},
		{
			name: "all custom values",
			env: map[string]string{
				"IAM_MODE":          "STRICT",
				"IAM_EMULATOR_HOST": "custom-host:1234",
				"IAM_TRACE":         "true",
			},
			expected: Config{
				Mode:  AuthModeStrict,
				Host:  "custom-host:1234",
				Trace: true,
			},
		},
		{
			name: "trace false explicit",
			env: map[string]string{
				"IAM_TRACE": "false",
			},
			expected: Config{
				Mode:  AuthModeOff,
				Host:  "localhost:8080",
				Trace: false,
			},
		},
		{
			name: "trace invalid value defaults to false",
			env: map[string]string{
				"IAM_TRACE": "yes",
			},
			expected: Config{
				Mode:  AuthModeOff,
				Host:  "localhost:8080",
				Trace: false,
			},
		},
		{
			name: "invalid mode defaults to off",
			env: map[string]string{
				"IAM_MODE": "invalid",
			},
			expected: Config{
				Mode:  AuthModeOff,
				Host:  "localhost:8080",
				Trace: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.env {
				os.Setenv(key, value)
			}

			// Load config
			got := LoadFromEnv()

			// Verify
			if got.Mode != tt.expected.Mode {
				t.Errorf("Mode = %v, want %v", got.Mode, tt.expected.Mode)
			}
			if got.Host != tt.expected.Host {
				t.Errorf("Host = %v, want %v", got.Host, tt.expected.Host)
			}
			if got.Trace != tt.expected.Trace {
				t.Errorf("Trace = %v, want %v", got.Trace, tt.expected.Trace)
			}
		})
	}

	// Clean up
	os.Clearenv()
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "env var not set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "empty default",
			key:          "TEST_KEY",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}

			got := getEnvWithDefault(tt.key, tt.defaultValue)
			if got != tt.expected {
				t.Errorf("getEnvWithDefault(%q, %q) = %q, want %q", tt.key, tt.defaultValue, got, tt.expected)
			}
		})
	}

	os.Clearenv()
}
