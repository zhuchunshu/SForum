package extensions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	capabilities "github.com/zhuchunshu/sforum/apps/api/app/Support/Capabilities"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

const DefaultTrustChallengeTTL = 5 * time.Minute

type ExecutableTrustService struct {
	extensions FrontendExtensionReader
	store      ExecutableTrustStore
	auditor    audit.Writer
	ttl        time.Duration
	now        func() time.Time
	random     io.Reader
}

func NewExecutableTrustService(extensions FrontendExtensionReader, store ExecutableTrustStore) *ExecutableTrustService {
	return &ExecutableTrustService{
		extensions: extensions,
		store:      store,
		ttl:        DefaultTrustChallengeTTL,
		now:        func() time.Time { return time.Now().UTC() },
		random:     rand.Reader,
	}
}

func (s *ExecutableTrustService) WithAuditor(writer audit.Writer) *ExecutableTrustService {
	s.auditor = writer
	return s
}

func (s *ExecutableTrustService) WithTTL(ttl time.Duration) *ExecutableTrustService {
	if ttl > 0 && ttl <= DefaultTrustChallengeTTL {
		s.ttl = ttl
	}
	return s
}

func (s *ExecutableTrustService) Impact(ctx context.Context, actor identity.Actor, extensionID string) (TrustImpact, error) {
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return TrustImpact{}, err
	}
	if !canViewExtensions(actor) && !canManagePlugins(actor) && !canManageThemes(actor) {
		return TrustImpact{}, identity.ErrPermissionDenied
	}
	return buildTrustImpact(trustReviewArtifact(extension), TrustActionEnable)
}

func (s *ExecutableTrustService) Status(ctx context.Context, actor identity.Actor, extensionID string) (ExecutableTrustStatus, error) {
	impact, err := s.Impact(ctx, actor, extensionID)
	if err != nil {
		return ExecutableTrustStatus{}, err
	}
	required := impact.Source == SourceUploaded && (len(impact.Binaries) > 0 || len(impact.Components) > 0 || len(impact.Migrations) > 0)
	trusted := !required
	if required {
		trusted, err = s.hasLiveGrant(ctx, impact)
		if err != nil {
			return ExecutableTrustStatus{}, err
		}
	}
	return ExecutableTrustStatus{Impact: impact, TrustRequired: required, Trusted: trusted}, nil
}

// TrustedArtifact 供同一进程内的执行入口复用整包信任结论，避免后端与前端各自授权。
func (s *ExecutableTrustService) TrustedArtifact(ctx context.Context, extension Extension) (bool, error) {
	if !RequiresExecutableTrust(extension) {
		return true, nil
	}
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		return false, err
	}
	return s.hasLiveGrant(ctx, impact)
}

// RuntimeIdentity returns the exact live trust row used by a v2 subprocess.
// The handshake may disclose this identity, but only the host-side grant lookup
// is authoritative for execution.
func (s *ExecutableTrustService) RuntimeIdentity(ctx context.Context, extension Extension) (RuntimeTrustIdentity, error) {
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		return RuntimeTrustIdentity{}, err
	}
	if !RequiresExecutableTrust(extension) {
		return RuntimeTrustIdentity{TrustGrantID: "builtin", ImpactDigest: impact.Digest}, nil
	}
	if s == nil || s.store == nil {
		return RuntimeTrustIdentity{}, ErrTrustGrantNotFound
	}
	grant, err := s.store.LiveGrant(ctx, trustIdentity(impact))
	if err != nil {
		return RuntimeTrustIdentity{}, err
	}
	return RuntimeTrustIdentity{TrustGrantID: strconv.FormatInt(grant.ID, 10), ImpactDigest: grant.ImpactDigest}, nil
}

func (s *ExecutableTrustService) hasLiveGrant(ctx context.Context, impact TrustImpact) (bool, error) {
	if s == nil || s.store == nil {
		return false, ErrTrustChallengeInvalid
	}
	return s.store.HasLiveGrant(ctx, trustIdentity(impact))
}

