package extensionmanifest

import (
	"encoding/json"
	"testing"
)

func TestManifestV3SEOContractNormalizesAndValidatesEveryFrozenFamily(t *testing.T) {
	manifest := completeV3Manifest()
	manifest.SEO = nil
	for index, kind := range []string{"TITLE", "meta", "canonical", "robots", "hreflang", "sitemap", "jsonld"} {
		id := manifest.ID + ".seo.family_" + string(rune('a'+index))
		manifest.SEO = append(manifest.SEO, ManifestSEO{
			ID: id, ContractVersion: id + "@1", Scope: " CORE.PAGE.TOPIC ", Kind: kind,
			Action: " FILTER ", Handler: " " + id + " ", FailurePolicy: " FALLBACK ",
		})
	}
	normalized := Normalize(manifest)
	if normalized.SEO[0].Scope != "core.page.topic" || normalized.SEO[0].Kind != "title" ||
		normalized.SEO[0].Action != "filter" || normalized.SEO[0].FailurePolicy != "fallback" ||
		normalized.SEO[0].TimeoutMS != ManifestSEODefaultTimeoutMS {
		t.Fatalf("normalized SEO declaration = %#v", normalized.SEO[0])
	}
	if err := Validate(normalized); err != nil {
		t.Fatalf("typed SEO families should validate: %v", err)
	}
	body, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("typed SEO JSON schema: %v", err)
	}
}

func TestManifestV3SEOContractRejectsAmbiguousOrUncallableDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Manifest)
	}{
		{name: "contract drift", change: func(value *Manifest) { value.SEO[0].ContractVersion = value.SEO[0].ID + "@2" }},
		{name: "unknown scope", change: func(value *Manifest) { value.SEO[0].Scope = "/topic" }},
		{name: "unknown kind", change: func(value *Manifest) { value.SEO[0].Kind = "html" }},
		{name: "unknown action", change: func(value *Manifest) { value.SEO[0].Action = "wrap" }},
		{name: "foreign handler", change: func(value *Manifest) { value.SEO[0].Handler = "other.seo.title" }},
		{name: "implicit failure policy", change: func(value *Manifest) { value.SEO[0].FailurePolicy = "" }},
		{name: "priority overflow", change: func(value *Manifest) { value.SEO[0].Priority = ManifestSEOMaximumPriority + 1 }},
		{name: "timeout overflow", change: func(value *Manifest) { value.SEO[0].TimeoutMS = ManifestSEOMaximumTimeoutMS + 1 }},
		{name: "unsupported protocol 1", change: func(value *Manifest) { value.Backend.ProtocolVersion = 1; value.Backend.HostAPIVersion = "" }},
		{name: "missing backend", change: func(value *Manifest) { value.Backend = ManifestBackend{} }},
		{name: "theme authority", change: func(value *Manifest) { value.Type = TypeTheme }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := completeV3Manifest()
			test.change(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("invalid SEO declaration was accepted")
			}
		})
	}
}

func TestManifestV3SEOShardUsesTheSameStrictSchema(t *testing.T) {
	manifest := completeV3Manifest()
	body, err := json.Marshal(manifest.SEO)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateV3JSONSchemaFragment(body, "seo"); err != nil {
		t.Fatalf("canonical SEO shard: %v", err)
	}
	var values []map[string]any
	if err := json.Unmarshal(body, &values); err != nil {
		t.Fatal(err)
	}
	values[0]["rawHtml"] = "<meta>"
	drifted, _ := json.Marshal(values)
	if err := validateV3JSONSchemaFragment(drifted, "seo"); err == nil {
		t.Fatal("unknown SEO shard field was accepted")
	}
}
