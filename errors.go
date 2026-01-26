package emulatorauth

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsConnectivityError returns true if the error is due to connectivity issues
// (IAM emulator unreachable, timeout, or cancelled context)
func IsConnectivityError(err error) bool {
	if err == nil {
		return false
	}

	code := status.Code(err)
	return code == codes.Unavailable ||
		code == codes.DeadlineExceeded ||
		code == codes.Canceled
}

// IsConfigError returns true if the error indicates a configuration problem
// that should always deny (in both permissive and strict modes)
func IsConfigError(err error) bool {
	if err == nil {
		return false
	}

	code := status.Code(err)
	// These errors indicate bugs or misconfigurations
	return code == codes.InvalidArgument ||
		code == codes.Internal ||
		code == codes.Unimplemented
}
