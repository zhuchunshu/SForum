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
type ProviderBinding struct {
	PageID        string `json:"pageId"`
	ExtensionID   string `json:"extensionId"`
	ContributionID string `json:"contributionId"`
	Version       string `json:"version"`
	PackageDigest string `json:"packageDigest"`
	ApprovedBy    int64  `json:"approvedBy,omitempty"`
	TemplatePath  string `json:"templatePath,omitempty"`
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
func (r *Registry) RegisterContributions(extensionID string, items []PageContribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 先清掉该扩展旧贡献
	r.clearExtensionLocked(extensionID)
	for _, item := range items {
		item.ExtensionID = extensionID
		if err := validateContribution(item); err != nil {
			return err
		}
		switch item.Action {
		case ActionReplace:
			target := strings.TrimSpace(item.Target)
			page, ok := Find(target)
			if !ok {
				return fmt.Errorf("%w: target %q", ErrUnknownPage, target)
			}
			if !page.Replaceable {
				return fmt.Errorf("%w: %s", ErrNotReplaceable, target)
			}
			r.contributions[target] = append(r.contributions[target], item)
		case ActionAdd:
			path := normalizePublicPath(item.Path)
			if IsReservedPath(path) {
				return fmt.Errorf("%w: %s", ErrReservedPath, path)
			}
			if existing, ok := r.addedPaths[path]; ok && existing.ExtensionID != extensionID {
				return fmt.Errorf("%w: path %s", ErrConflictProvider, path)
			}
			r.addedPaths[path] = item
			r.contributions[item.ID] = append(r.contributions[item.ID], item)
		default:
			return fmt.Errorf("%w: action %q", ErrInvalidContribution, item.Action)
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
func (r *Registry) ApproveReplace(ctx context.Context, binding ProviderBinding) error {
	if r == nil || r.store == nil {
		return fmt.Errorf("pages: registry store not configured")
	}
	page, ok := Find(binding.PageID)
	if !ok {
		return ErrUnknownPage
	}
	if !page.Replaceable {
		return ErrNotReplaceable
	}
	r.mu.RLock()
	found := false
	for _, c := range r.contributions[binding.PageID] {
		if c.ExtensionID == binding.ExtensionID && c.ID == binding.ContributionID {
			found = true
			if binding.Version == "" {
				binding.Version = c.Version
			}
			if binding.PackageDigest == "" {
				binding.PackageDigest = c.PackageDigest
			}
			if binding.TemplatePath == "" {
				binding.TemplatePath = c.Template
			}
			break
		}
	}
	r.mu.RUnlock()
	if !found {
		return fmt.Errorf("%w: contribution not registered", ErrInvalidContribution)
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
