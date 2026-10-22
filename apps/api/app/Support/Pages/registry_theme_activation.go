package pages

import (
	"context"
	"fmt"
)

// 主题激活/恢复的发布语义：切换核心页 replace 绑定、写审批来源、重启对账。
// 从 registry.go 抽出，保持注册表本体聚焦在通用贡献注册与解析。

// ReplaceThemeContributions 主题激活专用：用新主题贡献替换旧活动主题，
// 按「最终状态」校验（新旧主题相同 add 路径允许；与其它扩展冲突仍失败）。
// 失败时内存 snapshot 完全不变。
func (r *Registry) ReplaceThemeContributions(newThemeID string, newItems []PageContribution, oldThemeID string) error {
	prepared, err := prepareThemeContributions(newThemeID, newItems)
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
	prepared, err := prepareThemeContributions(newThemeID, newItems)
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
	return r.activateThemeContributions(
		ctx, newThemeID, newItems,
		bindingApproval{Source: BindingApprovalAdmin, Actor: approvedBy}, oldThemeID,
	)
}

// ActivateThemeContributionsHostApproved is reserved for the protected built-in
// default theme during startup repair / legacy genesis: the Host itself ships
// the artifact and its Core page mapping, so the bindings are written with
// host provenance instead of inventing an approving user.
func (r *Registry) ActivateThemeContributionsHostApproved(
	ctx context.Context,
	newThemeID string,
	newItems []PageContribution,
	oldThemeIDs ...string,
) error {
	return r.activateThemeContributions(
		ctx, newThemeID, newItems, bindingApproval{Source: BindingApprovalHost}, oldThemeIDs...,
	)
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
	return r.activateThemeContributions(
		ctx, newThemeID, newItems,
		bindingApproval{Source: BindingApprovalAdmin, Actor: approvedBy}, oldThemeIDs...,
	)
}

func (r *Registry) activateThemeContributions(
	ctx context.Context,
	newThemeID string,
	newItems []PageContribution,
	approval bindingApproval,
	oldThemeIDs ...string,
) error {
	switch approval.Source {
	case BindingApprovalAdmin:
		if approval.Actor <= 0 {
			return ErrApprovalRequired
		}
	case BindingApprovalHost:
		if approval.Actor != 0 {
			return ErrApprovalRequired
		}
	default:
		return ErrApprovalRequired
	}
	prepared, err := prepareThemeContributions(newThemeID, newItems)
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
			Version: item.Version, PackageDigest: item.PackageDigest,
			ApprovedBy: approval.Actor, ApprovalSource: approval.Source,
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
	prepared, err := prepareThemeContributions(activeThemeID, items)
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
