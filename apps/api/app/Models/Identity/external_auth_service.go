package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

// ExternalAuthService 是 Host 拥有的外部认证编排层。
//
// 职责边界（见 plans/2026-07-27-github-social-login-builtin-plugin.md）：
//   - Core 独占：用户创建/修改、默认角色/首用户规则、external link 持久化、
//     账号状态/权限检查、风险/会话策略评估、会话签发/续期/吊销、
//     返回路径校验、限流、审计、公开错误映射；
//   - 插件独占：外部 OAuth 行为、瞬时 vendor token 使用；
//   - 插件永不接收 password/password hash/raw cookie/raw session id/CSRF token/
//     PAT plaintext/role assignment/permission grant。
//
// 该服务只在 Host 进程内调用；raw subject 仅在 ConsumeAssertion 的输入参数里
// 短暂存在，立即被 ComputeSubjectDigest 转为 digest，之后不再保留。

const (
	// RecentAuthDefaultTTL 最近认证窗口（5 分钟）。
	RecentAuthDefaultTTL = 5 * time.Minute
)

// ExternalAuthAssertion 是 callback 处理后从插件 complete 得到的断言。
// ProviderSubject 是 raw 外部 subject，仅在此结构内传递，不出现在任何公开响应。
// SubjectDigest 兼容旧 fixture（membership-reference）路径：当 ProviderSubject 为空时使用。
// Owner* / ProviderContractVersion 绑定 ticket 签发时的 exact artifact，提交时必须与 live Registry 一致。
type ExternalAuthAssertion struct {
	ProviderID              string
	ProviderContractVersion string
	OwnerExtensionID        string
	OwnerExtensionVersion   string // 存在时与 live ExtensionVersion 精确比对
	OwnerPackageDigest      string
	Operation               ExternalAuthOperation
	ProviderSubject         string // raw subject，仅 Host 内部（Core-HMAC 模式）
	SubjectDigest           string // 兼容旧 fixture 路径的 plugin-computed digest
	DisplayName             string
	EmailHint               string
	CorrelationID           string
}

// MatchesLiveContribution 将断言/票据中的 provider、owner extension、package digest、
// provider contract 以及存在时的版本信息与 live Registry exact contribution 比较。
// disable / artifact upgrade / trust revoke / uninstall / Safe Mode 导致不可解析或漂移时 fail closed。
func (a ExternalAuthAssertion) MatchesLiveContribution(live identityregistry.ProviderContribution) bool {
	if strings.TrimSpace(a.ProviderID) == "" ||
		strings.TrimSpace(a.OwnerExtensionID) == "" ||
		strings.TrimSpace(a.OwnerPackageDigest) == "" {
		return false
	}
	if live.ID == "" || live.Artifact.Core || strings.TrimSpace(live.Artifact.RuntimeInstanceID) == "" {
		return false
	}
	if a.ProviderID != live.ID {
		return false
	}
	if a.OwnerExtensionID != live.Artifact.ExtensionID {
		return false
	}
	if a.OwnerPackageDigest != live.Artifact.PackageDigest {
		return false
	}
	// 版本信息存在时必须精确一致（ticket/assertion 可能携带）。
	if a.OwnerExtensionVersion != "" && a.OwnerExtensionVersion != live.Artifact.ExtensionVersion {
		return false
	}
	if a.ProviderContractVersion != "" && a.ProviderContractVersion != live.ContractVersion {
		return false
	}
	return true
}

// resolvedDigest 返回用于链接表的 digest：优先 Core-HMAC，否则用兼容 digest。
func (a ExternalAuthAssertion) resolvedDigest() (string, error) {
	if strings.TrimSpace(a.ProviderSubject) != "" {
		return ComputeSubjectDigest(a.ProviderID, a.ProviderSubject)
	}
	if strings.TrimSpace(a.SubjectDigest) != "" {
		return a.SubjectDigest, nil
	}
	return "", ErrExternalAuthOperationMismatch
}

// ExternalAuthLoginResult 是外部登录成功后的结果。
// SessionHash 仅为审计关联用的盐哈希，不返回浏览器；浏览器只看 CurrentUser。
type ExternalAuthLoginResult struct {
	User        CurrentUser
	SessionHash string
	NewlyLinked bool // 未知 link 不会发生：登录必须先有 link
	ProviderID  string
}

// ExternalAuthRegistrationResult 是外部注册（用户+默认角色+link 原子事务）结果。
type ExternalAuthRegistrationResult struct {
	User       CurrentUser
	ProviderID string
	LinkID     int64
}

// ExternalAuthLinkResult 是 link.complete 成功后的结果。
type ExternalAuthLinkResult struct {
	User       CurrentUser
	ProviderID string
	LinkID     int64
}

// UnlinkExternalIdentityInput 是 Host 解绑输入。
// IdempotencyKey 必须绑定 user/link/revision/request，不得使用 client IP。
type UnlinkExternalIdentityInput struct {
	UserID             int64
	LinkID             int64
	ExpectedRevision   int64
	SessionFingerprint string
	// RequestID 是本请求幂等后缀（例如 opaque request id 或 correlation）。
	// 与 user/link/revision 一并构成 idempotency key。
	RequestID string
}

