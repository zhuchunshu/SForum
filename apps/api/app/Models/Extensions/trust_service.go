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
	return buildTrustImpact(extension, TrustActionEnable)
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
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		impact = TrustImpact{
			SchemaVersion: TrustImpactSchemaV1, Action: TrustActionEnable,
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			ExtensionType: extension.Type, Source: extension.Source,
			PackageDigest: extension.PackageDigest,
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
	return hasExecutableBackend(extension.Manifest) || strings.TrimSpace(extension.AdminFrontendDigest) != "" || len(extension.Manifest.Migrations) > 0
}

func buildTrustImpact(extension Extension, action string) (TrustImpact, error) {
	if err := verifyInstalledPackageIdentity(extension); err != nil {
		return TrustImpact{}, err
	}
	manifest := extension.Manifest
	artifacts := map[string]string{"package": extension.PackageDigest}
	binaries := []TrustArtifact{}
	packageFiles := []string{}
	if entry := strings.TrimSpace(manifest.Backend.Entry); entry != "" {
		digest, err := digestInstalledFile(extension, entry)
		if err != nil {
			return TrustImpact{}, err
		}
		artifacts["backend"] = digest
		binaries = append(binaries, TrustArtifact{Kind: "backend", Path: entry, Digest: digest})
		packageFiles = append(packageFiles, entry)
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
		artifacts["migration:"+migration.Path] = digest
		migrations = append(migrations, TrustMigration{Path: migration.Path, Digest: digest})
		packageFiles = append(packageFiles, migration.Path)
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
		packageFiles = append(packageFiles, component.Entry)
		if component.CSS != "" {
			packageFiles = append(packageFiles, component.CSS)
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
	sort.Strings(packageFiles)
	impact := TrustImpact{
		SchemaVersion: TrustImpactSchemaV1, Action: action,
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		ExtensionType: extension.Type, Source: extension.Source, PackageDigest: extension.PackageDigest,
		ArtifactDigests: artifacts, Binaries: binaries,
		Routes: append([]ManifestRoute{}, manifest.Routes...), Guards: guards,
		Hooks: append([]ManifestHook{}, manifest.Hooks...), Events: append([]ManifestEvent{}, manifest.Events...),
		Migrations: migrations, Providers: append([]ManifestProvider{}, manifest.Providers...),
		Jobs: append([]ManifestJob{}, manifest.Jobs...), Schedules: []string{}, Components: components,
		Contributions: append([]ManifestContribution{}, manifest.Contributions...),
		Capabilities:  capabilities.GrantsFromKeys(capKeys, implied), Permissions: permissions,
		RequiredFeatures: requiredFeatures, Dependencies: []Dependency{},
		RequestedAuthority: TrustAuthority{
			BackendExecution:       hasExecutableBackend(manifest),
			AdminFrontendExecution: strings.TrimSpace(extension.AdminFrontendDigest) != "",
			OutboundNetwork:        capabilities.NewSet(capKeys).Has(capabilities.NetOutbound),
			PackageFiles:           packageFiles, Secrets: secrets,
		},
		Contracts: TrustContracts{HostAPI: "sforum.host/v1", FrontendAPI: frontendAPI},
	}
	digest, err := canonicalTrustImpactDigest(impact)
	if err != nil {
		return TrustImpact{}, err
	}
	impact.Digest = digest
	return impact, nil
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
