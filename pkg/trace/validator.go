package trace

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ValidationError struct {
	Line int
	Msg  string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("trace validation error at line %d: %s", e.Line, e.Msg)
}

type Validator struct {
	SupportedSchemaVersions map[string]bool
}

func NewValidator() *Validator {
	return &Validator{
		SupportedSchemaVersions: map[string]bool{
			SchemaV1_0: true,
		},
	}
}

func (v *Validator) ValidateEvent(ev *AuthzEvent) error {
	if ev.SchemaVersion == "" {
		return fmt.Errorf("missing schema_version")
	}
	if !v.SupportedSchemaVersions[ev.SchemaVersion] {
		return fmt.Errorf("unsupported schema_version: %s", ev.SchemaVersion)
	}
	if ev.EventType == "" {
		return fmt.Errorf("missing event_type")
	}
	if ev.Timestamp == "" {
		return fmt.Errorf("missing timestamp")
	}

	switch ev.EventType {
	case EventTypeAuthzCheck:
		// required fields per spec
		if ev.Actor == nil || strings.TrimSpace(ev.Actor.Principal) == "" {
			return fmt.Errorf("missing actor.principal")
		}
		if ev.Target == nil || strings.TrimSpace(ev.Target.Resource) == "" {
			return fmt.Errorf("missing target.resource")
		}
		if ev.Action == nil || strings.TrimSpace(ev.Action.Permission) == "" {
			return fmt.Errorf("missing action.permission")
		}
		if ev.Decision == nil || strings.TrimSpace(ev.Decision.Outcome) == "" {
			return fmt.Errorf("missing decision.outcome")
		}
		if ev.Decision.Outcome != OutcomeAllow && ev.Decision.Outcome != OutcomeDeny {
			return fmt.Errorf("invalid decision.outcome: %s", ev.Decision.Outcome)
		}
		return nil

	case EventTypeAuthzError:
		// For errors, we require error.kind/message.
		if ev.Error == nil {
			return fmt.Errorf("missing error object")
		}
		if strings.TrimSpace(ev.Error.Kind) == "" {
			return fmt.Errorf("missing error.kind")
		}
		if strings.TrimSpace(ev.Error.Message) == "" {
			return fmt.Errorf("missing error.message")
		}
		// decision is not required for authz_error
		return nil

	default:
		return fmt.Errorf("unknown event_type: %s", ev.EventType)
	}
}

// ValidateJSONLFile validates a JSONL trace file. Supports .gz by extension.
func (v *Validator) ValidateJSONLFile(path string) error {
	r, closeFn, err := openMaybeGzip(path)
	if err != nil {
		return err
	}
	defer closeFn()

	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var ev AuthzEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return ValidationError{Line: lineNum, Msg: fmt.Sprintf("invalid JSON: %v", err)}
		}

		if err := v.ValidateEvent(&ev); err != nil {
			return ValidationError{Line: lineNum, Msg: err.Error()}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// openMaybeGzip opens a file, decompressing if it ends with .gz
func openMaybeGzip(path string) (io.Reader, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	if filepath.Ext(path) == ".gz" {
		gr, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		return gr, func() error {
			gr.Close()
			return f.Close()
		}, nil
	}

	return f, f.Close, nil
}
