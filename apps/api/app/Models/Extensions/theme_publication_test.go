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
