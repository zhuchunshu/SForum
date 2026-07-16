package extensionmanifest

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestManifestV3ContentPublicationBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{name: "too many declarations", mutate: func(manifest *Manifest) {
			manifest.Content = make([]ManifestContent, ContentDeclarationsMaximum+1)
		}},
		{name: "handler too long", mutate: func(manifest *Manifest) {
			manifest.Content[0].Handler = strings.Repeat("h", HandlerReferenceMaximumLength+1)
		}},
		{name: "handler control character", mutate: func(manifest *Manifest) {
			manifest.Content[0].Handler = "content.\x00handler"
		}},
		{name: "schema reference too long", mutate: func(manifest *Manifest) {
			manifest.Content[0].Schema = strings.Repeat("s", SchemaReferenceMaximumLength) + "@1"
		}},
		{name: "schema path contains nul", mutate: func(manifest *Manifest) {
			manifest.Content[0].Schema = "schemas/content\x00.json"
		}},
		{name: "schema path traverses", mutate: func(manifest *Manifest) {
			manifest.Content[0].Schema = "schemas/../content.json"
		}},
		{name: "missing handler and renderer", mutate: func(manifest *Manifest) {
			manifest.Content[0].Handler = ""
			manifest.Content[0].Renderer = ""
		}},
		{name: "contract version too long", mutate: func(manifest *Manifest) {
			manifest.Content[0].ContractVersion = strings.Repeat("c", ContractVersionMaximumLength) + "@1"
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.mutate(&manifest)
			if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("Validate() error = %v, want ErrInvalidManifest", err)
			}
			body, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateV3JSONSchema(body); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("ValidateV3JSONSchema() error = %v, want ErrInvalidManifest", err)
			}
		})
	}
}

func TestManifestV3ContentPublicationAcceptsExactBounds(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.Content[0].Handler = strings.Repeat("h", HandlerReferenceMaximumLength)
	manifest.Content[0].Schema = strings.Repeat("s", SchemaReferenceMaximumLength-2) + "@1"
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate() exact bounds error = %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("ValidateV3JSONSchema() exact bounds error = %v", err)
	}
}

func TestManifestV3HandlerBackedContentRequiresExecutableBackend(t *testing.T) {
	digest := v3FixtureDigest()
	manifest := versionedTestManifest(ManifestVersionV3)
	manifest.ID = "renderer.content"
	manifest.PackageFiles = []ManifestPackageFile{{
		ID: "renderer.content.file.template", Kind: "template",
		Path: "templates/card.html", Digest: digest,
	}}
	manifest.Templates = []ManifestTemplate{{
		ID: "renderer.content.template.card", ContractVersion: "renderer.content.template.card@1",
		Action: "add", Path: "templates/card.html", Digest: digest,
		ViewModelSchema: "renderer.content.block.card.schema@1",
	}}
	manifest.Content = []ManifestContent{{
		ID: "renderer.content.block.card", ContractVersion: "renderer.content.block.card@1",
		Kind: "block", Schema: "renderer.content.block.card.schema@1",
		Renderer: "renderer.content.template.card",
	}}
	assertManifestAndSchemaValid(t, manifest)

	manifest.Content[0].Handler = "content.card"
	if err := Validate(manifest); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("Validate() handler without backend error = %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("ValidateV3JSONSchema() handler without backend error = %v", err)
	}
}

func assertManifestAndSchemaValid(t *testing.T, manifest Manifest) {
	t.Helper()
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("ValidateV3JSONSchema() error = %v", err)
	}
}
