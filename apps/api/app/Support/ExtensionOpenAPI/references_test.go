package extensionopenapi

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildRejectsUnsafeOrUnresolvedReferences(t *testing.T) {
	tests := []struct {
		name      string
		reference string
	}{
		{"url", "https://evil.example/schema.json#/Catalog"},
		{"absolute", "/etc/passwd#/Catalog"},
		{"escape", "../../outside.json#/Catalog"},
		{"encoded escape", "%2e%2e/outside.json#/Catalog"},
		{"undeclared", "undeclared.json#/Catalog"},
		{"missing pointer", "schemas/common.json#/Missing"},
		{"anchor", "schemas/common.json#Catalog"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := defaultFixtureOptions("refs." + strings.ReplaceAll(test.name, " ", "-"))
			options.document = strings.Replace(fixtureDocument(options), "schemas/common.json#/Catalog", test.reference, 1)
			fixture := buildFixture(t, options)
			if _, err := Build(BuildInput{Artifacts: []Artifact{fixture}}); !errors.Is(err, ErrUnsafeReference) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBuildRejectsYAMLAliasesWithoutExecutingAnything(t *testing.T) {
	options := defaultFixtureOptions("refs.alias")
	options.document = strings.Replace(fixtureDocument(options), "title: Fixture", "title: &title Fixture\n  description: *title", 1)
	fixture := buildFixture(t, options)
	if _, err := Build(BuildInput{Artifacts: []Artifact{fixture}}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("alias error = %v", err)
	}
}

func TestResolvePackageReferenceAllowsOnlyCanonicalPackageLocalPointers(t *testing.T) {
	target, pointer, err := resolvePackageReference("openapi/routes/root.yaml", "../schemas/common.json#/Catalog")
	if err != nil || target != "openapi/schemas/common.json" || pointer != "/Catalog" {
		t.Fatalf("package-local parent ref = %q %q %v", target, pointer, err)
	}
	target, pointer, err = resolvePackageReference("openapi/routes.yaml", "schemas/common.json#/Catalog")
	if err != nil || target != "openapi/schemas/common.json" || pointer != "/Catalog" {
		t.Fatalf("canonical ref = %q %q %v", target, pointer, err)
	}
	if _, _, err := resolvePackageReference("openapi/routes/root.yaml", "../../../outside.json#/Catalog"); !errors.Is(err, ErrUnsafeReference) {
		t.Fatalf("escaping parent ref error = %v", err)
	}
}
