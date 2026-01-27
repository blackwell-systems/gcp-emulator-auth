# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