func (s *ExecutableTrustService) Challenge(ctx context.Context, actor identity.Actor, extensionID string) (TrustChallenge, error) {
	if !actor.IsSuperAdmin() {
		return TrustChallenge{}, identity.ErrPermissionDenied
	}
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return TrustChallenge{}, err
	}
	extension = trustReviewArtifact(extension)
	if !RequiresExecutableTrust(extension) {
		return TrustChallenge{}, ErrTrustNotRequired
	}
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		return TrustChallenge{}, err
	}
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(s.random, tokenBytes); err != nil {
		return TrustChallenge{}, fmt.Errorf("create trust challenge token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := s.now().Add(s.ttl)
	if err := s.store.CreateChallenge(ctx, TrustChallengeRecord{
		TokenHash: tokenHash(token), ActorUserID: actor.ID,
		Identity: trustIdentity(impact), ArtifactDigests: impact.ArtifactDigests,
		Impact: impact, ExpiresAt: expiresAt,
	}); err != nil {
		return TrustChallenge{}, err
	}
	s.appendAudit(ctx, actor, audit.ActionExtensionTrustChallenge, impact, nil)
	return TrustChallenge{Token: token, Impact: impact, ExpiresAt: expiresAt}, nil
}

// ConfirmEnable 对同一摘要的已授权重启直接放行；新授权必须原子消费挑战。
func (s *ExecutableTrustService) ConfirmEnable(ctx context.Context, actor identity.Actor, extension Extension, token string) error {
	if !RequiresExecutableTrust(extension) {
		return nil
	}
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		return err
	}
	identityKey := trustIdentity(impact)
	granted, err := s.store.HasLiveGrant(ctx, identityKey)
	if err != nil {
		return err
	}
	if granted {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return ErrTrustChallengeRequired
	}
	if !actor.IsSuperAdmin() {
		return identity.ErrPermissionDenied
	}
	grant, err := s.store.ConsumeChallenge(ctx, TrustConsumeInput{
		TokenHash: tokenHash(strings.TrimSpace(token)), ActorUserID: actor.ID, Identity: identityKey,
	})
	if err != nil {
		s.appendAudit(ctx, actor, audit.ActionExtensionTrustDenied, impact, err)
		return err
	}
	s.appendAudit(ctx, actor, audit.ActionExtensionTrustGrant, impact, nil)
	_ = grant
	return nil
}

func (s *ExecutableTrustService) RevokeAllForExtension(ctx context.Context, extensionID string, actorUserID int64, reason string) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.RevokeAll(ctx, normalizeID(extensionID), actorUserID, strings.TrimSpace(reason))
}

func (s *ExecutableTrustService) Revoke(ctx context.Context, actor identity.Actor, extensionID string) (ExecutableTrustStatus, error) {
	if !actor.IsSuperAdmin() {
		return ExecutableTrustStatus{}, identity.ErrPermissionDenied
	}
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return ExecutableTrustStatus{}, err
	}
	if err := s.store.RevokeAll(ctx, extension.ID, actor.ID, "operator_revoked"); err != nil {
		return ExecutableTrustStatus{}, err
	}
	extension = trustReviewArtifact(extension)
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		impact = TrustImpact{
			SchemaVersion: TrustImpactSchemaV2, Action: TrustActionEnable,
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			ExtensionType: extension.Type, Source: extension.Source,
			PackageDigest:    extension.PackageDigest,
			ManifestContract: extensionmanifest.ManifestContract(extension.Manifest),
		}
	}
	s.appendAudit(ctx, actor, audit.ActionExtensionTrustRevoke, impact, nil)
	return ExecutableTrustStatus{Impact: impact, TrustRequired: RequiresExecutableTrust(extension), Trusted: false}, nil
}

func (s *ExecutableTrustService) extension(ctx context.Context, extensionID string) (Extension, error) {
	if s == nil || s.extensions == nil || s.store == nil {
		return Extension{}, ErrTrustChallengeInvalid
	}
	return s.extensions.Get(ctx, normalizeID(extensionID))
}

