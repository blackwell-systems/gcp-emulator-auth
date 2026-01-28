package emulatorauth

import "testing"

func TestParseAuthMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected AuthMode
	}{
		// Valid modes
		{"lowercase permissive", "permissive", AuthModePermissive},
		{"lowercase strict", "strict", AuthModeStrict},
		{"lowercase off", "off", AuthModeOff},

		// Case insensitive
		{"uppercase PERMISSIVE", "PERMISSIVE", AuthModePermissive},
		{"uppercase STRICT", "STRICT", AuthModeStrict},
		{"uppercase OFF", "OFF", AuthModeOff},
		{"mixed case Permissive", "Permissive", AuthModePermissive},
		{"mixed case StRiCt", "StRiCt", AuthModeStrict},

		// Whitespace handling
		{"leading space", "  permissive", AuthModePermissive},
		{"trailing space", "strict  ", AuthModeStrict},
		{"both spaces", "  off  ", AuthModeOff},

		// Invalid/empty defaults to off
		{"empty string", "", AuthModeOff},
		{"invalid value", "invalid", AuthModeOff},
		{"random string", "foobar", AuthModeOff},
		{"partial match", "permi", AuthModeOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAuthMode(tt.input)
			if got != tt.expected {
				t.Errorf("ParseAuthMode(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAuthModeString(t *testing.T) {
	tests := []struct {
		mode     AuthMode
		expected string
	}{
		{AuthModeOff, "off"},
		{AuthModePermissive, "permissive"},
		{AuthModeStrict, "strict"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.expected {
				t.Errorf("AuthMode(%v).String() = %q, want %q", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestAuthModeIsEnabled(t *testing.T) {
	tests := []struct {
		mode     AuthMode
		expected bool
	}{
		{AuthModeOff, false},
		{AuthModePermissive, true},
		{AuthModeStrict, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got := tt.mode.IsEnabled()
			if got != tt.expected {
				t.Errorf("AuthMode(%v).IsEnabled() = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

