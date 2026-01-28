package trace

import "time"

// Schema versioning
const (
	SchemaV1_0 = "1.0"
)

// Event types
const (
	EventTypeAuthzCheck = "authz_check"
	EventTypeAuthzError = "authz_error"
)

// Decision outcomes
const (
	OutcomeAllow = "ALLOW"
	OutcomeDeny  = "DENY"
)

// AuthzEvent is the union envelope. Fields not applicable to an event_type may be omitted.
type AuthzEvent struct {
	SchemaVersion string `json:"schema_version"`
	EventType     string `json:"event_type"`
	Timestamp     string `json:"timestamp"` // ISO-8601 with timezone

	Trace       *TraceContext `json:"trace,omitempty"`
	Actor       *Actor        `json:"actor,omitempty"`
	Target      *Target       `json:"target,omitempty"`
	Action      *Action       `json:"action,omitempty"`
	Decision    *Decision     `json:"decision,omitempty"`
	Policy      *Policy       `json:"policy,omitempty"`
	Environment *Environment  `json:"environment,omitempty"`

	Error *AuthzError `json:"error,omitempty"`
}

type TraceContext struct {
	TraceID      string `json:"trace_id,omitempty"`
	SpanID       string `json:"span_id,omitempty"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
}

type Actor struct {
	Principal     string   `json:"principal"`
	PrincipalType string   `json:"principal_type,omitempty"`
	Groups        []string `json:"groups,omitempty"`
	Source        *Source  `json:"source,omitempty"`
}

type Source struct {
	Channel string `json:"channel,omitempty"`
	Name    string `json:"name,omitempty"`
}

type Target struct {
	Resource     string  `json:"resource"`
	ResourceType string  `json:"resource_type,omitempty"`
	Project      string  `json:"project,omitempty"`
	Location     *string `json:"location,omitempty"`
	Service      string  `json:"service,omitempty"`
}

type Action struct {
	Permission string      `json:"permission"`
	Method     string      `json:"method,omitempty"`
	API        *APIDetails `json:"api,omitempty"`
}

type APIDetails struct {
	Protocol  string `json:"protocol,omitempty"`
	Operation string `json:"operation,omitempty"`
}

type Decision struct {
	Outcome     string `json:"outcome"`
	Reason      string `json:"reason,omitempty"`
	EvaluatedBy string `json:"evaluated_by,omitempty"`
	LatencyMS   int64  `json:"latency_ms,omitempty"`
}

type Policy struct {
	PolicyHash      string           `json:"policy_hash,omitempty"`
	MatchedBindings []MatchedBinding `json:"matched_bindings,omitempty"`
}

type MatchedBinding struct {
	Scope     string     `json:"scope,omitempty"`
	ScopeID   string     `json:"scope_id,omitempty"`
	Role      string     `json:"role,omitempty"`
	Member    string     `json:"member,omitempty"`
	Condition *Condition `json:"condition,omitempty"`
}

type Condition struct {
	Title      string `json:"title,omitempty"`
	Expression string `json:"expression,omitempty"`
	Result     bool   `json:"result,omitempty"`
}

type Environment struct {
	Mode      string `json:"mode,omitempty"`
	Component string `json:"component,omitempty"`
	Cluster   string `json:"cluster,omitempty"`
	CI        *CI    `json:"ci,omitempty"`
}

type CI struct {
	Provider string `json:"provider,omitempty"`
	RunID    string `json:"run_id,omitempty"`
	Job      string `json:"job,omitempty"`
}

type AuthzError struct {
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

// NowRFC3339Nano returns the current time in ISO-8601 format with nanosecond precision.
func NowRFC3339Nano() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
