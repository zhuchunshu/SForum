package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 外部认证 Provider Host 激活目录。
//
// 安全约束（见 plans/2026-07-27-github-social-login-builtin-plugin.md）：
//   - 内置发现（SyncBuiltins）只暂存包，绝不自动信任/启用/配置/公开激活；
//   - Host 拥有 durable、revisioned、audited 的 activation；默认全 off；
//   - 有效公开可见性需要：精确 active 字节 + 兼容 operations + 必需配置 + Host 激活 + 非 Safe Mode；
//   - artifact 变化使公开 activation 失效，直到新 digest 被显式确认；
//   - 所有 mutation 需要 CAS（WHERE revision = expected + 影响行检查）与 audit；
//   - admin 权限为 identity.provider.manage；executable trust 仍 super_admin-only。

// ProviderActivation 是一个 auth provider 的 Host-owned 激活状态。
type ProviderActivation struct {
	ProviderID              string     `json:"providerId"`
	OwnerExtensionID        string     `json:"ownerExtensionId"`
	OwnerExtensionVersionID int64      `json:"ownerExtensionVersionId,omitempty"`
	OwnerPackageDigest      string     `json:"ownerPackageDigest,omitempty"`
	LoginEnabled            bool       `json:"loginEnabled"`
	RegistrationEnabled     bool       `json:"registrationEnabled"`
	LinkEnabled             bool       `json:"linkEnabled"`
	Priority                int        `json:"priority"`
	Revision                int64      `json:"revision"`
	LastProbeAt             *time.Time `json:"lastProbeAt,omitempty"`
	LastProbeOK             *bool      `json:"lastProbeOk,omitempty"`
	LastProbeReason         string     `json:"lastProbeReason,omitempty"`
	SettingsRevision        int64      `json:"settingsRevision"`
	CreatedAt               time.Time  `json:"createdAt"`
	UpdatedAt               time.Time  `json:"updatedAt"`
}

// ProviderActivationInput 是 PATCH 输入；nil 指针字段表示不改。
// Owner* 必须由 Host 从 live Registry 派生，不得信任浏览器声明。
type ProviderActivationInput struct {
	ProviderID          string
	OwnerExtensionID    string
	OwnerPackageDigest  string
	OwnerExtensionVerID int64
	LoginEnabled        *bool
	RegistrationEnabled *bool
	LinkEnabled         *bool
	Priority            *int
	ExpectedRevision    int64 // CAS：新建必须为 0；更新必须等于当前 revision
}

// HasOperationMutation 是否携带任何可变更字段（不含 revision）。
func (in ProviderActivationInput) HasOperationMutation() bool {
	return in.LoginEnabled != nil || in.RegistrationEnabled != nil ||
		in.LinkEnabled != nil || in.Priority != nil
}

// ProviderActivationProbeResult 是探测结果的持久化输入。
// T8B 起由真实 provider.probe 运行时操作写入；probe_pending/unavailable 仍强制 ok=false。
type ProviderActivationProbeResult struct {
	ProviderID string
	OK         bool
	Reason     string // redacted, 安全短文
	At         time.Time
}

// ProviderActivationStore 持久化激活状态。
type ProviderActivationStore interface {
	Get(ctx context.Context, providerID string) (ProviderActivation, error)
	List(ctx context.Context) ([]ProviderActivation, error)
	// Upsert 以原子乐观并发写入；冲突返回 ErrProviderActivationCASConflict。
	// 若现有行与期望输入等价（无实质变更），返回 ErrProviderActivationNoMutation 且不递增 revision。
	Upsert(ctx context.Context, input ProviderActivationInput) (ProviderActivation, error)
	RecordProbe(ctx context.Context, result ProviderActivationProbeResult) error
	// Delete 当 provider 卸载/禁用时清理激活；保留 audit。
	Delete(ctx context.Context, providerID string) error
	// ResetOperationsToDefaults 重置 login/registration/link=false，priority=0；保留 secrets。
	// 已是默认状态时返回 ErrProviderActivationNoMutation。
	ResetOperationsToDefaults(ctx context.Context, providerID string) (ProviderActivation, error)
}

