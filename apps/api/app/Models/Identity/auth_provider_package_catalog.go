package identity

import (
	"context"
	"strings"
)

// AuthProviderPackageCandidate 是 Host 扩展/包目录中声明的 auth 提供方只读投影。
// 用于管理端 discovery；不授予 trust/enable 权威，也不可执行。
//
// live Identity Registry 仍是可执行权威：只有 Registry 中带 RuntimeInstanceID
// 的条目才能 start/complete/probe 到真实进程。
type AuthProviderPackageCandidate struct {
	ProviderID            string
	Kind                  string
	ContractVersion       string
	Priority              int
	Operations            []string
	OwnerExtensionID      string
	OwnerExtensionVersion string
	OwnerPackageDigest    string
	// VersionID 为扩展活动/暂存版本 id（若可知）；0 表示仅包目录元数据。
	VersionID int64
	Label     string
	// LabelLocales 插件声明的多语文案；Host 按 Accept-Language 解析。
	LabelLocales map[string]string
	Icon         string
	// Trusted 表示 exact-artifact 可执行信任已授予（或 Host 判定无需额外 grant）。
	// 权威仍属 extension lifecycle；此处仅供管理端展示。
	Trusted bool
	// Enabled 表示扩展 StatusEnabled（进程可能尚未发布到 Registry）。
	Enabled bool
	// Status 为 installed|enabled|disabled 等扩展生命周期状态。
	Status string
	// Source 为 builtin|uploaded 等包来源。
	Source string
}

// AuthProviderPackageCatalog 从 Host 扩展/包目录列出声明了 auth 的候选。
// 实现必须只读、fail-closed，且不得把 staged catalog 写入 live Registry。
type AuthProviderPackageCatalog interface {
	ListAuthProviderCandidates(ctx context.Context) ([]AuthProviderPackageCandidate, error)
}

// StaticAuthProviderPackageCatalog 测试用静态目录。
type StaticAuthProviderPackageCatalog struct {
	Items []AuthProviderPackageCandidate
	Err   error
}

func (c StaticAuthProviderPackageCatalog) ListAuthProviderCandidates(context.Context) ([]AuthProviderPackageCandidate, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	out := make([]AuthProviderPackageCandidate, 0, len(c.Items))
	for _, item := range c.Items {
		cloned := item
		if item.LabelLocales != nil {
			cloned.LabelLocales = make(map[string]string, len(item.LabelLocales))
			for k, v := range item.LabelLocales {
				cloned.LabelLocales[k] = v
			}
		}
		if item.Operations != nil {
			cloned.Operations = append([]string(nil), item.Operations...)
		}
		out = append(out, cloned)
	}
	return out, nil
}

// NormalizeAuthProviderPackageCandidate 规范化候选字段（小写 id、trim）。
func NormalizeAuthProviderPackageCandidate(in AuthProviderPackageCandidate) AuthProviderPackageCandidate {
	in.ProviderID = strings.ToLower(strings.TrimSpace(in.ProviderID))
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	in.ContractVersion = strings.TrimSpace(in.ContractVersion)
	in.OwnerExtensionID = strings.ToLower(strings.TrimSpace(in.OwnerExtensionID))
	in.OwnerExtensionVersion = strings.TrimSpace(in.OwnerExtensionVersion)
	in.OwnerPackageDigest = strings.ToLower(strings.TrimSpace(in.OwnerPackageDigest))
	in.Label = strings.TrimSpace(in.Label)
	in.Icon = strings.TrimSpace(in.Icon)
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	in.Source = strings.ToLower(strings.TrimSpace(in.Source))
	if len(in.Operations) > 0 {
		ops := make([]string, 0, len(in.Operations))
		for _, op := range in.Operations {
			op = strings.ToLower(strings.TrimSpace(op))
			if op != "" {
				ops = append(ops, op)
			}
		}
		in.Operations = ops
	}
	if len(in.LabelLocales) > 0 {
		locales := make(map[string]string, len(in.LabelLocales))
		for k, v := range in.LabelLocales {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k != "" && v != "" {
				locales[k] = v
			}
		}
		in.LabelLocales = locales
	}
	return in
}