func (s *ExecutableTrustService) appendAudit(ctx context.Context, actor identity.Actor, action string, impact TrustImpact, denied error) {
	if s == nil || s.auditor == nil {
		return
	}
	metadata := map[string]any{
		"extensionId":   impact.ExtensionID,
		"version":       impact.ExtensionVersion,
		"packageDigest": impact.PackageDigest,
		"impactDigest":  impact.Digest,
		"action":        impact.Action,
	}
	if denied != nil {
		metadata["reason"] = trustErrorCode(denied)
	}
	_ = s.auditor.Append(ctx, audit.Event{ActorUserID: actor.ID, Action: action, Metadata: metadata})
}

func RequiresExecutableTrust(extension Extension) bool {
	if extension.Source != SourceUploaded {
		return false
	}
	manifest := extension.Manifest
	return hasExecutableBackend(manifest) || strings.TrimSpace(extension.AdminFrontendDigest) != "" ||
		len(manifest.Migrations) > 0 || len(manifest.Guards) > 0 || hasL2Components(manifest) ||
		requestsRawRequest(manifest) || requestsRawCoreDatabase(manifest) || hasExecutableLifecycle(manifest)
}

func buildTrustImpact(extension Extension, action string) (TrustImpact, error) {
	if err := verifyInstalledPackageIdentity(extension); err != nil {
		return TrustImpact{}, err
	}
	manifest := extension.Manifest
	artifacts := map[string]string{"package": extension.PackageDigest}
	binaries := []TrustArtifact{}
	packageFileSet := map[string]struct{}{}
	addPackageFile := func(path string) {
		if path = strings.TrimSpace(path); path != "" {
			packageFileSet[path] = struct{}{}
		}
	}
	addExecutableArtifact := func(key, kind, path, declaredDigest string) error {
		digest, err := digestInstalledFile(extension, path)
		if err != nil {
			return err
		}
		declaredDigest = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(declaredDigest)), "sha256:")
		if declaredDigest != "" && declaredDigest != digest {
			return fmt.Errorf("%w: executable digest changed for %s", ErrFrontendPackageChanged, path)
		}
		artifacts[key] = digest
		binaries = append(binaries, TrustArtifact{Kind: kind, Path: path, Digest: digest})
		addPackageFile(path)
		return nil
	}
	if entry := strings.TrimSpace(manifest.Backend.Entry); entry != "" {
		if err := addExecutableArtifact("backend", "backend", entry, manifest.Backend.Digest); err != nil {
			return TrustImpact{}, err
		}
	}
	if digest := strings.TrimSpace(extension.AdminFrontendDigest); digest != "" {
		artifacts["adminFrontend"] = digest
	}
	migrations := make([]TrustMigration, 0, len(manifest.Migrations))
	for _, migration := range manifest.Migrations {
		digest, err := digestInstalledFile(extension, migration.Path)
		if err != nil {
			return TrustImpact{}, err
		}
		declaredDigest := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(migration.Digest)), "sha256:")
		if declaredDigest != "" && declaredDigest != digest {
			return TrustImpact{}, fmt.Errorf("%w: migration digest changed for %s", ErrFrontendPackageChanged, migration.Path)
		}
		artifacts["migration:"+migration.Path] = digest
		migrations = append(migrations, TrustMigration{Path: migration.Path, Digest: digest})
		addPackageFile(migration.Path)
	}
	for _, guard := range manifest.Guards {
		if err := addExecutableArtifact("guard:"+guard.ID, "guard", guard.Entry, guard.Digest); err != nil {
			return TrustImpact{}, err
		}
	}
	packageFilesByID := make(map[string]ManifestPackageFile, len(manifest.PackageFiles))
	for _, file := range manifest.PackageFiles {
		packageFilesByID[file.ID] = file
	}
	for _, component := range manifest.Components {
		if component.L2Component == "" {
			continue
		}
		file, ok := packageFilesByID[component.L2Component]
		if !ok {
			return TrustImpact{}, fmt.Errorf("%w: L2 package file %s is unavailable", ErrFrontendPackageChanged, component.L2Component)
		}
		if err := addExecutableArtifact("l2:"+file.ID, "l2", file.Path, file.Digest); err != nil {
			return TrustImpact{}, err
		}
	}
	guards := make([]TrustGuard, 0, len(manifest.Routes))
	for _, route := range manifest.Routes {
		guards = append(guards, TrustGuard{
			Path: route.Path, Methods: append([]string{}, route.Methods...),
			Access: route.Access, Permission: route.Permission,
		})
	}
	components := []SettingsComponent{}
	frontendAPI := ""
	if component := manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		components = append(components, *component)
		addPackageFile(component.Entry)
		if component.CSS != "" {
			addPackageFile(component.CSS)
		}
		frontendAPI = fmt.Sprintf("sforum.admin-component@%d", component.APIVersion)
	}
	capKeys, implied := extensionmanifest.ResolvedCapabilities(manifest)
	permissions := append([]string{}, manifest.Permissions...)
	requiredFeatures := append([]string{}, manifest.RequiresFeatures...)
	secrets := []string{}
	for _, field := range manifest.Settings {
		if field.Type == "secret" {
			secrets = append(secrets, field.Key)
		}
	}
	sort.Strings(permissions)
	sort.Strings(requiredFeatures)
	sort.Strings(secrets)
	packageFiles := make([]string, 0, len(packageFileSet))
	for path := range packageFileSet {
		packageFiles = append(packageFiles, path)
	}
	sort.Strings(packageFiles)
	hostAPI := strings.TrimSpace(manifest.Backend.HostAPIVersion)
	if hostAPI == "" {
		hostAPI = "sforum.host/v1"
	}
	impact := TrustImpact{
		SchemaVersion: TrustImpactSchemaV2, Action: action,
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		ExtensionType: extension.Type, Source: extension.Source, PackageDigest: extension.PackageDigest,
		ManifestContract: extensionmanifest.ManifestContract(manifest),
		ArtifactDigests:  artifacts, Binaries: binaries, Backend: manifest.Backend,
		Routes: append([]ManifestRoute{}, manifest.Routes...), Guards: guards,
		GuardDeclarations: append([]ManifestGuard{}, manifest.Guards...),
		Hooks:             append([]ManifestHook{}, manifest.Hooks...), Events: append([]ManifestEvent{}, manifest.Events...),
		Migrations: migrations, MigrationDeclarations: append([]ManifestMigration{}, manifest.Migrations...),
		Providers: append([]ManifestProvider{}, manifest.Providers...),
		Jobs:      append([]ManifestJob{}, manifest.Jobs...), Schedules: append([]ManifestSchedule{}, manifest.Schedules...),
		Components: components, RegistryComponents: append([]ManifestComponent{}, manifest.Components...),
		Templates: append([]ManifestTemplate{}, manifest.Templates...), Assets: append([]ManifestAsset{}, manifest.Assets...),
		Content: append([]ManifestContent{}, manifest.Content...), Database: manifest.Database,
		Cache: append([]ManifestCache{}, manifest.Cache...), Services: append([]ManifestService{}, manifest.Services...),
		Commands: append([]ManifestCommand{}, manifest.Commands...), AdminSurfaces: append([]ManifestAdminSurface{}, manifest.AdminSurfaces...),
		Queries: append([]ManifestQuery{}, manifest.Queries...), Identity: manifest.Identity,
		PermissionDefinitions: append([]ManifestPermissionDefinition{}, manifest.PermissionDefinitions...),
		Media:                 append([]ManifestMediaPipeline{}, manifest.Media...), Navigation: append([]ManifestNavigation{}, manifest.Navigation...),
		Regions:       append([]ManifestRegion{}, manifest.Regions...),
		Contributions: append([]ManifestContribution{}, manifest.Contributions...),
		Capabilities:  capabilities.GrantsFromKeys(capKeys, implied), Permissions: permissions,
		RequiredFeatures: requiredFeatures, Dependencies: append([]ManifestDependency{}, manifest.Dependencies...),
		Lifecycle: manifest.Lifecycle, OpenAPI: append([]ManifestOpenAPIFragment{}, manifest.OpenAPI...),
		PackageFiles: append([]ManifestPackageFile{}, manifest.PackageFiles...),
		RequestedAuthority: TrustAuthority{
			BackendExecution:       hasExecutableBackend(manifest),
			AdminFrontendExecution: strings.TrimSpace(extension.AdminFrontendDigest) != "",
			RawRequest:             requestsRawRequest(manifest),
			RawCoreDatabase:        requestsRawCoreDatabase(manifest),
			OutboundNetwork:        capabilities.NewSet(capKeys).Has(capabilities.NetOutbound),
			PackageFiles:           packageFiles, Secrets: secrets,
		},
		Contracts: TrustContracts{HostAPI: hostAPI, FrontendAPI: frontendAPI},
	}
	digest, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		return TrustImpact{}, err
	}
	impact.Digest = digest
	return impact, nil
}

