package emulatorauth

import (
	"context"
	"time"

	iampb "cloud.google.com/go/iam/apiv1/iampb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/blackwell-systems/gcp-emulator-auth/pkg/trace"
)

// Client is a lightweight IAM emulator client for permission checks
type Client struct {
	client      iampb.IAMPolicyClient
	conn        *grpc.ClientConn
	mode        AuthMode
	timeout     time.Duration
	traceWriter *trace.Writer
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

	// Initialize trace writer from environment
	traceWriter, _ := trace.NewWriterFromEnv()

	return &Client{
		client:      iampb.NewIAMPolicyClient(conn),
		conn:        conn,
		mode:        mode,
		timeout:     2 * time.Second,
		traceWriter: traceWriter,
	}, nil
}

// CheckPermission checks if the principal has the given permission on the resource
func (c *Client) CheckPermission(
	ctx context.Context,
	principal string,
	resource string,
	permission string,
) (bool, error) {
	start := time.Now()

	// Inject principal into outbound metadata
	ctx = InjectPrincipalToContext(ctx, principal)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
		Resource:    resource,
		Permissions: []string{permission},
	})

	duration := time.Since(start)

	if err != nil {
		// Emit error trace
		c.emitErrorTrace(principal, resource, permission, err, duration)

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
	allowed := len(resp.Permissions) == 1
	
	// Emit authorization trace
	c.emitAuthzTrace(principal, resource, permission, allowed, duration)

	return allowed, nil
}

func (c *Client) emitAuthzTrace(principal, resource, permission string, allowed bool, duration time.Duration) {
	if c.traceWriter == nil {
		return
	}

	outcome := trace.OutcomeDeny
	reason := "no_matching_binding"
	if allowed {
		outcome = trace.OutcomeAllow
		reason = "binding_match"
	}

	event := trace.AuthzEvent{
		SchemaVersion: trace.SchemaV1_0,
		EventType:     trace.EventTypeAuthzCheck,
		Timestamp:     trace.NowRFC3339Nano(),
		Actor: &trace.Actor{
			Principal: principal,
		},
		Target: &trace.Target{
			Resource: resource,
		},
		Action: &trace.Action{
			Permission: permission,
			Method:     "CheckPermission",
		},
		Decision: &trace.Decision{
			Outcome:     outcome,
			Reason:      reason,
			EvaluatedBy: "gcp-emulator-auth",
			LatencyMS:   duration.Milliseconds(),
		},
		Environment: &trace.Environment{
			Mode:      string(c.mode),
			Component: "gcp-emulator-auth",
		},
	}

	_ = c.traceWriter.Emit(event)
	_ = c.traceWriter.Flush()
}

func (c *Client) emitErrorTrace(principal, resource, permission string, err error, duration time.Duration) {
	if c.traceWriter == nil {
		return
	}

	kind := "policy_error"
	retryable := false

	if IsConnectivityError(err) {
		kind = "iam_unreachable"
		retryable = true
	} else if IsConfigError(err) {
		kind = "invalid_request"
		retryable = false
	}

	event := trace.AuthzEvent{
		SchemaVersion: trace.SchemaV1_0,
		EventType:     trace.EventTypeAuthzError,
		Timestamp:     trace.NowRFC3339Nano(),
		Error: &trace.AuthzError{
			Kind:      kind,
			Message:   err.Error(),
			Retryable: retryable,
		},
		Environment: &trace.Environment{
			Mode:      string(c.mode),
			Component: "gcp-emulator-auth",
		},
	}

	_ = c.traceWriter.Emit(event)
	_ = c.traceWriter.Flush()
}

// Close closes the IAM client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
