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
	ErrNotThemeable        = errors.New("pages: page does not allow a theme template")
	ErrReservedPath        = errors.New("pages: path is reserved")
	ErrConflictProvider    = errors.New("pages: multiple providers claim page")
	ErrApprovalRequired    = errors.New("pages: replace requires approval")
	ErrInvalidContribution = errors.New("pages: invalid page contribution")
	ErrInvalidAccess       = errors.New("pages: invalid access")
	ErrInvalidLayout       = errors.New("pages: invalid page layout")
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
	ID       string             `json:"id"`
	Action   ContributionAction `json:"action"`
	Target   string             `json:"target,omitempty"` // replace 目标 page id
	Path     string             `json:"path,omitempty"`   // add 路径
	Template string             `json:"template,omitempty"`
	// Label 页面展示标题（可选）：theme.json 声明，resolve-path 透传给前端做标题兜底。
	Label string `json:"label,omitempty"`
	// View: template | component（空值等价 template）。
	View PageView `json:"view,omitempty"`
	// ComponentID 指向包内已声明的公开 L2 组件；view=component 时必填。
	ComponentID string `json:"componentId,omitempty"`
	// SchemaDescriptorFile / SchemaContractRef 只在 view=schema 时有值：
	// 描述符文件（kind=schema packageFile id）与它必须满足的精确 schema 引用。
	SchemaDescriptorFile string     `json:"schemaDescriptorFile,omitempty"`
	SchemaContractRef    string     `json:"schemaContractRef,omitempty"`
	Contract             string     `json:"contract,omitempty"`
	Access               Access     `json:"access,omitempty"`
	Permission           string     `json:"permission,omitempty"` // access=permission 时必填
	Layout               PageLayout `json:"layout,omitempty"`     // site | bare（默认 bare）
	DataSource           string     `json:"dataSource,omitempty"` // core | plugin
	DataRoute            string     `json:"dataRoute,omitempty"`
	DataSchema           string     `json:"dataSchema,omitempty"` // 可选 JSON Schema 相对路径
	ExtensionID          string     `json:"extensionId"`
	Version              string     `json:"version"`
	PackageDigest        string     `json:"packageDigest,omitempty"`
	RuntimeInstanceID    string     `json:"runtimeInstanceId,omitempty"`
	// RouteSignature 注册时计算的语义签名（参数名无关）。
	RouteSignature string `json:"routeSignature,omitempty"`
}

// 审批来源：'admin' 由 super_admin 显式批准（ApprovedBy 必须是真实用户）；
// 'host' 只用于 Host 自带的受保护默认主题在首次安装/启动修复时的自动批准
// （ApprovedBy 为空）。上传与第三方主题永远只能是 'admin'。
const (
	BindingApprovalAdmin = "admin"
	BindingApprovalHost  = "host"
)

// bindingApproval 是一次主题切换要写进每条 replace 绑定的审批来源。
type bindingApproval struct {
	Source string
	Actor  int64
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
	// ApprovalSource 记录谁批准了这次核心页替换（见 BindingApproval* 常量）。
	// 它是审计属性，不参与制品身份比对（sameProviderArtifact）。
	ApprovalSource string `json:"approvalSource,omitempty"`
	TemplatePath   string `json:"templatePath,omitempty"`
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
		items, err := prepareThemeContributions(extensionID, contributions)
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
	return r.registerContributions(extensionID, items, contributionSourceExtension)
}

// RegisterThemeContributions 注册主题呈现候选。主题可以为 Themeable 页面
// 提供受限 L1 模板，但不能借此获得插件替换权限。
func (r *Registry) RegisterThemeContributions(extensionID string, items []PageContribution) error {
	return r.registerContributions(extensionID, items, contributionSourceTheme)
}

func (r *Registry) registerContributions(extensionID string, items []PageContribution, source contributionSource) error {
	prepared, err := prepareContributionsFor(extensionID, items, source)
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

// PreflightContributions 仅校验贡献是否可注册，不修改状态（激活预检用）。
// replaceExtensionID 非空时，预检按「替换该扩展」的最终状态计算（主题切换）。
func (r *Registry) PreflightContributions(extensionID string, items []PageContribution) error {
	return r.PreflightContributionsReplacing(extensionID, items, extensionID)
}

// PreflightThemeContributionsReplacing 校验主题最终态，允许 Themeable 页面
// 使用 Host 审核过的岛，同时保留普通扩展的 Replaceable 限制。
func (r *Registry) PreflightThemeContributionsReplacing(extensionID string, items []PageContribution, ignoreExtensionIDs ...string) error {
	return r.preflightContributionsReplacing(extensionID, items, contributionSourceTheme, ignoreExtensionIDs...)
}

// PreflightContributionsReplacing 按「忽略 ignoreExtensionIDs 后的最终状态」校验。
// 主题激活：ignore = [newTheme, oldTheme]，使新旧主题同路径可切换。
func (r *Registry) PreflightContributionsReplacing(extensionID string, items []PageContribution, ignoreExtensionIDs ...string) error {
	return r.preflightContributionsReplacing(extensionID, items, contributionSourceExtension, ignoreExtensionIDs...)
}

func (r *Registry) preflightContributionsReplacing(extensionID string, items []PageContribution, source contributionSource, ignoreExtensionIDs ...string) error {
	prepared, err := prepareContributionsFor(extensionID, items, source)
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
