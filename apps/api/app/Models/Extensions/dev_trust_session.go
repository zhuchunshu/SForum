package extensions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
	devhygiene "github.com/zhuchunshu/sforum/apps/api/app/Support/DevHygiene"
)

// DevTrustSession 是开发环境下 super_admin 一次性确认后授予的「会话级自动信任」：
// 之后同一 watch 循环内每个新 digest 由 GrantDevArtifact 自动写入 exact live grant。
// 仅 APP_ENV=development 且包位于 extensions/dev 时可用；其余情况一律 fail-closed
// 回到人工 trust challenge 路径。生产镜像没有 dev 包根，会话创建被 roots 校验拒绝。
type DevTrustSession struct {
	ID               string    `json:"id"`
	ExtensionID      string    `json:"extensionId"`
	PackageSlug      string    `json:"packageSlug"`
	RootPath         string    `json:"rootPath"`
	ActorUserID      int64     `json:"actorUserId"`
	GrantedAt        time.Time `json:"grantedAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
	LastPackageDigest string   `json:"lastPackageDigest,omitempty"`
	RevokedAt        *time.Time `json:"revokedAt,omitempty"`
	RevokedReason    string    `json:"revokedReason,omitempty"`
}

// DevTrustSessionListInput 列表过滤；ExtensionID 为空表示全部（含已撤销）。
type DevTrustSessionListInput struct {
	ExtensionID string
	IncludeRevoked bool
}

// DevTrustSessionGrantInput 创建会话：由携带确认 token 的 super_admin 请求发起。
type DevTrustSessionGrantInput struct {
	ExtensionID string
	PackageSlug string
	RootPath    string
	TTL         time.Duration
}

// DevTrustGrantInput 是 watch 循环每个新 digest 的自动信任写入请求。
type DevTrustGrantInput struct {
	ExtensionID   string
	PackageDigest string
	// ConfirmationToken 为空：自动路径依赖活跃会话而非一次性挑战。
}

var (
	ErrDevTrustDisabled        = errors.New("extensions: dev trust sessions require APP_ENV=development and a dev-root package")
	ErrDevTrustSessionNotFound = errors.New("extensions: dev trust session not found")
	ErrDevTrustSessionExists   = errors.New("extensions: an active dev trust session already exists for this package")
	ErrDevTrustSessionExpired  = errors.New("extensions: dev trust session expired")
	ErrDevTrustRootForbidden   = errors.New("extensions: dev trust root path is outside the allowed development roots")
)

const (
	CodeDevTrustDisabled        = "extension.dev_trust_disabled"
	CodeDevTrustSessionNotFound = "extension.dev_trust_session_not_found"
	CodeDevTrustSessionExists   = "extension.dev_trust_session_exists"
	CodeDevTrustRootForbidden   = "extension.dev_trust_root_forbidden"

	// DevTrustSessionDefaultTTL 是推荐默认值；配置可覆盖，UI 支持一键恢复默认。
	DevTrustSessionDefaultTTL = 8 * time.Hour
	// devTrustSessionMaxTTL 上限：会话是短命凭据，不允许配置成永久。
	devTrustSessionMaxTTL = 24 * time.Hour
)

// DevTrustSessionStore 持久化边界；生产实现为 Postgres（dev_trust_session_postgres.go）。
type DevTrustSessionStore interface {
	Create(ctx context.Context, session DevTrustSession) (DevTrustSession, error)
	List(ctx context.Context, input DevTrustSessionListInput) ([]DevTrustSession, error)
	Active(ctx context.Context, packageSlug string) (DevTrustSession, error)
	TouchDigest(ctx context.Context, sessionID, packageDigest string) error
	Revoke(ctx context.Context, sessionID string, actorUserID int64, reason string) error
	RevokeAllForExtension(ctx context.Context, extensionID string, actorUserID int64, reason string) error
	// RevokeAllActive 供 Safe Mode 启动时一次性吊销全部会话（开发装配专用）。
	RevokeAllActive(ctx context.Context, reason string) error
}

// DevTrustService 组合会话存储与 executable trust 写入路径。
// 独立小服务而不是塞回 Service/Manager God 对象（架构基线禁止增长）。
type DevTrustService struct {
	sessions DevTrustSessionStore
	trust    *ExecutableTrustService
	// devEnabled 与 watch loop / DevHygiene reaper 同一谓词。
	devEnabled bool
	// safeMode 打开时 Create/Grant 全部 fail-closed（服务层权威，UI 只是辅助）。
	safeMode bool
	now      func() time.Time
}

func NewDevTrustService(sessions DevTrustSessionStore, trust *ExecutableTrustService, appEnv string) *DevTrustService {
	return &DevTrustService{
		sessions:   sessions,
		trust:      trust,
		devEnabled: devhygiene.ShouldEnableDevelopmentOrphanReaper(appEnv),
		now:        time.Now,
	}
}

func (s *DevTrustService) WithClock(now func() time.Time) *DevTrustService {
	s.now = now
	return s
}

// WithSafeMode 注入 Host Safe Mode；打开时一切会话操作 fail-closed。
func (s *DevTrustService) WithSafeMode(enabled bool) *DevTrustService {
	s.safeMode = enabled
	return s
}

// RevokeAllOnSafeMode 在 Safe Mode 启动时一次性吊销全部会话，
// 确保 Safe Mode 结束后旧会话不会复活（逐包审计由调用方记录）。
func (s *DevTrustService) RevokeAllOnSafeMode(ctx context.Context) error {
	if s == nil || s.sessions == nil {
		return nil
	}
	return s.sessions.RevokeAllActive(ctx, "safe mode startup")
}

func newDevTrustSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate dev trust session id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// Create 在 super_admin 通过 dev trust challenge 后建立会话。
// root 必须位于允许的开发根内（extensions/dev），否则拒绝。
func (s *DevTrustService) Create(
	ctx context.Context,
	actor identity.Actor,
	input DevTrustSessionGrantInput,
) (DevTrustSession, error) {
	if err := s.requireDevTrustEligible(actor, input.RootPath); err != nil {
		return DevTrustSession{}, err
	}
	if _, err := s.trust.extension(ctx, input.ExtensionID); err != nil {
		return DevTrustSession{}, err
	}
	ttl := input.TTL
	if ttl <= 0 {
		ttl = DevTrustSessionDefaultTTL
	}
	if ttl > devTrustSessionMaxTTL {
		return DevTrustSession{}, ErrDevTrustDisabled
	}
	now := s.now()
	id, err := newDevTrustSessionID()
	if err != nil {
		return DevTrustSession{}, err
	}
	session := DevTrustSession{
		ID: id, ExtensionID: normalizeID(input.ExtensionID),
		PackageSlug: strings.TrimSpace(input.PackageSlug),
		RootPath:    strings.TrimSpace(input.RootPath),
		ActorUserID: actor.ID, GrantedAt: now, ExpiresAt: now.Add(ttl),
	}
	created, err := s.sessions.Create(ctx, session)
	if err != nil {
		return DevTrustSession{}, err
	}
	return created, nil
}

// List 暴露给 admin UI：活跃/历史会话可见，可一键撤销。
func (s *DevTrustService) List(ctx context.Context, actor identity.Actor, input DevTrustSessionListInput) ([]DevTrustSession, error) {
	if !canViewExtensions(actor) && !canManagePlugins(actor) {
		return nil, identity.ErrPermissionDenied
	}
	return s.sessions.List(ctx, input)
}

// Revoke 只允许创建者本人或 super_admin。
func (s *DevTrustService) Revoke(ctx context.Context, actor identity.Actor, sessionID string) error {
	sessions, err := s.sessions.List(ctx, DevTrustSessionListInput{IncludeRevoked: true})
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.ID != sessionID {
			continue
		}
		if !actor.IsSuperAdmin() && session.ActorUserID != actor.ID {
			return identity.ErrPermissionDenied
		}
		return s.sessions.Revoke(ctx, sessionID, actor.ID, "revoked by operator")
	}
	return ErrDevTrustSessionNotFound
}

// GrantDevArtifact 是 watch 循环每个新 digest 的自动信任写入：
// 活跃会话 + 同 actor + 未过期 + 未撤销时幂等写 exact live grant（审计标记 dev_session），
// 否则返回 ErrTrustChallengeRequired，与人工路径完全一致。
func (s *DevTrustService) GrantDevArtifact(
	ctx context.Context,
	actor identity.Actor,
	input DevTrustGrantInput,
) error {
	if !actor.IsSuperAdmin() {
		return identity.ErrPermissionDenied
	}
	if s == nil || s.sessions == nil {
		return ErrTrustChallengeRequired
	}
	extension, err := s.trust.extension(ctx, normalizeID(input.ExtensionID))
	if err != nil {
		return err
	}
	// Safe Mode 下一律回到人工路径（人工路径本身也会拒绝执行）。
	if s.safeMode || !s.devEnabled || extension.Source == SourceBuiltin {
		return ErrTrustChallengeRequired
	}
	if !RequiresExecutableTrust(extension) {
		return nil
	}
	slug := devTrustSessionKey(extension.ID, extension.PackagePath)
	session, err := s.sessions.Active(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrDevTrustSessionNotFound) {
			return ErrTrustChallengeRequired
		}
		return err
	}
	if session.ActorUserID != actor.ID || !s.now().Before(session.ExpiresAt) {
		return ErrTrustChallengeRequired
	}
	if extension.PackageDigest != strings.TrimSpace(input.PackageDigest) {
		// 摘要与当前落库制品不一致：staging 未完成或竞态，回到人工确认。
		return ErrTrustChallengeRequired
	}
	impact, err := buildTrustImpact(extension, TrustActionEnable)
	if err != nil {
		return err
	}
	granted, err := s.trust.store.HasLiveGrant(ctx, trustIdentity(impact))
	if err != nil {
		return err
	}
	if !granted {
		if _, err := s.trust.store.EnsureLiveGrant(ctx, TrustEnsureGrantInput{
			ActorUserID: actor.ID, Identity: trustIdentity(impact),
			ArtifactDigests: impact.ArtifactDigests, Impact: impact,
		}); err != nil {
			s.trust.appendAudit(ctx, actor, audit.ActionExtensionTrustDenied, impact, err)
			return err
		}
		s.trust.appendAudit(ctx, actor, audit.ActionExtensionTrustGrant, impact, nil)
	}
	if err := s.sessions.TouchDigest(ctx, session.ID, extension.PackageDigest); err != nil {
		return err
	}
	return nil
}

func (s *DevTrustService) requireDevTrustEligible(actor identity.Actor, rootPath string) error {
	if !actor.IsSuperAdmin() {
		return identity.ErrPermissionDenied
	}
	if s.safeMode {
		return ErrSafeModeActive
	}
	if !s.devEnabled {
		return ErrDevTrustDisabled
	}
	root := strings.TrimSpace(rootPath)
	if root == "" || !devTrustRootAllowed(root) {
		return ErrDevTrustRootForbidden
	}
	return nil
}

// devTrustSessionKey 会话唯一键：扩展 ID + 包根路径，避免同名包跨目录混淆。
func devTrustSessionKey(extensionID, packagePath string) string {
	return normalizeID(extensionID) + "@" + strings.TrimSpace(packagePath)
}

// devTrustRootAllowed 只接受位于仓库 extensions/dev 树下的绝对路径。
// 开发机（/home/…/SForum/extensions/dev/…）与容器（/app/extensions/dev/…）都合法。
func devTrustRootAllowed(root string) bool {
	normalized := strings.ReplaceAll(root, "\\", "/")
	if !strings.HasPrefix(normalized, "/") {
		return false
	}
	return strings.Contains(normalized+"/", "/extensions/dev/")
}
