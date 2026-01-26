package emulatorauth

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

const (
	// PrincipalMetadataKey is the gRPC metadata key for principal identity
	PrincipalMetadataKey = "x-emulator-principal"

	// PrincipalHeaderKey is the HTTP header key for principal identity
	PrincipalHeaderKey = "X-Emulator-Principal"
)

// ExtractPrincipalFromContext extracts the principal from gRPC incoming metadata
func ExtractPrincipalFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	principals := md.Get(PrincipalMetadataKey)
	if len(principals) == 0 {
		return ""
	}

	return principals[0]
}

// ExtractPrincipalFromRequest extracts the principal from HTTP request header
func ExtractPrincipalFromRequest(r *http.Request) string {
	return r.Header.Get(PrincipalHeaderKey)
}

// InjectPrincipalToContext adds the principal to outgoing gRPC metadata
func InjectPrincipalToContext(ctx context.Context, principal string) context.Context {
	if principal == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, PrincipalMetadataKey, principal)
}
