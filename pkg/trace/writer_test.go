package trace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriter_EmitToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "authz.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	ev := AuthzEvent{
		SchemaVersion: SchemaV1_0,
		EventType:     EventTypeAuthzCheck,
		Timestamp:     "2026-01-27T18:03:12.483Z",
		Actor:         &Actor{Principal: "serviceAccount:ci@test-project.iam.gserviceaccount.com"},
		Target:        &Target{Resource: "projects/test-project/secrets/prod-db-password"},
		Action:        &Action{Permission: "secretmanager.versions.access"},
		Decision:      &Decision{Outcome: OutcomeAllow},
	}

	if err := w.Emit(ev); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected file to contain data")
	}
}

func TestNewWriterFromEnv_Disabled(t *testing.T) {
	os.Unsetenv(EnvTraceOutput)

	w, err := NewWriterFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != nil {
		t.Fatal("expected nil writer when tracing disabled")
	}
}

func TestNewWriterFromEnv_Enabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.jsonl")

	os.Setenv(EnvTraceOutput, path)
	defer os.Unsetenv(EnvTraceOutput)

	w, err := NewWriterFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil writer when tracing enabled")
	}
	defer w.Close()
}

func TestWriter_EmitWhenNil(t *testing.T) {
	var w *Writer
	ev := AuthzEvent{
		SchemaVersion: SchemaV1_0,
		EventType:     EventTypeAuthzCheck,
		Timestamp:     NowRFC3339Nano(),
		Actor:         &Actor{Principal: "user:test@example.com"},
		Target:        &Target{Resource: "projects/test/secrets/foo"},
		Action:        &Action{Permission: "secretmanager.secrets.get"},
		Decision:      &Decision{Outcome: OutcomeAllow},
	}

	// Should not panic
	if err := w.Emit(ev); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
