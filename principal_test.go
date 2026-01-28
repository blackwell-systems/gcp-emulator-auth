package emulatorauth

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestExtractPrincipalFromContext(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "no metadata",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "empty metadata",
			ctx:      metadata.NewIncomingContext(context.Background(), metadata.MD{}),
			expected: "",
		},
		{
			name: "principal present",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				PrincipalMetadataKey: []string{"user:alice@example.com"},
			}),
			expected: "user:alice@example.com",
		},
		{
			name: "multiple principals takes first",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				PrincipalMetadataKey: []string{"user:alice@example.com", "user:bob@example.com"},
			}),
			expected: "user:alice@example.com",
		},
		{
			name: "uppercase key match works",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"X-Emulator-Principal": []string{"user:alice@example.com"},
			}),
			expected: "user:alice@example.com", // gRPC metadata keys are case-insensitive in MD creation
		},
		{
			name: "service account principal",
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				PrincipalMetadataKey: []string{"serviceAccount:sa@project.iam.gserviceaccount.com"},
			}),
			expected: "serviceAccount:sa@project.iam.gserviceaccount.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPrincipalFromContext(tt.ctx)
			if got != tt.expected {
				t.Errorf("ExtractPrincipalFromContext() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractPrincipalFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *http.Request
		expected string
	}{
		{
			name: "principal present",
			req: &http.Request{
				Header: http.Header{
					PrincipalHeaderKey: []string{"user:alice@example.com"},
				},
			},
			expected: "user:alice@example.com",
		},
		{
			name: "no header",
			req: &http.Request{
				Header: http.Header{},
			},
			expected: "",
		},
		{
			name: "empty header value",
			req: &http.Request{
				Header: http.Header{
					PrincipalHeaderKey: []string{""},
				},
			},
			expected: "",
		},
		{
			name: "multiple values takes first",
			req: &http.Request{
				Header: http.Header{
					PrincipalHeaderKey: []string{"user:alice@example.com", "user:bob@example.com"},
				},
			},
			expected: "user:alice@example.com",
		},
		{
			name: "lowercase header key",
			req: &http.Request{
				Header: http.Header{
					"x-emulator-principal": []string{"user:alice@example.com"},
				},
			},
			expected: "", // Header.Get() is case-sensitive with lowercase, need canonical form
		},
		{
			name: "service account principal",
			req: &http.Request{
				Header: http.Header{
					PrincipalHeaderKey: []string{"serviceAccount:sa@project.iam.gserviceaccount.com"},
				},
			},
			expected: "serviceAccount:sa@project.iam.gserviceaccount.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPrincipalFromRequest(tt.req)
			if got != tt.expected {
				t.Errorf("ExtractPrincipalFromRequest() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestInjectPrincipalToContext(t *testing.T) {
	tests := []struct {
		name      string
		principal string
		wantEmpty bool
	}{
		{
			name:      "inject user principal",
			principal: "user:alice@example.com",
			wantEmpty: false,
		},
		{
			name:      "inject service account",
			principal: "serviceAccount:sa@project.iam.gserviceaccount.com",
			wantEmpty: false,
		},
		{
			name:      "empty principal returns original context",
			principal: "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx := InjectPrincipalToContext(ctx, tt.principal)

			// Extract from outgoing metadata
			md, ok := metadata.FromOutgoingContext(newCtx)

			if tt.wantEmpty {
				// Should not have added metadata
				if ok && len(md.Get(PrincipalMetadataKey)) > 0 {
					t.Errorf("Expected no principal in metadata, got %v", md.Get(PrincipalMetadataKey))
				}
				return
			}

			// Should have metadata
			if !ok {
				t.Fatal("Expected outgoing metadata to be present")
			}

			principals := md.Get(PrincipalMetadataKey)
			if len(principals) == 0 {
				t.Fatal("Expected principal in metadata")
			}

			if principals[0] != tt.principal {
				t.Errorf("Principal in metadata = %q, want %q", principals[0], tt.principal)
			}
		})
	}
}

func TestInjectPrincipalToContext_Append(t *testing.T) {
	// Test that InjectPrincipalToContext appends to existing metadata
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "other-key", "other-value")
	ctx = InjectPrincipalToContext(ctx, "user:alice@example.com")

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("Expected outgoing metadata")
	}

	// Should have both keys
	if len(md.Get("other-key")) == 0 {
		t.Error("Expected existing metadata to be preserved")
	}
	if len(md.Get(PrincipalMetadataKey)) == 0 {
		t.Error("Expected principal metadata to be added")
	}

	if md.Get(PrincipalMetadataKey)[0] != "user:alice@example.com" {
		t.Errorf("Principal = %q, want %q", md.Get(PrincipalMetadataKey)[0], "user:alice@example.com")
	}
}
