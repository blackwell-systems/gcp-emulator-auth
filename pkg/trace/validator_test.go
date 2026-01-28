package trace

import (
	"path/filepath"
	"testing"
)

func TestValidator_ValidateEvent_Check_MinimumRequired(t *testing.T) {
	v := NewValidator()

	ev := AuthzEvent{
		SchemaVersion: SchemaV1_0,
		EventType:     EventTypeAuthzCheck,
		Timestamp:     "2026-01-27T18:03:12.483Z",
		Actor:         &Actor{Principal: "serviceAccount:ci@test-project.iam.gserviceaccount.com"},
		Target:        &Target{Resource: "projects/test-project/secrets/prod-db-password"},
		Action:        &Action{Permission: "secretmanager.versions.access"},
		Decision:      &Decision{Outcome: OutcomeAllow},
	}

	if err := v.ValidateEvent(&ev); err != nil {
		t.Fatalf("expected valid event, got: %v", err)
	}
}

func TestValidator_ValidateEvent_Error_MinimumRequired(t *testing.T) {
	v := NewValidator()

	ev := AuthzEvent{
		SchemaVersion: SchemaV1_0,
		EventType:     EventTypeAuthzError,
		Timestamp:     "2026-01-27T18:03:12.490Z",
		Error: &AuthzError{
			Kind:      "iam_unreachable",
			Message:   "connection refused",
			Retryable: true,
		},
	}

	if err := v.ValidateEvent(&ev); err != nil {
		t.Fatalf("expected valid error event, got: %v", err)
	}
}

func TestValidator_ValidateEvent_MissingSchemaVersion(t *testing.T) {
	v := NewValidator()

	ev := AuthzEvent{
		EventType: EventTypeAuthzCheck,
		Timestamp: "2026-01-27T18:03:12.483Z",
		Actor:     &Actor{Principal: "user:test@example.com"},
		Target:    &Target{Resource: "projects/test/secrets/foo"},
		Action:    &Action{Permission: "secretmanager.secrets.get"},
		Decision:  &Decision{Outcome: OutcomeAllow},
	}

	if err := v.ValidateEvent(&ev); err == nil {
		t.Fatal("expected error for missing schema_version")
	}
}

func TestValidator_ValidateEvent_InvalidOutcome(t *testing.T) {
	v := NewValidator()

	ev := AuthzEvent{
		SchemaVersion: SchemaV1_0,
		EventType:     EventTypeAuthzCheck,
		Timestamp:     "2026-01-27T18:03:12.483Z",
		Actor:         &Actor{Principal: "user:test@example.com"},
		Target:        &Target{Resource: "projects/test/secrets/foo"},
		Action:        &Action{Permission: "secretmanager.secrets.get"},
		Decision:      &Decision{Outcome: "INVALID"},
	}

	if err := v.ValidateEvent(&ev); err == nil {
		t.Fatal("expected error for invalid outcome")
	}
}

func TestValidator_ValidateJSONLFile_GoldenSample(t *testing.T) {
	v := NewValidator()
	path := filepath.Join("..", "..", "testdata", "sample-trace.jsonl")
	if err := v.ValidateJSONLFile(path); err != nil {
		t.Fatalf("expected golden sample to validate, got: %v", err)
	}
}
