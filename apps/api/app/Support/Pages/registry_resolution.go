package pages

import (
	"context"
	"strings"
)

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
			Page:                 core.Page,
			Provider:             binding.ExtensionID,
			ExtensionID:          binding.ExtensionID,
			ContributionID:       binding.ContributionID,
			Action:               string(ActionReplace),
			Fallback:             false,
			TemplatePath:         c.Template,
			Label:                c.Label,
			View:                 string(NormalizePageView(string(c.View))),
			Component:            c.ComponentID,
			SchemaDescriptorFile: c.SchemaDescriptorFile,
			SchemaContractRef:    c.SchemaContractRef,
			DataSource:           c.DataSource,
			DataRoute:            c.DataRoute,
			DataSchema:           c.DataSchema,
			Version:              c.Version,
			PackageDigest:        c.PackageDigest,
			RuntimeInstanceID:    c.RuntimeInstanceID,
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
