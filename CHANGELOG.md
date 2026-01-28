# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Client Trace Emission**: Authorization events emitted from CheckPermission
  - Emit `authz_check` events for every permission check (ALLOW/DENY)
  - Emit `authz_error` events for IAM connectivity failures
  - Captures principal, resource, permission, outcome, mode, and latency
  - Automatic classification: `iam_unreachable`, `invalid_request`, `policy_error`
  - Uses same `IAM_TRACE_OUTPUT` env var as IAM emulator
- **Trace Emission Package** (`pkg/trace/`):
  - JSONL trace schema v1.0 for authorization events
  - `AuthzEvent` struct with `authz_check` and `authz_error` event types
  - Buffered, thread-safe trace writer with `IAM_TRACE_OUTPUT` env var support
  - Schema validator with support for `.jsonl` and `.jsonl.gz` files
  - Golden test file (`testdata/sample-trace.jsonl`) for contract enforcement
- **Trace Writer Features**:
  - Opt-in tracing (off by default, enabled via `IAM_TRACE_OUTPUT`)
  - Supports `stdout` or file path destinations
  - Buffered writes for performance (no flush per event)
  - Thread-safe emission
  - Graceful handling when tracing disabled (nil writer pattern)
- **Schema Validation**:
  - Required fields enforcement per event type
  - Version compatibility checking (currently v1.0)
  - JSONL line-by-line validation with error line numbers
  - Support for gzip-compressed trace files

### Purpose
- **Authorization Observability**: Structured logging of IAM decisions for analysis and debugging
- **Audit Trail**: Deterministic record of permission checks with timestamps and outcomes
- **Testing & Analysis**: Enable trace-based testing and authorization behavior analysis
- **Schema Stability**: Semantic versioning prevents breaking changes

### Technical Details
- Minimum required fields: `schema_version`, `event_type`, `timestamp`, `actor.principal`, `target.resource`, `action.permission`, `decision.outcome`
- Optional enrichment fields: `trace.*`, `policy.*`, `environment.*`, `action.method`
- Event types: `authz_check` (permission checks), `authz_error` (IAM failures)
- Decision outcomes: `ALLOW`, `DENY`
- Error kinds: `iam_unreachable`, `iam_timeout`, `policy_error`, `invalid_principal`, `invalid_resource`, `invalid_permission`

### Usage
```go
import "github.com/blackwell-systems/gcp-emulator-auth/pkg/trace"

// Create writer from environment variable
writer, err := trace.NewWriterFromEnv()
if err != nil {
    return err
}
defer writer.Close()

// Emit authorization check event
event := trace.AuthzEvent{
    SchemaVersion: trace.SchemaV1_0,
    EventType:     trace.EventTypeAuthzCheck,
    Timestamp:     trace.NowRFC3339Nano(),
    Actor:         &trace.Actor{Principal: "user:alice@example.com"},
    Target:        &trace.Target{Resource: "projects/test/secrets/db-password"},
    Action:        &trace.Action{Permission: "secretmanager.secrets.get"},
    Decision:      &trace.Decision{Outcome: trace.OutcomeAllow},
}
if err := writer.Emit(event); err != nil {
    return err
}
```

## [0.1.1] - 2026-01-27

### Changed
- Downgraded Go version to 1.24 for CI compatibility
- Updated README with cross-references to related emulator projects

### Fixed
- Go module compatibility issues in CI environments
- Version format compatibility with older Go toolchains

## [0.1.0] - 2026-01-26

### Added
- **Initial Release**: Shared authentication library for GCP emulators
- **IAM Client**: Standardized client for IAM emulator integration
  - `NewClient(host, mode)` - Create IAM client with mode
  - `CheckPermission(ctx, principal, resource, permission)` - Check single permission
  - Connection pooling and error handling
- **Configuration Management**:
  - `LoadFromEnv()` - Load from environment variables
  - Support for `IAM_MODE` (off/permissive/strict)
  - Support for `IAM_HOST` (IAM emulator address)
- **Principal Extraction**:
  - `ExtractPrincipalFromContext(ctx)` - Extract from gRPC metadata
  - `ExtractPrincipalFromHTTPRequest(req)` - Extract from HTTP headers
  - Supports `x-emulator-principal` (gRPC) and `X-Emulator-Principal` (HTTP)
- **Authorization Modes**:
  - `AuthModeOff` - No permission checks (legacy)
  - `AuthModePermissive` - Fail-open on errors
  - `AuthModeStrict` - Fail-closed on errors
- **Error Classification**:
  - Connectivity errors (network issues)
  - Permission denied errors
  - Internal errors
  - Proper gRPC status code mapping

### Features
- **Consistent Behavior**: Same IAM integration pattern across all emulators
- **No Code Drift**: Shared library prevents copy/paste divergence
- **Maintained Error Handling**: Centralized error classification
- **Standard Principal Injection**: Uniform metadata/header extraction
- **Mode-Aware**: Respects fail-open vs fail-closed semantics

### Usage
```go
import emulatorauth "github.com/blackwell-systems/gcp-emulator-auth"

// Load config from environment
config := emulatorauth.LoadFromEnv()

// Create IAM client
iamClient, err := emulatorauth.NewClient(config.Host, config.Mode)

// Check permission
allowed, err := iamClient.CheckPermission(ctx, 
    "user:alice@example.com",
    "projects/test/secrets/db-password",
    "secretmanager.secrets.get")
```

### Adopted By
- gcp-secret-manager-emulator v1.2.0+
- gcp-kms-emulator v0.2.0+

### Design Goals
- Prevent IAM integration code drift
- Provide reference implementation
- Simplify emulator development
- Ensure consistent authorization behavior

[Unreleased]: https://github.com/blackwell-systems/gcp-emulator-auth/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/blackwell-systems/gcp-emulator-auth/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/blackwell-systems/gcp-emulator-auth/releases/tag/v0.1.0