// ErrProviderActivationNotFound 激活记录不存在（未激活）。
var ErrProviderActivationNotFound = errors.New("auth.provider_activation_not_found")

// ErrProviderActivationCASConflict CAS 修订冲突。
var ErrProviderActivationCASConflict = errors.New("auth.provider_activation_cas_conflict")

// ErrProviderActivationNoMutation 输入与当前状态等价，未写入。
var ErrProviderActivationNoMutation = errors.New("auth.provider_activation_no_mutation")

// ErrProviderActivationUnsupportedOperation 请求启用 live provider 未声明的操作。
var ErrProviderActivationUnsupportedOperation = errors.New("auth.provider_operation_unsupported")

// ErrProviderActivationOwnershipRejected 浏览器提交了 ownership/artifact 声明（Host 拒绝作为权威）。
var ErrProviderActivationOwnershipRejected = errors.New("auth.provider_ownership_rejected")

// PostgresProviderActivationStore 是 Host 拥有的 PostgreSQL 激活存储。
type PostgresProviderActivationStore struct {
	pool *pgxpool.Pool
}

func NewPostgresProviderActivationStore(pool *pgxpool.Pool) *PostgresProviderActivationStore {
	return &PostgresProviderActivationStore{pool: pool}
}

func (s *PostgresProviderActivationStore) Get(ctx context.Context, providerID string) (ProviderActivation, error) {
	row := s.pool.QueryRow(ctx, providerActivationSelectSQL+` WHERE provider_id = $1`, providerID)
	act, err := scanProviderActivation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProviderActivation{}, ErrProviderActivationNotFound
		}
		return ProviderActivation{}, err
	}
	return act, nil
}

