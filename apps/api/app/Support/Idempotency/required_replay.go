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
	RequiredReplayPolicy             = "required.24h@1"
	MaxRequiredReplayBody            = 8 << 20
	MaxRequiredReplayHeaders         = 64 << 10
	MaxRequiredReplayEvidence        = 8 << 20
	MaxRequiredReplayRecord          = 24 << 20
	MaxRequiredReplayPayload         = 22 << 20
	MaxRequiredReplayEncryptedRecord = 32 << 20
	MaxRequiredReplayCanonicalPath   = 8 << 10
	requiredReplaySchemaV1           = "sforum.required-route-replay@1"
	requiredReplaySchemaV2           = "sforum.required-route-replay@2"
	requiredReplaySchemaV3           = "sforum.required-route-replay@3"
	requiredReplayPayloadSchemaV1    = "sforum.required-route-replay-payload@1"
	requiredReplayPayloadSchemaV2    = "sforum.required-route-replay-payload@2"
	requiredReplayPending            = "pending"
	requiredReplayCompleted          = "completed"
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

type RequiredReplayBinding struct {
	Fingerprint            string
	PlanDigest             string
	CompatibleFingerprints []string
}

type RequiredReplayResponse struct {
	Status                int                             `json:"status"`
	Headers               http.Header                     `json:"headers,omitempty"`
	Body                  []byte                          `json:"body,omitempty"`
	CanonicalPath         string                          `json:"canonicalPath,omitempty"`
	ResponseContractKnown bool                            `json:"responseContractKnown,omitempty"`
	ResponseContract      *RequiredReplayResponseContract `json:"responseContract,omitempty"`
	Authorization         *RequiredReplayAuthorization    `json:"-"`
}

type RequiredReplayResponseContract struct {
	StepIndex       int    `json:"stepIndex"`
	InvocationStage string `json:"invocationStage"`
	RouteID         string `json:"routeId"`
	ContractVersion string `json:"contractVersion"`
	ResponseSchema  string `json:"responseSchema"`
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
	PayloadCiphertext       string                  `json:"payloadCiphertext,omitempty"`
}

type requiredReplayPayload struct {
	Schema        string                       `json:"schema"`
	PlanDigest    string                       `json:"planDigest"`
	Response      *RequiredReplayResponse      `json:"response"`
	Authorization *RequiredReplayAuthorization `json:"authorization,omitempty"`
}

// BeginRequiredReplay 保留 V2 调用兼容；新的 Route 调用必须改用带 plan digest 的 Bound 入口。
func (s *Store) BeginRequiredReplay(
	ctx context.Context,
	scope RequiredReplayScope,
	key string,
	fingerprint string,
	compatibleFingerprints ...string,
) (RequiredReplayLease, *RequiredReplayResponse, error) {
	return s.beginRequiredReplay(ctx, scope, key, RequiredReplayBinding{
		Fingerprint: fingerprint, CompatibleFingerprints: compatibleFingerprints,
	}, requiredReplaySchemaV2)
}

func (s *Store) BeginRequiredReplayBound(
	ctx context.Context,
	scope RequiredReplayScope,
	key string,
	binding RequiredReplayBinding,
) (RequiredReplayLease, *RequiredReplayResponse, error) {
	return s.beginRequiredReplay(ctx, scope, key, binding, requiredReplaySchemaV3)
}

