package cacheregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

const (
	maxPublications            = 512
	maxCachesTotal             = 4096
	maxCachesPerPublication    = 512
	maxTagsPerCache            = 64
	maxInvalidatorsPerCache    = 64
	maxTagsTotal               = 32768
	maxInvalidatorsTotal       = 32768
	maxIDLength                = 81
	maxContractVersionLength   = 256
	maxExtensionVersionLength  = 128
	maxRuntimeInstanceIDLength = 512
	maxFingerprintLength       = 128
	maxLocaleLength            = 128
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,80}$`)
	digestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contractPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
)

func normalizePublications(input []Publication) ([]Publication, error) {
	if len(input) > maxPublications {
		return nil, ErrInvalid
	}
	result := make([]Publication, 0, len(input))
	owners := make(map[string]struct{}, len(input))
	cacheCount, tagCount, invalidatorCount := 0, 0, 0
	for _, raw := range input {
		publication, err := normalizePublication(raw)
		if err != nil {
			return nil, err
		}
		if _, duplicate := owners[publication.Artifact.ExtensionID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate publication %s", ErrConflict, publication.Artifact.ExtensionID)
		}
		owners[publication.Artifact.ExtensionID] = struct{}{}
		result = append(result, publication)
		cacheCount += len(publication.Caches)
		for _, declaration := range publication.Caches {
			tagCount += len(declaration.Tags)
			invalidatorCount += len(declaration.Invalidators)
		}
		if cacheCount > maxCachesTotal || tagCount > maxTagsTotal || invalidatorCount > maxInvalidatorsTotal {
			return nil, ErrInvalid
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return artifactBefore(result[i].Artifact, result[j].Artifact)
	})
	return result, nil
}

func normalizePublication(input Publication) (Publication, error) {
	artifact, err := normalizeArtifact(input.Artifact)
	if err != nil || len(input.Caches) > maxCachesPerPublication {
		return Publication{}, ErrInvalid
	}
	result := Publication{Artifact: artifact}
	ids := make(map[string]struct{}, len(input.Caches))
	namespaces := make(map[string]struct{}, len(input.Caches))
	for _, raw := range input.Caches {
		declaration, err := normalizeDeclaration(artifact, raw)
		if err != nil {
			return Publication{}, err
		}
		if _, duplicate := ids[declaration.ID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate cache id %s", ErrConflict, declaration.ID)
		}
		if _, duplicate := namespaces[declaration.Namespace]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate cache namespace %s", ErrConflict, declaration.Namespace)
		}
		ids[declaration.ID] = struct{}{}
		namespaces[declaration.Namespace] = struct{}{}
		result.Caches = append(result.Caches, declaration)
	}
	sort.Slice(result.Caches, func(i, j int) bool {
		if result.Caches[i].ID != result.Caches[j].ID {
			return result.Caches[i].ID < result.Caches[j].ID
		}
		return result.Caches[i].Namespace < result.Caches[j].Namespace
	})
	return result, nil
}

func normalizeArtifact(input Artifact) (Artifact, error) {
	input.ExtensionID = strings.ToLower(strings.TrimSpace(input.ExtensionID))
	input.ExtensionVersion = strings.TrimSpace(input.ExtensionVersion)
	input.PackageDigest = normalizeDigest(input.PackageDigest)
	input.RuntimeInstanceID = strings.TrimSpace(input.RuntimeInstanceID)
	isCoreNamespace := strings.HasPrefix(input.ExtensionID, "core.")
	isSealedCore := validCoreArtifactSeal(input)
	if !idPattern.MatchString(input.ExtensionID) || input.ExtensionID == "core" ||
		len(input.ExtensionVersion) > maxExtensionVersionLength || !digestPattern.MatchString(input.PackageDigest) {
		return Artifact{}, ErrInvalid
	}
	if _, err := semver.StrictNewVersion(input.ExtensionVersion); err != nil {
		return Artifact{}, ErrInvalid
	}
	if input.Core {
		if !isCoreNamespace || !isSealedCore || input.VersionID != 0 || input.RuntimeInstanceID != "" {
			return Artifact{}, ErrInvalid
		}
	} else if input.coreSeal != [32]byte{} || input.VersionID <= 0 ||
		!validBoundedOpaque(input.RuntimeInstanceID, maxRuntimeInstanceIDLength) {
		return Artifact{}, ErrInvalid
	}
	return input, nil
}

// NewCoreArtifact is the explicit Host authority boundary. Extension-controlled
// JSON must never be passed to this constructor; decoded Core flags remain unsealed.
func NewCoreArtifact(extensionID, extensionVersion, packageDigest string) (Artifact, error) {
	artifact := Artifact{
		ExtensionID:      strings.ToLower(strings.TrimSpace(extensionID)),
		ExtensionVersion: strings.TrimSpace(extensionVersion),
		PackageDigest:    normalizeDigest(packageDigest),
		Core:             true,
	}
	artifact.coreSeal = coreArtifactSeal(artifact)
	return normalizeArtifact(artifact)
}

func coreArtifactSeal(artifact Artifact) [32]byte {
	material := SchemaVersion + "\x00core-artifact\x00" + artifact.ExtensionID + "\x00" +
		artifact.ExtensionVersion + "\x00" + artifact.PackageDigest
	return sha256.Sum256([]byte(material))
}

func validCoreArtifactSeal(artifact Artifact) bool {
	return artifact.Core && artifact.coreSeal != [32]byte{} && artifact.coreSeal == coreArtifactSeal(artifact)
}

func normalizeDeclaration(artifact Artifact, input Declaration) (Declaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Namespace = strings.ToLower(strings.TrimSpace(input.Namespace))
	input.Policy = strings.ToLower(strings.TrimSpace(input.Policy))
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	var err error
	if input.Tags, err = normalizeOwnedIDs(artifact.ExtensionID, input.Tags, maxTagsPerCache); err != nil {
		return Declaration{}, err
	}
	if input.Invalidators, err = normalizeOwnedIDs(artifact.ExtensionID, input.Invalidators, maxInvalidatorsPerCache); err != nil {
		return Declaration{}, err
	}
	if !validContributionID(artifact, input.ID) || !contractPattern.MatchString(input.ContractVersion) ||
		len(input.ContractVersion) > maxContractVersionLength || !idPattern.MatchString(input.Namespace) ||
		!strings.HasPrefix(input.Namespace, artifact.ExtensionID+".") || !validPolicy(input.Policy) ||
		(input.Provider != "" && !idPattern.MatchString(input.Provider)) {
		return Declaration{}, ErrInvalid
	}
	return input, nil
}

func normalizeOwnedIDs(extensionID string, input []string, maximum int) ([]string, error) {
	values, err := normalizeIDs(input, maximum)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		if !strings.HasPrefix(value, extensionID+".") {
			return nil, ErrInvalid
		}
	}
	return values, nil
}

func normalizeIDs(input []string, maximum int) ([]string, error) {
	if len(input) > maximum {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		value := strings.ToLower(strings.TrimSpace(raw))
		if !idPattern.MatchString(value) || len(value) > maxIDLength {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, ErrInvalid
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func validContributionID(artifact Artifact, id string) bool {
	return idPattern.MatchString(id) && len(id) <= maxIDLength && strings.HasPrefix(id, artifact.ExtensionID+".")
}

func validPolicy(value string) bool {
	switch value {
	case PolicyPrivate, PolicyActor, PolicyPermission, PolicyPublic:
		return true
	default:
		return false
	}
}

func artifactBefore(left, right Artifact) bool {
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
	if left.RuntimeInstanceID != right.RuntimeInstanceID {
		return left.RuntimeInstanceID < right.RuntimeInstanceID
	}
	return !left.Core && right.Core
}

func computeGraphDigest(publications []Publication, safeMode bool) string {
	if publications == nil {
		publications = []Publication{}
	}
	document := struct {
		SchemaVersion string        `json:"schemaVersion"`
		SafeMode      bool          `json:"safeMode"`
		Publications  []Publication `json:"publications"`
	}{SchemaVersion: SchemaVersion, SafeMode: safeMode, Publications: publications}
	body, _ := json.Marshal(document)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func isolationDigest(contribution Contribution, actor, permission, locale string) string {
	value := SchemaVersion + "\x00isolation\x00" + contribution.ID + "\x00" + contribution.ContractVersion + "\x00" +
		contribution.Namespace + "\x00" + contribution.Policy + "\x00" + contribution.Provider + "\x00" +
		contribution.Artifact.ExtensionID + "\x00" + contribution.Artifact.ExtensionVersion + "\x00" +
		contribution.Artifact.PackageDigest + "\x00" + strconv.FormatInt(contribution.Artifact.VersionID, 10) + "\x00" +
		contribution.Artifact.RuntimeInstanceID + "\x00" + strconv.FormatBool(contribution.Artifact.Core) + "\x00" +
		actor + "\x00" + permission + "\x00" + locale
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func normalizeDigest(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
}

func validBoundedOpaque(value string, maximum int) bool {
	if value == "" || len(value) > maximum {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func filterSafeModeInput(input []Publication, safeMode bool) []Publication {
	if !safeMode {
		return input
	}
	result := make([]Publication, 0, len(input))
	for _, publication := range input {
		// Filter before parsing declarations so corrupt third-party input cannot
		// block Host recovery. Core=true or a core.* prefix is not sufficient.
		if validCoreArtifactSeal(publication.Artifact) {
			result = append(result, publication)
		}
	}
	return result
}