// RecentAuthChecker 判断指定会话是否在 recent-auth 窗口内（敏感 link/unlink 前置）。
// 生产实现必须绑定 session fingerprint，不得仅按 user_id。
type RecentAuthChecker interface {
	IsSessionRecentlyAuthenticated(ctx context.Context, userID int64, sessionFingerprint string) (bool, error)
}

// RecentAuthMarker 在成功密码/外部认证后标记当前会话。
type RecentAuthMarker interface {
	MarkSessionRecentlyAuthenticated(ctx context.Context, userID int64, sessionFingerprint, method, providerID string, ttl time.Duration) error
}

// ExternalRegistrationValidator 复用权威注册字段/策略校验（无密码、无更弱副本）。
type ExternalRegistrationValidator interface {
	ValidateExternalRegister(ctx context.Context, input ExternalRegistrationInput) error
}

// ExternalAuthDeps 收集 ExternalAuthService 的依赖。
type ExternalAuthDeps struct {
	Pool                 *pgxpool.Pool
	LinkStore            ExternalIdentityLinkStore
	ActivationStore      ProviderActivationStore
	ExternalAuthStore    *PostgresExternalAuthStore
	RecentAuth           RecentAuthChecker
	RecentAuthMarker     RecentAuthMarker
	AuthProviderSource   AuthProviderSource
	AuthProviderInvoker  AuthProviderInvoker
	ProviderContribution func(providerID string) (identityregistry.ProviderContribution, error)
	// SafeMode 为 true 时第三方 auth 有效可用性一律关闭（Host 状态派生）。
	SafeMode func() bool
	// IsProviderConfigured 可选：settings 接线后用于“必需配置”门控；nil 时不阻断。
	IsProviderConfigured func(ctx context.Context, providerID string) (bool, error)
	RegistrationEnabled  func(ctx context.Context) (bool, error)
	// RegistrationEnabledTx 必须通过与外部注册写入相同的 pgx.Tx 读取权威运营策略，
	// 不得调用独立 pool 或进程缓存。
	RegistrationEnabledTx func(ctx context.Context, tx pgx.Tx) (bool, error)
	AnyUserExists         func(ctx context.Context) (bool, error)
	// LoadCurrentUser 走权威 CurrentUser 路径（token version/roles/permissions/avatar/status）。
	LoadCurrentUser func(ctx context.Context, userID int64) (CurrentUser, error)
	// ValidateRegistration 权威外部注册字段/策略校验；nil 时仅做最小非空检查（测试兼容）。
	ValidateRegistration ExternalRegistrationValidator
	// Events 可选；成功提交后发出 user.registered observe（与密码注册同一语义）。
	Events appevents.Publisher
}

// ExternalAuthService 编排外部登录/注册/link。
type ExternalAuthService struct {
	deps ExternalAuthDeps
}

// NewExternalAuthService 构造服务。
func NewExternalAuthService(deps ExternalAuthDeps) *ExternalAuthService {
	return &ExternalAuthService{deps: deps}
}

// WithCurrentUserLoader 注入权威 CurrentUser 加载器。
func (s *ExternalAuthService) WithCurrentUserLoader(fn func(context.Context, int64) (CurrentUser, error)) *ExternalAuthService {
	if s != nil {
		s.deps.LoadCurrentUser = fn
	}
	return s
}

// WithRegistrationValidator 注入权威外部注册校验器。
func (s *ExternalAuthService) WithRegistrationValidator(v ExternalRegistrationValidator) *ExternalAuthService {
	if s != nil {
		s.deps.ValidateRegistration = v
	}
	return s
}

// WithRegistrationPolicyTx 注入 CompleteRegistration 使用的权威 Options
// 事务读取器。事务外快速拒绝仍可保留，但账号创建必须依赖该读取器。
func (s *ExternalAuthService) WithRegistrationPolicyTx(fn func(context.Context, pgx.Tx) (bool, error)) *ExternalAuthService {
	if s != nil {
		s.deps.RegistrationEnabledTx = fn
	}
	return s
}

// ActivationStore 暴露激活目录给控制器（公开 catalog 过滤、admin 聚合）。
func (s *ExternalAuthService) ActivationStore() ProviderActivationStore {
	if s == nil {
		return nil
	}
	return s.deps.ActivationStore
}

// WithProviderConfiguredChecker 注入必需配置门控（settings 接线后）。
// nil 时 IsProviderConfigured 返回 false（admin 聚合不得臆测已配置）。
func (s *ExternalAuthService) WithProviderConfiguredChecker(fn func(ctx context.Context, providerID string) (bool, error)) *ExternalAuthService {
	if s != nil {
		s.deps.IsProviderConfigured = fn
	}
	return s
}

// WithEvents 注入 observe 事件发布器（user.registered 等）。
func (s *ExternalAuthService) WithEvents(publisher appevents.Publisher) *ExternalAuthService {
	if s != nil {
		s.deps.Events = appevents.EnsurePublisher(publisher)
	}
	return s
}

// IsProviderConfigured 返回 provider 必需配置是否齐全。
// 未接线时返回 (false, nil)：公开可用性不因“未知配置”而放行，admin 展示 configured=false。
func (s *ExternalAuthService) IsProviderConfigured(ctx context.Context, providerID string) (bool, error) {
	if s == nil || s.deps.IsProviderConfigured == nil {
		return false, nil
	}
	return s.deps.IsProviderConfigured(ctx, providerID)
}

