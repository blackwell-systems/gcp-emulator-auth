package emulatorauth

import (
	"context"
	"time"

	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a lightweight IAM emulator client for permission checks
type Client struct {
	client  iampb.IAMPolicyClient
	conn    *grpc.ClientConn
	mode    AuthMode
	timeout time.Duration
}

// NewClient creates a new IAM emulator client
func NewClient(host string, mode AuthMode) (*Client, error) {
	conn, err := grpc.NewClient(
		host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		client:  iampb.NewIAMPolicyClient(conn),
		conn:    conn,
		mode:    mode,
		timeout: 2 * time.Second,
	}, nil
}

// CheckPermission checks if the principal has the given permission on the resource
func (c *Client) CheckPermission(
	ctx context.Context,
	principal string,
	resource string,
	permission string,
) (bool, error) {
	// Inject principal into outbound metadata
	ctx = InjectPrincipalToContext(ctx, principal)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
		Resource:    resource,
		Permissions: []string{permission},
	})

	if err != nil {
		// Classify error type
		if IsConnectivityError(err) {
			// IAM emulator unreachable/timeout
			if c.mode == AuthModePermissive {
				// Fail-open: allow on connectivity issues
				return true, nil
			}
			// Strict mode: fail-closed
			return false, err
		}

		// Config/bad request error: always deny (both modes)
		// This indicates emulator misconfiguration that should be fixed
		return false, err
	}

	// Check if permission was granted
	return len(resp.Permissions) == 1, nil
}

// Close closes the IAM client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
