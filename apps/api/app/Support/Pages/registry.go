package pages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrUnknownPage       = errors.New("pages: unknown page")
	ErrNotReplaceable    = errors.New("pages: page is not replaceable")
	ErrReservedPath      = errors.New("pages: path is reserved")
	ErrConflictProvider  = errors.New("pages: multiple providers claim page")
	ErrApprovalRequired  = errors.New("pages: replace requires approval")
	ErrInvalidContribution = errors.New("pages: invalid page contribution")
)

// ContributionAction 主题/插件对页面的贡献动作。
type ContributionAction string

const (
	ActionAdd     ContributionAction = "add"
	ActionReplace ContributionAction = "replace"
)

// PageContribution 扩展声明的页面贡献（来自 theme.json / manifest）。
type PageContribution struct {
	ID           string             `json:"id"`
	Action       ContributionAction `json:"action"`
	Target       string             `json:"target,omitempty"` // replace 目标 page id
	Path         string             `json:"path,omitempty"`   // add 路径
	Template     string             `json:"template,omitempty"`
	Contract     string             `json:"contract,omitempty"`
	Access       Access             `json:"access,omitempty"`
	DataSource   string             `json:"dataSource,omitempty"` // core | plugin
	DataRoute    string             `json:"dataRoute,omitempty"`
	ExtensionID  string             `json:"extensionId"`
	Version      string             `json:"version"`
	PackageDigest string            `json:"packageDigest,omitempty"`
}

// ProviderBinding 管理员选择的生效提供者（或自动选中的唯一贡献）。
// 批准必须绑定 extension id、version、package digest、contribution id 与 contract version。
type ProviderBinding struct {
	PageID         string `json:"pageId"`
	ExtensionID    string `json:"extensionId"`
	ContributionID string `json:"contributionId"`
	Version        string `json:"version"`
	PackageDigest  string `json:"packageDigest"`
	ContractVersion string `json:"contractVersion,omitempty"`
	ApprovedBy     int64  `json:"approvedBy,omitempty"`
	TemplatePath   string `json:"templatePath,omitempty"`
}

// Store 持久化绑定与审批（可实现为内存或 Postgres）。
type Store interface {
	ListBindings(ctx context.Context) ([]ProviderBinding, error)
	GetBinding(ctx context.Context, pageID string) (ProviderBinding, bool, error)
	UpsertBinding(ctx context.Context, binding ProviderBinding) error
	DeleteBinding(ctx context.Context, pageID string) error
}

// Registry 解析页面提供者：目录 + 贡献 + 管理员绑定。
type Registry struct {
	store Store

	mu            sync.RWMutex
	contributions map[string][]PageContribution // pageId or contribution id for add
	addedPaths    map[string]PageContribution   // path -> contribution
}

