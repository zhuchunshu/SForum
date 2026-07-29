package pages

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
)

var (
	ErrUnknownPage         = errors.New("pages: unknown page")
	ErrNotReplaceable      = errors.New("pages: page is not replaceable")
	ErrReservedPath        = errors.New("pages: path is reserved")
	ErrConflictProvider    = errors.New("pages: multiple providers claim page")
	ErrApprovalRequired    = errors.New("pages: replace requires approval")
	ErrInvalidContribution = errors.New("pages: invalid page contribution")
	ErrInvalidAccess       = errors.New("pages: invalid access")
	ErrContractMismatch    = errors.New("pages: contract version mismatch")
	ErrRevisionConflict    = errors.New("pages: registry revision conflict")
	ErrArtifactConflict    = errors.New("pages: exact artifact conflict")
)

// ContributionAction 主题/插件对页面的贡献动作。
type ContributionAction string

const (
	ActionAdd     ContributionAction = "add"
	ActionReplace ContributionAction = "replace"
)

// PageContribution 扩展声明的页面贡献（来自 theme.json / manifest）。
type PageContribution struct {
	ID                string             `json:"id"`
	Action            ContributionAction `json:"action"`
	Target            string             `json:"target,omitempty"` // replace 目标 page id
	Path              string             `json:"path,omitempty"`   // add 路径
	Template          string             `json:"template,omitempty"`
	Contract          string             `json:"contract,omitempty"`
	Access            Access             `json:"access,omitempty"`
	Permission        string             `json:"permission,omitempty"` // access=permission 时必填
	DataSource        string             `json:"dataSource,omitempty"` // core | plugin
	DataRoute         string             `json:"dataRoute,omitempty"`
	DataSchema        string             `json:"dataSchema,omitempty"` // 可选 JSON Schema 相对路径
	ExtensionID       string             `json:"extensionId"`
	Version           string             `json:"version"`
	PackageDigest     string             `json:"packageDigest,omitempty"`
	RuntimeInstanceID string             `json:"runtimeInstanceId,omitempty"`
	// RouteSignature 注册时计算的语义签名（参数名无关）。
	RouteSignature string `json:"routeSignature,omitempty"`
}

// ProviderBinding 管理员选择的生效提供者（或自动选中的唯一贡献）。
// 批准必须绑定 extension id、version、package digest、contribution id 与 contract version。
type ProviderBinding struct {
	PageID          string `json:"pageId"`
	ExtensionID     string `json:"extensionId"`
	ContributionID  string `json:"contributionId"`
	Version         string `json:"version"`
	PackageDigest   string `json:"packageDigest"`
	ContractVersion string `json:"contractVersion"`
	ApprovedBy      int64  `json:"approvedBy,omitempty"`
	TemplatePath    string `json:"templatePath,omitempty"`
}

// RuntimeArtifact records the exact plugin process that owns one contribution
// set. Themes and legacy callers keep RuntimeInstanceID empty.
type RuntimeArtifact struct {
	ExtensionID       string `json:"extensionId"`
	ExtensionVersion  string `json:"extensionVersion"`
	PackageDigest     string `json:"packageDigest"`
	RuntimeInstanceID string `json:"runtimeInstanceId,omitempty"`
}

type ExtensionSnapshot struct {
	Revision      uint64
	Artifact      RuntimeArtifact
	Contributions []PageContribution
}

type ThemePublicationSnapshot struct {
	ExtensionIDs  []string
	Artifacts     map[string]RuntimeArtifact
	Contributions map[string][]PageContribution
	Bindings      []ProviderBinding
}

// Store 持久化绑定与审批（可实现为内存或 Postgres）。
type Store interface {
	ListBindings(ctx context.Context) ([]ProviderBinding, error)
	GetBinding(ctx context.Context, pageID string) (ProviderBinding, bool, error)
	UpsertBinding(ctx context.Context, binding ProviderBinding) error
	DeleteBinding(ctx context.Context, pageID string) error
	// ReplaceExtensionBindings atomically removes stale bindings owned by the
	// listed extensions and publishes the exact replacement set.
	ReplaceExtensionBindings(ctx context.Context, extensionIDs []string, bindings []ProviderBinding) error
	ReconcileExtensionBindings(ctx context.Context, activeExtensionID string, staleExtensionIDs []string, allowed []ProviderBinding) error
}

// Registry 解析页面提供者：目录 + 贡献 + 管理员绑定。
type Registry struct {
	store  Store
	logger *slog.Logger

	mu                  sync.RWMutex
	revision            uint64
	restoreBindingsOnce sync.Once
	restoreBindingsErr  error
	// bindings is copy-on-write. Reads hold mu.RLock while lifecycle changes,
	// approvals, and restores publish one coherent provider/contribution view.
	bindings      map[string]ProviderBinding
	contributions map[string][]PageContribution // pageId or contribution id for add
	// compiledAdds 确定性匹配表（按 Specificity 排序），不依赖 map 迭代顺序。
	compiledAdds []CompiledRoute
	// signatureOwners signature → 当前拥有该语义路径的扩展（用于冲突检测）。
	signatureOwners    map[string]string
	extensionArtifacts map[string]RuntimeArtifact
	admission          func(RuntimeArtifact) bool
}

