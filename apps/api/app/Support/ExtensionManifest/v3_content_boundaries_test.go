package extensionmanifest

import (
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
}
