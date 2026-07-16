package seoregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	semver "github.com/Masterminds/semver/v3"
)

const (
	maxPublications            = 512
	maxContributions           = 4096
	maxContributionsPerPackage = 512
	maxPriority                = 1_000_000
	maxRuntimeInstanceIDLength = 160
	maxTitleRunes              = 300
	maxMetaTags                = 128
	maxMetaContentRunes        = 4096
	maxHreflangLinks           = 64
	maxSitemapEntries          = 2048
	maxJSONLDDocuments         = 64
	maxJSONLDImages            = 32
	maxJSONLDParties           = 32
	maxJSONLDBreadcrumbs       = 64
	maxTextRunes               = 4096
	defaultProviderTimeout     = 500 * time.Millisecond
	maximumProviderTimeout     = 5 * time.Second
	defaultExecutionTimeout    = 3 * time.Second
	maximumExecutionTimeout    = 15 * time.Second
	defaultMaximumBytes        = 512 << 10
	maximumOutputBytes         = 2 << 20
)

var (
	idPattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,120}$`)
	digestPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contractPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*@[1-9][0-9]*$`)
	opaquePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
	metaKeyPattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9:._-]{0,127}$`)
	localePattern     = regexp.MustCompile(`^[A-Za-z]{2,8}(?:-[A-Za-z0-9]{1,8})*$`)
	schemaTypePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{0,127}$`)
	dnsLabelPattern   = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
)

func normalizePublications(input []Publication) ([]Publication, error) {
	if len(input) > maxPublications {
		return nil, ErrInvalid
	}
	result := make([]Publication, 0, len(input))
	owners := make(map[string]struct{}, len(input))
	total := 0
	for _, raw := range input {
		publication, err := normalizePublication(raw)
		if err != nil {
			return nil, err
		}
		if _, duplicate := owners[publication.Artifact.ExtensionID]; duplicate {
			return nil, fmt.Errorf("%w: duplicate publication %s", ErrConflict, publication.Artifact.ExtensionID)
		}
		owners[publication.Artifact.ExtensionID] = struct{}{}
		total += len(publication.Contributions)
		if total > maxContributions {
			return nil, ErrInvalid
		}
		result = append(result, publication)
	}
	sort.Slice(result, func(i, j int) bool { return artifactBefore(result[i].Artifact, result[j].Artifact) })
	return result, nil
}

func normalizePublication(input Publication) (Publication, error) {
	artifact, err := normalizeArtifact(input.Artifact)
	if err != nil || len(input.Contributions) > maxContributionsPerPackage {
		return Publication{}, ErrInvalid
	}
	result := Publication{Artifact: artifact}
	ids := make(map[string]struct{}, len(input.Contributions))
	for _, raw := range input.Contributions {
		declaration, declarationErr := normalizeDeclaration(artifact, raw)
		if declarationErr != nil {
			return Publication{}, declarationErr
		}
		if _, duplicate := ids[declaration.ID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate contribution %s", ErrConflict, declaration.ID)
		}
		ids[declaration.ID] = struct{}{}
		result.Contributions = append(result.Contributions, declaration)
	}
	sort.Slice(result.Contributions, func(i, j int) bool { return result.Contributions[i].ID < result.Contributions[j].ID })
	return result, nil
}

func normalizeArtifact(input Artifact) (Artifact, error) {
	input.ExtensionID = strings.ToLower(strings.TrimSpace(input.ExtensionID))
	input.ExtensionVersion = strings.TrimSpace(input.ExtensionVersion)
	input.PackageDigest = normalizeDigest(input.PackageDigest)
	input.ImpactDigest = normalizeDigest(input.ImpactDigest)
	input.RuntimeInstanceID = strings.TrimSpace(input.RuntimeInstanceID)
	isCoreNamespace := strings.HasPrefix(input.ExtensionID, "core.")
	if !idPattern.MatchString(input.ExtensionID) || input.ExtensionID == "core" ||
		!digestPattern.MatchString(input.PackageDigest) || !digestPattern.MatchString(input.ImpactDigest) {
		return Artifact{}, ErrInvalid
	}
	if _, err := semver.StrictNewVersion(input.ExtensionVersion); err != nil {
		return Artifact{}, ErrInvalid
	}
	if input.Core {
		if !isCoreNamespace || !validCoreArtifactSeal(input) || input.VersionID != 0 || input.RuntimeInstanceID != "" {
			return Artifact{}, ErrInvalid
		}
	} else if input.coreSeal != [32]byte{} || isCoreNamespace || input.VersionID <= 0 ||
		len(input.RuntimeInstanceID) > maxRuntimeInstanceIDLength || !opaquePattern.MatchString(input.RuntimeInstanceID) {
		return Artifact{}, ErrInvalid
	}
	return input, nil
}

