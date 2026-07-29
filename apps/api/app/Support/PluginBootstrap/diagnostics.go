package pluginbootstrap

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"
)

const DefaultDiagnosticLimit = 16 << 10

var (
	ErrBootstrapABIIncompatible = errors.New("plugin bootstrap ABI is incompatible")
	ErrExecutableArchitecture   = errors.New("plugin executable architecture is incompatible")
	ErrExecutableDependency     = errors.New("plugin executable dependency is unavailable")
	ErrExecutablePermission     = errors.New("plugin executable permission was denied")
	ErrProcessStart             = errors.New("plugin process failed to start")
)

type DiagnosticBuffer struct {
	mu        sync.Mutex
	remaining int
	body      strings.Builder
}

func NewDiagnosticBuffer(limit int) *DiagnosticBuffer {
	if limit <= 0 || limit > DefaultDiagnosticLimit {
		limit = DefaultDiagnosticLimit
	}
	return &DiagnosticBuffer{remaining: limit}
}

func (b *DiagnosticBuffer) Write(value []byte) (int, error) {
	if b == nil {
		return len(value), nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	write := len(value)
	if write > b.remaining {
		write = b.remaining
	}
	if write > 0 {
		_, _ = b.body.Write(value[:write])
		b.remaining -= write
	}
	return len(value), nil
}

func (b *DiagnosticBuffer) String() string {
	if b == nil {
		return ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.body.String()
}

func ClassifyStartError(cause error, diagnostic string) error {
	if cause == nil {
		return nil
	}
	causeText := strings.ToLower(cause.Error())
	diagnosticText := strings.ToLower(diagnostic)
	text := causeText + "\n" + diagnosticText
	kind := ErrProcessStart
	switch {
	case strings.Contains(text, "this binary is a plugin") || strings.Contains(text, "magic cookie"):
		kind = ErrBootstrapABIIncompatible
	case strings.Contains(text, "exec format") || strings.Contains(text, "bad cpu type") ||
		strings.Contains(text, "wrong architecture"):
		kind = ErrExecutableArchitecture
	case strings.Contains(text, "library not loaded") ||
		strings.Contains(text, "error while loading shared libraries") ||
		strings.Contains(text, "cannot open shared object file") ||
		strings.Contains(text, "symbol not found") || strings.Contains(text, "dll not found"):
		kind = ErrExecutableDependency
	case strings.Contains(text, "permission denied") || strings.Contains(diagnosticText, "not executable"):
		kind = ErrExecutablePermission
	}
	if diagnostic = SanitizedDiagnostic(diagnostic); diagnostic != "" {
		return fmt.Errorf("%w: %v; plugin stderr: %s", kind, cause, diagnostic)
	}
	return fmt.Errorf("%w: %v", kind, cause)
}

// SanitizedDiagnostic is intended for bounded local logs only. It removes
// control characters but deliberately is not included in public readiness.
func SanitizedDiagnostic(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
	return strings.TrimSpace(value)
}
