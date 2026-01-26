package emulatorauth

import "strings"

// AuthMode defines how IAM authorization is enforced
type AuthMode string

const (
	// AuthModeOff disables IAM checks entirely (legacy behavior, default)
	AuthModeOff AuthMode = "off"

	// AuthModePermissive enables IAM checks with fail-open behavior:
	// - IAM reachable: enforce permissions
	// - IAM unreachable: allow (fail-open)
	// - Config errors: deny
	// Use for development where IAM might not always be running
	AuthModePermissive AuthMode = "permissive"

	// AuthModeStrict enables IAM checks with fail-closed behavior:
	// - IAM reachable: enforce permissions
	// - IAM unreachable: deny (fail-closed)
	// - Config errors: deny
	// Recommended for CI/CD to catch permission issues
	AuthModeStrict AuthMode = "strict"
)

// ParseAuthMode parses an auth mode from string (case-insensitive)
func ParseAuthMode(s string) AuthMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "permissive":
		return AuthModePermissive
	case "strict":
		return AuthModeStrict
	default:
		return AuthModeOff
	}
}

// String returns the string representation of the auth mode
func (m AuthMode) String() string {
	return string(m)
}

// IsEnabled returns true if IAM checks are enabled (not off)
func (m AuthMode) IsEnabled() bool {
	return m != AuthModeOff
}