func (s *PostgresProviderActivationStore) List(ctx context.Context) ([]ProviderActivation, error) {
	rows, err := s.pool.Query(ctx, providerActivationSelectSQL+`
		ORDER BY priority DESC, provider_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ProviderActivation, 0)
	for rows.Next() {
		act, err := scanProviderActivation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, act)
	}
	return out, rows.Err()
}

// Upsert 使用 WHERE revision = expected 的原子乐观并发：
//  1. 不存在时仅允许 ExpectedRevision=0 插入；
//  2. 存在时 UPDATE ... WHERE provider_id AND revision = expected，检查影响行；
//  3. 无实质字段变化时返回 NoMutation，不递增 revision。
func (s *PostgresProviderActivationStore) Upsert(ctx context.Context, input ProviderActivationInput) (ProviderActivation, error) {
	if err := validateProviderActivationInput(input); err != nil {
		return ProviderActivation{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProviderActivation{}, err
	}
	defer tx.Rollback(ctx)

	existing, err := scanProviderActivation(tx.QueryRow(ctx,
		providerActivationSelectSQL+` WHERE provider_id = $1 FOR UPDATE`, input.ProviderID))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ProviderActivation{}, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if input.ExpectedRevision != 0 {
			return ProviderActivation{}, ErrProviderActivationCASConflict
		}
		priority := 0
		if input.Priority != nil {
			priority = *input.Priority
		}
		login := boolPtrOr(input.LoginEnabled, false)
		reg := boolPtrOr(input.RegistrationEnabled, false)
		link := boolPtrOr(input.LinkEnabled, false)
		// 新建且全部默认 off、priority=0 仍写入一行（建立 artifact 绑定），
		// 以便后续 CAS 与有效可用性比对有基线。
		_, err = tx.Exec(ctx, `
			INSERT INTO identity_provider_activations
			  (provider_id, owner_extension_id, owner_extension_version_id, owner_package_digest,
			   login_enabled, registration_enabled, link_enabled, priority, revision, settings_revision)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1, 0)
		`, input.ProviderID, input.OwnerExtensionID, nullInt64(input.OwnerExtensionVerID), input.OwnerPackageDigest,
			login, reg, link, priority)
		if err != nil {
			return ProviderActivation{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ProviderActivation{}, err
		}
		return s.Get(ctx, input.ProviderID)
	}

	// 更新：先应用输入得到期望下一状态，再判断是否 no-mutation。
	next := applyProviderActivationInput(existing, input)
	if providerActivationStateEqual(existing, next) {
		// 仍校验 CAS，避免调用方拿着陈旧 revision 误以为“无变更成功”。
		if input.ExpectedRevision != existing.Revision {
			return ProviderActivation{}, ErrProviderActivationCASConflict
		}
		return existing, ErrProviderActivationNoMutation
	}
	if input.ExpectedRevision != existing.Revision {
		return ProviderActivation{}, ErrProviderActivationCASConflict
	}

	tag, err := tx.Exec(ctx, `
		UPDATE identity_provider_activations
		SET owner_extension_id = $2,
		    owner_extension_version_id = $3,
		    owner_package_digest = $4,
		    login_enabled = $5,
		    registration_enabled = $6,
		    link_enabled = $7,
		    priority = $8,
		    revision = revision + 1,
		    updated_at = now()
		WHERE provider_id = $1 AND revision = $9
	`, input.ProviderID, next.OwnerExtensionID, nullInt64(next.OwnerExtensionVersionID), next.OwnerPackageDigest,
		next.LoginEnabled, next.RegistrationEnabled, next.LinkEnabled, next.Priority, input.ExpectedRevision)
	if err != nil {
		return ProviderActivation{}, err
	}
	if tag.RowsAffected() != 1 {
		return ProviderActivation{}, ErrProviderActivationCASConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderActivation{}, err
	}
	return s.Get(ctx, input.ProviderID)
}

func (s *PostgresProviderActivationStore) RecordProbe(ctx context.Context, result ProviderActivationProbeResult) error {
	if result.At.IsZero() {
		result.At = time.Now()
	}
	// 真实 probe RPC 之前禁止把 pending 写成 ok=true。
	ok := result.OK
	if result.Reason == ProbeReasonPending || result.Reason == ProbeReasonUnavailable {
		ok = false
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE identity_provider_activations
		SET last_probe_at = $2, last_probe_ok = $3, last_probe_reason = $4, updated_at = now()
		WHERE provider_id = $1
	`, result.ProviderID, result.At, ok, truncateForColumn(result.Reason, 500))
	return err
}

