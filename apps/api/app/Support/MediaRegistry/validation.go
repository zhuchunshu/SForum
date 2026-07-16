package mediaregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxPublications    = 512
	maxPoliciesTotal   = 2048
	maxProcessorsTotal = 4096
	maxVariantsTotal   = 4096
	maxPerPublication  = 512
	maxPatterns        = 128
	maxAliases         = 128
	maxStringBytes     = 512
	maxPermissionBytes = 192
	maxActorBytes      = 256
	maxReasonCodeBytes = 128
	maxMetadataEntries = 256
	maxURLBytes        = 4096
)

var (
	idPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)
	contractPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
	digestPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern   = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_-]{0,127}$`)
	extensionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9]{0,15}$`)
	handlePattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]{0,511}$`)
)

var hostMaximumBudget = Budget{
	MaxFileBytes: 2 << 30, MaxFiles: 1000, MaxDecompressedBytes: 8 << 30,
	MaxDecompressionRatio: 1000, MaxFilenameBytes: 255, MaxMIMECandidates: 16,
	MaxMetadataBytes: 1 << 20, MaxVariants: 128,
}

var recommendedBudget = Budget{
	MaxFileBytes: 64 << 20, MaxFiles: 20, MaxDecompressedBytes: 512 << 20,
	MaxDecompressionRatio: 100, MaxFilenameBytes: 180, MaxMIMECandidates: 4,
	MaxMetadataBytes: 64 << 10, MaxVariants: 16,
}

func MaximumBudget() Budget { return hostMaximumBudget }
func DefaultBudget() Budget { return recommendedBudget }

func NewCoreArtifact(extensionID, extensionVersion, packageDigest, impactDigest string) (Artifact, error) {
	artifact := Artifact{
		ExtensionID: extensionID, ExtensionVersion: extensionVersion, PackageDigest: packageDigest,
		ImpactDigest: impactDigest, RuntimeInstanceID: "core", Core: true,
	}
	normalized, err := normalizeArtifact(artifact)
	if err != nil {
		return Artifact{}, err
	}
	normalized.coreSeal = coreArtifactSeal(normalized)
	return normalized, nil
}

func normalizeArtifact(input Artifact) (Artifact, error) {
	input.ExtensionID = strings.ToLower(strings.TrimSpace(input.ExtensionID))
	input.ExtensionVersion = strings.TrimSpace(input.ExtensionVersion)
	input.PackageDigest = strings.ToLower(strings.TrimSpace(input.PackageDigest))
	input.ImpactDigest = strings.ToLower(strings.TrimSpace(input.ImpactDigest))
	input.RuntimeInstanceID = strings.TrimSpace(input.RuntimeInstanceID)
	isCoreNamespace := input.ExtensionID == "core" || strings.HasPrefix(input.ExtensionID, "core.")
	if !idPattern.MatchString(input.ExtensionID) || !versionPattern.MatchString(input.ExtensionVersion) ||
		!digestPattern.MatchString(input.PackageDigest) || !digestPattern.MatchString(input.ImpactDigest) ||
		len(input.RuntimeInstanceID) > maxStringBytes || !validPlainString(input.RuntimeInstanceID, maxStringBytes) {
		return Artifact{}, ErrInvalid
	}
	if input.Core {
		if !isCoreNamespace || input.VersionID != 0 || input.RuntimeInstanceID != "core" {
			return Artifact{}, ErrInvalid
		}
	} else if isCoreNamespace || input.coreSeal != [32]byte{} || input.VersionID <= 0 || input.RuntimeInstanceID == "" {
		return Artifact{}, ErrInvalid
	}
	return input, nil
}

func coreArtifactSeal(artifact Artifact) [32]byte {
	value := strings.Join([]string{SchemaVersion, artifact.ExtensionID, artifact.ExtensionVersion,
		artifact.PackageDigest, artifact.ImpactDigest, artifact.RuntimeInstanceID}, "\x00")
	return sha256.Sum256([]byte(value))
}

func validCoreArtifactSeal(artifact Artifact) bool {
	return artifact.Core && artifact.coreSeal != [32]byte{} && artifact.coreSeal == coreArtifactSeal(artifact)
}

func normalizePublication(input Publication) (Publication, error) {
	artifact, err := normalizeArtifact(input.Artifact)
	if err != nil || len(input.Policies) > maxPerPublication || len(input.Processors) > maxPerPublication ||
		len(input.Variants) > maxPerPublication {
		return Publication{}, ErrInvalid
	}
	if artifact.Core && !validCoreArtifactSeal(input.Artifact) {
		return Publication{}, ErrInvalid
	}
	result := Publication{Artifact: artifact}
	seen := make(map[string]struct{}, len(input.Policies)+len(input.Processors)+len(input.Variants))
	for _, raw := range input.Policies {
		value, err := normalizePolicy(raw)
		if err != nil || duplicateID(seen, value.ID) {
			return Publication{}, ErrInvalid
		}
		result.Policies = append(result.Policies, value)
	}
	for _, raw := range input.Processors {
		value, err := normalizeProcessor(raw)
		if err != nil || duplicateID(seen, value.ID) {
			return Publication{}, ErrInvalid
		}
		result.Processors = append(result.Processors, value)
	}
	for _, raw := range input.Variants {
		value, err := normalizeVariant(raw)
		if err != nil || duplicateID(seen, value.ID) {
			return Publication{}, ErrInvalid
		}
		result.Variants = append(result.Variants, value)
	}
	sort.Slice(result.Policies, func(i, j int) bool { return result.Policies[i].ID < result.Policies[j].ID })
	sort.Slice(result.Processors, func(i, j int) bool { return result.Processors[i].ID < result.Processors[j].ID })
	sort.Slice(result.Variants, func(i, j int) bool { return result.Variants[i].ID < result.Variants[j].ID })
	return result, nil
}

func normalizePolicy(input MIMEPolicyDeclaration) (MIMEPolicyDeclaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.ToLower(strings.TrimSpace(input.ContractVersion))
	input.Purpose = normalizePurpose(input.Purpose)
	input.RequiredPermission = strings.TrimSpace(input.RequiredPermission)
	if !idPattern.MatchString(input.ID) || !contractPattern.MatchString(input.ContractVersion) || input.Purpose == "" ||
		!validPermission(input.RequiredPermission) || input.Priority < -10000 || input.Priority > 10000 ||
		len(input.AllowedMIMEs) == 0 || len(input.AllowedMIMEs) > maxPatterns || len(input.DeniedMIMEs) > maxPatterns ||
		len(input.AllowedExtensions) > maxPatterns || len(input.MIMEAliases) > maxAliases || !validBudget(input.Budget) {
		return MIMEPolicyDeclaration{}, ErrInvalid
	}
	var err error
	input.AllowedMIMEs, err = normalizeMIMEPatterns(input.AllowedMIMEs, true)
	if err != nil {
		return MIMEPolicyDeclaration{}, err
	}
	input.DeniedMIMEs, err = normalizeMIMEPatterns(input.DeniedMIMEs, true)
	if err != nil {
		return MIMEPolicyDeclaration{}, err
	}
	input.AllowedExtensions, err = normalizeExtensions(input.AllowedExtensions)
	if err != nil {
		return MIMEPolicyDeclaration{}, err
	}
	aliases := make([]MIMEAlias, 0, len(input.MIMEAliases))
	aliasSeen := map[string]struct{}{}
	for _, alias := range input.MIMEAliases {
		alias.Declared, err = normalizeExactMIME(alias.Declared)
		if err != nil {
			return MIMEPolicyDeclaration{}, err
		}
		alias.Detected, err = normalizeExactMIME(alias.Detected)
		if err != nil || alias.Declared == alias.Detected {
			return MIMEPolicyDeclaration{}, ErrInvalid
		}
		key := alias.Declared + "\x00" + alias.Detected
		if _, exists := aliasSeen[key]; exists {
			continue
		}
		aliasSeen[key] = struct{}{}
		aliases = append(aliases, alias)
	}
	sort.Slice(aliases, func(i, j int) bool {
		if aliases[i].Declared != aliases[j].Declared {
			return aliases[i].Declared < aliases[j].Declared
		}
		return aliases[i].Detected < aliases[j].Detected
	})
	input.MIMEAliases = aliases
	return input, nil
}

func normalizeProcessor(input ProcessorDeclaration) (ProcessorDeclaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.ToLower(strings.TrimSpace(input.ContractVersion))
	input.Stage = strings.ToLower(strings.TrimSpace(input.Stage))
	input.Purpose = normalizePurpose(input.Purpose)
	input.Handler = strings.TrimSpace(input.Handler)
	input.Mode = strings.ToLower(strings.TrimSpace(input.Mode))
	input.Slot = strings.ToLower(strings.TrimSpace(input.Slot))
	input.Execution = strings.ToLower(strings.TrimSpace(input.Execution))
	input.FailureMode = strings.ToLower(strings.TrimSpace(input.FailureMode))
	input.RequiredPermission = strings.TrimSpace(input.RequiredPermission)
	if input.Mode == "" {
		input.Mode = ProcessorCompose
	}
	if input.Execution == "" {
		input.Execution = ExecutionSync
	}
	if input.FailureMode == "" {
		input.FailureMode = FailureFailClosed
	}
	if !idPattern.MatchString(input.ID) || !contractPattern.MatchString(input.ContractVersion) || !validStage(input.Stage) ||
		input.Purpose == "" || !validPlainString(input.Handler, maxStringBytes) || input.Handler == "" ||
		!validPermission(input.RequiredPermission) || input.Priority < -10000 || input.Priority > 10000 || len(input.MIMEs) == 0 ||
		len(input.MIMEs) > maxPatterns || !validRetryPolicy(input.Retry, input.Execution) {
		return ProcessorDeclaration{}, ErrInvalid
	}
	var err error
	input.MIMEs, err = normalizeMIMEPatterns(input.MIMEs, true)
	if err != nil {
		return ProcessorDeclaration{}, err
	}
	if input.Mode != ProcessorCompose && input.Mode != ProcessorExclusive ||
		input.Mode == ProcessorCompose && input.Slot != "" ||
		input.Mode == ProcessorExclusive && !idPattern.MatchString(input.Slot) {
		return ProcessorDeclaration{}, ErrInvalid
	}
	if input.Execution != ExecutionSync && input.Execution != ExecutionBackground ||
		input.Execution == ExecutionBackground && input.Stage == StageValidate {
		return ProcessorDeclaration{}, ErrInvalid
	}
	if input.FailureMode != FailureFailClosed && input.FailureMode != FailureSkip && input.FailureMode != FailureFallbackOriginal {
		return ProcessorDeclaration{}, ErrInvalid
	}
	if (input.Stage == StageValidate || input.Stage == StageScan) &&
		(input.Mode != ProcessorCompose || input.FailureMode != FailureFailClosed) {
		return ProcessorDeclaration{}, ErrInvalid
	}
	if input.Stage == StageCDN && input.Mode != ProcessorExclusive {
		return ProcessorDeclaration{}, ErrInvalid
	}
	return input, nil
}

func normalizeVariant(input VariantDeclaration) (VariantDeclaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.ToLower(strings.TrimSpace(input.ContractVersion))
	input.Purpose = normalizePurpose(input.Purpose)
	input.Name = strings.ToLower(strings.TrimSpace(input.Name))
	input.ProcessorID = strings.ToLower(strings.TrimSpace(input.ProcessorID))
	input.ProcessorContractVersion = strings.ToLower(strings.TrimSpace(input.ProcessorContractVersion))
	input.ProcessorOwnerExtensionID = strings.ToLower(strings.TrimSpace(input.ProcessorOwnerExtensionID))
	input.ProcessorPackageDigest = strings.ToLower(strings.TrimSpace(input.ProcessorPackageDigest))
	input.OutputMIME = strings.ToLower(strings.TrimSpace(input.OutputMIME))
	if !idPattern.MatchString(input.ID) || !contractPattern.MatchString(input.ContractVersion) || input.Purpose == "" ||
		!idPattern.MatchString(input.Name) || !idPattern.MatchString(input.ProcessorID) || input.Priority < -10000 || input.Priority > 10000 {
		return VariantDeclaration{}, ErrInvalid
	}
	if !contractPattern.MatchString(input.ProcessorContractVersion) || !idPattern.MatchString(input.ProcessorOwnerExtensionID) ||
		!digestPattern.MatchString(input.ProcessorPackageDigest) {
		return VariantDeclaration{}, ErrInvalid
	}
	value, err := normalizeExactMIME(input.OutputMIME)
	if err != nil {
		return VariantDeclaration{}, err
	}
	input.OutputMIME = value
	return input, nil
}

func normalizePublications(input []Publication, safeMode bool) ([]Publication, error) {
	if len(input) > maxPublications {
		return nil, ErrInvalid
	}
	result := make([]Publication, 0, len(input))
	owners := map[string]struct{}{}
	policies, processors, variants := 0, 0, 0
	for _, raw := range input {
		if safeMode && !validCoreArtifactSeal(raw.Artifact) {
			continue
		}
		publication, err := normalizePublication(raw)
		if err != nil {
			return nil, err
		}
		if _, duplicate := owners[publication.Artifact.ExtensionID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate publication %s", ErrConflict, publication.Artifact.ExtensionID)
		}
		owners[publication.Artifact.ExtensionID] = struct{}{}
		result = append(result, publication)
		policies += len(publication.Policies)
		processors += len(publication.Processors)
		variants += len(publication.Variants)
		if policies > maxPoliciesTotal || processors > maxProcessorsTotal || variants > maxVariantsTotal {
			return nil, ErrInvalid
		}
	}
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result, nil
}

func validBudget(value Budget) bool {
	return value.MaxFileBytes > 0 && value.MaxFileBytes <= hostMaximumBudget.MaxFileBytes &&
		value.MaxFiles > 0 && value.MaxFiles <= hostMaximumBudget.MaxFiles &&
		value.MaxDecompressedBytes > 0 && value.MaxDecompressedBytes <= hostMaximumBudget.MaxDecompressedBytes &&
		value.MaxDecompressionRatio > 0 && value.MaxDecompressionRatio <= hostMaximumBudget.MaxDecompressionRatio &&
		value.MaxFilenameBytes > 0 && value.MaxFilenameBytes <= hostMaximumBudget.MaxFilenameBytes &&
		value.MaxMIMECandidates > 0 && value.MaxMIMECandidates <= hostMaximumBudget.MaxMIMECandidates &&
		value.MaxMetadataBytes > 0 && value.MaxMetadataBytes <= hostMaximumBudget.MaxMetadataBytes &&
		value.MaxVariants > 0 && value.MaxVariants <= hostMaximumBudget.MaxVariants
}

func validRetryPolicy(value RetryPolicy, execution string) bool {
	if value.MaxAttempts == 0 && value.BaseDelaySeconds == 0 && value.MaxDelaySeconds == 0 {
		return execution == ExecutionSync
	}
	return value.MaxAttempts >= 1 && value.MaxAttempts <= 25 && value.BaseDelaySeconds >= 1 &&
		value.BaseDelaySeconds <= 3600 && value.MaxDelaySeconds >= value.BaseDelaySeconds && value.MaxDelaySeconds <= 86400
}

func normalizeMIMEPatterns(input []string, allowWildcard bool) ([]string, error) {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(input))
	for _, raw := range input {
		value := strings.ToLower(strings.TrimSpace(raw))
		if allowWildcard && value == "*/*" {
			// valid
		} else if allowWildcard && strings.HasSuffix(value, "/*") {
			major := strings.TrimSuffix(value, "/*")
			if major == "" || strings.ContainsAny(major, "/; \t") {
				return nil, ErrInvalid
			}
		} else {
			normalized, err := normalizeExactMIME(value)
			if err != nil {
				return nil, err
			}
			value = normalized
		}
		if len(value) > 127 {
			return nil, ErrInvalid
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeExactMIME(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" || len(input) > 127 || !validPlainString(input, 127) {
		return "", ErrInvalid
	}
	mediaType, _, err := mime.ParseMediaType(input)
	mediaType = strings.ToLower(mediaType)
	if err != nil || mediaType == "" || strings.Contains(mediaType, "*") || len(mediaType) > 127 {
		return "", ErrInvalid
	}
	return mediaType, nil
}

func normalizeExtensions(input []string) ([]string, error) {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(input))
	for _, raw := range input {
		value := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(raw)), ".")
		if !extensionPattern.MatchString(value) {
			return nil, ErrInvalid
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizePurpose(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "*" || idPattern.MatchString(value) {
		return value
	}
	return ""
}

func validPermission(value string) bool {
	return value != "" && validPlainString(value, maxPermissionBytes) && idPattern.MatchString(value)
}

func validPlainString(value string, limit int) bool {
	if !utf8.ValidString(value) || len(value) > limit || strings.ContainsRune(value, '\x00') {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func validStage(stage string) bool {
	switch stage {
	case StageValidate, StageScan, StageMetadata, StageTransform, StageCDN, StageRetention, StageBeforeDelete, StageAfterDelete:
		return true
	default:
		return false
	}
}

func duplicateID(seen map[string]struct{}, id string) bool {
	if _, exists := seen[id]; exists {
		return true
	}
	seen[id] = struct{}{}
	return false
}

func artifactBefore(left, right Artifact) bool {
	if left.Core != right.Core {
		return left.Core
	}
	if left.ExtensionID != right.ExtensionID {
		return left.ExtensionID < right.ExtensionID
	}
	if left.ExtensionVersion != right.ExtensionVersion {
		return left.ExtensionVersion < right.ExtensionVersion
	}
	if left.PackageDigest != right.PackageDigest {
		return left.PackageDigest < right.PackageDigest
	}
	if left.VersionID != right.VersionID {
		return left.VersionID < right.VersionID
	}
	return left.RuntimeInstanceID < right.RuntimeInstanceID
}

func graphDigest(publications []Publication, selections []ProviderSelection, safeMode bool) string {
	payload := struct {
		Schema       string              `json:"schema"`
		SafeMode     bool                `json:"safeMode"`
		Publications []Publication       `json:"publications"`
		Selections   []ProviderSelection `json:"selections"`
	}{
		Schema: SchemaVersion, SafeMode: safeMode, Publications: publications, Selections: selections,
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