// NewRegistry 创建注册表；store 可为 nil（仅 core）。
func NewRegistry(store Store) *Registry {
	return &Registry{
		store:         store,
		contributions: map[string][]PageContribution{},
		addedPaths:    map[string]PageContribution{},
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
	// 路径冲突：与其它扩展已注册路径冲突则拒绝（本扩展旧路径会先清掉）
	for _, item := range prepared {
		if item.Action != ActionAdd {
			continue
		}
		path := normalizePublicPath(item.Path)
		if existing, ok := r.addedPaths[path]; ok && existing.ExtensionID != extensionID {
			return fmt.Errorf("%w: path %s", ErrConflictProvider, path)
		}
	}
	r.clearExtensionLocked(extensionID)
	for _, item := range prepared {
		switch item.Action {
		case ActionReplace:
			target := strings.TrimSpace(item.Target)
			r.contributions[target] = append(r.contributions[target], item)
		case ActionAdd:
			path := normalizePublicPath(item.Path)
			r.addedPaths[path] = item
			r.contributions[item.ID] = append(r.contributions[item.ID], item)
		}
	}
	return nil
}

// prepareContributions 校验并规范化贡献列表（不修改 Registry）。
func prepareContributions(extensionID string, items []PageContribution) ([]PageContribution, error) {
	prepared := make([]PageContribution, 0, len(items))
	seenAddPaths := map[string]struct{}{}
	for _, item := range items {
		item.ExtensionID = extensionID
		if err := validateContribution(item); err != nil {
			return nil, err
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
			// constrained 页面（login 等）允许 replace 外观，但合同版本必须匹配目录
			if item.Contract == "" {
				item.Contract = page.ContractVersion
			}
		case ActionAdd:
			path := normalizePublicPath(item.Path)
			item.Path = path
			if IsReservedPath(path) {
				return nil, fmt.Errorf("%w: %s", ErrReservedPath, path)
			}
			// 与核心目录路径冲突
			if _, ok := MatchPath(path); ok {
				return nil, fmt.Errorf("%w: path %s collides with core page", ErrConflictProvider, path)
			}
			if _, ok := seenAddPaths[path]; ok {
				return nil, fmt.Errorf("%w: duplicate add path %s", ErrInvalidContribution, path)
			}
			seenAddPaths[path] = struct{}{}
		default:
			return nil, fmt.Errorf("%w: action %q", ErrInvalidContribution, item.Action)
		}
		prepared = append(prepared, item)
	}
	return prepared, nil
}

// PreflightContributions 仅校验贡献是否可注册，不修改状态（激活预检用）。
func (r *Registry) PreflightContributions(extensionID string, items []PageContribution) error {
	prepared, err := prepareContributions(extensionID, items)
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range prepared {
		if item.Action != ActionAdd {
			continue
		}
		path := normalizePublicPath(item.Path)
		if existing, ok := r.addedPaths[path]; ok && existing.ExtensionID != extensionID {
			return fmt.Errorf("%w: path %s", ErrConflictProvider, path)
		}
	}
	return nil
}

// ClearExtension 移除扩展的全部页面贡献。
func (r *Registry) ClearExtension(extensionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearExtensionLocked(extensionID)
}

func (r *Registry) clearExtensionLocked(extensionID string) {
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
	for path, c := range r.addedPaths {
		if c.ExtensionID == extensionID {
			delete(r.addedPaths, path)
		}
	}
}

// Resolve 解析页面 id 的当前提供者；失败时回退 core。
func (r *Registry) Resolve(ctx context.Context, pageID string) (ResolvedPage, error) {
	core, err := ResolveCore(pageID)
	if err != nil {
		return ResolvedPage{}, err
	}
	if r == nil || r.store == nil {
		return core, nil
	}
	binding, ok, err := r.store.GetBinding(ctx, pageID)
	if err != nil {
		return core, nil // 存储故障时回退 core
	}
	if !ok || strings.TrimSpace(binding.ExtensionID) == "" || binding.ExtensionID == ProviderCore {
		return core, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 确认贡献仍在线且 digest/version 匹配（更新后需重新审批）
	list := r.contributions[pageID]
	for _, c := range list {
		if c.ExtensionID == binding.ExtensionID && c.ID == binding.ContributionID {
			if binding.PackageDigest != "" && c.PackageDigest != "" && binding.PackageDigest != c.PackageDigest {
				// digest 变化：失效绑定，回退 core
				return core, nil
			}
			return ResolvedPage{
				Page:           core.Page,
				Provider:       binding.ExtensionID,
				ExtensionID:    binding.ExtensionID,
				ContributionID: binding.ContributionID,
				Action:         string(ActionReplace),
				Fallback:       false,
				TemplatePath:   firstNonEmpty(binding.TemplatePath, c.Template),
				DataSource:     c.DataSource,
				DataRoute:      c.DataRoute,
			}, nil
		}
	}
	// 绑定存在但贡献已卸载 → core
	return core, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ListProviders 管理端只读列表：每个核心页 + 当前 provider + 候选贡献。
type ProviderListItem struct {
	Page           PageDefinition     `json:"page"`
	Provider       string             `json:"provider"`
	ExtensionID    string             `json:"extensionId,omitempty"`
	ContributionID string             `json:"contributionId,omitempty"`
	Candidates     []PageContribution `json:"candidates,omitempty"`
}

func (r *Registry) ListProviders(ctx context.Context) ([]ProviderListItem, error) {
	items := make([]ProviderListItem, 0, len(coreCatalog))
	for _, page := range Catalog() {
		resolved, _ := r.Resolve(ctx, page.ID)
		item := ProviderListItem{
			Page:        page,
			Provider:    resolved.Provider,
			ExtensionID: resolved.ExtensionID,
		}
		if r != nil {
			r.mu.RLock()
			item.Candidates = append([]PageContribution(nil), r.contributions[page.ID]...)
			r.mu.RUnlock()
		}
		if r != nil && r.store != nil {
			if b, ok, _ := r.store.GetBinding(ctx, page.ID); ok {
				item.ContributionID = b.ContributionID
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// ApproveReplace 超级管理员批准 replace 并写入绑定。
// 必须提供 ApprovedBy>0，且 version/packageDigest 与在线贡献精确匹配（禁止空 digest 自动填充）。
func (r *Registry) ApproveReplace(ctx context.Context, binding ProviderBinding) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("pages: registry store not configured")
	}
	if binding.ApprovedBy <= 0 {
		return fmt.Errorf("%w: approvedBy required", ErrApprovalRequired)
	}
	if strings.TrimSpace(binding.ExtensionID) == "" || strings.TrimSpace(binding.ContributionID) == "" {
		return fmt.Errorf("%w: extensionId and contributionId required", ErrInvalidContribution)
	}
	if strings.TrimSpace(binding.Version) == "" || strings.TrimSpace(binding.PackageDigest) == "" {
		return fmt.Errorf("%w: version and packageDigest required", ErrInvalidContribution)
	}
	page, ok := Find(binding.PageID)
	if !ok {
		return ErrUnknownPage
	}
	if !page.Replaceable {
		return ErrNotReplaceable
	}
	r.mu.RLock()
	var matched *PageContribution
	for i := range r.contributions[binding.PageID] {
		c := &r.contributions[binding.PageID][i]
		if c.ExtensionID == binding.ExtensionID && c.ID == binding.ContributionID {
			matched = c
			break
		}
	}
	r.mu.RUnlock()
	if matched == nil {
		return fmt.Errorf("%w: contribution not registered", ErrInvalidContribution)
	}
	// 精确绑定：升级后 digest 变化则旧批准不可复用
	if matched.Version != binding.Version {
		return fmt.Errorf("%w: version mismatch", ErrInvalidContribution)
	}
	if matched.PackageDigest != binding.PackageDigest {
		return fmt.Errorf("%w: package digest mismatch", ErrInvalidContribution)
	}
	if binding.TemplatePath == "" {
		binding.TemplatePath = matched.Template
	}
	if binding.ContractVersion == "" {
		binding.ContractVersion = firstNonEmpty(matched.Contract, page.ContractVersion)
	}
	// 合同版本必须与目录或贡献声明一致
	if page.ContractVersion != "" && binding.ContractVersion != "" &&
		matched.Contract != "" && matched.Contract != binding.ContractVersion &&
		matched.Contract != page.ContractVersion {
		return fmt.Errorf("%w: contract version mismatch", ErrInvalidContribution)
	}
	return r.store.UpsertBinding(ctx, binding)
}

// RestoreCore 一键恢复核心页面。
func (r *Registry) RestoreCore(ctx context.Context, pageID string) error {
	if r == nil || r.store == nil {
		return nil
	}
	if _, ok := Find(pageID); !ok {
		return ErrUnknownPage
	}
	return r.store.DeleteBinding(ctx, pageID)
}

// AddedPages 返回插件/主题 add 的动态路径贡献。
func (r *Registry) AddedPages() []PageContribution {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PageContribution, 0, len(r.addedPaths))
	for _, c := range r.addedPaths {
		out = append(out, c)
	}
	return out
}

// ResolveAddedPath 按公开路径匹配 add 贡献（locale 已剥离）。
func (r *Registry) ResolveAddedPath(requestPath string) (PageContribution, bool) {
	if r == nil {
		return PageContribution{}, false
	}
	path := normalizePublicPath(stripLocalePrefix(requestPath))
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 精确匹配
	if c, ok := r.addedPaths[path]; ok {
		return c, true
	}
	// 参数化路径：简单前缀段匹配（:param / *）
	for pattern, c := range r.addedPaths {
		if pathMatches(pattern, path) {
			return c, true
		}
	}
	return PageContribution{}, false
}

// Snapshot 返回当前贡献的只读拷贝（测试/诊断）。
func (r *Registry) Snapshot() (replace map[string][]PageContribution, added map[string]PageContribution) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	replace = make(map[string][]PageContribution, len(r.contributions))
	for k, v := range r.contributions {
		replace[k] = append([]PageContribution(nil), v...)
	}
	added = make(map[string]PageContribution, len(r.addedPaths))
	for k, v := range r.addedPaths {
		added[k] = v
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
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	return p
}