// IsOperationActivated 校验 provider + operation 是否有效可用。
// T1C 起这是完整有效可用性（activation flag + live artifact + Safe Mode + ops），
// 不再只看激活目录布尔位。默认全 off。
func (s *ExternalAuthService) IsOperationActivated(ctx context.Context, providerID string, op ExternalAuthOperation) (bool, error) {
	return s.IsEffectivelyAvailable(ctx, providerID, op)
}

// RequireActivated 校验有效可用，否则返回稳定错误。
// Host 状态查询失败向上返回，供 HTTP 层 fail closed。
func (s *ExternalAuthService) RequireActivated(ctx context.Context, providerID string, op ExternalAuthOperation) error {
	av, err := s.EvaluateOperationAvailability(ctx, providerID, op)
	if err != nil {
		return err
	}
	if av.Available {
		return nil
	}
	switch av.Reason {
	case "auth.provider_not_enabled":
		return ErrExternalAuthOperationNotActivated
	case "auth.provider_not_found":
		return ErrAuthProviderNotFound
	default:
		return ErrExternalAuthProviderUnavailable
	}
}

// ValidateCallbackBeforeEffect 在 callback 消费 state 之后、任何业务效应之前：
//  1. 重新解析 live Registry provider；
//  2. 比对 provider/operation/owner extension/version/package digest；
//  3. 重新校验 Host 有效激活。
//
// 失败时调用方不得调用插件 complete，更不得写 link/用户/会话。
func (s *ExternalAuthService) ValidateCallbackBeforeEffect(
	ctx context.Context,
	tx CallbackTransaction,
	routeProviderID string,
) (identityregistry.ProviderContribution, error) {
	routeProviderID = strings.ToLower(strings.TrimSpace(routeProviderID))
	if routeProviderID == "" || tx.ProviderID == "" {
		return identityregistry.ProviderContribution{}, ErrCallbackStateInvalid
	}
	if tx.ProviderID != routeProviderID {
		return identityregistry.ProviderContribution{}, ErrCallbackStateInvalid
	}
	live, err := s.providerContribution(tx.ProviderID)
	if err != nil {
		// 禁用/卸载/Registry 不可见 → fail closed，与无效 callback 同形。
		return identityregistry.ProviderContribution{}, ErrExternalAuthProviderUnavailable
	}
	if live.Artifact.Core || strings.TrimSpace(live.Artifact.RuntimeInstanceID) == "" {
		return identityregistry.ProviderContribution{}, ErrExternalAuthProviderUnavailable
	}
	if !tx.MatchesLiveArtifact(live, tx.Operation) {
		return identityregistry.ProviderContribution{}, ErrExternalAuthArtifactMismatch
	}
	// 操作名必须仍被 live provider 声明。
	if !authProviderHasOperation(live, externalOpToCompleteName(tx.Operation)) {
		return identityregistry.ProviderContribution{}, ErrExternalAuthProviderUnavailable
	}
	if err := s.RequireActivated(ctx, tx.ProviderID, tx.Operation); err != nil {
		return identityregistry.ProviderContribution{}, err
	}
	return live, nil
}

// ValidateLoginEffect fences a completed external login before a Host session
// effect. The successful check immediately before session persistence is the
// login-effect linearization point: it observes the current Registry,
// activation and Safe Mode state, but cannot make those independent stores
// globally atomic with the session store.
//
// Callers invoke it once after provider complete returns and again inside the
// Host session-effect admission callback. Keeping the matching rules here
// prevents controllers from drifting from the Registry contract.
func (s *ExternalAuthService) ValidateLoginEffect(ctx context.Context, assertion ExternalAuthAssertion) error {
	if assertion.Operation != ExternalAuthOperationLogin {
		return ErrExternalAuthOperationMismatch
	}
	live, err := s.providerContribution(assertion.ProviderID)
	if err != nil {
		return ErrExternalAuthProviderUnavailable
	}
	if !assertion.MatchesLiveContribution(live) ||
		!authProviderHasOperation(live, AuthOperationLoginComplete) {
		return ErrExternalAuthArtifactMismatch
	}
	if err := s.RequireActivated(ctx, assertion.ProviderID, ExternalAuthOperationLogin); err != nil {
		return err
	}
	return nil
}

// AuthorizeLinkBeforePersist 在 link 持久化之前校验：
// 当前会话 actor 与事务绑定一致，且具备 session-bound recent-auth/step-up。
// 任一失败必须零写入。
func (s *ExternalAuthService) AuthorizeLinkBeforePersist(
	ctx context.Context,
	tx CallbackTransaction,
	sessionUserID int64,
	sessionFingerprint string,
) error {
	if tx.Operation != ExternalAuthOperationLink {
		return ErrExternalAuthOperationMismatch
	}
	if sessionUserID <= 0 {
		return ErrExternalAuthActorRequired
	}
	if !tx.MatchesActor(ExternalAuthOperationLink, sessionUserID) {
		return ErrExternalAuthActorMismatch
	}
	recent, err := s.isSessionRecentlyAuthenticated(ctx, sessionUserID, sessionFingerprint)
	if err != nil {
		return err
	}
	if !recent {
		return ErrExternalAuthRecentAuthRequired
	}
	return nil
}