func (s *Store) beginRequiredReplay(
	ctx context.Context,
	scope RequiredReplayScope,
	key string,
	binding RequiredReplayBinding,
	writeSchema string,
) (RequiredReplayLease, *RequiredReplayResponse, error) {
	if s == nil || s.backend == nil {
		return RequiredReplayLease{}, nil, ErrRequiredReplayUnavailable
	}
	if ctx == nil || !validRequiredReplayScope(scope) ||
		!validRequiredReplayKey(key) || !validRequiredReplayFingerprint(binding.Fingerprint) ||
		writeSchema == requiredReplaySchemaV3 && !validRequiredReplayFingerprint(binding.PlanDigest) ||
		writeSchema != requiredReplaySchemaV3 && binding.PlanDigest != "" ||
		!validRequiredReplayCompatibleFingerprints(binding.CompatibleFingerprints) {
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
			legacyCompatible := requiredReplayLegacyFingerprintCompatible(record, writeSchema) &&
				requiredReplayFingerprintCompatible(record.Fingerprint, binding.CompatibleFingerprints)
			if record.Fingerprint != binding.Fingerprint && !legacyCompatible {
				return RequiredReplayLease{}, nil, ErrRequiredReplayFingerprintConflict
			}
			if record.Schema == requiredReplaySchemaV3 && writeSchema != requiredReplaySchemaV3 {
				return RequiredReplayLease{}, nil, ErrRequiredReplayUnavailable
			}
			if record.Schema == requiredReplaySchemaV3 && record.PlanDigest != binding.PlanDigest {
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
		if writeSchema == requiredReplaySchemaV3 && !s.replayCipher.Enabled() {
			// 先允许读取历史 V1/V2 response-only 记录；没有现存记录时，
			// 缺少密钥必须在 SetNX 和任何插件执行前失败，绝不新增明文回放。
			return RequiredReplayLease{}, nil, errors.Join(
				ErrRequiredReplayUnavailable, ErrRequiredReplayCipherInvalid,
			)
		}
		pending, marshalErr := s.newRequiredReplayPending(
			binding.Fingerprint, binding.PlanDigest, writeSchema,
		)
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
	completed, err := s.newRequiredReplayCompleted(lease.storageKey, pending, response)
	if err != nil {
		return err
	}
	replacement, err := json.Marshal(completed)
	if err != nil {
		return errors.Join(ErrRequiredReplayUnavailable, err)
	}
	recordLimit := MaxRequiredReplayRecord
	if completed.Schema == requiredReplaySchemaV3 {
		recordLimit = MaxRequiredReplayEncryptedRecord
	}
	if len(replacement) > recordLimit {
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

func (s *Store) newRequiredReplayPending(fingerprint, planDigest, schema string) ([]byte, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	return json.Marshal(requiredReplayRecord{
		// Older binaries reject V3 pending rows, so they cannot complete a bound
		// lease as a plaintext V1/V2 response during a rolling deployment.
		Schema: schema, State: requiredReplayPending,
		Fingerprint: fingerprint, PlanDigest: planDigest, LeaseToken: hex.EncodeToString(token),
	})
}

func (s *Store) newRequiredReplayCompleted(
	storageKey string,
	pending requiredReplayRecord,
	response RequiredReplayResponse,
) (requiredReplayRecord, error) {
	storedResponse := cloneRequiredReplayResponse(&response)
	storedResponse.Authorization = nil
	completed := requiredReplayRecord{
		Schema: pending.Schema, State: requiredReplayCompleted, Fingerprint: pending.Fingerprint,
	}
	switch pending.Schema {
	case requiredReplaySchemaV2:
		if response.ResponseContractKnown || response.ResponseContract != nil {
			return requiredReplayRecord{}, ErrRequiredReplayInvalid
		}
		completed.Response = storedResponse
		if response.Authorization == nil {
			return completed, nil
		}
		plaintext, err := json.Marshal(response.Authorization)
		if err != nil || len(plaintext) > MaxRequiredReplayEvidence {
			return requiredReplayRecord{}, ErrRequiredReplayInvalid
		}
		completed.PlanDigest = response.Authorization.PlanDigest
		completed.AuthorizationCiphertext, err = s.replayCipher.Encrypt(
			storageKey, pending.Fingerprint, completed.PlanDigest, plaintext,
		)
		if err != nil {
			return requiredReplayRecord{}, errors.Join(ErrRequiredReplayUnavailable, err)
		}
		return completed, nil
	case requiredReplaySchemaV3:
		if response.Authorization != nil && response.Authorization.PlanDigest != pending.PlanDigest {
			return requiredReplayRecord{}, ErrRequiredReplayInvalid
		}
		completed.PlanDigest = pending.PlanDigest
		payload, err := json.Marshal(requiredReplayPayload{
			Schema: requiredReplayPayloadSchemaV2, PlanDigest: pending.PlanDigest, Response: storedResponse,
			Authorization: cloneRequiredReplayAuthorization(response.Authorization),
		})
		if err != nil || len(payload) == 0 || len(payload) > MaxRequiredReplayPayload {
			return requiredReplayRecord{}, ErrRequiredReplayInvalid
		}
		completed.PayloadCiphertext, err = s.replayCipher.EncryptReplay(
			storageKey, pending.Fingerprint, pending.PlanDigest, payload,
		)
		if err != nil {
			return requiredReplayRecord{}, errors.Join(ErrRequiredReplayUnavailable, err)
		}
		return completed, nil
	default:
		return requiredReplayRecord{}, ErrRequiredReplayInvalid
	}
}

func decodeRequiredReplayRecord(raw []byte) (requiredReplayRecord, error) {
	var record requiredReplayRecord
	if len(raw) == 0 || len(raw) > MaxRequiredReplayEncryptedRecord || json.Unmarshal(raw, &record) != nil ||
		(record.Schema != requiredReplaySchemaV1 && record.Schema != requiredReplaySchemaV2 &&
			record.Schema != requiredReplaySchemaV3) ||
		!validRequiredReplayFingerprint(record.Fingerprint) {
		return requiredReplayRecord{}, ErrRequiredReplayUnavailable
	}
	if record.Schema != requiredReplaySchemaV3 && len(raw) > MaxRequiredReplayRecord {
		return requiredReplayRecord{}, ErrRequiredReplayUnavailable
	}
	switch record.State {
	case requiredReplayPending:
		if len(record.LeaseToken) != 32 || record.Response != nil ||
			record.AuthorizationCiphertext != "" || record.PayloadCiphertext != "" ||
			(record.Schema == requiredReplaySchemaV3 && !validRequiredReplayFingerprint(record.PlanDigest)) ||
			(record.Schema != requiredReplaySchemaV3 && record.PlanDigest != "") {
			return requiredReplayRecord{}, ErrRequiredReplayUnavailable
		}
	case requiredReplayCompleted:
		switch record.Schema {
		case requiredReplaySchemaV1, requiredReplaySchemaV2:
			if record.LeaseToken != "" || record.PayloadCiphertext != "" ||
				validateRequiredReplayResponsePointer(record.Response) != nil ||
				record.Schema == requiredReplaySchemaV1 &&
					(record.PlanDigest != "" || record.AuthorizationCiphertext != "") ||
				(record.PlanDigest == "") != (record.AuthorizationCiphertext == "") ||
				record.PlanDigest != "" && !validRequiredReplayFingerprint(record.PlanDigest) {
				return requiredReplayRecord{}, ErrRequiredReplayUnavailable
			}
		case requiredReplaySchemaV3:
			if record.LeaseToken != "" || record.Response != nil || !validRequiredReplayFingerprint(record.PlanDigest) ||
				record.AuthorizationCiphertext != "" || record.PayloadCiphertext == "" {
				return requiredReplayRecord{}, ErrRequiredReplayUnavailable
			}
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
	if record.Schema == requiredReplaySchemaV3 {
		return s.requiredReplayResponseV3(storageKey, record)
	}
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

func (s *Store) requiredReplayResponseV3(
	storageKey string,
	record requiredReplayRecord,
) (*RequiredReplayResponse, error) {
	plaintext, err := s.replayCipher.DecryptReplay(
		storageKey, record.Fingerprint, record.PlanDigest, record.PayloadCiphertext,
	)
	if err != nil {
		return nil, errors.Join(ErrRequiredReplayUnavailable, err)
	}
	var payload requiredReplayPayload
	if json.Unmarshal(plaintext, &payload) != nil || payload.PlanDigest != record.PlanDigest ||
		validateRequiredReplayResponsePointer(payload.Response) != nil ||
		validateRequiredReplayAuthorization(payload.Authorization) != nil ||
		payload.Authorization != nil && payload.Authorization.PlanDigest != record.PlanDigest {
		return nil, ErrRequiredReplayUnavailable
	}
	switch payload.Schema {
	case requiredReplayPayloadSchemaV1:
		if payload.Response.ResponseContractKnown || payload.Response.ResponseContract != nil {
			return nil, ErrRequiredReplayUnavailable
		}
	case requiredReplayPayloadSchemaV2:
	default:
		return nil, ErrRequiredReplayUnavailable
	}
	response := cloneRequiredReplayResponse(payload.Response)
	response.Authorization = cloneRequiredReplayAuthorization(payload.Authorization)
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

func requiredReplayLegacyFingerprintCompatible(record requiredReplayRecord, writeSchema string) bool {
	if record.Schema == requiredReplaySchemaV1 {
		return true
	}
	// V2 rolling fingerprints are a one-way migration bridge for Bound/V3
	// callers. The deprecated V2 writer may still read an exact V2 record, but
	// must not expand its identity through caller-supplied aliases.
	return writeSchema == requiredReplaySchemaV3 && record.Schema == requiredReplaySchemaV2 && record.PlanDigest == "" &&
		record.AuthorizationCiphertext == ""
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
	if err := validateRequiredReplayResponseContract(response.ResponseContractKnown, response.ResponseContract); err != nil {
		return err
	}
	return validateRequiredReplayAuthorization(response.Authorization)
}

func validateRequiredReplayResponseContract(
	known bool,
	contract *RequiredReplayResponseContract,
) error {
	if !known {
		if contract != nil {
			return ErrRequiredReplayInvalid
		}
		return nil
	}
	if contract == nil {
		return nil
	}
	if contract.StepIndex < 0 || contract.StepIndex > 65535 ||
		(contract.InvocationStage != "handler" && contract.InvocationStage != "response") {
		return ErrRequiredReplayInvalid
	}
	for _, value := range []string{contract.RouteID, contract.ContractVersion, contract.ResponseSchema} {
		if value == "" || value != strings.TrimSpace(value) || len(value) > 512 {
			return ErrRequiredReplayInvalid
		}
	}
	return nil
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
		CanonicalPath: response.CanonicalPath, ResponseContractKnown: response.ResponseContractKnown,
		ResponseContract: cloneRequiredReplayResponseContract(response.ResponseContract),
		Authorization:    cloneRequiredReplayAuthorization(response.Authorization),
	}
}

func cloneRequiredReplayResponseContract(
	value *RequiredReplayResponseContract,
) *RequiredReplayResponseContract {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
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