func hasL2Components(manifest Manifest) bool {
	for _, component := range manifest.Components {
		if strings.TrimSpace(component.L2Component) != "" {
			return true
		}
	}
	return false
}

func requestsRawRequest(manifest Manifest) bool {
	for _, guard := range manifest.Guards {
		if guard.Kind == "raw_request" {
			return true
		}
	}
	for _, route := range manifest.Routes {
		if route.Guard == extensionmanifest.GuardCoreRaw {
			return true
		}
	}
	return false
}

func requestsRawCoreDatabase(manifest Manifest) bool {
	if manifest.Database == nil {
		return false
	}
	return manifest.Database.Authority == "raw_core" || manifest.Database.Authority == "kernel"
}

func hasExecutableLifecycle(manifest Manifest) bool {
	if manifest.Lifecycle == nil {
		return false
	}
	for _, operation := range []*extensionmanifest.ManifestLifecycleOperation{
		manifest.Lifecycle.Install, manifest.Lifecycle.Enable, manifest.Lifecycle.Disable,
		manifest.Lifecycle.Upgrade, manifest.Lifecycle.Rollback, manifest.Lifecycle.Uninstall,
	} {
		if operation != nil && strings.TrimSpace(operation.Execute) != "" {
			return true
		}
	}
	return false
}

