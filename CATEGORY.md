# Local IAM Control Plane — Category Definition

> **This document defines the infrastructure category that Blackwell Systems created. Use this language consistently across all repositories.**

---

## One-Line Definition

**Local IAM Control Plane** — Enforce real GCP IAM policies before requests reach emulators, making local environments fail exactly like production.

---

## The Three Defining Properties

### 1. Pre-Flight Enforcement
Authorization happens **before** data access, not after.

- Requests are **denied**, not observed
- Failure is part of the execution path
- No post-hoc analysis required

### 2. Production Semantics
Policy resolution matches real GCP behavior exactly.

- Same permission names (`secretmanager.secrets.get`)
- Same role bindings and inheritance
- Same group membership resolution
- Same failure modes (`PermissionDenied`)

### 3. Cross-Service Consistency
One policy engine, many consumers, deterministic behavior.

- Single IAM emulator evaluates all policies
- All service emulators enforce identically
- Fail-open (permissive) and fail-closed (strict) modes
- Works across Secret Manager, KMS, and future services

---

## Category Contrast (Use This Table Everywhere)

| Approach | Example | When | Behavior |
|----------|---------|------|----------|
| Mock | Standard emulators | Never | Always allows |
| Observer | iamlive (AWS) | After | Records what you used |
| **Control Plane** | **Blackwell IAM** | **Before** | **Denies unauthorized** |

**Key insight:** Most tools are either permissive mocks or passive observers. Control planes actively enforce policies before data access.

---

## The Architecture (Canonical Diagram)

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
│  1. Extract principal from request      │
│  2. Check permission via auth library   │
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
│  - Custom roles                         │
│  - Policy inheritance                   │
└─────────────────────────────────────────┘
```

**Critical insight:** This replicates GCP's actual topology. IAM is upstream of every API. Enforcement happens at the control plane boundary.

---

## The Blackwell Security Trinity

When you use these components together, you create a **hermetic environment** — tests are completely sealed and behave exactly like real Google Cloud, but entirely on your machine.

| Component | Role | Repository | Purpose |
|-----------|------|------------|---------|
| **IAM Emulator** | **The Brain** | `gcp-iam-emulator` | Stores policies, evaluates permissions, handles `testIamPermissions` |
| **Emulator Auth** | **The Guard** | `gcp-emulator-auth` | Enforcement proxy that checks permissions before data access |
| **Service Emulators** | **The Workers** | `gcp-secret-manager-emulator`, `gcp-kms-emulator` | Data plane services secured by IAM layer |
| **Control Plane CLI** | **The Orchestrator** | `gcp-emulator-control-plane` | Manages the entire stack (start/stop/configure) |

---

## Why This Is a "Killer App" for CI/CD

Most developers hate IAM because you only find out you've missed a permission **after** you deploy to production.

With the Blackwell IAM control plane, you can write deterministic tests:

```bash
# Test A: Run with admin role → Expect Success
IAM_MODE=strict ./run-tests-with-role admin

