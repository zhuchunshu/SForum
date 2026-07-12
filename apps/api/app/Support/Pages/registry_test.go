package pages

import (
	"context"
	"testing"
)

func TestRegistryReplaceAndRestore(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := context.Background()

	err := reg.RegisterContributions("demo.theme", []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "abc",
	}})
	if err != nil {
		t.Fatal(err)
	}

	// 未批准 → core
	r, err := reg.Resolve(ctx, "forum.home")
	if err != nil || r.Provider != ProviderCore {
		t.Fatalf("expected core before approval, got %#v err=%v", r, err)
	}

	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "abc", ContractVersion: "sforum.page.home@1", ApprovedBy: 1,
	}); err != nil {
		t.Fatal(err)
	}
	r, err = reg.Resolve(ctx, "forum.home")
	if err != nil || r.Provider != "demo.theme" {
		t.Fatalf("expected demo.theme, got %#v err=%v", r, err)
	}

	if err := reg.RestoreCore(ctx, "forum.home"); err != nil {
		t.Fatal(err)
	}
	r, _ = reg.Resolve(ctx, "forum.home")
	if r.Provider != ProviderCore {
		t.Fatalf("expected core after restore, got %s", r.Provider)
	}
}

func TestRegistryRejectsReservedAdd(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	err := reg.RegisterContributions("x", []PageContribution{{
		ID: "x.admin", Action: ActionAdd, Path: "/admin/hack", ExtensionID: "x",
	}})
	if err == nil {
		t.Fatal("expected reserved path error")
	}
}

func TestRegistryRejectsNonReplaceable(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	err := reg.RegisterContributions("x", []PageContribution{{
		ID: "x.mod", Action: ActionReplace, Target: "moderation.review", ExtensionID: "x",
	}})
	if err == nil {
		t.Fatal("expected not replaceable")
	}
}

func TestDigestMismatchFallsBack(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := context.Background()
	_ = reg.RegisterContributions("demo.theme", []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "new",
	}})
	_ = store.UpsertBinding(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "old", ContractVersion: "sforum.page.home@1",
	})
	r, _ := reg.Resolve(ctx, "forum.home")
	if r.Provider != ProviderCore {
		t.Fatalf("digest mismatch should fallback core, got %s", r.Provider)
	}
}

func TestApproveReplaceRequiresActorAndExactDigest(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := context.Background()
	_ = reg.RegisterContributions("demo.theme", []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "abc",
	}})
	// 无 actor
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "abc", ContractVersion: "sforum.page.home@1", ApprovedBy: 0,
	}); err == nil {
		t.Fatal("expected approvedBy required")
	}
	// digest 不匹配
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "wrong", ContractVersion: "sforum.page.home@1", ApprovedBy: 1,
	}); err == nil {
		t.Fatal("expected digest mismatch")
	}
	// 空 digest 不得自动填充
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "", ContractVersion: "sforum.page.home@1", ApprovedBy: 1,
	}); err == nil {
		t.Fatal("expected missing digest reject")
	}
	// 缺少 contract
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "abc", ContractVersion: "", ApprovedBy: 1,
	}); err == nil {
		t.Fatal("expected missing contract reject")
	}
	// 错误 contract
	if err := reg.ApproveReplace(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "abc", ContractVersion: "evil@9", ApprovedBy: 1,
	}); err == nil {
		t.Fatal("expected contract mismatch")
	}
}

func TestRegisterContributionsAtomicOnError(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	_ = reg.RegisterContributions("demo.theme", []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1", PackageDigest: "d",
	}})
	// 第二条非法：整批失败，旧贡献应仍在
	err := reg.RegisterContributions("demo.theme", []PageContribution{
		{ID: "demo.home2", Action: ActionReplace, Target: "forum.home", Contract: "sforum.page.home@1", Version: "2", PackageDigest: "e"},
		{ID: "bad", Action: ActionAdd, Path: "/admin/x", Version: "2", PackageDigest: "e"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	_, added := reg.Snapshot()
	if len(added) != 0 {
		t.Fatalf("should not partial-register adds: %#v", added)
	}
	// 旧 replace 候选仍在
	list, _ := reg.ListProviders(context.Background())
	found := false
	for _, item := range list {
		if item.Page.ID == "forum.home" {
			for _, c := range item.Candidates {
				if c.ID == "demo.home" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("old contribution should remain after failed re-register")
	}
}

func TestResolveAddedPath(t *testing.T) {
	reg := NewRegistry(NewMemoryStore())
	_ = reg.RegisterContributions("plug", []PageContribution{{
		ID: "plug.docs", Action: ActionAdd, Path: "/docs/:slug",
		ExtensionID: "plug", Version: "1", PackageDigest: "d", Access: AccessPublic,
	}})
	c, ok := reg.ResolveAddedPath("/docs/hello")
	if !ok || c.ID != "plug.docs" {
		t.Fatalf("expected match, got %#v ok=%v", c, ok)
	}
	if _, ok := reg.ResolveAddedPath("/admin/x"); ok {
		t.Fatal("reserved must not match")
	}
}
