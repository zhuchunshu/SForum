package pages

import (
	"context"
	"fmt"
	"strings"
)

// ProviderListItem describes the active provider and replacement candidates
// for one Core page in the administrative provider selector.
type ProviderListItem struct {
	Page            PageDefinition     `json:"page"`
	Provider        string             `json:"provider"`
	ExtensionID     string             `json:"extensionId,omitempty"`
	ContributionID  string             `json:"contributionId,omitempty"`
	ContractVersion string             `json:"contractVersion,omitempty"`
	Candidates      []PageContribution `json:"candidates,omitempty"`
}

func (r *Registry) ListProviders(_ context.Context) ([]ProviderListItem, error) {
	items := make([]ProviderListItem, 0, len(coreCatalog))
	if r == nil {
		for _, page := range Catalog() {
			items = append(items, ProviderListItem{Page: page, Provider: ProviderCore, ContractVersion: page.ContractVersion})
		}
		return items, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, page := range Catalog() {
		core, _ := ResolveCore(page.ID)
		resolved := r.resolveLocked(core)
		item := ProviderListItem{
			Page:            page,
			Provider:        resolved.Provider,
			ExtensionID:     resolved.ExtensionID,
			ContractVersion: page.ContractVersion,
		}
		item.Candidates = append([]PageContribution(nil), r.contributions[page.ID]...)
		if binding, ok := r.bindings[page.ID]; ok {
			item.ContributionID = binding.ContributionID
			if binding.ContractVersion != "" {
				item.ContractVersion = binding.ContractVersion
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// ApproveReplace persists an exact super-admin-approved replacement binding.
// TemplatePath is always taken from the registered contribution.
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
	reqContract := strings.TrimSpace(binding.ContractVersion)
	if reqContract == "" {
		return fmt.Errorf("%w: contractVersion required", ErrContractMismatch)
	}
	if page.ContractVersion != "" && reqContract != page.ContractVersion {
		return fmt.Errorf("%w: request contract %q != core %q", ErrContractMismatch, reqContract, page.ContractVersion)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	var matched PageContribution
	matchedFound := false
	for _, contribution := range r.contributions[binding.PageID] {
		if contribution.ExtensionID == binding.ExtensionID && contribution.ID == binding.ContributionID {
			matched = contribution
			matchedFound = true
			break
		}
	}
	if !matchedFound {
		return fmt.Errorf("%w: contribution not registered", ErrInvalidContribution)
	}
	if matched.Version != binding.Version {
		return fmt.Errorf("%w: version mismatch", ErrInvalidContribution)
	}
	if matched.PackageDigest != binding.PackageDigest {
		return fmt.Errorf("%w: package digest mismatch", ErrInvalidContribution)
	}
	if strings.TrimSpace(matched.Contract) == "" {
		return fmt.Errorf("%w: contribution missing contract", ErrContractMismatch)
	}
	if matched.Contract != reqContract {
		return fmt.Errorf("%w: request contract %q != contribution %q", ErrContractMismatch, reqContract, matched.Contract)
	}
	if page.ContractVersion != "" && matched.Contract != page.ContractVersion {
		return fmt.Errorf("%w: contribution contract %q != core %q", ErrContractMismatch, matched.Contract, page.ContractVersion)
	}

	binding.TemplatePath = matched.Template
	binding.ContractVersion = matched.Contract
	if err := r.store.UpsertBinding(ctx, binding); err != nil {
		return err
	}
	next := cloneBindings(r.bindings)
	next[binding.PageID] = cloneBinding(binding)
	r.bindings = next
	r.revision++
	return nil
}

// RestoreCore removes an operator binding so the Core page becomes active.
func (r *Registry) RestoreCore(ctx context.Context, pageID string) error {
	if r == nil || r.store == nil {
		return nil
	}
	if _, ok := Find(pageID); !ok {
		return ErrUnknownPage
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.store.DeleteBinding(ctx, pageID); err != nil {
		return err
	}
	next := cloneBindings(r.bindings)
	delete(next, pageID)
	r.bindings = next
	r.revision++
	return nil
}
