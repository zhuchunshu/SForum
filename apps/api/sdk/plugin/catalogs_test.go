package plugin

import (
	"testing"
)

func TestCatalogsNonEmpty(t *testing.T) {
	if len(EventCatalog()) == 0 {
		t.Fatal("EventCatalog empty")
	}
	if len(CapabilityCatalog()) == 0 {
		t.Fatal("CapabilityCatalog empty")
	}
	if len(ContributionPoints()) == 0 {
		t.Fatal("ContributionPoints empty")
	}
	if len(CoreSchedules()) == 0 {
		t.Fatal("CoreSchedules empty")
	}
	slots := KnownProviderSlots()
	if len(slots) < 3 {
		t.Fatalf("KnownProviderSlots too small: %v", slots)
	}
	// 关键事件与能力应在目录中。
	if !KnownEvent("topic.created") {
		t.Fatal("topic.created should be known")
	}
	if !KnownCapability("host.api") {
		t.Fatal("host.api should be known")
	}
}

func TestHostMethodConstantsStable(t *testing.T) {
	// 防止 SDK 与 Host API 方法名漂移。
	want := map[string]string{
		MethodPing:            "Ping",
		MethodCheckPermission: "CheckPermission",
		MethodGetSettings:     "GetSettings",
		MethodEnqueueOwnJob:   "EnqueueOwnJob",
		MethodAppendAudit:     "AppendAudit",
		MethodGetUserSafe:     "GetUserSafe",
	}
	for got, expect := range want {
		if got != expect {
			t.Fatalf("method constant %q != %q", got, expect)
		}
	}
	if HostAPIVersion != "sforum.host/v1" {
		t.Fatalf("HostAPIVersion=%s", HostAPIVersion)
	}
}
