package contentregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultMaxInputBytes      = 1 << 20
	defaultMaxOutputBytes     = 2 << 20
	defaultMaxJSONDepth       = 32
	defaultMaxJSONNodes       = 10_000
	defaultMaxSegments        = 512
	defaultMaxBindings        = 256
	defaultMaxCacheTags       = 64
	defaultMaxConcurrentCalls = 64
	defaultCallTimeout        = 2 * time.Second

	hardMaxInputBytes      = 8 << 20
	hardMaxOutputBytes     = 8 << 20
	hardMaxJSONDepth       = 64
	hardMaxJSONNodes       = 100_000
	hardMaxSegments        = 4096
	hardMaxBindings        = 4096
	hardMaxCacheTags       = 256
	hardMaxConcurrentCalls = 1024
	hardMaxCallTimeout     = 10 * time.Second

	maxExecutionStringLength = 512
	maxCacheTagLength        = 128
	maxPriority              = 10_000
)

var (
	storageVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	cacheTagPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{0,127}$`)
)

func normalizeExecutionLimits(input ExecutionLimits) (ExecutionLimits, error) {
	defaults := ExecutionLimits{
		MaxInputBytes: defaultMaxInputBytes, MaxOutputBytes: defaultMaxOutputBytes,
		MaxJSONDepth: defaultMaxJSONDepth, MaxJSONNodes: defaultMaxJSONNodes,
		MaxSegments: defaultMaxSegments, MaxBindings: defaultMaxBindings,
		MaxCacheTags: defaultMaxCacheTags, MaxConcurrentCalls: defaultMaxConcurrentCalls,
		CallTimeout: defaultCallTimeout,
	}
	values := []*int{
		&input.MaxInputBytes, &input.MaxOutputBytes, &input.MaxJSONDepth, &input.MaxJSONNodes,
		&input.MaxSegments, &input.MaxBindings, &input.MaxCacheTags, &input.MaxConcurrentCalls,
	}
	defaultValues := []int{
		defaults.MaxInputBytes, defaults.MaxOutputBytes, defaults.MaxJSONDepth, defaults.MaxJSONNodes,
		defaults.MaxSegments, defaults.MaxBindings, defaults.MaxCacheTags, defaults.MaxConcurrentCalls,
	}
	hardValues := []int{
		hardMaxInputBytes, hardMaxOutputBytes, hardMaxJSONDepth, hardMaxJSONNodes,
		hardMaxSegments, hardMaxBindings, hardMaxCacheTags, hardMaxConcurrentCalls,
	}
	for index, value := range values {
		if *value == 0 {
			*value = defaultValues[index]
		}
		if *value < 0 || *value > hardValues[index] {
			return ExecutionLimits{}, ErrExecutionInvalid
		}
	}
	if input.CallTimeout == 0 {
		input.CallTimeout = defaults.CallTimeout
	}
	if input.CallTimeout < time.Millisecond || input.CallTimeout > hardMaxCallTimeout {
		return ExecutionLimits{}, ErrExecutionInvalid
	}
	return input, nil
}

func normalizeExecutionBindings(input []ExecutionBinding, limits ExecutionLimits) ([]ExecutionBinding, string, error) {
	if len(input) == 0 || len(input) > limits.MaxBindings {
		return nil, "", ErrCompositionInvalid
	}
	result := make([]ExecutionBinding, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	totalCacheTags := 0
	for _, raw := range input {
		binding, err := normalizeExecutionBinding(raw, limits)
		if err != nil {
			return nil, "", err
		}
		key := binding.TargetID + "\x00" + binding.DeclarationID + "\x00" + binding.Action
		if _, duplicate := seen[key]; duplicate {
			return nil, "", fmt.Errorf("%w: duplicate binding %s", ErrCompositionInvalid, binding.DeclarationID)
		}
		seen[key] = struct{}{}
		totalCacheTags += len(binding.CacheTags)
		if totalCacheTags > limits.MaxCacheTags {
			return nil, "", ErrExecutionLimit
		}
		result = append(result, binding)
	}
	sort.Slice(result, func(i, j int) bool { return executionBindingBefore(result[i], result[j]) })
	return result, executionBindingDigest(result), nil
}

func normalizeExecutionBinding(input ExecutionBinding, limits ExecutionLimits) (ExecutionBinding, error) {
	input.TargetID = strings.ToLower(strings.TrimSpace(input.TargetID))
	input.TargetContractVersion = strings.TrimSpace(input.TargetContractVersion)
	input.DeclarationID = strings.ToLower(strings.TrimSpace(input.DeclarationID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.Fallback = strings.ToLower(strings.TrimSpace(input.Fallback))
	artifact, err := normalizeArtifact(input.Artifact)
	if err != nil || !idPattern.MatchString(input.TargetID) ||
		!contractPattern.MatchString(input.TargetContractVersion) || !idPattern.MatchString(input.DeclarationID) ||
		!contractPattern.MatchString(input.ContractVersion) || input.Priority < -maxPriority || input.Priority > maxPriority ||
		!validExecutionAction(input.Action) {
		return ExecutionBinding{}, ErrCompositionInvalid
	}
	input.Artifact = artifact
	if input.Fallback == "" {
		input.Fallback = defaultFallback(input.Action)
	}
	if !validExecutionFallback(input.Action, input.Fallback) {
		return ExecutionBinding{}, ErrCompositionInvalid
	}
	input.CacheTags, err = normalizeCacheTags(input.CacheTags, limits.MaxCacheTags)
	if err != nil {
		return ExecutionBinding{}, err
	}
	switch input.Action {
	case ActionHide:
		if input.Providers.Editor != nil || input.Providers.Validator != nil || input.Providers.Serializer != nil ||
			input.Providers.Renderer != nil || input.Providers.Filter != nil {
			return ExecutionBinding{}, ErrCompositionInvalid
		}
	case ActionFilter:
		if input.Providers.Filter == nil || input.Providers.Renderer != nil {
			return ExecutionBinding{}, ErrCompositionInvalid
		}
	default:
		if input.Providers.Renderer == nil || input.Providers.Filter != nil {
			return ExecutionBinding{}, ErrCompositionInvalid
		}
	}
	return input, nil
}

func normalizeExecutionRequest(input ExecutionRequest, limits ExecutionLimits) (ExecutionRequest, error) {
	// Snapshot caller-owned source before any asynchronous permission/runtime
	// callback can create an aliasing window.
	input.Document.Value = append([]byte(nil), input.Document.Value...)
	input.TargetID = strings.ToLower(strings.TrimSpace(input.TargetID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.Locale = strings.TrimSpace(input.Locale)
	input.Scope = strings.TrimSpace(input.Scope)
	input.Permission.ActorFingerprint = strings.TrimSpace(input.Permission.ActorFingerprint)
	input.Permission.PolicyFingerprint = strings.TrimSpace(input.Permission.PolicyFingerprint)
	if !idPattern.MatchString(input.TargetID) || !contractPattern.MatchString(input.ContractVersion) ||
		input.Permission.Recheck == nil || input.Permission.PolicyFingerprint == "" ||
		!validExecutionString(input.ResourceID) || !validExecutionString(input.Locale) || !validExecutionString(input.Scope) ||
		!validExecutionString(input.Permission.ActorFingerprint) || !validExecutionString(input.Permission.PolicyFingerprint) {
		return ExecutionRequest{}, ErrExecutionInvalid
	}
	var err error
	input.CacheTags, err = normalizeCacheTags(input.CacheTags, limits.MaxCacheTags)
	if err != nil {
		return ExecutionRequest{}, err
	}
	return input, nil
}

func normalizeEditorDocument(input EditorDocument, target Contribution, limits ExecutionLimits) (EditorDocument, any, error) {
	// Provider-returned slices may be retained by their owner. All subsequent
	// validation and release work operates on a private snapshot.
	input.Value = append([]byte(nil), input.Value...)
	if input.SchemaVersion == "" {
		input.SchemaVersion = EditorDocumentSchemaVersion
	}
	if input.StorageVersion == "" {
		input.StorageVersion = "1"
	}
	input.ContentID = strings.ToLower(strings.TrimSpace(input.ContentID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Schema = strings.TrimSpace(input.Schema)
	input.StorageVersion = strings.TrimSpace(input.StorageVersion)
	if input.SchemaVersion != EditorDocumentSchemaVersion || input.ContentID != target.ID ||
		input.ContractVersion != target.ContractVersion || input.Schema != target.Schema ||
		input.StorageVersion != "1" || !storageVersionPattern.MatchString(input.StorageVersion) ||
		len(input.Value) == 0 || len(input.Value) > limits.MaxInputBytes {
		return EditorDocument{}, nil, ErrExecutionInvalid
	}
	value, err := decodeBoundedJSON(input.Value, limits)
	if err != nil {
		return EditorDocument{}, nil, err
	}
	canonical, err := json.Marshal(value)
	if err != nil || len(canonical) > limits.MaxInputBytes {
		return EditorDocument{}, nil, ErrExecutionLimit
	}
	input.Value = canonical
	return input, value, nil
}

func decodeBoundedJSON(raw []byte, limits ExecutionLimits) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, ErrExecutionInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, ErrExecutionInvalid
	}
	nodes := 0
	budget := limits.MaxInputBytes
	if !boundedJSONValue(value, 1, &nodes, &budget, limits) {
		return nil, ErrExecutionLimit
	}
	return value, nil
}

func boundedJSONValue(value any, depth int, nodes *int, budget *int, limits ExecutionLimits) bool {
	(*nodes)++
	if depth > limits.MaxJSONDepth || *nodes > limits.MaxJSONNodes {
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		if !consumeExecutionJSONBudget(budget, 2) {
			return false
		}
		index := 0
		for key, child := range typed {
			if index > 0 && !consumeExecutionJSONBudget(budget, 1) {
				return false
			}
			if !validExecutionString(key) || !consumeExecutionJSONString(budget, key) ||
				!consumeExecutionJSONBudget(budget, 1) ||
				!boundedJSONValue(child, depth+1, nodes, budget, limits) {
				return false
			}
			index++
		}
	case []any:
		if !consumeExecutionJSONBudget(budget, 2) {
			return false
		}
		for index, child := range typed {
			if index > 0 && !consumeExecutionJSONBudget(budget, 1) {
				return false
			}
			if !boundedJSONValue(child, depth+1, nodes, budget, limits) {
				return false
			}
		}
	case string:
		return len(typed) <= limits.MaxInputBytes && consumeExecutionJSONString(budget, typed)
	case json.Number:
		return len(typed.String()) <= 128 && consumeExecutionJSONBudget(budget, len(typed.String()))
	case nil:
		return consumeExecutionJSONBudget(budget, 4)
	case bool:
		return consumeExecutionJSONBudget(budget, 5)
	default:
		return false
	}
	return true
}

func consumeExecutionJSONBudget(remaining *int, size int) bool {
	if remaining == nil || size < 0 || size > *remaining {
		return false
	}
	*remaining -= size
	return true
}

// consumeExecutionJSONString mirrors encoding/json's escaping size without
// allocating the encoded string. It is deliberately conservative for invalid
// UTF-8, U+2028, and U+2029.
func consumeExecutionJSONString(remaining *int, value string) bool {
	if !consumeExecutionJSONBudget(remaining, 2) {
		return false
	}
	for len(value) > 0 {
		current := value[0]
		switch {
		case current < 0x20 || current == '<' || current == '>' || current == '&':
			if !consumeExecutionJSONBudget(remaining, 6) {
				return false
			}
			value = value[1:]
		case current == '\\' || current == '"':
			if !consumeExecutionJSONBudget(remaining, 2) {
				return false
			}
			value = value[1:]
		case current < utf8.RuneSelf:
			if !consumeExecutionJSONBudget(remaining, 1) {
				return false
			}
			value = value[1:]
		default:
			runeValue, size := utf8.DecodeRuneInString(value)
			encodedSize := size
			if runeValue == utf8.RuneError && size == 1 || runeValue == '\u2028' || runeValue == '\u2029' {
				encodedSize = 6
			}
			if !consumeExecutionJSONBudget(remaining, encodedSize) {
				return false
			}
			value = value[size:]
		}
	}
	return true
}

func normalizeCacheTags(input []string, limit int) ([]string, error) {
	if len(input) > limit {
		return nil, ErrExecutionLimit
	}
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))
	for _, raw := range input {
		value := strings.ToLower(strings.TrimSpace(raw))
		if len(value) > maxCacheTagLength || !cacheTagPattern.MatchString(value) {
			return nil, ErrExecutionInvalid
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func validExecutionString(value string) bool {
	if len(value) > maxExecutionStringLength || strings.ContainsRune(value, '\x00') {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func validExecutionAction(action string) bool {
	switch action {
	case ActionAdd, ActionBefore, ActionAfter, ActionWrap, ActionReplace, ActionHide, ActionFilter:
		return true
	default:
		return false
	}
}

// IsExactAdmissionRequest validates the complete Host/runtime identity before
// HostAPI consults Manager. It accepts only already-normalized Registry facts.
func IsExactAdmissionRequest(input AdmissionRequest) bool {
	if !IsExactArtifact(input.TargetArtifact) || !IsExactArtifact(input.Artifact) ||
		!validContributionIdentity(input.TargetArtifact, input.TargetID, input.TargetContractVersion) ||
		!validContributionIdentity(input.Artifact, input.ContentID, input.ContractVersion) ||
		input.TargetID != strings.ToLower(strings.TrimSpace(input.TargetID)) ||
		input.ContentID != strings.ToLower(strings.TrimSpace(input.ContentID)) ||
		input.TargetContractVersion != strings.TrimSpace(input.TargetContractVersion) ||
		input.ContractVersion != strings.TrimSpace(input.ContractVersion) ||
		input.TargetSchema != strings.TrimSpace(input.TargetSchema) || !validSchemaRef(input.TargetSchema) ||
		input.Action != strings.ToLower(strings.TrimSpace(input.Action)) || !validExecutionAction(input.Action) ||
		input.Operation != strings.ToLower(strings.TrimSpace(input.Operation)) ||
		!validExecutionOperation(input.Operation) || input.Operation == OperationSource {
		return false
	}
	if input.HandlerReference != strings.TrimSpace(input.HandlerReference) ||
		input.RendererReference != strings.TrimSpace(input.RendererReference) ||
		input.HandlerReference != "" && !validHandler(input.HandlerReference) ||
		input.RendererReference != "" && !validOpaqueRef(input.RendererReference) ||
		input.HandlerReference == "" && input.RendererReference == "" ||
		input.HandlerReference != "" && !input.Artifact.Core && input.Artifact.RuntimeInstanceID == "" {
		return false
	}
	switch input.Operation {
	case OperationEditor, OperationValidator, OperationSerializer, OperationRenderer, OperationFilter:
		return input.HandlerReference != ""
	default:
		return true
	}
}

// IsExactPermissionClaim applies the same target/provider identity fence to
// permission callbacks. Actor authority still comes only from Host context.
func IsExactPermissionClaim(input PermissionClaim) bool {
	return IsExactArtifact(input.TargetArtifact) && IsExactArtifact(input.Artifact) &&
		validContributionIdentity(input.TargetArtifact, input.TargetID, input.TargetContractVersion) &&
		validContributionIdentity(input.Artifact, input.ContentID, input.ContractVersion) &&
		input.TargetID == strings.ToLower(strings.TrimSpace(input.TargetID)) &&
		input.ContentID == strings.ToLower(strings.TrimSpace(input.ContentID)) &&
		input.TargetContractVersion == strings.TrimSpace(input.TargetContractVersion) &&
		input.ContractVersion == strings.TrimSpace(input.ContractVersion) &&
		input.TargetSchema == strings.TrimSpace(input.TargetSchema) && validSchemaRef(input.TargetSchema) &&
		input.Schema == strings.TrimSpace(input.Schema) && validSchemaRef(input.Schema) &&
		input.Action == strings.ToLower(strings.TrimSpace(input.Action)) && validExecutionAction(input.Action) &&
		input.Operation == strings.ToLower(strings.TrimSpace(input.Operation)) && validExecutionOperation(input.Operation) &&
		input.ResourceID == strings.TrimSpace(input.ResourceID) && validExecutionString(input.ResourceID) &&
		input.Locale == strings.TrimSpace(input.Locale) && validExecutionString(input.Locale) &&
		input.Scope == strings.TrimSpace(input.Scope) && validExecutionString(input.Scope)
}

func validExecutionFallback(action, fallback string) bool {
	switch fallback {
	case FallbackClosed:
		return true
	case FallbackOmit:
		return action == ActionBefore || action == ActionAfter || action == ActionWrap || action == ActionFilter
	case FallbackBase:
		return action == ActionReplace || action == ActionBefore || action == ActionAfter || action == ActionWrap || action == ActionFilter
	case FallbackPreserveSource:
		return action == ActionAdd || action == ActionReplace
	default:
		return false
	}
}

func defaultFallback(action string) string {
	switch action {
	case ActionAdd:
		return FallbackPreserveSource
	case ActionReplace:
		return FallbackBase
	case ActionBefore, ActionAfter, ActionWrap, ActionFilter:
		return FallbackOmit
	default:
		return FallbackClosed
	}
}

func executionBindingBefore(left, right ExecutionBinding) bool {
	if left.TargetID != right.TargetID {
		return left.TargetID < right.TargetID
	}
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.DeclarationID != right.DeclarationID {
		return left.DeclarationID < right.DeclarationID
	}
	if left.Action != right.Action {
		return left.Action < right.Action
	}
	return artifactBefore(left.Artifact, right.Artifact)
}

func executionBindingDigest(bindings []ExecutionBinding) string {
	digest := sha256.New()
	writeExecutionDigestString(digest, ExecutionSchemaVersion)
	writeExecutionDigestUint64(digest, uint64(len(bindings)))
	for _, binding := range bindings {
		writeExecutionDigestString(digest, binding.TargetID)
		writeExecutionDigestString(digest, binding.TargetContractVersion)
		writeExecutionDigestString(digest, binding.DeclarationID)
		writeExecutionDigestString(digest, binding.ContractVersion)
		writeExecutionDigestString(digest, binding.Action)
		writeExecutionDigestString(digest, binding.Fallback)
		writeExecutionDigestArtifact(digest, binding.Artifact)
		writeExecutionDigestUint64(digest, uint64(int64(binding.Priority)))
		writeExecutionDigestUint64(digest, uint64(len(binding.CacheTags)))
		for _, tag := range binding.CacheTags {
			writeExecutionDigestString(digest, tag)
		}
		shape := byte(0)
		if binding.Providers.Editor != nil {
			shape |= 1 << 0
		}
		if binding.Providers.Validator != nil {
			shape |= 1 << 1
		}
		if binding.Providers.Serializer != nil {
			shape |= 1 << 2
		}
		if binding.Providers.Renderer != nil {
			shape |= 1 << 3
		}
		if binding.Providers.Filter != nil {
			shape |= 1 << 4
		}
		_, _ = digest.Write([]byte{shape})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func writeExecutionDigestString(target hash.Hash, value string) {
	writeExecutionDigestUint64(target, uint64(len(value)))
	_, _ = target.Write([]byte(value))
}

func writeExecutionDigestUint64(target hash.Hash, value uint64) {
	_ = binary.Write(target, binary.BigEndian, value)
}

func writeExecutionDigestArtifact(target hash.Hash, artifact Artifact) {
	writeExecutionDigestString(target, artifact.ExtensionID)
	writeExecutionDigestString(target, artifact.ExtensionVersion)
	writeExecutionDigestString(target, artifact.PackageDigest)
	writeExecutionDigestString(target, artifact.RuntimeInstanceID)
	writeExecutionDigestUint64(target, uint64(artifact.VersionID))
	if artifact.Core {
		_, _ = target.Write([]byte{1})
	} else {
		_, _ = target.Write([]byte{0})
	}
}
