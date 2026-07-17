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
	"net/url"
	"strings"

	"golang.org/x/net/http/httpguts"
)

const (
	RequiredReplayPolicy           = "required.24h@1"
	MaxRequiredReplayBody          = 8 << 20
	MaxRequiredReplayHeaders       = 64 << 10
	MaxRequiredReplayEvidence      = 8 << 20
	MaxRequiredReplayRecord        = 24 << 20
	MaxRequiredReplayCanonicalPath = 8 << 10
	requiredReplaySchemaV1         = "sforum.required-route-replay@1"
	requiredReplaySchemaV2         = "sforum.required-route-replay@2"
	requiredReplayPending          = "pending"
	requiredReplayCompleted        = "completed"
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
	Status        int                          `json:"status"`
	Headers       http.Header                  `json:"headers,omitempty"`
	Body          []byte                       `json:"body,omitempty"`
	CanonicalPath string                       `json:"canonicalPath,omitempty"`
	Authorization *RequiredReplayAuthorization `json:"-"`
}

type RequiredReplayAuthorization struct {
	Schema           string                          `json:"schema"`
	PlanDigest       string                          `json:"planDigest"`
	BaseDigest       string                          `json:"baseDigest"`
	RequestMutations []RequiredReplayRequestMutation `json:"requestMutations"`
}

type RequiredReplayRequestMutation struct {
	StepIndex    int                            `json:"stepIndex"`
	BeforeDigest string                         `json:"beforeDigest"`
	AfterDigest  string                         `json:"afterDigest"`
	Operations   []RequiredReplayPatchOperation `json:"operations"`
}

