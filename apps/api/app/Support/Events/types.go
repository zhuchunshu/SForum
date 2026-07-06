package events

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

const (
	KindObserve  = "observe"
	KindValidate = "validate"
	KindFilter   = "filter"

	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusSkipped   = "skipped"

	DefaultSyncTimeoutMS  = 2000
	DefaultAsyncTimeoutMS = 5000
)

type Definition struct {
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	Description   string   `json:"description"`
	PayloadFields []string `json:"payloadFields,omitempty"`
	PatchFields   []string `json:"patchFields,omitempty"`
	TimeoutMS     int      `json:"timeoutMs"`
}

type Envelope struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	ActorUserID   int64          `json:"actorUserId,omitempty"`
	ResourceType  string         `json:"resourceType,omitempty"`
	ResourceID    string         `json:"resourceId,omitempty"`
	CorrelationID string         `json:"correlationId,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	PatchFields   []string       `json:"patchFields,omitempty"`
	OccurredAt    time.Time      `json:"occurredAt"`
}

type Result struct {
	OK      bool           `json:"ok"`
	Reason  string         `json:"reason,omitempty"`
	Message string         `json:"message,omitempty"`
	Patch   map[string]any `json:"patch,omitempty"`
}

type RejectedError struct {
	Reason  string
	Message string
}

func (e *RejectedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "event rejected"
}

func Reject(result Result) error {
	reason := result.Reason
	if reason == "" {
		reason = "extension.event_rejected"
	}
	return &RejectedError{Reason: reason, Message: result.Message}
}

type Publisher interface {
	Emit(ctx context.Context, envelope Envelope) Result
}

type NoopPublisher struct{}

func (NoopPublisher) Emit(context.Context, Envelope) Result {
	return Result{OK: true}
}

func NewEnvelope(name string, payload map[string]any) Envelope {
	definition, _ := FindDefinition(name)
	return Envelope{
		ID:            NewID(),
		Name:          name,
		Kind:          definition.Kind,
		CorrelationID: NewID(),
		Payload:       payload,
		PatchFields:   append([]string{}, definition.PatchFields...),
		OccurredAt:    time.Now().UTC(),
	}
}

func EnsurePublisher(publisher Publisher) Publisher {
	if publisher == nil {
		return NoopPublisher{}
	}
	return publisher
}

func NewID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(value[:])
}