func (s *PostgresProviderActivationStore) Delete(ctx context.Context, providerID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM identity_provider_activations WHERE provider_id = $1`, providerID)
	return err
}

func (s *PostgresProviderActivationStore) ResetOperationsToDefaults(ctx context.Context, providerID string) (ProviderActivation, error) {
	// 原子：仅当当前不是默认状态时才递增 revision。
	tag, err := s.pool.Exec(ctx, `
		UPDATE identity_provider_activations
		SET login_enabled = FALSE,
		    registration_enabled = FALSE,
		    link_enabled = FALSE,
		    priority = 0,
		    revision = revision + 1,
		    updated_at = now()
		WHERE provider_id = $1
		  AND (login_enabled OR registration_enabled OR link_enabled OR priority <> 0)
	`, providerID)
	if err != nil {
		return ProviderActivation{}, err
	}
	act, getErr := s.Get(ctx, providerID)
	if getErr != nil {
		return ProviderActivation{}, getErr
	}
	if tag.RowsAffected() == 0 {
		return act, ErrProviderActivationNoMutation
	}
	return act, nil
}

const providerActivationSelectSQL = `
	SELECT provider_id, owner_extension_id, owner_extension_version_id, owner_package_digest,
	       login_enabled, registration_enabled, link_enabled, priority, revision,
	       last_probe_at, last_probe_ok, last_probe_reason, settings_revision,
	       created_at, updated_at
	FROM identity_provider_activations
`

type providerActivationScanner interface {
	Scan(dest ...any) error
}

func scanProviderActivation(scanner providerActivationScanner) (ProviderActivation, error) {
	var act ProviderActivation
	var lastProbeAt *time.Time
	var lastProbeOK *bool
	var lastProbeReason *string
	var ownerExtVerID *int64
	if err := scanner.Scan(
		&act.ProviderID, &act.OwnerExtensionID, &ownerExtVerID, &act.OwnerPackageDigest,
		&act.LoginEnabled, &act.RegistrationEnabled, &act.LinkEnabled, &act.Priority, &act.Revision,
		&lastProbeAt, &lastProbeOK, &lastProbeReason, &act.SettingsRevision,
		&act.CreatedAt, &act.UpdatedAt,
	); err != nil {
		return ProviderActivation{}, err
	}
	if ownerExtVerID != nil {
		act.OwnerExtensionVersionID = *ownerExtVerID
	}
	act.LastProbeAt = lastProbeAt
	act.LastProbeOK = lastProbeOK
	if lastProbeReason != nil {
		act.LastProbeReason = *lastProbeReason
	}
	return act, nil
}

func validateProviderActivationInput(input ProviderActivationInput) error {
	if strings.TrimSpace(input.ProviderID) == "" {
		return fmt.Errorf("providerId required")
	}
	if strings.TrimSpace(input.OwnerExtensionID) == "" {
		return fmt.Errorf("ownerExtensionId required")
	}
	if strings.TrimSpace(input.OwnerPackageDigest) == "" {
		return fmt.Errorf("ownerPackageDigest required")
	}
	return nil
}

func applyProviderActivationInput(existing ProviderActivation, input ProviderActivationInput) ProviderActivation {
	next := existing
	// Host 派生的 live artifact 总是写入下一状态（激活绑定到精确 live digest）。
	if input.OwnerExtensionID != "" {
		next.OwnerExtensionID = input.OwnerExtensionID
	}
	if input.OwnerPackageDigest != "" {
		next.OwnerPackageDigest = input.OwnerPackageDigest
	}
	if input.OwnerExtensionVerID != 0 {
		next.OwnerExtensionVersionID = input.OwnerExtensionVerID
	}
	if input.LoginEnabled != nil {
		next.LoginEnabled = *input.LoginEnabled
	}
	if input.RegistrationEnabled != nil {
		next.RegistrationEnabled = *input.RegistrationEnabled
	}
	if input.LinkEnabled != nil {
		next.LinkEnabled = *input.LinkEnabled
	}
	if input.Priority != nil {
		next.Priority = *input.Priority
	}
	return next
}

// providerActivationStateEqual 比较影响公开可用性与排序的持久字段（不含 revision/时间/probe）。
func providerActivationStateEqual(a, b ProviderActivation) bool {
	return a.OwnerExtensionID == b.OwnerExtensionID &&
		a.OwnerExtensionVersionID == b.OwnerExtensionVersionID &&
		a.OwnerPackageDigest == b.OwnerPackageDigest &&
		a.LoginEnabled == b.LoginEnabled &&
		a.RegistrationEnabled == b.RegistrationEnabled &&
		a.LinkEnabled == b.LinkEnabled &&
		a.Priority == b.Priority
}

func boolPtrOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func nullInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func truncateForColumn(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// ---------------------------------------------------------------------------
// 内存实现：测试与无 DB 路径；语义对齐 Postgres（mutex + CAS + no-mutation）。
// ---------------------------------------------------------------------------

// MemoryProviderActivationStore 是并发安全的内存激活目录。
type MemoryProviderActivationStore struct {
	mu    sync.Mutex
	items map[string]ProviderActivation
}

// NewMemoryProviderActivationStore 构造空内存激活目录。
func NewMemoryProviderActivationStore() *MemoryProviderActivationStore {
	return &MemoryProviderActivationStore{items: map[string]ProviderActivation{}}
}

func (s *MemoryProviderActivationStore) Get(_ context.Context, providerID string) (ProviderActivation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.items[providerID]; ok {
		return a, nil
	}
	return ProviderActivation{}, ErrProviderActivationNotFound
}

func (s *MemoryProviderActivationStore) List(_ context.Context) ([]ProviderActivation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ProviderActivation, 0, len(s.items))
	for _, a := range s.items {
		out = append(out, a)
	}
	return out, nil
}

func (s *MemoryProviderActivationStore) Upsert(_ context.Context, input ProviderActivationInput) (ProviderActivation, error) {
	if err := validateProviderActivationInput(input); err != nil {
		return ProviderActivation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.items[input.ProviderID]
	if !ok {
		if input.ExpectedRevision != 0 {
			return ProviderActivation{}, ErrProviderActivationCASConflict
		}
		now := time.Now()
		a := ProviderActivation{
			ProviderID:              input.ProviderID,
			OwnerExtensionID:        input.OwnerExtensionID,
			OwnerPackageDigest:      input.OwnerPackageDigest,
			OwnerExtensionVersionID: input.OwnerExtensionVerID,
			LoginEnabled:            boolPtrOr(input.LoginEnabled, false),
			RegistrationEnabled:     boolPtrOr(input.RegistrationEnabled, false),
			LinkEnabled:             boolPtrOr(input.LinkEnabled, false),
			Priority:                0,
			Revision:                1,
			CreatedAt:               now,
			UpdatedAt:               now,
		}
		if input.Priority != nil {
			a.Priority = *input.Priority
		}
		s.items[input.ProviderID] = a
		return a, nil
	}
	next := applyProviderActivationInput(existing, input)
	if input.ExpectedRevision != existing.Revision {
		return ProviderActivation{}, ErrProviderActivationCASConflict
	}
	if providerActivationStateEqual(existing, next) {
		return existing, ErrProviderActivationNoMutation
	}
	next.Revision = existing.Revision + 1
	next.UpdatedAt = time.Now()
	// 保留 probe / settings 字段。
	next.LastProbeAt = existing.LastProbeAt
	next.LastProbeOK = existing.LastProbeOK
	next.LastProbeReason = existing.LastProbeReason
	next.SettingsRevision = existing.SettingsRevision
	next.CreatedAt = existing.CreatedAt
	s.items[input.ProviderID] = next
	return next, nil
}

func (s *MemoryProviderActivationStore) RecordProbe(_ context.Context, result ProviderActivationProbeResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.items[result.ProviderID]
	if !ok {
		return nil
	}
	at := result.At
	if at.IsZero() {
		at = time.Now()
	}
	okVal := result.OK
	if result.Reason == ProbeReasonPending || result.Reason == ProbeReasonUnavailable {
		okVal = false
	}
	a.LastProbeAt = &at
	a.LastProbeOK = &okVal
	a.LastProbeReason = result.Reason
	a.UpdatedAt = time.Now()
	s.items[result.ProviderID] = a
	return nil
}

func (s *MemoryProviderActivationStore) Delete(_ context.Context, providerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, providerID)
	return nil
}

func (s *MemoryProviderActivationStore) ResetOperationsToDefaults(_ context.Context, providerID string) (ProviderActivation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.items[providerID]
	if !ok {
		return ProviderActivation{}, ErrProviderActivationNotFound
	}
	if !a.LoginEnabled && !a.RegistrationEnabled && !a.LinkEnabled && a.Priority == 0 {
		return a, ErrProviderActivationNoMutation
	}
	a.LoginEnabled = false
	a.RegistrationEnabled = false
	a.LinkEnabled = false
	a.Priority = 0
	a.Revision++
	a.UpdatedAt = time.Now()
	s.items[providerID] = a
	return a, nil
}

// Probe 稳定 reason 常量。
const (
	// ProbeReasonPending 历史占位；T8B 起不是产品 probe 实现，不得作为成功路径写入。
	ProbeReasonPending = "probe_pending"
	// ProbeReasonUnavailable Host 无法执行探测（缺 runtime / 未声明操作 / Safe Mode 等）。
	ProbeReasonUnavailable = "probe_unavailable"
)

var _ ProviderActivationStore = (*PostgresProviderActivationStore)(nil)
var _ ProviderActivationStore = (*MemoryProviderActivationStore)(nil)
