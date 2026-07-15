package idempotency

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	RequiredReplayPolicy     = "required.24h@1"
	MaxRequiredReplayBody    = 8 << 20
	MaxRequiredReplayHeaders = 64 << 10
	requiredReplaySchema     = "sforum.required-route-replay@1"
	requiredReplayPending    = "pending"
	requiredReplayCompleted  = "completed"
)

var (
	ErrRequiredReplayInvalid             = errors.New("idempotency: required replay input is invalid")
	ErrRequiredReplayInProgress          = errors.New("idempotency: required replay is in progress")
	ErrRequiredReplayFingerprintConflict = errors.New("idempotency: required replay fingerprint conflict")
	ErrRequiredReplayUnavailable         = errors.New("idempotency: required replay storage is unavailable")
	ErrRequiredReplayLeaseLost           = errors.New("idempotency: required replay lease was lost")
)

// RequiredReplayScope binds one key to the live actor and exact route artifact.
// The serialized scope is hashed into the Redis key and is never stored as data.
type RequiredReplayScope struct {
	ActorScope       string `json:"actorScope"`
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	PackageDigest    string `json:"packageDigest"`
	RouteID          string `json:"routeId"`
	ContractVersion  string `json:"contractVersion"`
	Method           string `json:"method"`
}

type RequiredReplayResponse struct {
	Status  int         `json:"status"`
	Headers http.Header `json:"headers,omitempty"`
	Body    []byte      `json:"body,omitempty"`
}

// RequiredReplayLease is opaque proof that this request owns the pending row.
type RequiredReplayLease struct {
	storageKey string
	pending    []byte
}

type requiredReplayRecord struct {
	Schema      string                  `json:"schema"`
	State       string                  `json:"state"`
	Fingerprint string                  `json:"fingerprint"`
	LeaseToken  string                  `json:"leaseToken,omitempty"`
	Response    *RequiredReplayResponse `json:"response,omitempty"`
}

func (s *Store) BeginRequiredReplay(
	ctx context.Context,
	scope RequiredReplayScope,
	key string,
	fingerprint string,
) (RequiredReplayLease, *RequiredReplayResponse, error) {
	if s == nil || s.backend == nil {
		return RequiredReplayLease{}, nil, ErrRequiredReplayUnavailable
	}
	if ctx == nil || !validRequiredReplayScope(scope) ||
		!validRequiredReplayKey(key) || !validRequiredReplayFingerprint(fingerprint) {
		return RequiredReplayLease{}, nil, ErrRequiredReplayInvalid
	}
	storageKey, err := requiredReplayStorageKey(scope, key)
	if err != nil {
		return RequiredReplayLease{}, nil, ErrRequiredReplayInvalid
	}
	for range 4 {
		raw, found, getErr := s.backend.Get(ctx, s.fullKey(storageKey))
		if getErr != nil {
			return RequiredReplayLease{}, nil, errors.Join(ErrRequiredReplayUnavailable, getErr)
		}
		if found {
			record, decodeErr := decodeRequiredReplayRecord(raw)
			if decodeErr != nil {
				return RequiredReplayLease{}, nil, errors.Join(ErrRequiredReplayUnavailable, decodeErr)
			}
			if record.Fingerprint != fingerprint {
				return RequiredReplayLease{}, nil, ErrRequiredReplayFingerprintConflict
			}
			switch record.State {
			case requiredReplayPending:
				return RequiredReplayLease{}, nil, ErrRequiredReplayInProgress
			case requiredReplayCompleted:
				return RequiredReplayLease{}, cloneRequiredReplayResponse(record.Response), nil
			default:
				return RequiredReplayLease{}, nil, ErrRequiredReplayUnavailable
			}
		}
		pending, marshalErr := newRequiredReplayPending(fingerprint)
		if marshalErr != nil {
			return RequiredReplayLease{}, nil, errors.Join(ErrRequiredReplayUnavailable, marshalErr)
		}
		acquired, setErr := s.backend.SetNX(ctx, s.fullKey(storageKey), pending, s.ttl)
		if setErr != nil {
			return RequiredReplayLease{}, nil, errors.Join(ErrRequiredReplayUnavailable, setErr)
		}
		if acquired {
			return RequiredReplayLease{storageKey: storageKey, pending: pending}, nil, nil
		}
	}
	return RequiredReplayLease{}, nil, ErrRequiredReplayUnavailable
}

func (s *Store) CompleteRequiredReplay(
	ctx context.Context,
	lease RequiredReplayLease,
	response RequiredReplayResponse,
) error {
	if s == nil || s.backend == nil {
		return ErrRequiredReplayUnavailable
	}
	if ctx == nil || lease.storageKey == "" || len(lease.pending) == 0 ||
		validateRequiredReplayResponse(response) != nil {
		return ErrRequiredReplayInvalid
	}
	pending, err := decodeRequiredReplayRecord(lease.pending)
	if err != nil || pending.State != requiredReplayPending {
		return ErrRequiredReplayInvalid
	}
	completed := requiredReplayRecord{
		Schema: requiredReplaySchema, State: requiredReplayCompleted,
		Fingerprint: pending.Fingerprint, Response: cloneRequiredReplayResponse(&response),
	}
	replacement, err := json.Marshal(completed)
	if err != nil {
		return errors.Join(ErrRequiredReplayUnavailable, err)
	}
	swapped, err := s.backend.CompareAndSwap(
		ctx, s.fullKey(lease.storageKey), lease.pending, replacement, s.ttl,
	)
	if err != nil {
		return errors.Join(ErrRequiredReplayUnavailable, err)
	}
	if !swapped {
		return ErrRequiredReplayLeaseLost
	}
	return nil
}

