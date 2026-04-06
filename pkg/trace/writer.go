package trace

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const EnvTraceOutput = "IAM_TRACE_OUTPUT"

type Writer struct {
	mu     sync.Mutex
	out    io.WriteCloser
	bw     *bufio.Writer
	closed bool
}

// NewWriterFromEnv returns (nil, nil) if tracing is disabled (env var not set).
// Supported values:
// - "stdout"
// - "/path/to/authz.jsonl"
func NewWriterFromEnv() (*Writer, error) {
	dest := os.Getenv(EnvTraceOutput)
	if dest == "" {
		return nil, nil
	}
	return NewWriter(dest)
}

// NewWriter creates a trace writer.
// Supported destinations:
// - "stdout" → writes to os.Stdout
// - "/path/to/file.jsonl" → creates/appends to file
func NewWriter(dest string) (*Writer, error) {
	if dest == "" {
		return nil, errors.New("trace destination cannot be empty")
	}

	var out io.WriteCloser

	if strings.ToLower(dest) == "stdout" {
		out = os.Stdout
	} else {
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open trace file: %w", err)
		}
		out = f
	}

	return &Writer{
		out: out,
		bw:  bufio.NewWriter(out),
	}, nil
}

// Emit writes an event to the trace output as a single JSON line.
// Thread-safe. Does not flush automatically (use Flush or defer Close).
func (w *Writer) Emit(ev AuthzEvent) error {
	if w == nil {
		return nil // tracing disabled
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer is closed")
	}

	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal trace event: %w", err)
	}

	if _, err := w.bw.Write(data); err != nil {
		return fmt.Errorf("failed to write trace event: %w", err)
	}
	if err := w.bw.WriteByte('\n'); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// Flush flushes the buffered writer.
func (w *Writer) Flush() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	return w.bw.Flush()
}

// Close flushes and closes the writer.
// Safe to call multiple times.
func (w *Writer) Close() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true

	if err := w.bw.Flush(); err != nil {
		return err
	}

	// Don't close stdout
	if w.out != os.Stdout {
		return w.out.Close()
	}

	return nil
}
