// Package hosthttp is the Host-owned outbound HTTP client for plugins and core.
//
// Default path uses SSRF-safe dialing from OutboundHTTP. Fully trusted processes
// may opt into raw network authority only when the Host grants it explicitly.
// Credential values are injected from SecretStore references — never from plugin
// plaintext settings on the wire.
package hosthttp

import (
	"errors"
	"time"
)

// SchemaVersion is the Host HTTP client contract identity.
const SchemaVersion = "sforum.host-http@1"

const (
	DefaultTimeout       = 15 * time.Second
	DefaultMaxRedirects  = 3
	DefaultMaxRetries    = 2
	DefaultMaxBodyBytes  = 2 * 1024 * 1024 // 2 MiB
	MaxMaxBodyBytes      = 16 * 1024 * 1024
	DefaultRetryBackoff  = 100 * time.Millisecond
	MaxHeaderBytesStored = 8 * 1024
)

var (
	ErrInvalid          = errors.New("host http request is invalid")
	ErrPermissionDenied = errors.New("host http permission denied")
	ErrSSRF             = errors.New("host http url is not public")
	ErrResponseTooLarge = errors.New("host http response exceeds limit")
	ErrRetriesExhausted = errors.New("host http retries exhausted")
	ErrSecret           = errors.New("host http secret resolve failed")
	ErrRawDenied        = errors.New("host http raw network authority denied")
)

// Authority selects the network path.
const (
	// AuthoritySafe is the default SSRF-hardened public-only client.
	AuthoritySafe = "safe"
	// AuthorityRaw requires Host grant for fully trusted processes (no SSRF).
	AuthorityRaw = "raw"
)

// Request is a Host-mediated outbound call.
type Request struct {
	// Method defaults to GET.
	Method string
	// URL is the absolute target. Safe authority rejects private/link-local hosts.
	URL string
	// Headers are copied; Host may inject Authorization from SecretRef.
	Headers map[string]string
	// Body is optional request body (cloned on send).
	Body []byte
	// Timeout overrides client default when > 0.
	Timeout time.Duration
	// MaxBodyBytes caps response body; 0 uses DefaultMaxBodyBytes.
	MaxBodyBytes int64
	// MaxRetries is additional attempts after the first (0 = no retry).
	MaxRetries int
	// Authority is safe|raw. Empty defaults to safe.
	Authority string
	// SecretRef injects Authorization: Bearer <value> when set.
	// Format: sforum.secret://namespace/id or empty.
	SecretRef string
	// SecretPurpose is required when SecretRef is set (default http.credential).
	SecretPurpose string
	// Actor is optional audit identity.
	Actor string
	// ExtensionID admits SecretStore namespace for plugin callers.
	ExtensionID string
	// TraceID is echoed into response trace for correlation.
	TraceID string
}

// Response is a bounded, non-streaming result.
type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"-"`
	Attempts   int               `json:"attempts"`
	Duration   time.Duration     `json:"duration"`
	Trace      Trace             `json:"trace"`
}

// Trace records operator-safe outbound evidence (no bodies/secrets).
type Trace struct {
	SchemaVersion string        `json:"schemaVersion"`
	TraceID       string        `json:"traceId,omitempty"`
	Method        string        `json:"method"`
	Host          string        `json:"host"`
	Authority     string        `json:"authority"`
	StatusCode    int           `json:"statusCode,omitempty"`
	Attempts      int           `json:"attempts"`
	Duration      time.Duration `json:"duration"`
	ErrorClass    string        `json:"errorClass,omitempty"`
	SecretUsed    bool          `json:"secretUsed,omitempty"`
	ExtensionID   string        `json:"extensionId,omitempty"`
	Actor         string        `json:"actor,omitempty"`
}

// Metrics is process-local inspector counters.
type Metrics struct {
	Requests   uint64 `json:"requests"`
	Successes  uint64 `json:"successes"`
	Failures   uint64 `json:"failures"`
	SSRFDenies uint64 `json:"ssrfDenies"`
	Retries    uint64 `json:"retries"`
	BytesIn    uint64 `json:"bytesIn"`
}
