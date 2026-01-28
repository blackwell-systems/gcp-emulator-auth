# Local IAM Control Plane — Enforcement Proxy

[![Blackwell Systems](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![CI](https://github.com/blackwell-systems/gcp-emulator-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/blackwell-systems/gcp-emulator-auth/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-97.9%25-brightgreen)](https://github.com/blackwell-systems/gcp-emulator-auth/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/gcp-emulator-auth.svg)](https://pkg.go.dev/github.com/blackwell-systems/gcp-emulator-auth)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> **Enforce real GCP IAM policies before requests reach emulators — make local environments fail exactly like production.**

This is the **enforcement proxy** component of the Blackwell Local IAM Control Plane. It sits between your application and service emulators (Secret Manager, KMS, etc.), checking permissions before allowing data access.

## What This Is

Unlike mocks (which allow everything) or observers like iamlive (which record after the fact), this library **actively denies unauthorized requests** using real IAM policy evaluation.

| Approach | Example | When | Behavior |
|----------|---------|------|----------|
| Mock | Standard emulators | Never | Always allows |
| Observer | iamlive (AWS) | After | Records what you used |
| **Control Plane** | **Blackwell IAM** | **Before** | **Denies unauthorized** |

Pre-flight enforcement catches permission bugs in development and CI, not production.

## Architecture

```
┌─────────────────────────────────────────┐
│  Your Application Code                  │
│  (GCP client libraries)                 │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│  DATA PLANE                             │
│  Service Emulator (Secret Manager, KMS) │
│                                         │
│  Uses THIS LIBRARY to:                  │
│  1. Extract principal from request      │
│  2. Check permission                    │
│  3. Deny if unauthorized                │
└────────────────┬────────────────────────┘
                 │
                 │ CheckPermission(principal, resource, permission)
                 ▼
┌─────────────────────────────────────────┐
│  CONTROL PLANE                          │
│  IAM Emulator (Policy Engine)           │
│                                         │
│  - Role bindings                        │
│  - Group memberships                    │
│  - Policy inheritance                   │
└─────────────────────────────────────────┘
```

This library is used by all Blackwell service emulators to provide consistent IAM enforcement.

## Who Uses This

**Service emulator developers:** This library enforces IAM policies in your emulator.

**End users:** Use the [control plane CLI](https://github.com/blackwell-systems/gcp-emulator-control-plane) instead — it manages the entire stack for you.

**Currently integrated:**
- [GCP Secret Manager Emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator)
- [GCP KMS Emulator](https://github.com/blackwell-systems/gcp-kms-emulator)

**Building a new emulator?** See the [Integration Contract](https://github.com/blackwell-systems/gcp-iam-emulator/blob/main/docs/INTEGRATION.md).

## Installation

```bash
go get github.com/blackwell-systems/gcp-emulator-auth
```

## Usage

### Basic Setup

```go
import (
    emulatorauth "github.com/blackwell-systems/gcp-emulator-auth"
)

func main() {
    // Load config from environment
    config := emulatorauth.LoadFromEnv()
    
    if config.Mode.IsEnabled() {
        // Connect to IAM emulator
        iamClient, err := emulatorauth.NewClient(config.Host, config.Mode)
        if err != nil {
            log.Fatal(err)
        }
        defer iamClient.Close()
        
        // Use in handlers...
    }
}
```

### In gRPC Handler

```go
func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.Secret, error) {
    // Extract principal from incoming request
    principal := emulatorauth.ExtractPrincipalFromContext(ctx)
    
    // Check permission if IAM is enabled
    if s.iamClient != nil {
        allowed, err := s.iamClient.CheckPermission(
            ctx,
            principal,
            req.Name, // resource
            "secretmanager.secrets.get", // permission
        )
        if err != nil {
            return nil, status.Error(codes.Internal, "IAM check failed")
        }
        if !allowed {
            return nil, status.Error(codes.PermissionDenied, "Permission denied")
        }
    }
    
    // Proceed with operation
    return s.storage.GetSecret(req.Name)
}
```

### In HTTP Handler

```go
func (s *Server) handleGetSecret(w http.ResponseWriter, r *http.Request) {
    // Extract principal from HTTP header
    principal := emulatorauth.ExtractPrincipalFromRequest(r)
    
    // Check permission if IAM is enabled
    if s.iamClient != nil {
        allowed, err := s.iamClient.CheckPermission(
            r.Context(),
            principal,
            resourceName,
            "secretmanager.secrets.get",
        )
        if err != nil {
            http.Error(w, "IAM check failed", http.StatusInternalServerError)
            return
        }
        if !allowed {
            http.Error(w, "Permission denied", http.StatusForbidden)
            return
        }
    }
    
    // Proceed with operation
    // ...
}
```

## Environment Variables

| Variable | Purpose | Default | Values |
|----------|---------|---------|--------|
| `IAM_MODE` | Authorization mode | `off` | `off`, `permissive`, `strict` |
| `IAM_EMULATOR_HOST` | IAM emulator gRPC endpoint | `localhost:8080` | `host:port` |
| `IAM_TRACE` | Enable IAM decision logging | `false` | `true`, `false` |

## Auth Modes

### Off (default)
No IAM checks, all requests allowed (legacy behavior).

```bash
# Default - no environment variables needed
./server
```

### Permissive
IAM checks enabled with fail-open behavior:
- IAM reachable → enforce permissions
- IAM unreachable → allow (fail-open)
- Config errors → deny

```bash
IAM_MODE=permissive IAM_EMULATOR_HOST=localhost:8080 ./server
```

**Use for:** Development environments where IAM might not always be running.

### Strict
IAM checks enabled with fail-closed behavior:
- IAM reachable → enforce permissions
- IAM unreachable → deny (fail-closed)
- Config errors → deny

```bash
IAM_MODE=strict IAM_EMULATOR_HOST=localhost:8080 ./server
```

**Use for:** CI/CD to catch permission issues before production.

## Error Handling

The package classifies errors into three categories:

### 1. Permission Denied (Normal)
IAM evaluated and denied permission. Always deny.

### 2. Connectivity Errors
IAM unreachable, timeout, or cancelled:
- **Permissive mode:** Allow (fail-open)
- **Strict mode:** Deny (fail-closed)

### 3. Configuration Errors
Invalid resource, bad permission format, internal errors:
- **Both modes:** Deny (indicates bug/misconfiguration)

## API Reference

### Functions

#### `ParseAuthMode(s string) AuthMode`
Parse auth mode from string (case-insensitive).

#### `LoadFromEnv() Config`
Load configuration from environment variables.

#### `ExtractPrincipalFromContext(ctx context.Context) string`
Extract principal from gRPC incoming metadata.

#### `ExtractPrincipalFromRequest(r *http.Request) string`
Extract principal from HTTP request header.

#### `InjectPrincipalToContext(ctx context.Context, principal string) context.Context`
Add principal to outgoing gRPC metadata.

#### `IsConnectivityError(err error) bool`
Check if error is due to connectivity issues.

#### `IsConfigError(err error) bool`
Check if error indicates configuration problem.

### Types

#### `type AuthMode string`
Authorization mode: `off`, `permissive`, or `strict`.

#### `type Config struct`
IAM emulator configuration.

#### `type Client struct`
IAM emulator client for permission checks.

## Maintained By

Maintained by **Dayna Blackwell** — founder of Blackwell Systems, building reference infrastructure for cloud-native development.

[GitHub](https://github.com/blackwell-systems) · [LinkedIn](https://linkedin.com/in/dayna-blackwell) · [Blog](https://blog.blackwell-systems.com)

## Trademarks

**Blackwell Systems™** and the **Blackwell Systems logo** are trademarks of Dayna Blackwell. You may use the name "Blackwell Systems" to refer to this project, but you may not use the name or logo in a way that suggests endorsement or official affiliation without prior written permission. See [BRAND.md](BRAND.md) for usage guidelines.

## Related Projects

- [**GCP Emulator Control Plane**](https://github.com/blackwell-systems/gcp-emulator-control-plane) - CLI tool to run the emulator mesh
- [GCP IAM Emulator](https://github.com/blackwell-systems/gcp-iam-emulator) - The IAM policy engine
- [GCP Secret Manager Emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) - Secrets management
- [GCP KMS Emulator](https://github.com/blackwell-systems/gcp-kms-emulator) - Key management and crypto operations

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
