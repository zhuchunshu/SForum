package entitlements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ScopeResource   = "resource"
	ScopeCapability = "capability"

	StatusActive  = "active"
	StatusRevoked = "revoked"
	StatusExpired = "expired"

	ActionGrant  = "grant"
	ActionRevoke = "revoke"
	ActionExpire = "expire"
)

var (
	ErrInvalidInput        = errors.New("entitlements: invalid input")
	ErrNotFound            = errors.New("entitlements: not found")
	ErrStateConflict       = errors.New("entitlements: lifecycle state conflict")
	ErrIdempotencyConflict = errors.New("entitlements: idempotency fingerprint conflict")
	ErrNotYetExpired       = errors.New("entitlements: validity window has not expired")
)

type Subject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Scope struct {
	Kind         string `json:"kind"`
	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
	Capability   string `json:"capability,omitempty"`
}

type Source struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Entitlement struct {
	ID         int64      `json:"id"`
	Subject    Subject    `json:"subject"`
	Scope      Scope      `json:"scope"`
	Status     string     `json:"status"`
	Source     Source     `json:"source"`
	ValidFrom  time.Time  `json:"validFrom"`
	ValidUntil *time.Time `json:"validUntil,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
	ExpiredAt  *time.Time `json:"expiredAt,omitempty"`
	Revision   int64      `json:"revision"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Event struct {
	ID                 int64     `json:"id"`
	EntitlementID      int64     `json:"entitlementId"`
	Action             string    `json:"action"`
	IdempotencyKey     string    `json:"idempotencyKey"`
	RequestFingerprint string    `json:"requestFingerprint"`
	PreviousStatus     string    `json:"previousStatus,omitempty"`
	NextStatus         string    `json:"nextStatus"`
	ActorUserID        int64     `json:"actorUserId,omitempty"`
	AuditEventID       int64     `json:"auditEventId"`
	CreatedAt          time.Time `json:"createdAt"`
}

type MutationResult struct {
	Entitlement Entitlement `json:"entitlement"`
	Event       Event       `json:"event"`
	Replayed    bool        `json:"replayed"`
}

type GrantInput struct {
	Subject        Subject
	Scope          Scope
	Source         Source
	ValidFrom      time.Time
	ValidUntil     *time.Time
	ActorUserID    int64
	IdempotencyKey string
}

type TransitionInput struct {
	EntitlementID  int64
	ActorUserID    int64
	IdempotencyKey string
}

type EffectiveInput struct {
	Subject Subject
	Scope   Scope
	At      time.Time
}

// ValidateGrant validates the provider-neutral grant contract without writing
// data. Transactional Host Commands use it for dry-run plans before calling
// GrantTx inside the authoritative transaction.
func ValidateGrant(input GrantInput) error {
	_, _, err := prepareGrant(input)
	return err
}

// ValidateTransition validates a revoke/expire request without reading or
// mutating lifecycle state. Current state remains authoritative in the Tx path.
func ValidateTransition(action string, input TransitionInput) error {
	_, _, err := prepareTransition(action, input)
	return err
}

type preparedGrant struct {
	Subject        Subject    `json:"subject"`
	Scope          Scope      `json:"scope"`
	Source         Source     `json:"source"`
	ValidFrom      time.Time  `json:"validFrom"`
	ValidUntil     *time.Time `json:"validUntil,omitempty"`
	ActorUserID    int64      `json:"actorUserId,omitempty"`
	IdempotencyKey string     `json:"-"`
}

type preparedTransition struct {
	Action         string `json:"action"`
	EntitlementID  int64  `json:"entitlementId"`
	ActorUserID    int64  `json:"actorUserId,omitempty"`
	IdempotencyKey string `json:"-"`
}

type grantFingerprint struct {
	Action      string     `json:"action"`
	Subject     Subject    `json:"subject"`
	Scope       Scope      `json:"scope"`
	Source      Source     `json:"source"`
	ValidFrom   time.Time  `json:"validFrom"`
	ValidUntil  *time.Time `json:"validUntil,omitempty"`
	ActorUserID int64      `json:"actorUserId,omitempty"`
}

func prepareGrant(input GrantInput) (preparedGrant, string, error) {
	result := preparedGrant{
		Subject: normalizeSubject(input.Subject), Scope: normalizeScope(input.Scope),
		Source: normalizeSource(input.Source), ValidFrom: input.ValidFrom.UTC(),
		ActorUserID: input.ActorUserID, IdempotencyKey: input.IdempotencyKey,
	}
	if input.ValidUntil != nil {
		value := input.ValidUntil.UTC()
		result.ValidUntil = &value
	}
	if err := validateSubject(result.Subject); err != nil {
		return preparedGrant{}, "", err
	}
	if err := validateScope(result.Scope); err != nil {
		return preparedGrant{}, "", err
	}
	if err := validateSource(result.Source); err != nil {
		return preparedGrant{}, "", err
	}
	if input.ValidFrom.IsZero() {
		return preparedGrant{}, "", fmt.Errorf("%w: validFrom is required", ErrInvalidInput)
	}
	if result.ValidUntil != nil && !result.ValidUntil.After(result.ValidFrom) {
		return preparedGrant{}, "", fmt.Errorf("%w: validUntil must be after validFrom", ErrInvalidInput)
	}
	if err := validateActorAndKey(result.ActorUserID, result.IdempotencyKey); err != nil {
		return preparedGrant{}, "", err
	}
	fingerprint, err := requestFingerprint(grantFingerprint{
		Action: ActionGrant, Subject: result.Subject, Scope: result.Scope, Source: result.Source,
		ValidFrom: result.ValidFrom, ValidUntil: result.ValidUntil, ActorUserID: result.ActorUserID,
	})
	return result, fingerprint, err
}

func prepareTransition(action string, input TransitionInput) (preparedTransition, string, error) {
	result := preparedTransition{
		Action: action, EntitlementID: input.EntitlementID,
		ActorUserID: input.ActorUserID, IdempotencyKey: input.IdempotencyKey,
	}
	if action != ActionRevoke && action != ActionExpire {
		return preparedTransition{}, "", fmt.Errorf("%w: unsupported action", ErrInvalidInput)
	}
	if result.EntitlementID <= 0 {
		return preparedTransition{}, "", fmt.Errorf("%w: entitlement id is required", ErrInvalidInput)
	}
	if err := validateActorAndKey(result.ActorUserID, result.IdempotencyKey); err != nil {
		return preparedTransition{}, "", err
	}
	fingerprint, err := requestFingerprint(result)
	return result, fingerprint, err
}

func prepareEffective(input EffectiveInput) (EffectiveInput, error) {
	input.Subject = normalizeSubject(input.Subject)
	input.Scope = normalizeScope(input.Scope)
	input.At = input.At.UTC()
	if err := validateSubject(input.Subject); err != nil {
		return EffectiveInput{}, err
	}
	if err := validateScope(input.Scope); err != nil {
		return EffectiveInput{}, err
	}
	if input.At.IsZero() {
		return EffectiveInput{}, fmt.Errorf("%w: check time is required", ErrInvalidInput)
	}
	return input, nil
}

func requestFingerprint(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("%w: encode request fingerprint: %v", ErrInvalidInput, err)
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func normalizeSubject(value Subject) Subject {
	return Subject{Type: strings.TrimSpace(value.Type), ID: strings.TrimSpace(value.ID)}
}

func normalizeScope(value Scope) Scope {
	return Scope{
		Kind: strings.TrimSpace(value.Kind), ResourceType: strings.TrimSpace(value.ResourceType),
		ResourceID: strings.TrimSpace(value.ResourceID), Capability: strings.TrimSpace(value.Capability),
	}
}

func normalizeSource(value Source) Source {
	return Source{Type: strings.TrimSpace(value.Type), ID: strings.TrimSpace(value.ID)}
}

func validateSubject(value Subject) error {
	if !validText(value.Type, 100) || !validText(value.ID, 512) {
		return fmt.Errorf("%w: subject type and id are required", ErrInvalidInput)
	}
	return nil
}

func validateScope(value Scope) error {
	switch value.Kind {
	case ScopeResource:
		if !validText(value.ResourceType, 100) || !validText(value.ResourceID, 512) || value.Capability != "" {
			return fmt.Errorf("%w: resource scope requires only resource type and id", ErrInvalidInput)
		}
	case ScopeCapability:
		if !validText(value.Capability, 200) || value.ResourceType != "" || value.ResourceID != "" {
			return fmt.Errorf("%w: capability scope requires only capability", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: scope kind must be resource or capability", ErrInvalidInput)
	}
	return nil
}

func validateSource(value Source) error {
	if !validText(value.Type, 100) || !validText(value.ID, 512) {
		return fmt.Errorf("%w: source type and id are required", ErrInvalidInput)
	}
	return nil
}

func validateActorAndKey(actorUserID int64, key string) error {
	if actorUserID < 0 {
		return fmt.Errorf("%w: actor user id cannot be negative", ErrInvalidInput)
	}
	if key == "" || len(key) > 128 || strings.TrimSpace(key) != key {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidInput)
	}
	for index := range len(key) {
		if key[index] < '!' || key[index] > '~' {
			return fmt.Errorf("%w: idempotency key must contain visible ASCII", ErrInvalidInput)
		}
	}
	return nil
}

func validText(value string, max int) bool {
	return value != "" && len(value) <= max && strings.TrimSpace(value) == value
}
