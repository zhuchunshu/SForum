package systemtier

import "testing"

func TestSafeModeBypassAndCLIDisable(t *testing.T) {
	reg := New()
	if err := reg.Upsert(Member{ExtensionID: "sys.cache", Role: RoleCache, Priority: 10, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Upsert(Member{ExtensionID: "sys.auth", Role: RoleAuth, Priority: 1, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	order := reg.LoadOrder(false)
	if len(order) != 2 || order[0].ExtensionID != "sys.auth" {
		t.Fatalf("order = %#v", order)
	}
	// Safe Mode never loads system tier.
	if got := reg.LoadOrder(true); len(got) != 0 {
		t.Fatalf("safe mode load = %#v", got)
	}
	// CLI disable without loading code.
	if err := reg.Disable("sys.cache"); err != nil {
		t.Fatal(err)
	}
	order = reg.LoadOrder(false)
	if len(order) != 1 || order[0].ExtensionID != "sys.auth" {
		t.Fatalf("after disable = %#v", order)
	}
	snap := reg.Snapshot()
	if !snap.SafeModeBypass || len(snap.Members) != 2 {
		t.Fatalf("snap = %#v", snap)
	}
}
