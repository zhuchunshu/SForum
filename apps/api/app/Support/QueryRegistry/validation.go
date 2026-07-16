package queryregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

// Bounds keep publication graphs and plan inputs finite. Individually valid
// declarations can otherwise explode startup work or plan cost.
const (
	maxPublications            = 512
	maxQueriesTotal            = 4096
	maxQueriesPerPublication   = 512
	maxFieldsPerQuery          = 128
	maxRelationsPerQuery       = 32
	maxFiltersPerQuery         = 64
	maxSortsPerQuery           = 16
	maxCacheTagsPerQuery       = 32
	maxIDLength                = 121
	maxSchemaRefLength         = 256
	maxOpaqueNameLength        = 256
	maxFilterValueLength       = 512
	maxLocaleLength            = 32
	maxScopeLength             = 128
	maxFingerprintLength       = 128
	maxExtensionVersionLength  = 128
	maxRuntimeInstanceIDLength = 512
	// Cost values cross JSON/protocol boundaries. This representational bound is
	// not the still-open product cost model; it prevents architecture-dependent
	// int ranges and pathological policy output from entering a durable plan.
	maxCostValue     = 1<<31 - 1
	maxPlanFields    = maxFieldsPerQuery
	maxPlanRelations = maxRelationsPerQuery
	maxPlanFilters   = maxFiltersPerQuery
	maxPlanSorts     = maxSortsPerQuery

	defaultPageLimit = 20
	maximumPageLimit = 100
	maximumOffset    = 1_000_000
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,120}$`)
	digestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contractPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
	schemaPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
	localePattern   = regexp.MustCompile(`^[A-Za-z]{2,3}([-_][A-Za-z0-9]{2,8}){0,3}$`)
)

func normalizePublications(input []Publication) ([]Publication, error) {
	if len(input) > maxPublications {
		return nil, ErrInvalid
	}
	result := make([]Publication, 0, len(input))
	seen := map[string]bool{}
	queryCount := 0
	for _, publication := range input {
		normalized, err := normalizePublication(publication)
		if err != nil {
			return nil, err
		}
		if seen[normalized.Artifact.ExtensionID] {
			return nil, fmt.Errorf("%w: duplicate publication %s", ErrConflict, normalized.Artifact.ExtensionID)
		}
		seen[normalized.Artifact.ExtensionID] = true
		queryCount += len(normalized.Queries)
		if queryCount > maxQueriesTotal {
			return nil, ErrInvalid
		}
		result = append(result, normalized)
	}
	sort.Slice(result, func(i, j int) bool {
		return artifactBefore(result[i].Artifact, result[j].Artifact)
	})
	return result, nil
}

func normalizePublication(input Publication) (Publication, error) {
	artifact, err := normalizeArtifact(input.Artifact)
	if err != nil {
		return Publication{}, ErrInvalid
	}
	if len(input.Queries) > maxQueriesPerPublication {
		return Publication{}, ErrInvalid
	}
	result := Publication{Artifact: artifact}
	seen := map[string]bool{}
	for _, raw := range input.Queries {
		declaration, queryErr := normalizeQueryDeclaration(artifact, raw)
		if queryErr != nil {
			return Publication{}, queryErr
		}
		if seen["query\x00"+declaration.ID] {
			return Publication{}, fmt.Errorf("%w: duplicate query %s in publication", ErrConflict, declaration.ID)
		}
		seen["query\x00"+declaration.ID] = true
		result.Queries = append(result.Queries, declaration)
	}
	sort.Slice(result.Queries, func(i, j int) bool {
		return result.Queries[i].ID < result.Queries[j].ID
	})
	return result, nil
}

func normalizeArtifact(input Artifact) (Artifact, error) {
	input.ExtensionID = strings.ToLower(strings.TrimSpace(input.ExtensionID))
	input.ExtensionVersion = strings.TrimSpace(input.ExtensionVersion)
	input.PackageDigest = normalizeDigest(input.PackageDigest)
	input.RuntimeInstanceID = strings.TrimSpace(input.RuntimeInstanceID)
	isCoreNamespace := strings.HasPrefix(input.ExtensionID, "core.")
	isTrustedCore := validCoreArtifactSeal(input)
	if !idPattern.MatchString(input.ExtensionID) || input.ExtensionID == "core" ||
		len(input.ExtensionVersion) > maxExtensionVersionLength ||
		!digestPattern.MatchString(input.PackageDigest) {
		return Artifact{}, ErrInvalid
	}
	if _, err := semver.StrictNewVersion(input.ExtensionVersion); err != nil {
		return Artifact{}, ErrInvalid
	}
	if input.Core {
		if !isCoreNamespace || !isTrustedCore || input.VersionID != 0 || input.RuntimeInstanceID != "" {
			return Artifact{}, ErrInvalid
		}
	} else if input.coreSeal != [32]byte{} || input.VersionID <= 0 ||
		!validBoundedOpaque(input.RuntimeInstanceID, maxRuntimeInstanceIDLength) {
		return Artifact{}, ErrInvalid
	}
	return input, nil
}

// NewCoreArtifact is the explicit Host-only authority boundary for core query
// catalogs. Callers must never feed extension-controlled values into it. The
// private seal prevents JSON or an Artifact literal in another package from
// promoting a publication by setting Core=true or choosing a core.* ID.
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

func normalizeQueryDeclaration(artifact Artifact, input QueryDeclaration) (QueryDeclaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Entity = strings.ToLower(strings.TrimSpace(input.Entity))
	input.PlanVersion = strings.TrimSpace(input.PlanVersion)
	input.Pagination = strings.ToLower(strings.TrimSpace(input.Pagination))
	input.ResultSchema = strings.TrimSpace(input.ResultSchema)
	input.PermissionPolicy = strings.ToLower(strings.TrimSpace(input.PermissionPolicy))

	fields, err := normalizeNameList(input.Fields, maxFieldsPerQuery, true)
	if err != nil {
		return QueryDeclaration{}, err
	}
	relations, err := normalizeNameList(input.Relations, maxRelationsPerQuery, false)
	if err != nil {
		return QueryDeclaration{}, err
	}
	filters, err := normalizeNameList(input.Filters, maxFiltersPerQuery, false)
	if err != nil {
		return QueryDeclaration{}, err
	}
	sorts, err := normalizeNameList(input.Sort, maxSortsPerQuery, false)
	if err != nil {
		return QueryDeclaration{}, err
	}
	cacheTags, err := normalizeNameList(input.CacheTags, maxCacheTagsPerQuery, false)
	if err != nil {
		return QueryDeclaration{}, err
	}
	if !validContributionIdentity(artifact, input.ID, input.ContractVersion) ||
		!idPattern.MatchString(input.Entity) || !contractPattern.MatchString(input.PlanVersion) ||
		len(input.PlanVersion) > maxSchemaRefLength ||
		!validSchemaRef(input.ResultSchema) || len(fields) == 0 ||
		!validPagination(input.Pagination) || !validPermissionPolicy(input.PermissionPolicy) {
		return QueryDeclaration{}, ErrInvalid
	}
	input.Fields = fields
	input.Relations = relations
	input.Filters = filters
	input.Sort = sorts
	input.CacheTags = cacheTags
	return input, nil
}

func normalizeNameList(input []string, limit int, required bool) ([]string, error) {
	if len(input) > limit {
		return nil, ErrInvalid
	}
	if required && len(input) == 0 {
		return nil, ErrInvalid
	}
	result := make([]string, 0, len(input))
	seen := map[string]bool{}
	for _, value := range input {
		value = strings.TrimSpace(value)
		if !validOpaqueName(value) || seen[value] {
			return nil, ErrInvalid
		}
		seen[value] = true
		result = append(result, value)
	}
	// Stable declaration order is declaration order after lowercasing; do not
	// re-sort names so plan selection can preserve request order later.
	return result, nil
}

func validContributionIdentity(artifact Artifact, id, contract string) bool {
	return idPattern.MatchString(id) && strings.HasPrefix(id, artifact.ExtensionID+".") &&
		contractPattern.MatchString(contract) &&
		len(id) <= maxIDLength && len(contract) <= maxSchemaRefLength
}

func validPagination(value string) bool {
	switch value {
	case PaginationNone, PaginationOffset, PaginationCursor:
		return true
	default:
		return false
	}
}

func validPermissionPolicy(value string) bool {
	if value == PermissionPolicyPublic || value == PermissionPolicyLogin {
		return true
	}
	return idPattern.MatchString(value)
}

func validSchemaRef(value string) bool {
	if len(value) > maxSchemaRefLength {
		return false
	}
	if schemaPattern.MatchString(value) {
		return true
	}
	clean, ok := extensionmanifest.SafeArchivePath(value)
	return ok && clean == value && strings.HasSuffix(value, ".json")
}

func validOpaqueName(value string) bool {
	return validBoundedOpaque(value, maxOpaqueNameLength)
}

func validBoundedOpaque(value string, limit int) bool {
	if value == "" || len(value) > limit {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func normalizeDigest(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
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
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
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

// filterSafeModeInput drops non-core publications before validation so a
// corrupt third-party query cannot block Host Safe Mode recovery.
func filterSafeModeInput(publications []Publication, safeMode bool) []Publication {
	if !safeMode {
		return publications
	}
	result := make([]Publication, 0, len(publications))
	for _, publication := range publications {
		if validCoreArtifactSeal(publication.Artifact) {
			result = append(result, publication)
		}
	}
	return result
}
