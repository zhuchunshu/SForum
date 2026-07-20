package mediaregistry

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckUploadMIMENoPolicyIsNoop(t *testing.T) {
	t.Parallel()
	registry := New()
	if err := registry.CheckUploadMIME("general", "image/png", "png"); err != nil {
		t.Fatalf("empty registry must not reject: %v", err)
	}
	if err := (*Registry)(nil).CheckUploadMIME("general", "image/png", "png"); err != nil {
		t.Fatalf("nil registry must not reject: %v", err)
	}
}

func TestCheckUploadMIMEEnforcesPublishedPolicy(t *testing.T) {
	t.Parallel()
	registry := New()
	packageDigest := strings.Repeat("ab", 32)
	impactDigest := strings.Repeat("cd", 32)
	if _, err := registry.Publish(Publication{
		Artifact: Artifact{
			ExtensionID: "demo.media", ExtensionVersion: "1.0.0",
			PackageDigest: packageDigest, ImpactDigest: impactDigest,
			VersionID: 1, RuntimeInstanceID: "demo.media-runtime",
		},
		Policies: []MIMEPolicyDeclaration{{
			ID: "demo.media.policy", ContractVersion: "demo.media.policy@1",
			Purpose: "general", Priority: 10, RequiredPermission: "attachment.upload",
			AllowedMIMEs: []string{"image/png"}, DeniedMIMEs: []string{"image/svg+xml"},
			AllowedExtensions: []string{"png"}, StrictDeclaredMIME: true, Budget: DefaultBudget(),
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := registry.CheckUploadMIME("general", "image/png", "png"); err != nil {
		t.Fatalf("png allowed: %v", err)
	}
	if err := registry.CheckUploadMIME("general", "image/jpeg", "jpg"); !errors.Is(err, ErrMediaRejected) {
		t.Fatalf("jpeg denied by allowlist, got %v", err)
	}
	if err := registry.CheckUploadMIME("general", "image/svg+xml", "svg"); !errors.Is(err, ErrMediaRejected) {
		t.Fatalf("svg denied explicitly, got %v", err)
	}
}