// NewRegistry 创建注册表；store 可为 nil（仅 core）。
func NewRegistry(store Store) *Registry {
	return &Registry{
		store:              store,
		bindings:           map[string]ProviderBinding{},
		contributions:      map[string][]PageContribution{},
		compiledAdds:       nil,
		signatureOwners:    map[string]string{},
		extensionArtifacts: map[string]RuntimeArtifact{},
	}
}

func (r *Registry) CaptureThemePublication(extensionIDs ...string) ThemePublicationSnapshot {
	snapshot := ThemePublicationSnapshot{
		Artifacts: make(map[string]RuntimeArtifact), Contributions: make(map[string][]PageContribution),
	}
	if r == nil {
		return snapshot
	}
	owners := make(map[string]struct{}, len(extensionIDs))
	for _, extensionID := range extensionIDs {
		if extensionID != "" {
			owners[extensionID] = struct{}{}
			snapshot.ExtensionIDs = append(snapshot.ExtensionIDs, extensionID)
		}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for extensionID := range owners {
		if artifact, ok := r.extensionArtifacts[extensionID]; ok {
			snapshot.Artifacts[extensionID] = artifact
		}
	}
	for _, list := range r.contributions {
		for _, contribution := range list {
			if _, ok := owners[contribution.ExtensionID]; ok {
				snapshot.Contributions[contribution.ExtensionID] = append(snapshot.Contributions[contribution.ExtensionID], contribution)
			}
		}
	}
	for _, binding := range r.bindings {
		if _, ok := owners[binding.ExtensionID]; ok {
			snapshot.Bindings = append(snapshot.Bindings, cloneBinding(binding))
		}
	}
	return snapshot
}

func (r *Registry) RestoreThemePublication(ctx context.Context, snapshot ThemePublicationSnapshot, currentExtensionIDs ...string) error {
	if r == nil {
		return nil
	}
	prepared := make(map[string][]PageContribution, len(snapshot.Contributions))
	for extensionID, contributions := range snapshot.Contributions {
		items, err := prepareContributions(extensionID, contributions)
		if err != nil {
			return err
		}
		prepared[extensionID] = items
	}
	owners := append([]string(nil), currentExtensionIDs...)
	owners = append(owners, snapshot.ExtensionIDs...)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store != nil {
		if err := r.store.ReplaceExtensionBindings(ctx, owners, snapshot.Bindings); err != nil {
			return err
		}
	}
	for _, extensionID := range owners {
		r.clearExtensionLocked(extensionID)
	}
	for extensionID, contributions := range prepared {
		r.applyContributionsLocked(contributions)
		if artifact, ok := snapshot.Artifacts[extensionID]; ok {
			r.extensionArtifacts[extensionID] = artifact
		}
	}
	next := cloneBindings(r.bindings)
	ownerSet := make(map[string]struct{}, len(owners))
	for _, owner := range owners {
		ownerSet[owner] = struct{}{}
	}
	for pageID, binding := range next {
		if _, ok := ownerSet[binding.ExtensionID]; ok {
			delete(next, pageID)
		}
	}
	for _, binding := range snapshot.Bindings {
		next[binding.PageID] = cloneBinding(binding)
	}
	r.bindings = next
	r.revision++
	return nil
}

// RestoreBindings loads durable provider choices exactly once during boot.
// Request resolution never falls back to Store.GetBinding.
func (r *Registry) RestoreBindings(ctx context.Context) error {
	if r == nil || r.store == nil {
		return nil
	}
	r.restoreBindingsOnce.Do(func() {
		r.restoreBindingsErr = r.restoreBindings(ctx)
	})
	return r.restoreBindingsErr
}

func (r *Registry) restoreBindings(ctx context.Context) error {
	items, err := r.store.ListBindings(ctx)
	if err != nil {
		return err
	}
	next := make(map[string]ProviderBinding, len(items))
	for _, binding := range items {
		binding.PageID = strings.TrimSpace(binding.PageID)
		if binding.PageID == "" {
			return fmt.Errorf("%w: stored page binding has no page id", ErrInvalidContribution)
		}
		if _, exists := next[binding.PageID]; exists {
			return fmt.Errorf("%w: duplicate stored page binding %s", ErrInvalidContribution, binding.PageID)
		}
		next[binding.PageID] = cloneBinding(binding)
	}
	r.mu.Lock()
	r.bindings = next
	r.revision++
	r.mu.Unlock()
	return nil
}

func cloneBindings(input map[string]ProviderBinding) map[string]ProviderBinding {
	result := make(map[string]ProviderBinding, len(input))
	for pageID, binding := range input {
		result[pageID] = cloneBinding(binding)
	}
	return result
}

// WithRuntimeAdmission hides exact-runtime contributions while their Manager
// gate is staged or draining. It does not replace the execution-time lease.
func (r *Registry) WithRuntimeAdmission(admission func(RuntimeArtifact) bool) *Registry {
	if r == nil {
		return r
	}
	r.mu.Lock()
	r.admission = admission
	r.mu.Unlock()
	return r
}

// WithLogger 注入安全诊断日志（contract 回退等）。
func (r *Registry) WithLogger(logger *slog.Logger) *Registry {
	if r != nil {
		r.logger = logger
	}
	return r
}

func (r *Registry) logWarn(msg string, args ...any) {
	if r != nil && r.logger != nil {
		r.logger.Warn(msg, args...)
		return
	}
}

// RegisterContributions 注册某扩展的页面贡献（启用/激活时调用；禁用时 ClearExtension）。
// 先完整校验全部条目，再原子替换该扩展的旧贡献，避免半注册状态。
func (r *Registry) RegisterContributions(extensionID string, items []PageContribution) error {
	prepared, err := prepareContributions(extensionID, items)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.preflightAddsLocked(extensionID, prepared); err != nil {
		return err
	}
	r.clearExtensionLocked(extensionID)
	if len(prepared) == 0 {
		// This legacy entry point has no exact RuntimeArtifact input. An empty
		// contribution set therefore means absence, not an artifact with blank
		// version, digest, and runtime fields. Lifecycle publication uses
		// PublishExtensionIfRevision when it needs an exact empty set.
		r.revision++
		return nil
	}
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[extensionID] = pageArtifactFromContributions(extensionID, prepared)
	r.revision++
	return nil
}

// ReplaceExtensionContributions 原子替换某扩展的贡献快照。
// 用于主题切换：在「最终状态」视角下校验，允许新旧主题共享相同 add 路径。
// 失败时内存状态不变。
func (r *Registry) ReplaceExtensionContributions(extensionID string, items []PageContribution) error {
	prepared, err := prepareContributions(extensionID, items)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.preflightAddsLocked(extensionID, prepared); err != nil {
		return err
	}
	r.clearExtensionLocked(extensionID)
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[extensionID] = pageArtifactFromContributions(extensionID, prepared)
	r.revision++
	return nil
}

// ReplaceThemeContributions 主题激活专用：用新主题贡献替换旧活动主题，
// 按「最终状态」校验（新旧主题相同 add 路径允许；与其它扩展冲突仍失败）。
// 失败时内存 snapshot 完全不变。
func (r *Registry) ReplaceThemeContributions(newThemeID string, newItems []PageContribution, oldThemeID string) error {
	prepared, err := prepareContributions(newThemeID, newItems)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.preflightThemeAddsLocked(newThemeID, prepared, oldThemeID); err != nil {
		return err
	}
	if oldThemeID != "" && oldThemeID != newThemeID {
		r.clearExtensionLocked(oldThemeID)
	}
	r.clearExtensionLocked(newThemeID)
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[newThemeID] = pageArtifactFromContributions(newThemeID, prepared)
	r.revision++
	return nil
}

// SwitchThemeContributions removes old theme approvals and publishes the new
// theme as presentation candidates without granting any Core replacement.
func (r *Registry) SwitchThemeContributions(ctx context.Context, newThemeID string, newItems []PageContribution, oldThemeID string) error {
	prepared, err := prepareContributions(newThemeID, newItems)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.preflightThemeAddsLocked(newThemeID, prepared, oldThemeID); err != nil {
		return err
	}
	removeOwners := []string{newThemeID}
	if oldThemeID != "" && oldThemeID != newThemeID {
		removeOwners = append(removeOwners, oldThemeID)
	}
	if r.store != nil {
		if err := r.store.ReplaceExtensionBindings(ctx, removeOwners, nil); err != nil {
			return err
		}
	}
	if oldThemeID != "" && oldThemeID != newThemeID {
		r.clearExtensionLocked(oldThemeID)
	}
	r.clearExtensionLocked(newThemeID)
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[newThemeID] = pageArtifactFromContributions(newThemeID, prepared)
	next := cloneBindings(r.bindings)
	for pageID, binding := range next {
		for _, owner := range removeOwners {
			if binding.ExtensionID == owner {
				delete(next, pageID)
				break
			}
		}
	}
	r.bindings = next
	r.revision++
	return nil
}

// ActivateThemeContributions publishes theme contributions and their automatic
// replace bindings as one process-local revision. Durable bindings are swapped
// first, so a failed store transaction cannot expose a partial runtime view.
func (r *Registry) ActivateThemeContributions(
	ctx context.Context,
	newThemeID string,
	newItems []PageContribution,
	oldThemeID string,
	approvedBy int64,
) error {
	if approvedBy <= 0 {
		return ErrApprovalRequired
	}
	return r.ActivateThemeContributionsReplacing(ctx, newThemeID, newItems, approvedBy, oldThemeID)
}

func (r *Registry) ActivateThemeContributionsReplacing(
	ctx context.Context,
	newThemeID string,
	newItems []PageContribution,
	approvedBy int64,
	oldThemeIDs ...string,
) error {
	if approvedBy <= 0 {
		return ErrApprovalRequired
	}
	prepared, err := prepareContributions(newThemeID, newItems)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// 模拟最终状态：去掉旧主题 + 新主题旧贡献，再校验新主题 adds
	ignore := map[string]struct{}{newThemeID: {}}
	for _, oldThemeID := range oldThemeIDs {
		if oldThemeID != "" && oldThemeID != newThemeID {
			ignore[oldThemeID] = struct{}{}
		}
	}
	for _, item := range prepared {
		if item.Action != ActionAdd {
			continue
		}
		sig := item.RouteSignature
		if owner, ok := r.signatureOwners[sig]; ok {
			if _, skip := ignore[owner]; !skip {
				return fmt.Errorf("%w: path signature %s owned by %s", ErrConflictProvider, sig, owner)
			}
		}
	}

	bindings := make([]ProviderBinding, 0, len(prepared))
	for _, item := range prepared {
		if item.Action != ActionReplace {
			continue
		}
		bindings = append(bindings, ProviderBinding{
			PageID: item.Target, ExtensionID: newThemeID, ContributionID: item.ID,
			Version: item.Version, PackageDigest: item.PackageDigest, ApprovedBy: approvedBy,
			TemplatePath: item.Template, ContractVersion: item.Contract,
		})
	}
	removeOwners := []string{newThemeID}
	for _, oldThemeID := range oldThemeIDs {
		if oldThemeID != "" && oldThemeID != newThemeID {
			removeOwners = append(removeOwners, oldThemeID)
		}
	}
	if r.store != nil {
		if err := r.store.ReplaceExtensionBindings(ctx, removeOwners, bindings); err != nil {
			return err
		}
	}

	for _, oldThemeID := range oldThemeIDs {
		if oldThemeID != "" && oldThemeID != newThemeID {
			r.clearExtensionLocked(oldThemeID)
		}
	}
	r.clearExtensionLocked(newThemeID)
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[newThemeID] = pageArtifactFromContributions(newThemeID, prepared)
	nextBindings := cloneBindings(r.bindings)
	for pageID, binding := range nextBindings {
		for _, extensionID := range removeOwners {
			if binding.ExtensionID == extensionID {
				delete(nextBindings, pageID)
				break
			}
		}
	}
	for _, binding := range bindings {
		nextBindings[binding.PageID] = cloneBinding(binding)
	}
	r.bindings = nextBindings
	r.revision++
	return nil
}

// RestoreThemeContributions never invents approval. It preserves only durable
// bindings that still match the active exact artifact and removes stale theme
// owners before publishing the restored contribution snapshot.
func (r *Registry) RestoreThemeContributions(
	ctx context.Context,
	activeThemeID string,
	items []PageContribution,
	staleThemeIDs []string,
) error {
	prepared, err := prepareContributions(activeThemeID, items)
	if err != nil {
		return err
	}
	allowed := make([]ProviderBinding, 0, len(prepared))
	for _, item := range prepared {
		if item.Action == ActionReplace {
			allowed = append(allowed, ProviderBinding{
				PageID: item.Target, ExtensionID: activeThemeID, ContributionID: item.ID,
				Version: item.Version, PackageDigest: item.PackageDigest,
				TemplatePath: item.Template, ContractVersion: item.Contract,
			})
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.preflightThemeAddsLocked(activeThemeID, prepared, staleThemeIDs...); err != nil {
		return err
	}
	if r.store != nil {
		if err := r.store.ReconcileExtensionBindings(ctx, activeThemeID, staleThemeIDs, allowed); err != nil {
			return err
		}
	}
	for _, staleID := range staleThemeIDs {
		if staleID != "" && staleID != activeThemeID {
			r.clearExtensionLocked(staleID)
		}
	}
	r.clearExtensionLocked(activeThemeID)
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[activeThemeID] = pageArtifactFromContributions(activeThemeID, prepared)
	allowedByPage := make(map[string]ProviderBinding, len(allowed))
	for _, binding := range allowed {
		allowedByPage[binding.PageID] = binding
	}
	next := cloneBindings(r.bindings)
	for pageID, binding := range next {
		if binding.ExtensionID == activeThemeID {
			exact, ok := allowedByPage[pageID]
			if !ok || !sameProviderArtifact(binding, exact) {
				delete(next, pageID)
			}
			continue
		}
		for _, staleID := range staleThemeIDs {
			if binding.ExtensionID == staleID {
				delete(next, pageID)
				break
			}
		}
	}
	r.bindings = next
	r.revision++
	return nil
}

func (r *Registry) preflightThemeAddsLocked(newThemeID string, prepared []PageContribution, oldThemeIDs ...string) error {
	ignore := map[string]struct{}{newThemeID: {}}
	for _, oldThemeID := range oldThemeIDs {
		if oldThemeID != "" {
			ignore[oldThemeID] = struct{}{}
		}
	}
	for _, item := range prepared {
		if item.Action != ActionAdd {
			continue
		}
		if owner, ok := r.signatureOwners[item.RouteSignature]; ok {
			if _, ignored := ignore[owner]; !ignored {
				return fmt.Errorf("%w: path signature %s owned by %s", ErrConflictProvider, item.RouteSignature, owner)
			}
		}
	}
	return nil
}

func sameProviderArtifact(current, exact ProviderBinding) bool {
	return current.PageID == exact.PageID && current.ExtensionID == exact.ExtensionID &&
		current.ContributionID == exact.ContributionID && current.Version == exact.Version &&
		current.PackageDigest == exact.PackageDigest && current.TemplatePath == exact.TemplatePath &&
		current.ContractVersion == exact.ContractVersion
}

func (r *Registry) preflightAddsLocked(extensionID string, prepared []PageContribution) error {
	// 本批次内 signature 去重
	seenSig := map[string]struct{}{}
	for _, item := range prepared {
		if item.Action != ActionAdd {
			continue
		}
		sig := item.RouteSignature
		if _, ok := seenSig[sig]; ok {
			return fmt.Errorf("%w: duplicate route signature %s", ErrInvalidContribution, sig)
		}
		seenSig[sig] = struct{}{}
		if owner, ok := r.signatureOwners[sig]; ok && owner != extensionID {
			return fmt.Errorf("%w: path signature %s owned by %s", ErrConflictProvider, sig, owner)
		}
	}
	return nil
}

func (r *Registry) applyContributionsLocked(prepared []PageContribution) {
	for _, item := range prepared {
		switch item.Action {
		case ActionReplace:
			target := strings.TrimSpace(item.Target)
			r.contributions[target] = append(r.contributions[target], item)
		case ActionAdd:
			r.contributions[item.ID] = append(r.contributions[item.ID], item)
			compiled, err := CompileRoute(item.Path, item)
			if err != nil {
				// prepare 已校验，理论上不会到这里
				continue
			}
			r.signatureOwners[item.RouteSignature] = item.ExtensionID
			// 替换同扩展旧 route（clear 已做过）；追加新 route
			r.compiledAdds = append(r.compiledAdds, compiled)
		}
	}
	SortCompiledRoutes(r.compiledAdds)
}

// prepareContributions 校验并规范化贡献列表（不修改 Registry）。
func prepareContributions(extensionID string, items []PageContribution) ([]PageContribution, error) {
	prepared := make([]PageContribution, 0, len(items))
	seenAddSigs := map[string]struct{}{}
	for _, item := range items {
		item.ExtensionID = extensionID
		// access 规范化（空 → public；未知 → 失败）
		access, err := NormalizeAccess(string(item.Access))
		if err != nil {
			return nil, err
		}
		item.Access = access
		if access == AccessPermission {
			if strings.TrimSpace(item.Permission) == "" {
				return nil, fmt.Errorf("%w: permission key required for access=permission", ErrInvalidAccess)
			}
		}
		if err := validateContribution(item); err != nil {
			return nil, err
		}
		// data route 预检
		if strings.TrimSpace(item.DataRoute) != "" {
			if err := ValidateDataRoute(item.DataRoute); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidContribution, err)
			}
			if item.DataSource == "" {
				item.DataSource = "plugin"
			}
		}
		switch item.Action {
		case ActionReplace:
			target := strings.TrimSpace(item.Target)
			page, ok := Find(target)
			if !ok {
				return nil, fmt.Errorf("%w: target %q", ErrUnknownPage, target)
			}
			if !page.Replaceable {
				return nil, fmt.Errorf("%w: %s", ErrNotReplaceable, target)
			}
			// replace 必须声明 contract，且与核心目录一致
			if strings.TrimSpace(item.Contract) == "" {
				return nil, fmt.Errorf("%w: replace requires contract", ErrInvalidContribution)
			}
			if page.ContractVersion != "" && item.Contract != page.ContractVersion {
				return nil, fmt.Errorf("%w: contribution contract %q != core %q", ErrContractMismatch, item.Contract, page.ContractVersion)
			}
		case ActionAdd:
			path := normalizePublicPath(item.Path)
			item.Path = path
			if IsReservedPath(path) {
				return nil, fmt.Errorf("%w: %s", ErrReservedPath, path)
			}
			// 与核心目录路径冲突（精确或参数化语义）
			if _, ok := MatchPath(path); ok {
				return nil, fmt.Errorf("%w: path %s collides with core page", ErrConflictProvider, path)
			}
			// 也检查与核心 path pattern 的 signature 冲突
			sig, err := CanonicalRouteSignature(path)
			if err != nil {
				return nil, err
			}
			item.RouteSignature = sig
			for _, page := range Catalog() {
				if page.PathPattern == "" {
					continue
				}
				coreSig, err := CanonicalRouteSignature(page.PathPattern)
				if err != nil {
					continue
				}
				if signaturesConflict(sig, coreSig) {
					return nil, fmt.Errorf("%w: path signature %s collides with core page %s", ErrConflictProvider, sig, page.ID)
				}
			}
			if _, ok := seenAddSigs[sig]; ok {
				return nil, fmt.Errorf("%w: duplicate add signature %s", ErrInvalidContribution, sig)
			}
			seenAddSigs[sig] = struct{}{}
			// 编译校验
			if _, err := CompileRoute(path, item); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%w: action %q", ErrInvalidContribution, item.Action)
		}
		prepared = append(prepared, item)
	}
	return prepared, nil
}

// PreflightContributions 仅校验贡献是否可注册，不修改状态（激活预检用）。
// replaceExtensionID 非空时，预检按「替换该扩展」的最终状态计算（主题切换）。
func (r *Registry) PreflightContributions(extensionID string, items []PageContribution) error {
	return r.PreflightContributionsReplacing(extensionID, items, extensionID)
}

// PreflightContributionsReplacing 按「忽略 ignoreExtensionIDs 后的最终状态」校验。
// 主题激活：ignore = [newTheme, oldTheme]，使新旧主题同路径可切换。
func (r *Registry) PreflightContributionsReplacing(extensionID string, items []PageContribution, ignoreExtensionIDs ...string) error {
	prepared, err := prepareContributions(extensionID, items)
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	ignore := map[string]struct{}{}
	for _, id := range ignoreExtensionIDs {
		if id != "" {
			ignore[id] = struct{}{}
		}
	}
	ignore[extensionID] = struct{}{}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range prepared {
		if item.Action != ActionAdd {
			continue
		}
		if owner, ok := r.signatureOwners[item.RouteSignature]; ok {
			if _, skip := ignore[owner]; !skip {
				return fmt.Errorf("%w: path signature %s owned by %s", ErrConflictProvider, item.RouteSignature, owner)
			}
		}
	}
	return nil
}

// ClearExtension 移除扩展的全部页面贡献。
func (r *Registry) ClearExtension(extensionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearExtensionLocked(extensionID)
	r.revision++
}

func (r *Registry) clearExtensionLocked(extensionID string) {
	delete(r.extensionArtifacts, extensionID)
	for key, list := range r.contributions {
		filtered := list[:0]
		for _, c := range list {
			if c.ExtensionID != extensionID {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			delete(r.contributions, key)
		} else {
			r.contributions[key] = filtered
		}
	}
	// 重建 compiledAdds 与 signatureOwners（去掉该扩展）
	kept := r.compiledAdds[:0]
	for _, cr := range r.compiledAdds {
		if cr.Contribution.ExtensionID == extensionID {
			delete(r.signatureOwners, cr.Signature)
			continue
		}
		kept = append(kept, cr)
	}
	r.compiledAdds = kept
	// 确保 owners 与 routes 一致
	r.rebuildOwnersLocked()
}

// PublishExtensionIfRevision atomically replaces one exact plugin page set.
// The revision CAS lets a lifecycle aggregate preserve unrelated extensions.
func (r *Registry) PublishExtensionIfRevision(
	artifact RuntimeArtifact,
	items []PageContribution,
	expectedRevision uint64,
) (uint64, error) {
	if r == nil || !validRuntimeArtifact(artifact, true) {
		return 0, ErrArtifactConflict
	}
	exact := append([]PageContribution(nil), items...)
	for index := range exact {
		exact[index].ExtensionID = artifact.ExtensionID
		exact[index].Version = artifact.ExtensionVersion
		exact[index].PackageDigest = artifact.PackageDigest
		exact[index].RuntimeInstanceID = artifact.RuntimeInstanceID
	}
	prepared, err := prepareContributions(artifact.ExtensionID, exact)
	if err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.revision != expectedRevision {
		return r.revision, fmt.Errorf("%w: expected %d, current %d", ErrRevisionConflict, expectedRevision, r.revision)
	}
	if err := r.preflightAddsLocked(artifact.ExtensionID, prepared); err != nil {
		return r.revision, err
	}
	r.clearExtensionLocked(artifact.ExtensionID)
	r.applyContributionsLocked(prepared)
	r.extensionArtifacts[artifact.ExtensionID] = artifact
	r.revision++
	return r.revision, nil
}

func (r *Registry) RemoveExtensionIfRevision(
	extensionID string,
	expected RuntimeArtifact,
	expectedRevision uint64,
) (uint64, error) {
	if r == nil || strings.TrimSpace(extensionID) == "" {
		return 0, ErrArtifactConflict
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.revision != expectedRevision {
		return r.revision, fmt.Errorf("%w: expected %d, current %d", ErrRevisionConflict, expectedRevision, r.revision)
	}
	current, exists := r.extensionArtifacts[extensionID]
	if validRuntimeArtifact(expected, false) && (!exists || current != expected) {
		return r.revision, ErrArtifactConflict
	}
	r.clearExtensionLocked(extensionID)
	r.revision++
	return r.revision, nil
}

func (r *Registry) ExtensionSnapshot(extensionID string) (ExtensionSnapshot, bool) {
	if r == nil {
		return ExtensionSnapshot{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	artifact, ok := r.extensionArtifacts[extensionID]
	if !ok {
		return ExtensionSnapshot{Revision: r.revision}, false
	}
	result := ExtensionSnapshot{Revision: r.revision, Artifact: artifact}
	for _, list := range r.contributions {
		for _, item := range list {
			if item.ExtensionID == extensionID {
				result.Contributions = append(result.Contributions, item)
			}
		}
	}
	sortPageContributions(result.Contributions)
	return result, true
}

func (r *Registry) Revision() uint64 {
	if r == nil {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.revision
}

func (r *Registry) rebuildOwnersLocked() {
	r.signatureOwners = map[string]string{}
	for _, cr := range r.compiledAdds {
		r.signatureOwners[cr.Signature] = cr.Contribution.ExtensionID
	}
	SortCompiledRoutes(r.compiledAdds)
}

// Resolve 解析页面 id 的当前提供者；失败时回退 core。
// 绑定的 version/digest/contract 与在线贡献、核心目录任一不匹配立即回退。
func (r *Registry) Resolve(_ context.Context, pageID string) (ResolvedPage, error) {
	core, err := ResolveCore(pageID)
	if err != nil {
		return ResolvedPage{}, err
	}
	if r == nil {
		return core, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resolveLocked(core), nil
}

func (r *Registry) resolveLocked(core ResolvedPage) ResolvedPage {
	pageID := core.Page.ID
	binding, ok := r.bindings[pageID]
	if !ok || strings.TrimSpace(binding.ExtensionID) == "" || binding.ExtensionID == ProviderCore {
		return core
	}
	list := r.contributions[pageID]
	for _, c := range list {
		if c.ExtensionID != binding.ExtensionID || c.ID != binding.ContributionID {
			continue
		}
		if !r.contributionAdmittedLocked(c) {
			return core
		}
		// 精确匹配 version / digest / contract
		if binding.Version != "" && c.Version != "" && binding.Version != c.Version {
			r.logWarn("pages.resolve version mismatch, fallback core",
				"pageId", pageID, "binding", binding.Version, "contrib", c.Version)
			return core
		}
		if binding.PackageDigest != "" && c.PackageDigest != "" && binding.PackageDigest != c.PackageDigest {
			r.logWarn("pages.resolve digest mismatch, fallback core",
				"pageId", pageID, "binding", binding.PackageDigest, "contrib", c.PackageDigest)
			return core
		}
		// contract：binding、贡献、核心目录三者一致
		coreContract := core.Page.ContractVersion
		contribContract := c.Contract
		bindContract := binding.ContractVersion
		if coreContract != "" {
			if contribContract != "" && contribContract != coreContract {
				r.logWarn("pages.resolve contrib contract != core, fallback",
					"pageId", pageID, "contrib", contribContract, "core", coreContract)
				return core
			}
			if bindContract != "" && bindContract != coreContract {
				r.logWarn("pages.resolve binding contract != core, fallback",
					"pageId", pageID, "binding", bindContract, "core", coreContract)
				return core
			}
		}
		if bindContract != "" && contribContract != "" && bindContract != contribContract {
			r.logWarn("pages.resolve binding contract != contrib, fallback",
				"pageId", pageID, "binding", bindContract, "contrib", contribContract)
			return core
		}
		// 模板路径与 data schema 仅从已注册贡献读取，忽略客户端伪造
		return ResolvedPage{
			Page:              core.Page,
			Provider:          binding.ExtensionID,
			ExtensionID:       binding.ExtensionID,
			ContributionID:    binding.ContributionID,
			Action:            string(ActionReplace),
			Fallback:          false,
			TemplatePath:      c.Template,
			DataSource:        c.DataSource,
			DataRoute:         c.DataRoute,
			DataSchema:        c.DataSchema,
			Version:           c.Version,
			PackageDigest:     c.PackageDigest,
			RuntimeInstanceID: c.RuntimeInstanceID,
		}
	}
	// 绑定存在但贡献已卸载 → core
	r.logWarn("pages.resolve contribution offline, fallback core", "pageId", pageID, "extensionId", binding.ExtensionID)
	return core
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// AddedPages 返回插件/主题 add 的动态路径贡献（稳定排序）。
func (r *Registry) AddedPages() []PageContribution {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PageContribution, 0, len(r.compiledAdds))
	for _, cr := range r.compiledAdds {
		if r.contributionAdmittedLocked(cr.Contribution) {
			out = append(out, cr.Contribution)
		}
	}
	return out
}

// ResolveAddedPath 按公开路径确定性匹配 add 贡献（locale 已剥离）。
// 优先静态、其次参数、最后 catch-all；不依赖 map 迭代。
func (r *Registry) ResolveAddedPath(requestPath string) (PageContribution, bool) {
	match, ok := r.ResolveAddedPathMatch(requestPath)
	if !ok {
		return PageContribution{}, false
	}
	return match.Contribution, true
}

// ResolveAddedPathMatch 返回贡献 + 实际 route params。
func (r *Registry) ResolveAddedPathMatch(requestPath string) (RouteMatch, bool) {
	if r == nil {
		return RouteMatch{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	visible := make([]CompiledRoute, 0, len(r.compiledAdds))
	for _, route := range r.compiledAdds {
		if r.contributionAdmittedLocked(route.Contribution) {
			visible = append(visible, route)
		}
	}
	return MatchRequestPath(visible, requestPath)
}

func (r *Registry) contributionAdmittedLocked(contribution PageContribution) bool {
	if contribution.RuntimeInstanceID == "" || r.admission == nil {
		return true
	}
	return r.admission(RuntimeArtifact{
		ExtensionID: contribution.ExtensionID, ExtensionVersion: contribution.Version,
		PackageDigest: contribution.PackageDigest, RuntimeInstanceID: contribution.RuntimeInstanceID,
	})
}

func validRuntimeArtifact(artifact RuntimeArtifact, requireRuntime bool) bool {
	if artifact.ExtensionID == "" || artifact.ExtensionID != strings.TrimSpace(artifact.ExtensionID) ||
		artifact.ExtensionVersion == "" || artifact.ExtensionVersion != strings.TrimSpace(artifact.ExtensionVersion) ||
		artifact.PackageDigest == "" || artifact.PackageDigest != strings.TrimSpace(artifact.PackageDigest) {
		return false
	}
	return !requireRuntime || (artifact.RuntimeInstanceID != "" && artifact.RuntimeInstanceID == strings.TrimSpace(artifact.RuntimeInstanceID))
}

func pageArtifactFromContributions(extensionID string, items []PageContribution) RuntimeArtifact {
	artifact := RuntimeArtifact{ExtensionID: extensionID}
	if len(items) > 0 {
		artifact.ExtensionVersion = items[0].Version
		artifact.PackageDigest = items[0].PackageDigest
		artifact.RuntimeInstanceID = items[0].RuntimeInstanceID
	}
	return artifact
}

func sortPageContributions(items []PageContribution) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}
		if items[i].Action != items[j].Action {
			return items[i].Action < items[j].Action
		}
		return items[i].Path < items[j].Path
	})
}

// Snapshot 返回当前贡献的只读拷贝（测试/诊断）。
func (r *Registry) Snapshot() (replace map[string][]PageContribution, added map[string]PageContribution) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	replace = make(map[string][]PageContribution, len(r.contributions))
	for k, v := range r.contributions {
		replace[k] = append([]PageContribution(nil), v...)
	}
	added = make(map[string]PageContribution, len(r.compiledAdds))
	for _, cr := range r.compiledAdds {
		added[cr.Pattern] = cr.Contribution
	}
	return replace, added
}

func validateContribution(item PageContribution) error {
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("%w: missing id", ErrInvalidContribution)
	}
	if strings.TrimSpace(item.ExtensionID) == "" {
		return fmt.Errorf("%w: missing extensionId", ErrInvalidContribution)
	}
	switch item.Action {
	case ActionReplace:
		if strings.TrimSpace(item.Target) == "" {
			return fmt.Errorf("%w: replace needs target", ErrInvalidContribution)
		}
	case ActionAdd:
		if strings.TrimSpace(item.Path) == "" {
			return fmt.Errorf("%w: add needs path", ErrInvalidContribution)
		}
	default:
		return fmt.Errorf("%w: action", ErrInvalidContribution)
	}
	return nil
}

func normalizePublicPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// 折叠重复斜杠
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	return p
}