// MarkSessionAuthenticated 在成功密码或外部认证后标记当前会话的 recent-auth。
func (s *ExternalAuthService) MarkSessionAuthenticated(
	ctx context.Context,
	userID int64,
	sessionFingerprint, method, providerID string,
) error {
	if userID <= 0 || !validSessionFingerprint(strings.ToLower(strings.TrimSpace(sessionFingerprint))) {
		return nil // best-effort：无 session 上下文时不写（不阻断登录）
	}
	if s.deps.RecentAuthMarker != nil {
		return s.deps.RecentAuthMarker.MarkSessionRecentlyAuthenticated(
			ctx, userID, sessionFingerprint, method, providerID, RecentAuthDefaultTTL,
		)
	}
	if s.deps.ExternalAuthStore != nil {
		return s.deps.ExternalAuthStore.MarkSessionRecentlyAuthenticated(
			ctx, userID, sessionFingerprint, method, providerID, RecentAuthDefaultTTL,
		)
	}
	return nil
}

func (s *ExternalAuthService) isSessionRecentlyAuthenticated(
	ctx context.Context,
	userID int64,
	sessionFingerprint string,
) (bool, error) {
	sessionFingerprint = strings.ToLower(strings.TrimSpace(sessionFingerprint))
	if !validSessionFingerprint(sessionFingerprint) {
		return false, nil
	}
	if s.deps.RecentAuth != nil {
		return s.deps.RecentAuth.IsSessionRecentlyAuthenticated(ctx, userID, sessionFingerprint)
	}
	if s.deps.ExternalAuthStore != nil {
		return s.deps.ExternalAuthStore.IsSessionRecentlyAuthenticated(ctx, userID, sessionFingerprint)
	}
	// 无 checker 时 fail closed：敏感操作不能在未知 recent-auth 状态下写入。
	return false, nil
}

func externalOpToCompleteName(op ExternalAuthOperation) string {
	switch op {
	case ExternalAuthOperationLogin:
		return AuthOperationLoginComplete
	case ExternalAuthOperationRegistration:
		return AuthOperationRegistrationComplete
	case ExternalAuthOperationLink:
		return AuthOperationLinkComplete
	default:
		return ""
	}
}

// CompleteLogin 从一次成功的 login.complete 断言建立会话前置状态。
// 步骤：
//  1. 计算 digest；
//  2. FindActive 找到已绑定用户（未知 link → 返回 ErrExternalIdentityUnlinked，
//     不创建账号、不暴露存在性）；
//  3. 校验账号状态 active、未 banned/disabled；
//  4. 通过权威 CurrentUser 路径重载完整会话 claims。
//
// Controller 负责 risk/session/audit；本方法只做账号解析。
func (s *ExternalAuthService) CompleteLogin(ctx context.Context, assertion ExternalAuthAssertion) (ExternalAuthLoginResult, error) {
	if assertion.Operation != ExternalAuthOperationLogin {
		return ExternalAuthLoginResult{}, ErrExternalAuthOperationMismatch
	}
	digest, err := assertion.resolvedDigest()
	if err != nil {
		return ExternalAuthLoginResult{}, err
	}
	link, err := s.deps.LinkStore.FindActive(ctx, assertion.ProviderID, digest)
	if err != nil {
		if errors.Is(err, ErrExternalIdentityLinkNotFound) {
			// 未绑定：泛化错误，不暴露存在性。
			return ExternalAuthLoginResult{}, ErrExternalIdentityUnlinked
		}
		return ExternalAuthLoginResult{}, err
	}
	current, err := s.loadCurrentUser(ctx, link.UserID)
	if err != nil {
		return ExternalAuthLoginResult{}, err
	}
	if current.Status != UserStatusActive {
		// 禁用/封禁：泛化错误。
		return ExternalAuthLoginResult{}, ErrExternalIdentityUnlinked
	}
	return ExternalAuthLoginResult{
		User:        current,
		ProviderID:  assertion.ProviderID,
		NewlyLinked: false,
	}, nil
}