// NewCoreArtifact is the Host-only construction boundary. Extension-controlled
// values must never be passed through this constructor.
func NewCoreArtifact(extensionID, extensionVersion, packageDigest, impactDigest string) (Artifact, error) {
	artifact := Artifact{
		ExtensionID: strings.ToLower(strings.TrimSpace(extensionID)), ExtensionVersion: strings.TrimSpace(extensionVersion),
		PackageDigest: normalizeDigest(packageDigest), ImpactDigest: normalizeDigest(impactDigest), Core: true,
	}
	artifact.coreSeal = coreArtifactSeal(artifact)
	return normalizeArtifact(artifact)
}

func coreArtifactSeal(artifact Artifact) [32]byte {
	material := SchemaVersion + "\x00core-artifact\x00" + artifact.ExtensionID + "\x00" +
		artifact.ExtensionVersion + "\x00" + artifact.PackageDigest + "\x00" + artifact.ImpactDigest
	return sha256.Sum256([]byte(material))
}

func validCoreArtifactSeal(artifact Artifact) bool {
	return artifact.Core && artifact.coreSeal != [32]byte{} && artifact.coreSeal == coreArtifactSeal(artifact)
}

func filterSafeModeInput(input []Publication, safeMode bool) []Publication {
	if !safeMode {
		return input
	}
	result := make([]Publication, 0, len(input))
	for _, publication := range input {
		// Filter before declaration validation so corrupt third-party input cannot
		// block Host recovery. Core=true or a core.* prefix is not sufficient.
		if validCoreArtifactSeal(publication.Artifact) {
			result = append(result, publication)
		}
	}
	return result
}

func normalizeDeclaration(artifact Artifact, input Declaration) (Declaration, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	input.ContractVersion = strings.TrimSpace(input.ContractVersion)
	input.Scope = strings.ToLower(strings.TrimSpace(input.Scope))
	input.Kind = strings.ToLower(strings.TrimSpace(input.Kind))
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.Handler = strings.ToLower(strings.TrimSpace(input.Handler))
	input.FailurePolicy = strings.ToLower(strings.TrimSpace(input.FailurePolicy))
	if input.Timeout == 0 {
		input.Timeout = defaultProviderTimeout
	}
	if !validContributionIdentity(artifact, input.ID, input.ContractVersion) ||
		(input.Scope != GlobalScope && !idPattern.MatchString(input.Scope)) || !validKind(input.Kind) ||
		!validAction(input.Action) || !idPattern.MatchString(input.Handler) ||
		!strings.HasPrefix(input.Handler, artifact.ExtensionID+".") || input.Priority < -maxPriority ||
		input.Priority > maxPriority || (input.FailurePolicy != FailurePolicyFailClosed && input.FailurePolicy != FailurePolicyFallback) ||
		input.Timeout < time.Millisecond || input.Timeout > maximumProviderTimeout {
		return Declaration{}, ErrInvalid
	}
	return input, nil
}

func validContributionIdentity(artifact Artifact, id, contract string) bool {
	return idPattern.MatchString(id) && contractPattern.MatchString(contract) && contract == id+"@1" &&
		strings.HasPrefix(id, artifact.ExtensionID+".")
}

func validKind(value string) bool {
	switch value {
	case KindTitle, KindMeta, KindCanonical, KindRobots, KindHreflang, KindSitemap, KindJSONLD:
		return true
	default:
		return false
	}
}

func validAction(value string) bool {
	return value == ActionAdd || value == ActionFilter || value == ActionReplace
}

func normalizeDigest(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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
	if left.ImpactDigest != right.ImpactDigest {
		return left.ImpactDigest < right.ImpactDigest
	}
	if left.VersionID != right.VersionID {
		return left.VersionID < right.VersionID
	}
	return left.RuntimeInstanceID < right.RuntimeInstanceID
}

