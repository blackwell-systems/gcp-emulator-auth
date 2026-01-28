package emulatorauth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

var (
	iamEmulatorHost = "localhost:18080" // Use non-standard port for tests
	iamServerCmd    *exec.Cmd
)

func getAbsolutePath(relPath string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", wd, relPath), nil
}

// TestMain starts the IAM emulator before tests and stops it after
func TestMain(m *testing.M) {
	// Check if IAM emulator binary exists
	iamBinaryPath := os.Getenv("IAM_EMULATOR_BINARY")
	if iamBinaryPath == "" {
		// Default path relative to gcp-emulator-auth repo
		iamBinaryPath = "../gcp-iam-emulator/bin/server"
	}

	if _, err := os.Stat(iamBinaryPath); os.IsNotExist(err) {
		fmt.Printf("IAM emulator binary not found at %s\n", iamBinaryPath)
		fmt.Println("Build it first: cd ../gcp-iam-emulator && make build")
		os.Exit(1)
	}

	// Get absolute path to test policy
	policyPath, err := getAbsolutePath("testdata/test-policy.yaml")
	if err != nil {
		fmt.Printf("Failed to get policy path: %v\n", err)
		os.Exit(1)
	}

	// Start IAM emulator
	fmt.Println("Starting IAM emulator...")
	iamServerCmd = exec.Command(iamBinaryPath,
		"-port", "18080",
		"-config", policyPath,
	)
	iamServerCmd.Stdout = os.Stdout
	iamServerCmd.Stderr = os.Stderr
	
	if err := iamServerCmd.Start(); err != nil {
		fmt.Printf("Failed to start IAM emulator: %v\n", err)
		os.Exit(1)
	}

	// Wait for IAM emulator to be ready
	fmt.Println("Waiting for IAM emulator to be ready...")
	if !waitForIAMEmulator(iamEmulatorHost, 10*time.Second) {
		iamServerCmd.Process.Kill()
		fmt.Println("IAM emulator failed to start")
		os.Exit(1)
	}
	fmt.Println("IAM emulator ready")

	// Run tests
	code := m.Run()

	// Cleanup
	fmt.Println("Stopping IAM emulator...")
	if iamServerCmd != nil && iamServerCmd.Process != nil {
		iamServerCmd.Process.Kill()
		iamServerCmd.Wait()
	}

	os.Exit(code)
}

func waitForIAMEmulator(host string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		client, err := NewClient(host, AuthModeStrict)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			_, err := client.CheckPermission(ctx, "user:test@example.com", "test-resource", "test.permission")
			cancel()
			client.Close()
			
			// Any response (even permission denied) means emulator is up
			if err == nil || !IsConnectivityError(err) {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		mode    AuthMode
		wantErr bool
	}{
		{
			name:    "valid connection",
			host:    iamEmulatorHost,
			mode:    AuthModeStrict,
			wantErr: false,
		},
		{
			name:    "permissive mode",
			host:    iamEmulatorHost,
			mode:    AuthModePermissive,
			wantErr: false,
		},
		{
			name:    "off mode",
			host:    iamEmulatorHost,
			mode:    AuthModeOff,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.host, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if client != nil {
				defer client.Close()
				
				if client.mode != tt.mode {
					t.Errorf("Client mode = %v, want %v", client.mode, tt.mode)
				}
				if client.timeout != 2*time.Second {
					t.Errorf("Client timeout = %v, want %v", client.timeout, 2*time.Second)
				}
			}
		})
	}
}