// CompleteRegistration 原子地创建用户 + 默认角色 + external link。
// 不创建密码凭据（外部账号）。首用户 bootstrap 规则仍生效：零用户站点
// 禁止外部注册（首用户必须用 Core 密码 bootstrap）。
//
// T8A 授权/事务边界：
//  1. 任何账号创建效果前重新校验 registration 操作的 Host 有效激活；
//  2. 将 ticket/assertion 的 provider/owner/digest/contract/version 与 live Registry 精确比对；
//  3. 在创建 user/default role/external link 的同一事务内重新读取权威 registration policy；
//  4. 成功提交后恰好一次发出 user.registered observe；
//  5. 注册 mutation audit 写入同一事务（复用 audit_events，不另建审计子系统）。
//
// 调用方应在消费票据前完成可编辑字段校验；本方法在事务内再次检查权威策略。
func (s *ExternalAuthService) CompleteRegistration(ctx context.Context, assertion ExternalAuthAssertion, input ExternalRegistrationInput) (ExternalAuthRegistrationResult, error) {
	if assertion.Operation != ExternalAuthOperationRegistration {
		return ExternalAuthRegistrationResult{}, ErrExternalAuthOperationMismatch
	}
	input = input.Normalized()
	// 事务外权威字段/策略校验（调用方通常已做过；此处纵深）。
	if err := s.validateExternalRegistrationInput(ctx, input); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}

	// 1. 任何账号效果前：registration 操作必须仍有效激活（含 Safe Mode / disable / artifact）。
	if err := s.RequireActivated(ctx, assertion.ProviderID, ExternalAuthOperationRegistration); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}

	// 2. live Registry exact contribution 与 ticket/assertion 绑定比对。
	contribution, err := s.providerContribution(assertion.ProviderID)
	if err != nil {
		// 卸载 / trust revoke / Safe Mode / 不可见 → fail closed。
		return ExternalAuthRegistrationResult{}, ErrExternalAuthProviderUnavailable
	}
	if !assertion.MatchesLiveContribution(contribution) {
		return ExternalAuthRegistrationResult{}, ErrExternalAuthArtifactMismatch
	}
	if !authProviderHasOperation(contribution, AuthOperationRegistrationComplete) {
		return ExternalAuthRegistrationResult{}, ErrExternalAuthProviderUnavailable
	}

	digest, err := assertion.resolvedDigest()
	if err != nil {
		return ExternalAuthRegistrationResult{}, err
	}

	// 事务外快速拒绝（权威校验仍在事务内）。
	if err := s.ensureExternalRegistrationPolicy(ctx, false); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}

	postgresLink, ok := s.deps.LinkStore.(*PostgresExternalIdentityLinkStore)
	if !ok || s.deps.Pool == nil {
		return ExternalAuthRegistrationResult{}, ErrExternalIdentityLinkStoreUnavailable
	}

	// 3. 事务：policy 再读 + user + 默认角色 + link + registration audit 原子。
	tx, err := s.deps.Pool.Begin(ctx)
	if err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	defer tx.Rollback(ctx)

	if err := s.ensureExternalRegistrationPolicyTx(ctx, tx); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	// 冲突校验（username/email）—— 非枚举：字段级 taken，与 Core 注册一致。
	if err := registrationConflictsTx(ctx, tx, input.Username, input.Email); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	current, err := createUserWithoutCredentialTx(ctx, tx, CreateUserInput{
		Username:    input.Username,
		Email:       input.Email,
		DisplayName: input.DisplayName,
		Locale:      input.Locale,
	})
	if err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	// 默认角色必须恰好影响 1 行，否则回滚 user/link/audit。
	if err := assignDefaultRoleTx(ctx, tx, current.ID); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}

	linkInput := LinkExternalIdentityInput{
		UserID:                current.ID,
		Provider:              contribution,
		ProviderOperation:     AuthOperationRegistrationComplete,
		ProviderSubjectDigest: digest,
		ActorUserID:           0,
		IdempotencyKey:        "registration:" + assertion.CorrelationID,
	}
	mutation, err := postgresLink.LinkTx(ctx, tx, linkInput)
	if err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	// 注册 mutation audit 与 user/role/link 同事务；后续 session audit 不能替代。
	if _, err := insertExternalRegistrationAuditTx(ctx, tx, externalRegistrationAuditInput{
		UserID:           current.ID,
		ProviderID:       contribution.ID,
		OwnerExtensionID: contribution.Artifact.ExtensionID,
		CorrelationID:    assertion.CorrelationID,
	}); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExternalAuthRegistrationResult{}, err
	}

	// 4. 成功提交后恰好一次发出 user.registered（与密码注册同一 observe 语义）。
	s.emitUserRegistered(ctx, current, input.Email)

	// 通过权威 CurrentUser 路径重载完整 claims 后再签发会话。
	current, err = s.loadCurrentUser(ctx, current.ID)
	if err != nil {
		return ExternalAuthRegistrationResult{}, err
	}
	return ExternalAuthRegistrationResult{
		User:       current,
		ProviderID: assertion.ProviderID,
		LinkID:     mutation.Link.ID,
	}, nil
}

// ensureExternalRegistrationPolicy 事务外快速策略检查。
// insideTx=false 时使用 pool/依赖；权威结果仍以 ensureExternalRegistrationPolicyTx 为准。
func (s *ExternalAuthService) ensureExternalRegistrationPolicy(ctx context.Context, _ bool) error {
	hasAny, err := s.anyUser(ctx)
	if err != nil {
		return err
	}
	if !hasAny {
		return ErrExternalAuthBootstrapRequired
	}
	enabled, err := s.registrationEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrRegistrationDisabled
	}
	return nil
}

// ensureExternalRegistrationPolicyTx 在创建 user 的同一事务内重新读取权威策略。
// 用户计数来自事务快照；开放注册意图再读运营配置（与密码注册 WithBootstrapTx 一致）。
func (s *ExternalAuthService) ensureExternalRegistrationPolicyTx(ctx context.Context, tx pgx.Tx) error {
	var txCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&txCount); err != nil {
		return err
	}
	if txCount == 0 {
		return ErrExternalAuthBootstrapRequired
	}
	// 运营策略必须通过同一个 pgx.Tx 读取；独立 pool/cache 会把“同一事务”
	// 的声明变成谎言。生产 Options 实现用 advisory xact lock 与更新串行化。
	if s.deps.RegistrationEnabledTx == nil {
		return fmt.Errorf("transactional registration policy is unavailable")
	}
	enabled, err := s.deps.RegistrationEnabledTx(ctx, tx)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrRegistrationDisabled
	}
	return nil
}

