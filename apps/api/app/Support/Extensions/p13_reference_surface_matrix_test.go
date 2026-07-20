package extensionsruntime_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestP13ReferenceSurfaceMatrixFamilies maps each of the five reference plugin
// classes onto the Extension Surface Matrix families it is required to prove.
// This is a package-declaration gate; product execution remains in the named
// *reference_plugin_integration_test.go files.
func TestP13ReferenceSurfaceMatrixFamilies(t *testing.T) {
	root := p13RepoRoot(t)
	// family -> required substrings that must appear in at least one package
	// for that class (Manifest V3 surface keys / declarations).
	type classSpec struct {
		paths    []string
		families map[string][]string
	}
	specs := map[string]classSpec{
		"seo": {
			paths: []string{"extensions/fixtures/plugins/sforum-seo-reference/sforum.extension.json.tmpl"},
			families: map[string][]string{
				// Host-owned SEO admin routes; plugin proves multi-kind SEO Registry.
				"seo": {`"seo"`, `"kind": "title"`, `"kind": "jsonld"`, `"kind": "sitemap"`},
			},
		},
		"identity": {
			paths: []string{"extensions/fixtures/plugins/sforum-membership-reference/sforum.extension.json.tmpl"},
			families: map[string][]string{
				"identityPermissions": {`"identity"`, `"permissionDefinitions"`, `"assignmentPolicy"`},
			},
		},
		"custom-content": {
			paths: []string{"extensions/fixtures/plugins/sforum-custom-content/sforum.extension.json.tmpl"},
			families: map[string][]string{
				"entities":           {`"entities"`},
				"content":            {`"content"`},
				"editor":             {`"editor"`},
				"queries":            {`"queries"`},
				"navigationRegions":  {`"navigation"`},
			},
		},
		"media": {
			paths: []string{"extensions/fixtures/plugins/sforum-media-optimize/sforum.extension.json.tmpl"},
			families: map[string][]string{
				"media": {`"media"`},
				"jobs":  {`"jobs"`},
			},
		},
		"commerce": {
			paths: []string{
				"extensions/fixtures/plugins/sforum-commerce-workflow/sforum.extension.json.tmpl",
				"extensions/fixtures/plugins/sforum-commerce-workflow-ext/sforum.extension.json.tmpl",
			},
			families: map[string][]string{
				"routes":              {`"routes"`, `"action"`},
				"database":            {`"database"`},
				"hooks":               {`"hooks"`},
				"jobs":                {`"jobs"`},
				"cacheInvalidation":   {`"cache"`},
				"services":            {`"services"`},
				"openapi":             {`"openapi"`},
				"publicComponents":    {`"components"`},
				"lifecycle":           {`"lifecycle"`},
			},
		},
	}

	// Union of ESM-relevant families proved by the five-class set.
	// Eleven core ESM families: routes, hooks, queries, adminComponents,
	// publicComponents, identityPermissions, media, navigationRegions,
	// cacheInvalidation, jobs, lifecycle — plus SEO/content/editor/database
	// which the plan treats as first-class surfaces for reference plugins.
	covered := map[string]bool{}
	for class, spec := range specs {
		var body strings.Builder
		found := false
		for _, rel := range spec.paths {
			raw, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				continue
			}
			found = true
			body.Write(raw)
			body.WriteByte('\n')
		}
		if !found {
			t.Fatalf("P13 %s: no fixture package found", class)
		}
		text := body.String()
		for family, markers := range spec.families {
			for _, marker := range markers {
				if !strings.Contains(text, marker) {
					t.Fatalf("P13 %s missing ESM family %s marker %s", class, family, marker)
				}
			}
			covered[family] = true
		}
	}

	// Required union across the five classes (plan P13 final gate).
	requiredUnion := []string{
		"routes", "hooks", "queries", "publicComponents", "identityPermissions",
		"media", "navigationRegions", "cacheInvalidation", "jobs", "lifecycle",
		"seo", "entities", "content", "editor", "database", "services", "openapi",
	}
	for _, family := range requiredUnion {
		if !covered[family] {
			t.Fatalf("five-class union missing ESM family %s", family)
		}
	}
}