func contributionBefore(left, right Contribution) bool {
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.Scope != right.Scope {
		// Exact scope runs before global scope at equal Host priority.
		return left.Scope != GlobalScope
	}
	if left.Artifact != right.Artifact {
		return artifactBefore(left.Artifact, right.Artifact)
	}
	return left.ID < right.ID
}

func computeGraphDigest(publications []Publication, safeMode bool) string {
	if publications == nil {
		publications = []Publication{}
	}
	body, _ := json.Marshal(struct {
		SchemaVersion string        `json:"schemaVersion"`
		SafeMode      bool          `json:"safeMode"`
		Publications  []Publication `json:"publications"`
	}{SchemaVersion: SchemaVersion, SafeMode: safeMode, Publications: publications})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func validateDocument(input Document) error {
	if err := validateText(input.Title, maxTitleRunes, true); err != nil {
		return fmt.Errorf("%w: title", ErrOutputInvalid)
	}
	if len(input.Meta) > maxMetaTags || len(input.Hreflang) > maxHreflangLinks ||
		len(input.Sitemap) > maxSitemapEntries || len(input.JSONLD) > maxJSONLDDocuments {
		return ErrOutputTooLarge
	}
	if input.CanonicalURL != "" {
		canonical, err := canonicalAbsoluteURL(input.CanonicalURL, false)
		if err != nil || canonical != input.CanonicalURL {
			return fmt.Errorf("%w: canonical URL", ErrOutputInvalid)
		}
	}
	if err := validateRobots(input.Robots); err != nil {
		return err
	}
	metaKeys := make(map[string]struct{}, len(input.Meta))
	for _, item := range input.Meta {
		if item.Attribute != "name" && item.Attribute != "property" || !metaKeyPattern.MatchString(item.Key) ||
			validateText(item.Content, maxMetaContentRunes, false) != nil {
			return fmt.Errorf("%w: meta tag", ErrOutputInvalid)
		}
		key := strings.ToLower(item.Attribute + "\x00" + item.Key)
		if _, duplicate := metaKeys[key]; duplicate {
			return fmt.Errorf("%w: duplicate meta tag", ErrOutputInvalid)
		}
		metaKeys[key] = struct{}{}
	}
	locales := make(map[string]struct{}, len(input.Hreflang))
	for _, link := range input.Hreflang {
		locale := strings.ToLower(link.Locale)
		if !validCanonicalLocale(link.Locale) {
			return fmt.Errorf("%w: hreflang locale", ErrOutputInvalid)
		}
		if _, duplicate := locales[locale]; duplicate {
			return fmt.Errorf("%w: duplicate hreflang locale", ErrOutputInvalid)
		}
		locales[locale] = struct{}{}
		canonical, err := canonicalAbsoluteURL(link.URL, false)
		if err != nil || canonical != link.URL {
			return fmt.Errorf("%w: hreflang URL", ErrOutputInvalid)
		}
	}
	sitemapURLs := make(map[string]struct{}, len(input.Sitemap))
	for _, entry := range input.Sitemap {
		canonical, err := canonicalAbsoluteURL(entry.URL, false)
		if err != nil || canonical != entry.URL {
			return fmt.Errorf("%w: sitemap URL", ErrOutputInvalid)
		}
		if _, duplicate := sitemapURLs[canonical]; duplicate {
			return fmt.Errorf("%w: duplicate sitemap URL", ErrOutputInvalid)
		}
		sitemapURLs[canonical] = struct{}{}
		if entry.LastModified != "" {
			if !validCanonicalDate(entry.LastModified) {
				return fmt.Errorf("%w: sitemap last-modified", ErrOutputInvalid)
			}
		}
		if entry.ChangeFrequency != "" && !validChangeFrequency(entry.ChangeFrequency) {
			return fmt.Errorf("%w: sitemap change frequency", ErrOutputInvalid)
		}
		if entry.Priority != nil && (math.IsNaN(*entry.Priority) || math.IsInf(*entry.Priority, 0) || *entry.Priority < 0 || *entry.Priority > 1) {
			return fmt.Errorf("%w: sitemap priority", ErrOutputInvalid)
		}
	}
	jsonIDs := make(map[string]struct{}, len(input.JSONLD))
	for _, document := range input.JSONLD {
		if err := validateJSONLDDocument(document); err != nil {
			return err
		}
		key := document.ID
		if key == "" {
			key = document.Type + "\x00" + document.URL
		}
		if _, duplicate := jsonIDs[key]; duplicate {
			return fmt.Errorf("%w: duplicate JSON-LD identity", ErrOutputInvalid)
		}
		jsonIDs[key] = struct{}{}
	}
	return nil
}

func validateRobots(value RobotsDirectives) error {
	if value == (RobotsDirectives{}) {
		return nil
	}
	if (value.Indexing != RobotsIndex && value.Indexing != RobotsNoIndex) ||
		(value.Following != RobotsFollow && value.Following != RobotsNoFollow) {
		return fmt.Errorf("%w: robots directives", ErrOutputInvalid)
	}
	return nil
}

func validChangeFrequency(value string) bool {
	switch value {
	case SitemapAlways, SitemapHourly, SitemapDaily, SitemapWeekly, SitemapMonthly, SitemapYearly, SitemapNever:
		return true
	default:
		return false
	}
}

func validateJSONLDDocument(document JSONLDDocument) error {
	if document.Context != "https://schema.org" || !schemaTypePattern.MatchString(document.Type) {
		return fmt.Errorf("%w: JSON-LD context or type", ErrOutputInvalid)
	}
	for _, value := range []string{document.Name, document.Headline, document.Description} {
		if validateText(value, maxTextRunes, true) != nil {
			return fmt.Errorf("%w: JSON-LD text", ErrOutputInvalid)
		}
	}
	for _, value := range []string{document.ID, document.URL} {
		if value != "" {
			canonical, err := canonicalAbsoluteURL(value, true)
			if err != nil || canonical != value {
				return fmt.Errorf("%w: JSON-LD URL", ErrOutputInvalid)
			}
		}
	}
	if len(document.ImageURLs) > maxJSONLDImages || len(document.Author) > maxJSONLDParties ||
		len(document.Breadcrumbs) > maxJSONLDBreadcrumbs {
		return ErrOutputTooLarge
	}
	for _, imageURL := range document.ImageURLs {
		canonical, err := canonicalAbsoluteURL(imageURL, false)
		if err != nil || canonical != imageURL {
			return fmt.Errorf("%w: JSON-LD image URL", ErrOutputInvalid)
		}
	}
	for _, value := range []string{document.DatePublished, document.DateModified} {
		if value != "" {
			if !validCanonicalDate(value) {
				return fmt.Errorf("%w: JSON-LD date", ErrOutputInvalid)
			}
		}
	}
	for _, party := range document.Author {
		if err := validateJSONLDParty(party); err != nil {
			return err
		}
	}
	if document.Publisher != nil {
		if err := validateJSONLDParty(*document.Publisher); err != nil {
			return err
		}
	}
	for index, breadcrumb := range document.Breadcrumbs {
		if breadcrumb.Type != "ListItem" || breadcrumb.Position != index+1 || validateText(breadcrumb.Name, maxTextRunes, false) != nil {
			return fmt.Errorf("%w: JSON-LD breadcrumb", ErrOutputInvalid)
		}
		canonical, err := canonicalAbsoluteURL(breadcrumb.URL, false)
		if err != nil || canonical != breadcrumb.URL {
			return fmt.Errorf("%w: JSON-LD breadcrumb URL", ErrOutputInvalid)
		}
	}
	return nil
}

func validateJSONLDParty(party JSONLDParty) error {
	if (party.Type != "Person" && party.Type != "Organization") || validateText(party.Name, maxTextRunes, false) != nil {
		return fmt.Errorf("%w: JSON-LD party", ErrOutputInvalid)
	}
	for _, value := range []string{party.ID, party.URL, party.LogoURL} {
		if value != "" {
			canonical, err := canonicalAbsoluteURL(value, true)
			if err != nil || canonical != value {
				return fmt.Errorf("%w: JSON-LD party URL", ErrOutputInvalid)
			}
		}
	}
	return nil
}

func validateText(value string, maxRunes int, emptyAllowed bool) error {
	if value == "" {
		if emptyAllowed {
			return nil
		}
		return ErrOutputInvalid
	}
	if !utf8.ValidString(value) || strings.TrimSpace(value) != value || utf8.RuneCountInString(value) > maxRunes {
		return ErrOutputInvalid
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ErrOutputInvalid
		}
	}
	return nil
}

// canonicalAbsoluteURL rejects ambiguous, credential-bearing, fragment, and
// non-HTTP(S) URLs. Callers compare the returned value to input where exact
// canonical form is required (notably sitemap output).
