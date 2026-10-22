package extensions

import (
	"strings"
	"testing"
)

func TestValidThemeRuntimePublicationRequiresCanonicalHexAndExactSourceApproval(t *testing.T) {
	base := ThemeRuntimePublication{
		DesiredState: ThemeRuntimePublicationActive,
		ThemeID:      "target.theme", ThemeVersion: "2.0.0", PackageDigest: strings.Repeat("a", 64),
		SourceThemeID: "source.theme", SourceThemeVersion: "1.0.0", SourcePackageDigest: strings.Repeat("b", 64),
		SourceCoreReplacementsApproved: true, SourceActorUserID: 42,
		CoreReplacementsApproved: true, ActorUserID: 99,
		Reason: ThemeRuntimePublicationActivation,
	}
	if !validThemeRuntimePublication(base) {
		t.Fatal("valid exact publication was rejected")
	}
	for name, mutate := range map[string]func(*ThemeRuntimePublication){
		"non-hex target":   func(value *ThemeRuntimePublication) { value.PackageDigest = strings.Repeat("g", 64) },
		"uppercase target": func(value *ThemeRuntimePublication) { value.PackageDigest = strings.Repeat("A", 64) },
		"source actor without approval": func(value *ThemeRuntimePublication) {
			value.SourceCoreReplacementsApproved = false
		},
		"source approval without actor": func(value *ThemeRuntimePublication) { value.SourceActorUserID = 0 },
		"source approval without tuple": func(value *ThemeRuntimePublication) {
			value.SourceThemeID, value.SourceThemeVersion, value.SourcePackageDigest = "", "", ""
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if validThemeRuntimePublication(value) {
				t.Fatalf("invalid publication accepted: %#v", value)
			}
		})
	}
}

func TestValidThemeRuntimePublicationApprovalProvenance(t *testing.T) {
	digest := strings.Repeat("c", 64)
	source := strings.Repeat("d", 64)
	base := ThemeRuntimePublication{
		DesiredState:             ThemeRuntimePublicationActive,
		ThemeID:                  DefaultThemeID,
		ThemeVersion:             "1.0.0",
		PackageDigest:            digest,
		CoreReplacementsApproved: true,
		ApprovalSource:           ThemeApprovalSourceHost,
		Reason:                   ThemeRuntimePublicationStartupRepair,
	}
	if !validThemeRuntimePublication(base) {
		t.Fatal("host-approved protected default publication was rejected")
	}
	for name, mutate := range map[string]func(*ThemeRuntimePublication){
		"host approval with actor": func(value *ThemeRuntimePublication) { value.ActorUserID = 7 },
		"host approval for uploaded theme": func(value *ThemeRuntimePublication) {
			value.ThemeID = "acme.theme"
		},
		"unknown approval source": func(value *ThemeRuntimePublication) { value.ApprovalSource = "system" },
		"admin approval without actor": func(value *ThemeRuntimePublication) {
			value.ApprovalSource = ThemeApprovalSourceAdmin
		},
		"host source approval without tuple": func(value *ThemeRuntimePublication) {
			value.SourceCoreReplacementsApproved = true
			value.SourceApprovalSource = ThemeApprovalSourceHost
		},
		"host source approval with actor": func(value *ThemeRuntimePublication) {
			value.SourceThemeID = "acme.old"
			value.SourceThemeVersion = "1.0.0"
			value.SourcePackageDigest = source
			value.SourceCoreReplacementsApproved = true
			value.SourceApprovalSource = ThemeApprovalSourceHost
			value.SourceActorUserID = 7
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if validThemeRuntimePublication(value) {
				t.Fatalf("invalid publication accepted: %#v", value)
			}
		})
	}

	admin := base
	admin.ApprovalSource = ThemeApprovalSourceAdmin
	admin.ActorUserID = 12
	if !validThemeRuntimePublication(admin) {
		t.Fatal("admin-approved publication was rejected")
	}
	sourceHost := base
	sourceHost.SourceThemeID = DefaultThemeID
	sourceHost.SourceThemeVersion = "0.9.0"
	sourceHost.SourcePackageDigest = source
	sourceHost.SourceCoreReplacementsApproved = true
	sourceHost.SourceApprovalSource = ThemeApprovalSourceHost
	if !validThemeRuntimePublication(sourceHost) {
		t.Fatal("host-approved source tuple was rejected")
	}
}