func (s *Store) AbortRequiredReplay(ctx context.Context, lease RequiredReplayLease) error {
	if s == nil || s.backend == nil {
		return ErrRequiredReplayUnavailable
	}
	if ctx == nil || lease.storageKey == "" || len(lease.pending) == 0 {
		return ErrRequiredReplayInvalid
	}
	_, err := s.backend.CompareAndSwap(ctx, s.fullKey(lease.storageKey), lease.pending, nil, s.ttl)
	if err != nil {
		return errors.Join(ErrRequiredReplayUnavailable, err)
	}
	return nil
}

func newRequiredReplayPending(fingerprint string) ([]byte, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	return json.Marshal(requiredReplayRecord{
		Schema: requiredReplaySchema, State: requiredReplayPending,
		Fingerprint: fingerprint, LeaseToken: hex.EncodeToString(token),
	})
}

func decodeRequiredReplayRecord(raw []byte) (requiredReplayRecord, error) {
	var record requiredReplayRecord
	if len(raw) == 0 || json.Unmarshal(raw, &record) != nil || record.Schema != requiredReplaySchema ||
		!validRequiredReplayFingerprint(record.Fingerprint) {
		return requiredReplayRecord{}, ErrRequiredReplayUnavailable
	}
	switch record.State {
	case requiredReplayPending:
		if len(record.LeaseToken) != 32 || record.Response != nil {
			return requiredReplayRecord{}, ErrRequiredReplayUnavailable
		}
	case requiredReplayCompleted:
		if record.LeaseToken != "" || validateRequiredReplayResponsePointer(record.Response) != nil {
			return requiredReplayRecord{}, ErrRequiredReplayUnavailable
		}
	default:
		return requiredReplayRecord{}, ErrRequiredReplayUnavailable
	}
	return record, nil
}

func requiredReplayStorageKey(scope RequiredReplayScope, key string) (string, error) {
	document, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	_, _ = digest.Write(document)
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(key))
	return "plugin-route:" + RequiredReplayPolicy + ":" + hex.EncodeToString(digest.Sum(nil)), nil
}

func validRequiredReplayScope(scope RequiredReplayScope) bool {
	values := []string{
		scope.ActorScope, scope.ExtensionID, scope.ExtensionVersion, scope.PackageDigest,
		scope.RouteID, scope.ContractVersion, scope.Method,
	}
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) || len(value) > 256 {
			return false
		}
	}
	if !strings.HasPrefix(scope.ActorScope, "actor:") && !strings.HasPrefix(scope.ActorScope, "anonymous:") {
		return false
	}
	if len(scope.PackageDigest) != 64 || scope.PackageDigest != strings.ToLower(scope.PackageDigest) ||
		scope.Method != strings.ToUpper(scope.Method) {
		return false
	}
	_, err := hex.DecodeString(scope.PackageDigest)
	return err == nil
}

func validRequiredReplayKey(key string) bool {
	if len(key) == 0 || len(key) > MaxKeyLength {
		return false
	}
	for index := 0; index < len(key); index++ {
		if key[index] < 0x21 || key[index] > 0x7e {
			return false
		}
	}
	return true
}

func validRequiredReplayFingerprint(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validateRequiredReplayResponsePointer(response *RequiredReplayResponse) error {
	if response == nil {
		return ErrRequiredReplayInvalid
	}
	return validateRequiredReplayResponse(*response)
}

func validateRequiredReplayResponse(response RequiredReplayResponse) error {
	if response.Status < http.StatusOK || response.Status >= http.StatusMultipleChoices ||
		len(response.Body) > MaxRequiredReplayBody {
		return ErrRequiredReplayInvalid
	}
	headerBytes := 0
	for name, values := range response.Headers {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if canonical == "" || canonical == "set-cookie" || canonical == "cookie" ||
			canonical == "authorization" || canonical == "proxy-authorization" {
			return ErrRequiredReplayInvalid
		}
		headerBytes += len(name)
		for _, value := range values {
			headerBytes += len(value)
		}
		if headerBytes > MaxRequiredReplayHeaders {
			return fmt.Errorf("%w: response headers exceed %d bytes", ErrRequiredReplayInvalid, MaxRequiredReplayHeaders)
		}
	}
	return nil
}

func cloneRequiredReplayResponse(response *RequiredReplayResponse) *RequiredReplayResponse {
	if response == nil {
		return nil
	}
	return &RequiredReplayResponse{
		Status: response.Status, Headers: response.Headers.Clone(), Body: append([]byte(nil), response.Body...),
	}
}