func (s *ExternalAuthService) emitUserRegistered(ctx context.Context, current CurrentUser, email string) {
	publisher := appevents.EnsurePublisher(s.deps.Events)
	publisher.Emit(ctx, appevents.Envelope{
		Name:          appevents.UserRegistered,
		Kind:          appevents.KindObserve,
		ActorUserID:   current.ID,
		ResourceType:  "user",
		ResourceID:    strconv.FormatInt(current.ID, 10),
		CorrelationID: appevents.NewID(),
		Payload: map[string]any{
			"userId":   current.ID,
			"username": current.Username,
			"email":    email,
			"locale":   current.Locale,
		},
		OccurredAt: time.Now().UTC(),
	})
}

// externalRegistrationAuditInput 是注册提交边界的 mutation audit（无 raw subject/token）。
type externalRegistrationAuditInput struct {
	UserID           int64
	ProviderID       string
	OwnerExtensionID string
	CorrelationID    string
}

// insertExternalRegistrationAuditTx 复用 audit_events 事务写入路径。
// action = auth.external_register.success；metadata 仅含 provider/owner 公开绑定字段。
func insertExternalRegistrationAuditTx(ctx context.Context, tx pgx.Tx, input externalRegistrationAuditInput) (int64, error) {
	if input.UserID <= 0 || strings.TrimSpace(input.ProviderID) == "" {
		return 0, fmt.Errorf("external registration audit input invalid")
	}
	metadata, err := json.Marshal(map[string]any{
		"providerId":       input.ProviderID,
		"ownerExtensionId": input.OwnerExtensionID,
		// correlation 仅作请求关联；不得含 subject/digest/token/state/verifier。
		"correlationId": input.CorrelationID,
	})
	if err != nil {
		return 0, fmt.Errorf("encode external registration audit: %w", err)
	}
	var auditID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO audit_events (actor_user_id, target_user_id, action, metadata)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id
	`, input.UserID, input.UserID, AuditActionExternalRegister, metadata).Scan(&auditID); err != nil {
		return 0, fmt.Errorf("record external registration audit: %w", err)
	}
	return auditID, nil
}

// CompleteLink 把一次 link.complete 断言绑定到已登录用户。
//
// 调用方必须先通过 AuthorizeLinkBeforePersist + ValidateCallbackBeforeEffect；
// 本方法在写库前再次校验 actor、session-bound recent-auth、activation 与 live artifact。
func (s *ExternalAuthService) CompleteLink(
	ctx context.Context,
	assertion ExternalAuthAssertion,
	actorUserID int64,
	sessionFingerprint string,
) (ExternalAuthLinkResult, error) {
	if assertion.Operation != ExternalAuthOperationLink {
		return ExternalAuthLinkResult{}, ErrExternalAuthOperationMismatch
	}
	if actorUserID <= 0 {
		return ExternalAuthLinkResult{}, ErrExternalAuthActorRequired
	}
	// 纵深：写库前再次确认激活与 session-bound recent-auth。
	if err := s.RequireActivated(ctx, assertion.ProviderID, ExternalAuthOperationLink); err != nil {
		return ExternalAuthLinkResult{}, err
	}
	recent, err := s.isSessionRecentlyAuthenticated(ctx, actorUserID, sessionFingerprint)
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	if !recent {
		return ExternalAuthLinkResult{}, ErrExternalAuthRecentAuthRequired
	}
	digest, err := assertion.resolvedDigest()
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	// 检查 subject 是否已被绑定。
	existing, err := s.deps.LinkStore.FindActive(ctx, assertion.ProviderID, digest)
	if err == nil && existing.UserID != actorUserID {
		return ExternalAuthLinkResult{}, ErrExternalIdentitySubjectConflict
	}
	if err != nil && !errors.Is(err, ErrExternalIdentityLinkNotFound) {
		return ExternalAuthLinkResult{}, err
	}
	contribution, err := s.providerContribution(assertion.ProviderID)
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	// live artifact 与断言中的 owner 绑定必须一致（callback 后再次确认）。
	if assertion.OwnerPackageDigest != "" && assertion.OwnerPackageDigest != contribution.Artifact.PackageDigest {
		return ExternalAuthLinkResult{}, ErrExternalAuthArtifactMismatch
	}
	if assertion.OwnerExtensionID != "" && assertion.OwnerExtensionID != contribution.Artifact.ExtensionID {
		return ExternalAuthLinkResult{}, ErrExternalAuthArtifactMismatch
	}
	current, err := s.loadCurrentUser(ctx, actorUserID)
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	if current.Status != UserStatusActive {
		return ExternalAuthLinkResult{}, ErrExternalAuthActorInactive
	}
	mutation, err := s.deps.LinkStore.Link(ctx, LinkExternalIdentityInput{
		UserID:                actorUserID,
		Provider:              contribution,
		ProviderOperation:     AuthOperationLinkComplete,
		ProviderSubjectDigest: digest,
		ActorUserID:           actorUserID,
		IdempotencyKey:        "link:" + assertion.CorrelationID,
	}, func() error { return nil })
	if err != nil {
		return ExternalAuthLinkResult{}, err
	}
	return ExternalAuthLinkResult{
		User:       current,
		ProviderID: assertion.ProviderID,
		LinkID:     mutation.Link.ID,
	}, nil
}

// Unlink 在同一事务内：加载目标 link、校验所有权/active/revision、
// last-login-method 保护、执行 unlink。idempotency key 绑定 user/link/revision/request。
func (s *ExternalAuthService) Unlink(ctx context.Context, input UnlinkExternalIdentityInput) (ExternalIdentityLinkMutation, error) {
	if input.UserID <= 0 || input.LinkID <= 0 || input.ExpectedRevision <= 0 {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}
	recent, err := s.isSessionRecentlyAuthenticated(ctx, input.UserID, input.SessionFingerprint)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if !recent {
		return ExternalIdentityLinkMutation{}, ErrExternalAuthRecentAuthRequired
	}

	// 事务外快速加载：所有权/状态/revision 预检（事务内再锁）。
	link, err := s.deps.LinkStore.Get(ctx, input.LinkID)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if link.UserID != input.UserID || link.Status != ExternalIdentityLinkStatusActive {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkNotFound
	}
	if link.Revision != input.ExpectedRevision {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkStateConflict
	}

	idempotencyKey := unlinkIdempotencyKey(input.UserID, input.LinkID, input.ExpectedRevision, input.RequestID)
	if !validExternalIdentityIdempotencyKey(idempotencyKey) {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkInvalid
	}

	postgresLink, ok := s.deps.LinkStore.(*PostgresExternalIdentityLinkStore)
	if !ok || s.deps.Pool == nil {
		// 非 Postgres 测试路径：仍做 last-method 检查后调用 Unlink。
		if err := s.CanUnlink(ctx, input.UserID, input.LinkID); err != nil {
			return ExternalIdentityLinkMutation{}, err
		}
		return s.deps.LinkStore.Unlink(ctx, TransitionExternalIdentityLinkInput{
			LinkID:           input.LinkID,
			ExpectedRevision: input.ExpectedRevision,
			ActorUserID:      input.UserID,
			IdempotencyKey:   idempotencyKey,
		})
	}

	tx, err := s.deps.Pool.Begin(ctx)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	defer tx.Rollback(ctx)

	// 事务内：FOR UPDATE 加载 + 所有权/状态/revision + last-method + unlink。
	locked, err := scanExternalIdentityLink(tx.QueryRow(ctx, externalIdentityLinkSelect+`
		WHERE id = $1 FOR UPDATE
	`, input.LinkID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkNotFound
	}
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if locked.UserID != input.UserID || locked.Status != ExternalIdentityLinkStatusActive {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkNotFound
	}
	if locked.Revision != input.ExpectedRevision {
		return ExternalIdentityLinkMutation{}, ErrExternalIdentityLinkStateConflict
	}

	// last-login-method 与 unlink 同事务：删除后是否仍有可用登录方式。
	hasPassword, err := hasPasswordCredentialTx(ctx, tx, input.UserID)
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if !hasPassword {
		remaining, err := countActiveExternalLinksTx(ctx, tx, input.UserID, input.LinkID)
		if err != nil {
			return ExternalIdentityLinkMutation{}, err
		}
		if remaining < 1 {
			return ExternalIdentityLinkMutation{}, ErrExternalAuthLastLoginMethodRequired
		}
	}

	mutation, err := postgresLink.TransitionTx(ctx, tx, ExternalIdentityLinkActionUnlink, TransitionExternalIdentityLinkInput{
		LinkID:           input.LinkID,
		ExpectedRevision: input.ExpectedRevision,
		ActorUserID:      input.UserID,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ExternalIdentityLinkMutation{}, err
	}
	return mutation, nil
}

// unlinkIdempotencyKey 绑定 user/link/revision/request（非 client IP）。
func unlinkIdempotencyKey(userID, linkID, revision int64, requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		requestID = "once"
	}
	// 限制长度以符合 idempotency key 上限 128。
	key := fmt.Sprintf("unlink:%d:%d:r%d:%s", userID, linkID, revision, requestID)
	if len(key) > 128 {
		// 缩短：复用不可逆 digest 工具（输入非 SID，仅作长度夹紧）。
		key = "unlink:" + SessionFingerprint(key)
	}
	return key
}

// CanUnlink 校验：删除该 link 后用户是否仍有可用登录方式（事务外预检）。
func (s *ExternalAuthService) CanUnlink(ctx context.Context, userID, linkID int64) error {
	link, err := s.deps.LinkStore.Get(ctx, linkID)
	if err != nil {
		return err
	}
	if link.UserID != userID || link.Status != ExternalIdentityLinkStatusActive {
		return ErrExternalIdentityLinkNotFound
	}
	if s.deps.ExternalAuthStore == nil {
		return nil
	}
	hasPassword, err := s.deps.ExternalAuthStore.HasPasswordCredential(ctx, userID)
	if err != nil {
		return err
	}
	if hasPassword {
		return nil
	}
	count, err := s.deps.ExternalAuthStore.CountActiveExternalLinks(ctx, userID, "")
	if err != nil {
		return err
	}
	if count <= 1 {
		return ErrExternalAuthLastLoginMethodRequired
	}
	return nil
}

// CanRemovePassword 校验：删除密码后是否仍有外部 link。
func (s *ExternalAuthService) CanRemovePassword(ctx context.Context, userID int64) error {
	if s.deps.ExternalAuthStore == nil {
		return ErrExternalAuthLastLoginMethodRequired
	}
	count, err := s.deps.ExternalAuthStore.CountActiveExternalLinks(ctx, userID, "")
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrExternalAuthLastLoginMethodRequired
	}
	return nil
}

// --- 辅助方法 ---

func (s *ExternalAuthService) loadCurrentUser(ctx context.Context, userID int64) (CurrentUser, error) {
	if s.deps.LoadCurrentUser != nil {
		return s.deps.LoadCurrentUser(ctx, userID)
	}
	// 测试/未接线路径：仅基础字段（生产必须注入权威 loader）。
	if s.deps.Pool == nil {
		return CurrentUser{}, fmt.Errorf("identity pool unavailable")
	}
	row := s.deps.Pool.QueryRow(ctx, `
		SELECT id, username, display_name, locale, status, is_initial_super_admin, current_token_version
		FROM users WHERE id = $1
	`, userID)
	var current CurrentUser
	if err := row.Scan(
		&current.ID, &current.Username, &current.DisplayName, &current.Locale,
		&current.Status, &current.IsInitialSuperAdmin, &current.CurrentTokenVersion,
	); err != nil {
		return CurrentUser{}, fmt.Errorf("load external auth user: %w", err)
	}
	return current, nil
}

func (s *ExternalAuthService) anyUser(ctx context.Context) (bool, error) {
	if s.deps.AnyUserExists != nil {
		return s.deps.AnyUserExists(ctx)
	}
	if s.deps.Pool == nil {
		return false, fmt.Errorf("identity pool unavailable")
	}
	var count int
	err := s.deps.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count > 0, err
}

func (s *ExternalAuthService) registrationEnabled(ctx context.Context) (bool, error) {
	if s.deps.RegistrationEnabled != nil {
		return s.deps.RegistrationEnabled(ctx)
	}
	return true, nil
}

func (s *ExternalAuthService) providerContribution(providerID string) (identityregistry.ProviderContribution, error) {
	if s.deps.ProviderContribution != nil {
		return s.deps.ProviderContribution(providerID)
	}
	return identityregistry.ProviderContribution{}, ErrExternalIdentityLinkStoreUnavailable
}

func (s *ExternalAuthService) validateExternalRegistrationInput(ctx context.Context, input ExternalRegistrationInput) error {
	if s.deps.ValidateRegistration != nil {
		return s.deps.ValidateRegistration.ValidateExternalRegister(ctx, input)
	}
	// 测试兼容：最小非空检查（生产注入权威 Service）。
	return validateRegistrationLocalInputMinimal(input)
}

// ExternalRegistrationInput 是外部注册时收集的本地必填字段。
type ExternalRegistrationInput struct {
	Username    string
	Email       string
	DisplayName string
	Locale      string
}

// Normalized 返回 trim 后的字段，displayName 缺省回落到 username。
func (input ExternalRegistrationInput) Normalized() ExternalRegistrationInput {
	username := strings.TrimSpace(input.Username)
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = username
	}
	locale := strings.TrimSpace(input.Locale)
	if locale == "" {
		locale = "zh-CN"
	}
	return ExternalRegistrationInput{
		Username:    username,
		Email:       strings.TrimSpace(input.Email),
		DisplayName: displayName,
		Locale:      locale,
	}
}

func validateRegistrationLocalInputMinimal(input ExternalRegistrationInput) error {
	input = input.Normalized()
	if input.Username == "" {
		return ErrExternalRegistrationFieldUsername
	}
	if input.Email == "" {
		return ErrExternalRegistrationFieldEmail
	}
	return nil
}

// registrationConflictsTx 在事务里检查 username/email 冲突。
func registrationConflictsTx(ctx context.Context, tx pgx.Tx, username, email string) error {
	var usernameTaken, emailTaken bool
	err := tx.QueryRow(ctx, `
		SELECT
		  EXISTS (SELECT 1 FROM users WHERE username_lower = lower($1)),
		  EXISTS (SELECT 1 FROM users WHERE email_lower = lower($2))
	`, username, email).Scan(&usernameTaken, &emailTaken)
	if err != nil {
		return err
	}
	if usernameTaken {
		return ErrExternalRegistrationFieldUsername
	}
	if emailTaken {
		return ErrExternalRegistrationFieldEmail
	}
	return nil
}

// --- 错误定义 ---

var (
	ErrExternalAuthOperationNotActivated   = errors.New("auth.provider_not_enabled")
	ErrExternalAuthOperationMismatch       = errors.New("auth.provider_operation_mismatch")
	ErrExternalIdentityUnlinked            = errors.New("auth.external_identity_unlinked")
	ErrExternalAuthBootstrapRequired       = errors.New("auth.external_bootstrap_required")
	ErrExternalAuthActorRequired           = errors.New("auth.external_actor_required")
	ErrExternalAuthActorInactive           = errors.New("auth.external_actor_inactive")
	ErrExternalAuthActorMismatch           = errors.New("auth.external_actor_mismatch")
	ErrExternalAuthRecentAuthRequired      = errors.New("auth.recent_auth_required")
	ErrExternalAuthArtifactMismatch        = errors.New("auth.provider_artifact_mismatch")
	ErrExternalAuthProviderUnavailable     = errors.New("auth.provider_unavailable")
	ErrExternalAuthLastLoginMethodRequired = errors.New("auth.last_login_method_required")
	ErrExternalAuthDefaultRoleFailed       = errors.New("auth.external_default_role_failed")
	ErrExternalRegistrationFieldUsername   = errors.New("auth.register_invalid:username")
	ErrExternalRegistrationFieldEmail      = errors.New("auth.register_invalid:email")
)
