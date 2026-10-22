package pages

import (
	"context"
	"errors"
	"testing"
)

// Host 批准只属于受保护内置默认主题：绑定写 approval_source=host 且没有审批用户，
// 并且立刻让核心页解析到该主题。
func TestRegistryHostApprovedActivationPublishesBindingsWithoutActor(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := context.Background()

	items := []PageContribution{{
		ID: "default.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		ExtensionID: "sforum.default-theme", Version: "1.0.0", PackageDigest: "abc",
	}}
	if err := reg.ActivateThemeContributionsHostApproved(ctx, "sforum.default-theme", items); err != nil {
		t.Fatal(err)
	}
	binding, ok, err := store.GetBinding(ctx, "forum.home")
	if err != nil || !ok {
		t.Fatalf("host-approved binding missing: ok=%v err=%v", ok, err)
	}
	if binding.ApprovalSource != BindingApprovalHost || binding.ApprovedBy != 0 {
		t.Fatalf("binding provenance=%q actor=%d", binding.ApprovalSource, binding.ApprovedBy)
	}
	resolved, err := reg.Resolve(ctx, "forum.home")
	if err != nil || resolved.Provider != "sforum.default-theme" || resolved.Fallback {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}

	// 重启后（启动恢复路径：先加载持久化绑定，再重新登记贡献并对账）host 来源必须原样保留。
	restored := NewRegistry(store)
	if err := restored.RestoreBindings(ctx); err != nil {
		t.Fatal(err)
	}
	if err := restored.RestoreThemeContributions(ctx, "sforum.default-theme", items, nil); err != nil {
		t.Fatal(err)
	}
	restoredResolved, err := restored.Resolve(ctx, "forum.home")
	if err != nil || restoredResolved.Provider != "sforum.default-theme" {
		t.Fatalf("restored resolve=%#v err=%v", restoredResolved, err)
	}
	kept, ok, err := store.GetBinding(ctx, "forum.home")
	if err != nil || !ok || kept.ApprovalSource != BindingApprovalHost {
		t.Fatalf("restore dropped host provenance: %#v ok=%v err=%v", kept, ok, err)
	}
}

// 人工批准路径不受影响：approvedBy<=0 仍然被拒绝，admin 绑定带审批人。
func TestRegistryAdminApprovalStillRequiresActor(t *testing.T) {
	store := NewMemoryStore()
	reg := NewRegistry(store)
	ctx := context.Background()
	items := []PageContribution{{
		ID: "demo.home", Action: ActionReplace, Target: "forum.home",
		Template: "templates/home.html", Contract: "sforum.page.home@1",
		ExtensionID: "demo.theme", Version: "1.0.0", PackageDigest: "abc",
	}}
	if err := reg.ActivateThemeContributions(ctx, "demo.theme", items, "", 0); !errors.Is(err, ErrApprovalRequired) {
		t.Fatalf("approvedBy=0 error=%v", err)
	}
	if err := reg.ActivateThemeContributions(ctx, "demo.theme", items, "", 7); err != nil {
		t.Fatal(err)
	}
	binding, ok, err := store.GetBinding(ctx, "forum.home")
	if err != nil || !ok {
		t.Fatalf("admin binding missing: ok=%v err=%v", ok, err)
	}
	if binding.ApprovalSource != BindingApprovalAdmin || binding.ApprovedBy != 7 {
		t.Fatalf("binding provenance=%q actor=%d", binding.ApprovalSource, binding.ApprovedBy)
	}
}
