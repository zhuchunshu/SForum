package pages

import (
	"context"
	"testing"
)

// BenchmarkThemeResolveV1Baseline 覆盖当前绑定、版本、摘要和契约三方匹配路径。
func BenchmarkThemeResolveV1Baseline(b *testing.B) {
	ctx := context.Background()
	store := NewMemoryStore()
	registry := NewRegistry(store)
	contribution := PageContribution{
		ID: "benchmark.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		ExtensionID: "benchmark.theme", Version: "1.0.0", PackageDigest: "benchmark-digest",
	}
	if err := registry.RegisterContributions(contribution.ExtensionID, []PageContribution{contribution}); err != nil {
		b.Fatal(err)
	}
	if err := registry.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: contribution.ExtensionID, ContributionID: contribution.ID,
		Version: contribution.Version, PackageDigest: contribution.PackageDigest,
		ContractVersion: contribution.Contract, ApprovedBy: 1,
	}); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		resolved, err := registry.Resolve(ctx, "forum.home")
		if err != nil || resolved.Provider != contribution.ExtensionID {
			b.Fatalf("resolve provider=%q err=%v", resolved.Provider, err)
		}
	}
}
