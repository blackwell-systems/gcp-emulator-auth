package emulatorauth

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsConnectivityError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		// Connectivity errors (should return true)
		{
			name:     "Unavailable",
			err:      status.Error(codes.Unavailable, "service unavailable"),
			expected: true,
		},
		{
			name:     "DeadlineExceeded",
			err:      status.Error(codes.DeadlineExceeded, "timeout"),
			expected: true,
		},
		{
			name:     "Canceled",
			err:      status.Error(codes.Canceled, "context canceled"),
			expected: true,
		},

		// Non-connectivity errors (should return false)
		{
			name:     "PermissionDenied",
			err:      status.Error(codes.PermissionDenied, "permission denied"),
			expected: false,
		},
		{
			name:     "InvalidArgument",
			err:      status.Error(codes.InvalidArgument, "invalid argument"),
			expected: false,
		},
		{
			name:     "Internal",
			err:      status.Error(codes.Internal, "internal error"),
			expected: false,
		},
		{
			name:     "NotFound",
			err:      status.Error(codes.NotFound, "not found"),
			expected: false,
		},
		{
			name:     "Unauthenticated",
			err:      status.Error(codes.Unauthenticated, "unauthenticated"),
			expected: false,
		},

		// Edge cases
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-gRPC error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "OK status",
			err:      status.Error(codes.OK, "ok"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectivityError(tt.err)
			if got != tt.expected {
				t.Errorf("IsConnectivityError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsConfigError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		// Config errors (should return true)
		{
			name:     "InvalidArgument",
			err:      status.Error(codes.InvalidArgument, "bad resource format"),
			expected: true,
		},
		{
			name:     "Internal",
			err:      status.Error(codes.Internal, "internal server error"),
			expected: true,
		},
		{
			name:     "Unimplemented",
			err:      status.Error(codes.Unimplemented, "not implemented"),
			expected: true,
		},

		// Non-config errors (should return false)
		{
			name:     "PermissionDenied",
			err:      status.Error(codes.PermissionDenied, "permission denied"),
			expected: false,
		},
		{
			name:     "Unavailable",
			err:      status.Error(codes.Unavailable, "service unavailable"),
			expected: false,
		},
		{
			name:     "DeadlineExceeded",
			err:      status.Error(codes.DeadlineExceeded, "timeout"),
			expected: false,
		},
		{
			name:     "Canceled",
			err:      status.Error(codes.Canceled, "canceled"),
			expected: false,
		},
		{
			name:     "NotFound",
			err:      status.Error(codes.NotFound, "not found"),
			expected: false,
		},

		// Edge cases
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-gRPC error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "OK status",
			err:      status.Error(codes.OK, "ok"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConfigError(tt.err)
			if got != tt.expected {
				t.Errorf("IsConfigError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestErrorClassification_MutuallyExclusive(t *testing.T) {
	// Verify that connectivity and config errors don't overlap
	connectivityCodes := []codes.Code{
		codes.Unavailable,
		codes.DeadlineExceeded,
		codes.Canceled,
	}

	configCodes := []codes.Code{
		codes.InvalidArgument,
		codes.Internal,
		codes.Unimplemented,
	}

	// Check connectivity codes are not classified as config errors
	for _, code := range connectivityCodes {
		err := status.Error(code, "test")
		if IsConfigError(err) {
			t.Errorf("Connectivity error %v should not be classified as config error", code)
		}
	}

	// Check config codes are not classified as connectivity errors
	for _, code := range configCodes {
		err := status.Error(code, "test")
		if IsConnectivityError(err) {
			t.Errorf("Config error %v should not be classified as connectivity error", code)
		}
	}
}