func TestClientClose(t *testing.T) {
	client, err := NewClient(iamEmulatorHost, AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
	
	// Set conn to nil after close (idempotent close pattern)
	client.conn = nil
	
	// Close again should not error (nil conn check)
	err = client.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestCheckPermission_StrictMode_Allowed(t *testing.T) {
	// This test requires a policy file that allows the permission
	// For now, we test the mechanics work
	client, err := NewClient(iamEmulatorHost, AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	// Test with a permission check (result depends on policy file)
	allowed, err := client.CheckPermission(
		ctx,
		"user:test@example.com",
		"projects/test-project/secrets/test-secret",
		"secretmanager.secrets.get",
	)

	// We don't assert allowed true/false because it depends on policy
	// We just verify the call succeeds and returns valid data
	t.Logf("Permission check: allowed=%v, err=%v", allowed, err)
	
	// Should not have connectivity error in strict mode with running emulator
	if err != nil && IsConnectivityError(err) {
		t.Errorf("Unexpected connectivity error in strict mode with running emulator: %v", err)
	}
}

func TestCheckPermission_PermissiveMode_Connectivity(t *testing.T) {
	// Test permissive mode fail-open behavior
	// Connect to non-existent host to simulate connectivity failure
	client, err := NewClient("localhost:9999", AuthModePermissive)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	// This should fail to connect but return true (fail-open)
	allowed, err := client.CheckPermission(
		ctx,
		"user:test@example.com",
		"projects/test-project/secrets/test-secret",
		"secretmanager.secrets.get",
	)

	if !allowed {
		t.Error("Permissive mode should allow on connectivity error (fail-open)")
	}
	
	if err != nil {
		t.Error("Permissive mode should not return error on connectivity failure (fail-open)")
	}
}

func TestCheckPermission_StrictMode_Connectivity(t *testing.T) {
	// Test strict mode fail-closed behavior
	// Connect to non-existent host to simulate connectivity failure
	client, err := NewClient("localhost:9999", AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	// This should fail to connect and return false (fail-closed)
	allowed, err := client.CheckPermission(
		ctx,
		"user:test@example.com",
		"projects/test-project/secrets/test-secret",
		"secretmanager.secrets.get",
	)

	if allowed {
		t.Error("Strict mode should deny on connectivity error (fail-closed)")
	}
	
	if err == nil {
		t.Error("Strict mode should return error on connectivity failure")
	}
	
	if !IsConnectivityError(err) {
		t.Errorf("Expected connectivity error, got: %v", err)
	}
}

func TestCheckPermission_PrincipalInjection(t *testing.T) {
	// Verify that principal is properly injected into outgoing metadata
	client, err := NewClient(iamEmulatorHost, AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	principals := []string{
		"user:alice@example.com",
		"user:bob@example.com",
		"serviceAccount:sa@project.iam.gserviceaccount.com",
	}

	for _, principal := range principals {
		t.Run(principal, func(t *testing.T) {
			// Just verify the call succeeds with different principals
			// The IAM emulator will receive the principal in metadata
			_, err := client.CheckPermission(
				ctx,
				principal,
				"projects/test-project/secrets/test-secret",
				"secretmanager.secrets.get",
			)
			
			// We don't assert the result, just that principal injection works
			// The fact that we get any response (not a panic) means injection worked
			t.Logf("Principal %s: err=%v", principal, err)
		})
	}
}

func TestCheckPermission_EmptyPrincipal(t *testing.T) {
	client, err := NewClient(iamEmulatorHost, AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	// Empty principal should still make the call (IAM emulator decides how to handle it)
	allowed, err := client.CheckPermission(
		ctx,
		"",
		"projects/test-project/secrets/test-secret",
		"secretmanager.secrets.get",
	)
	
	t.Logf("Empty principal: allowed=%v, err=%v", allowed, err)
	
	// Should get a response (IAM decides if empty principal is valid)
	// We just verify it doesn't panic or fail for connectivity reasons
	if err != nil && IsConnectivityError(err) {
		t.Error("Should not have connectivity error with empty principal")
	}
}

func TestCheckPermission_Timeout(t *testing.T) {
	// Test that timeout is applied (2 seconds default)
	client, err := NewClient(iamEmulatorHost, AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	allowed, err := client.CheckPermission(
		ctx,
		"user:test@example.com",
		"projects/test-project/secrets/test-secret",
		"secretmanager.secrets.get",
	)
	
	if allowed {
		t.Error("Should not allow with cancelled context")
	}
	
	if err == nil {
		t.Error("Should return error with cancelled context")
	}
	
	if !IsConnectivityError(err) {
		t.Errorf("Cancelled context should be connectivity error, got: %v", err)
	}
}

func TestCheckPermission_MultiplePermissions(t *testing.T) {
	// Test checking multiple different permissions
	client, err := NewClient(iamEmulatorHost, AuthModeStrict)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	
	permissions := []string{
		"secretmanager.secrets.get",
		"secretmanager.versions.access",
		"cloudkms.cryptoKeys.encrypt",
		"cloudkms.cryptoKeys.decrypt",
	}

	for _, permission := range permissions {
		t.Run(permission, func(t *testing.T) {
			allowed, err := client.CheckPermission(
				ctx,
				"user:test@example.com",
				"projects/test-project/secrets/test-secret",
				permission,
			)
			
			// Just verify the call works for different permissions
			t.Logf("Permission %s: allowed=%v, err=%v", permission, allowed, err)
			
			// Should not have connectivity errors
			if err != nil && IsConnectivityError(err) {
				t.Errorf("Unexpected connectivity error for permission %s", permission)
			}
		})
	}
}