type RequiredReplayPatchOperation struct {
	Kind  string          `json:"kind"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

// RequiredReplayLease is opaque proof that this request owns the pending row.
type RequiredReplayLease struct {
	storageKey string
	pending    []byte
}

type requiredReplayRecord struct {
	Schema                  string                  `json:"schema"`
	State                   string                  `json:"state"`
	Fingerprint             string                  `json:"fingerprint"`
	LeaseToken              string                  `json:"leaseToken,omitempty"`
	Response                *RequiredReplayResponse `json:"response,omitempty"`
	PlanDigest              string                  `json:"planDigest,omitempty"`
	AuthorizationCiphertext string                  `json:"authorizationCiphertext,omitempty"`
}

func (s *Store) BeginRequiredReplay(
	ctx context.Context,
	scope RequiredReplayScope,
	key string,
	fingerprint string,
	compatibleFingerprints ...string,
) (RequiredReplayLease, *RequiredReplayResponse, error) {
	if s == nil || s.backend == nil {
		return RequiredReplayLease{}, nil, ErrRequiredReplayUnavailable
	}
	if ctx == nil || !validRequiredReplayScope(scope) ||
		!validRequiredReplayKey(key) || !validRequiredReplayFingerprint(fingerprint) ||
		!validRequiredReplayCompatibleFingerprints(compatibleFingerprints) {
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
			legacyCompatible := record.Schema == requiredReplaySchemaV1 &&
				requiredReplayFingerprintCompatible(record.Fingerprint, compatibleFingerprints)
			if record.Fingerprint != fingerprint && !legacyCompatible {
				return RequiredReplayLease{}, nil, ErrRequiredReplayFingerprintConflict
			}
			switch record.State {
			case requiredReplayPending:
				return RequiredReplayLease{}, nil, ErrRequiredReplayInProgress
			case requiredReplayCompleted:
				response, replayErr := s.requiredReplayResponse(storageKey, record)
				if replayErr != nil {
					return RequiredReplayLease{}, nil, replayErr
				}
				return RequiredReplayLease{}, response, nil
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
	storedResponse := cloneRequiredReplayResponse(&response)
	storedResponse.Authorization = nil
	completed := requiredReplayRecord{
		Schema: pending.Schema, State: requiredReplayCompleted,
		Fingerprint: pending.Fingerprint, Response: storedResponse,
	}
	if response.Authorization != nil {
		plaintext, marshalErr := json.Marshal(response.Authorization)
		if marshalErr != nil || len(plaintext) > MaxRequiredReplayEvidence {
			return ErrRequiredReplayInvalid
		}
		completed.PlanDigest = response.Authorization.PlanDigest
		completed.AuthorizationCiphertext, err = s.replayCipher.Encrypt(
			lease.storageKey, pending.Fingerprint, completed.PlanDigest, plaintext,
		)
		if err != nil {
			return errors.Join(ErrRequiredReplayUnavailable, err)
		}
	}
	replacement, err := json.Marshal(completed)
	if err != nil {
		return errors.Join(ErrRequiredReplayUnavailable, err)
	}
	if len(replacement) > MaxRequiredReplayRecord {
		return ErrRequiredReplayInvalid
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
		// V2 pending makes old binaries fail closed during a rolling deployment;
		// the Redis key and public required.24h@1 policy remain unchanged.
		Schema: requiredReplaySchemaV2, State: requiredReplayPending,
		Fingerprint: fingerprint, LeaseToken: hex.EncodeToString(token),
	})
}

func decodeRequiredReplayRecord(raw []byte) (requiredReplayRecord, error) {
	var record requiredReplayRecord
	if len(raw) == 0 || len(raw) > MaxRequiredReplayRecord || json.Unmarshal(raw, &record) != nil ||
		(record.Schema != requiredReplaySchemaV1 && record.Schema != requiredReplaySchemaV2) ||
		!validRequiredReplayFingerprint(record.Fingerprint) {
		return requiredReplayRecord{}, ErrRequiredReplayUnavailable
	}
	switch record.State {
	case requiredReplayPending:
		if len(record.LeaseToken) != 32 || record.Response != nil || record.PlanDigest != "" ||
			record.AuthorizationCiphertext != "" {
			return requiredReplayRecord{}, ErrRequiredReplayUnavailable
		}
	case requiredReplayCompleted:
		if record.LeaseToken != "" || validateRequiredReplayResponsePointer(record.Response) != nil ||
			record.Schema == requiredReplaySchemaV1 &&
				(record.PlanDigest != "" || record.AuthorizationCiphertext != "") ||
			(record.PlanDigest == "") != (record.AuthorizationCiphertext == "") ||
			record.PlanDigest != "" && !validRequiredReplayFingerprint(record.PlanDigest) {
			return requiredReplayRecord{}, ErrRequiredReplayUnavailable
		}
	default:
		return requiredReplayRecord{}, ErrRequiredReplayUnavailable
	}
	return record, nil
}

func (s *Store) requiredReplayResponse(
	storageKey string,
	record requiredReplayRecord,
) (*RequiredReplayResponse, error) {
	response := cloneRequiredReplayResponse(record.Response)
	if record.AuthorizationCiphertext == "" {
		return response, nil
	}
	plaintext, err := s.replayCipher.Decrypt(
		storageKey, record.Fingerprint, record.PlanDigest, record.AuthorizationCiphertext,
	)
	if err != nil {
		return nil, errors.Join(ErrRequiredReplayUnavailable, err)
	}
	var authorization RequiredReplayAuthorization
	if json.Unmarshal(plaintext, &authorization) != nil ||
		validateRequiredReplayAuthorization(&authorization) != nil || authorization.PlanDigest != record.PlanDigest {
		return nil, ErrRequiredReplayUnavailable
	}
	response.Authorization = cloneRequiredReplayAuthorization(&authorization)
	return response, nil
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

func validRequiredReplayCompatibleFingerprints(values []string) bool {
	if len(values) > 2 {
		return false
	}
	for _, value := range values {
		if !validRequiredReplayFingerprint(value) {
			return false
		}
	}
	return true
}

func requiredReplayFingerprintCompatible(value string, compatible []string) bool {
	for _, candidate := range compatible {
		if value == candidate {
			return true
		}
	}
	return false
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
	if len(response.CanonicalPath) > MaxRequiredReplayCanonicalPath {
		return ErrRequiredReplayInvalid
	}
	if response.CanonicalPath != "" {
		parsed, err := url.ParseRequestURI(response.CanonicalPath)
		if err != nil || !strings.HasPrefix(response.CanonicalPath, "/") || strings.HasPrefix(response.CanonicalPath, "//") ||
			parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return ErrRequiredReplayInvalid
		}
	}
	headerBytes := 0
	for name, values := range response.Headers {
		canonical := strings.ToLower(strings.TrimSpace(name))
		if canonical == "" || canonical == "set-cookie" || canonical == "cookie" ||
			canonical == "authorization" || canonical == "proxy-authorization" ||
			!httpguts.ValidHeaderFieldName(name) {
			return ErrRequiredReplayInvalid
		}
		headerBytes += len(name)
		for _, value := range values {
			if !httpguts.ValidHeaderFieldValue(value) {
				return ErrRequiredReplayInvalid
			}
			headerBytes += len(value)
		}
		if headerBytes > MaxRequiredReplayHeaders {
			return fmt.Errorf("%w: response headers exceed %d bytes", ErrRequiredReplayInvalid, MaxRequiredReplayHeaders)
		}
	}
	return validateRequiredReplayAuthorization(response.Authorization)
}

func validateRequiredReplayAuthorization(value *RequiredReplayAuthorization) error {
	if value == nil {
		return nil
	}
	if value.Schema != "sforum.route-replay-authorization@1" ||
		!validRequiredReplayFingerprint(value.PlanDigest) || !validRequiredReplayFingerprint(value.BaseDigest) ||
		len(value.RequestMutations) == 0 {
		return ErrRequiredReplayInvalid
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > MaxRequiredReplayEvidence {
		return ErrRequiredReplayInvalid
	}
	previousIndex := -1
	for _, mutation := range value.RequestMutations {
		if mutation.StepIndex <= previousIndex || !validRequiredReplayFingerprint(mutation.BeforeDigest) ||
			!validRequiredReplayFingerprint(mutation.AfterDigest) {
			return ErrRequiredReplayInvalid
		}
		previousIndex = mutation.StepIndex
		for _, operation := range mutation.Operations {
			switch operation.Kind {
			case "add", "replace":
				if len(operation.Value) == 0 || !json.Valid(operation.Value) {
					return ErrRequiredReplayInvalid
				}
			case "remove":
				if len(operation.Value) != 0 {
					return ErrRequiredReplayInvalid
				}
			default:
				return ErrRequiredReplayInvalid
			}
			if requiredReplayPatchTargetsCredential(operation.Path) {
				return ErrRequiredReplayInvalid
			}
		}
	}
	return nil
}

func requiredReplayPatchTargetsCredential(path string) bool {
	for _, header := range []string{
		"cookie", "authorization", "proxy-authorization", "x-api-key", "x-auth-token", "x-csrf-token", "idempotency-key",
	} {
		prefix := "/headers/" + header
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func cloneRequiredReplayResponse(response *RequiredReplayResponse) *RequiredReplayResponse {
	if response == nil {
		return nil
	}
	return &RequiredReplayResponse{
		Status: response.Status, Headers: response.Headers.Clone(), Body: append([]byte(nil), response.Body...),
		CanonicalPath: response.CanonicalPath,
		Authorization: cloneRequiredReplayAuthorization(response.Authorization),
	}
}

func cloneRequiredReplayAuthorization(value *RequiredReplayAuthorization) *RequiredReplayAuthorization {
	if value == nil {
		return nil
	}
	result := &RequiredReplayAuthorization{
		Schema: value.Schema, PlanDigest: value.PlanDigest, BaseDigest: value.BaseDigest,
		RequestMutations: make([]RequiredReplayRequestMutation, len(value.RequestMutations)),
	}
	for index, mutation := range value.RequestMutations {
		result.RequestMutations[index].StepIndex = mutation.StepIndex
		result.RequestMutations[index].BeforeDigest = mutation.BeforeDigest
		result.RequestMutations[index].AfterDigest = mutation.AfterDigest
		result.RequestMutations[index].Operations = make([]RequiredReplayPatchOperation, len(mutation.Operations))
		for operationIndex, operation := range mutation.Operations {
			result.RequestMutations[index].Operations[operationIndex] = RequiredReplayPatchOperation{
				Kind: operation.Kind, Path: operation.Path, Value: append([]byte(nil), operation.Value...),
			}
		}
	}
	return result
}
