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
		Template: "templates/home.html", ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "abc",
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
		Version: "1.0.0", PackageDigest: "abc", ApprovedBy: 1,
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
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "new",
	}})
	_ = store.UpsertBinding(ctx, ProviderBinding{
		PageID: "forum.home", ExtensionID: "demo.theme", ContributionID: "demo.home",
		Version: "1.0.0", PackageDigest: "old",
	})
	r, _ := reg.Resolve(ctx, "forum.home")
	if r.Provider != ProviderCore {
		t.Fatalf("digest mismatch should fallback core, got %s", r.Provider)
	}
}