# Test B: Run with read-only role → Expect Failure on write
IAM_MODE=strict ./run-tests-with-role reader
# Expected: PermissionDenied when trying to create secret
```

**Deterministic enforcement** means:
- No IAM eventual consistency delays (instant policy changes)
- No flaky tests due to cloud timing
- Same behavior every time
- Catch permission bugs before deployment

---

## Enforcement Modes (Key Differentiator)

| Mode | Behavior | Use Case |
|------|----------|----------|
| **Off** | No IAM checks (legacy) | Fast iteration, prototyping |
| **Permissive** | Enforce when IAM available (fail-open) | Development, graceful degradation |
| **Strict** | Always enforce (fail-closed) | CI/CD, pre-production validation |

**Fail-open vs fail-closed** is the control plane's security posture. This lets you choose between availability (permissive) and security (strict).

---

## How It Works (Technical Flow)

### 1. Request arrives at service emulator
```go
// Secret Manager receives GetSecret request
func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) {
    principal := emulatorauth.ExtractPrincipalFromContext(ctx)
    // principal = "user:alice@example.com"
```

### 2. Emulator auth checks permission
```go
    allowed, err := s.iamClient.CheckPermission(
        ctx,
        principal,                              // who
        req.Name,                               // what resource
        "secretmanager.secrets.get",            // which permission
    )
```

### 3. IAM emulator evaluates policy
```yaml
# Policy evaluated by IAM emulator
projects/my-project:
  bindings:
    - role: roles/secretmanager.secretAccessor
      members:
        - user:alice@example.com
```

### 4. Result enforced
```go
    if !allowed {
        return nil, status.Error(codes.PermissionDenied, "Permission denied")
    }
    // Only reaches here if authorized
    return s.storage.GetSecret(req.Name)
}
```

---

## What This Is NOT

### ❌ Not a Mock
- Real policies, real enforcement, real denials
- Not just returning fake data

### ❌ Not an Observer
- Denies requests in real-time, doesn't just log them
- Not post-hoc analysis (like iamlive)

### ❌ Not SDK Stubs
- Actual gRPC/HTTP servers with real protocol implementations
- Not in-process fakes

### ❌ Not Permissive by Default
- Strict mode fails-closed like production
- Configurable security posture

---

## Category Language (Use Verbatim)

### Primary Descriptor
"Local IAM Control Plane for GCP Emulators"

### Elevator Pitch
"Enforce real GCP IAM policies in local development and CI — make your emulators fail exactly like production would."

### Technical Description
"Pre-flight IAM enforcement layer that evaluates permissions before data access, using production policy semantics across all service emulators."

### One-Sentence Comparison
"Unlike mocks (which allow everything) or observers (which record after the fact), this control plane denies unauthorized requests before they reach emulators."

---

## Positioning Against iamlive

iamlive is the only well-known tool in this space, but it's fundamentally different:

| Dimension | iamlive | Blackwell IAM |
|-----------|---------|---------------|
| **Approach** | Passive observer | Active enforcer |
| **Timing** | After request completes | Before request executes |
| **Action** | Records permissions used | Denies unauthorized requests |
| **Cloud** | AWS only | GCP |
| **Testing** | Discovers what you need | Validates what you have |
| **CI/CD** | Audit trail | Enforcement gate |
| **Failure** | Never fails requests | Fails like production |

**Key distinction:** iamlive tells you what permissions you *used*. Blackwell IAM tells you if you're *allowed* to use them.

---

## Repository Roles (Explicit Hierarchy)

### 1. Control Plane (The Product)
- **`gcp-iam-emulator`** - Policy engine (the brain)
- **`gcp-emulator-auth`** - Enforcement proxy (the guard)

These define the category. Everything else depends on them.

### 2. Data Planes (Consumers)
- **`gcp-secret-manager-emulator`** - Secured by IAM layer
- **`gcp-kms-emulator`** - Secured by IAM layer
- Future: Pub/Sub, Tasks, Spanner, etc.

These prove the control plane works across services.

### 3. Orchestration (UX)
- **`gcp-emulator-control-plane`** - CLI to manage the stack

This is the "kubectl" of the system — important, but not the thing.

---

## SEO Keywords (Natural Usage)

Include these naturally in repository READMEs:
- Local IAM control plane
- IAM enforcement proxy
- GCP permission testing
- Test IAM policies locally
- Fail-open fail-closed
- IAM emulator
- Local GCP development
- CI/CD permission testing
- Deterministic IAM testing
- Pre-flight authorization

---

## What This Unlocks

Once you adopt this framing consistently:

1. **Clarity** - One-sentence explanation of the entire system
2. **Differentiation** - Clear separation from mocks and observers
3. **Discoverability** - High-traffic emulators funnel to control plane
4. **Extensibility** - Future emulators snap into place naturally
5. **Credibility** - Infrastructure primitive, not just a library
6. **Monetization** - Control planes are products, libraries are features

---

## Usage Instructions

### For README Updates

1. Lead with: "Local IAM Control Plane"
2. Include the category contrast table
3. Show the architecture diagram
4. Reference this document: `See [CATEGORY.md](../gcp-emulator-auth/CATEGORY.md) for the full category definition`

### For Documentation

Use the exact phrasing:
- "Local IAM Control Plane" (not "IAM library" or "auth helper")
- "Enforcement proxy" (not "auth middleware")
- "Pre-flight authorization" (not "permission checking")
- "Production semantics" (not "realistic behavior")

### For Comparisons

Always use the three-column table:
- Mock / Observer / **Control Plane**
- Never / After / **Before**

---

## Why This Matters

**You didn't just build a better emulator library. You created a new infrastructure primitive.**

Control planes define categories. CLIs never do.

- Kubernetes is not `kubectl`
- Terraform is not `terraform plan`
- GCP is not `gcloud`

Your system is the same class of thing.

The category existed in reality before it existed in language. Now it has both.

---

## License

This category definition is part of the Blackwell Systems documentation.
Use this language freely across all Blackwell repositories.
