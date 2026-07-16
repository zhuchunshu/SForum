package extensionmanifest

import (
	"errors"
	"testing"
)

func TestV3PermissionRecommendationsCannotTargetSuperAdmin(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.PermissionDefinitions[0].RecommendedRoles = []string{"super_admin"}
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("Validate() error = %v, want ErrInvalidManifest", err)
	}
}

func TestV3PermissionRecommendationsRemainDescriptiveHostPolicy(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.PermissionDefinitions[0].RecommendedRoles = []string{" Member ", "OPERATOR"}
	manifest.Identity.RiskHooks = []string{" Demo.V3.Risk.Login "}
	manifest = Normalize(manifest)
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := manifest.PermissionDefinitions[0].RecommendedRoles; len(got) != 2 || got[0] != "member" || got[1] != "operator" {
		t.Fatalf("recommended roles = %#v", got)
	}
	if got := manifest.Identity.RiskHooks; len(got) != 1 || got[0] != "demo.v3.risk.login" {
		t.Fatalf("risk hooks = %#v", got)
	}
}

func TestV3IdentityCatalogRejectsForeignAndAmbiguousAuthority(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{name: "invalid recommended role", mutate: func(manifest *Manifest) {
			manifest.PermissionDefinitions[0].RecommendedRoles = []string{"bad role"}
		}},
		{name: "too many recommended roles", mutate: func(manifest *Manifest) {
			manifest.PermissionDefinitions[0].RecommendedRoles = make([]string, manifestIdentityMaximumRoleSuggestions+1)
			for index := range manifest.PermissionDefinitions[0].RecommendedRoles {
				manifest.PermissionDefinitions[0].RecommendedRoles[index] = "role_" + string(rune('a'+index%26)) + string(rune('a'+index/26))
			}
		}},
		{name: "foreign risk hook", mutate: func(manifest *Manifest) {
			manifest.Identity.RiskHooks = []string{"other.identity.risk.login"}
		}},
		{name: "duplicate risk hook", mutate: func(manifest *Manifest) {
			manifest.Identity.RiskHooks = []string{"demo.v3.risk.login", "demo.v3.risk.login"}
		}},
		{name: "too many risk hooks", mutate: func(manifest *Manifest) {
			manifest.Identity.RiskHooks = make([]string, manifestIdentityMaximumRiskHooks+1)
			for index := range manifest.Identity.RiskHooks {
				manifest.Identity.RiskHooks[index] = "demo.v3.risk.hook_" + string(rune('a'+index%26)) + string(rune('a'+index/26))
			}
		}},
		{name: "foreign session policy", mutate: func(manifest *Manifest) {
			manifest.Identity.SessionPolicy = "other.identity.session"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.mutate(&manifest)
			if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