// canonicalTrustImpactDigest 必须覆盖 TrustImpact 的每个字段；P2 新增声明时由测试防止遗漏。
func canonicalTrustImpactDigest(impact TrustImpact) (string, error) {
	impact.Digest = ""
	body, err := json.Marshal(impact)
	if err != nil {
		return "", fmt.Errorf("marshal extension trust impact: %w", err)
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func digestInstalledFile(extension Extension, relative string) (string, error) {
	path, ok := installedFilePath(extension, relative)
	if !ok {
		return "", fmt.Errorf("%w: unsafe executable path %s", ErrFrontendPackageChanged, relative)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%w: executable path %s is unavailable", ErrFrontendPackageChanged, relative)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func trustIdentity(impact TrustImpact) TrustIdentity {
	return TrustIdentity{
		ExtensionID: impact.ExtensionID, ExtensionVersion: impact.ExtensionVersion,
		PackageDigest: impact.PackageDigest, Action: impact.Action, ImpactDigest: impact.Digest,
	}
}

func tokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func trustErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrTrustChallengeExpired):
		return CodeTrustChallengeExpired
	case errors.Is(err, ErrTrustChallengeReplayed):
		return CodeTrustChallengeReplayed
	case errors.Is(err, ErrTrustChallengeStale):
		return CodeTrustChallengeStale
	default:
		return CodeTrustChallengeInvalid
	}
}
