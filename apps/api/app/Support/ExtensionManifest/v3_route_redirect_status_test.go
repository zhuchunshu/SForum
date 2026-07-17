package extensionmanifest

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestManifestV3RedirectStatusDefaultsAndAllowsOnlyPermanentRedirects(t *testing.T) {
	manifest := redirectStatusManifest()
	if got := Normalize(manifest).Routes[0].StatusCode; got != RouteRedirectStatusDefault {
		t.Fatalf("default redirect status = %d", got)
	}
	if err := Validate(manifest); err != nil {
		t.Fatalf("omitted redirect status should default safely: %v", err)
	}

	for _, status := range []int{301, RouteRedirectStatusDefault} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			candidate := redirectStatusManifest()
			candidate.Routes[0].StatusCode = status
			if err := Validate(candidate); err != nil {
				t.Fatalf("status %d should validate: %v", status, err)
			}
		})
	}

	for _, status := range []int{-1, 200, 302, 307, 309} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			candidate := redirectStatusManifest()
			candidate.Routes[0].StatusCode = status
			if err := Validate(candidate); err == nil {
				t.Fatalf("status %d must fail", status)
			}
		})
	}

	nonRedirect := completeV3Manifest()
	nonRedirect.Routes[0].StatusCode = 301
	if err := Validate(nonRedirect); err == nil {
		t.Fatal("non-redirect action must reject statusCode")
	}
}

func TestManifestV3RedirectStatusJSONSchemaMatchesRuntimeContract(t *testing.T) {
	for _, status := range []int{0, 301, RouteRedirectStatusDefault} {
		manifest := redirectStatusManifest()
		manifest.Routes[0].StatusCode = status
		body, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateV3JSONSchema(body); err != nil {
			t.Fatalf("valid redirect status %d rejected: %v", status, err)
		}
	}

	invalid := redirectStatusManifest()
	invalid.Routes[0].StatusCode = 302
	nonRedirect := completeV3Manifest()
	nonRedirect.Routes[0].StatusCode = 301
	for _, manifest := range []Manifest{invalid, nonRedirect} {
		body, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateV3JSONSchema(body); err == nil {
			t.Fatal("invalid redirect status schema must fail")
		}
	}
}

func TestManifestV3RedirectDefaultIsCanonical(t *testing.T) {
	normalized := Normalize(redirectStatusManifest())
	body, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateV3JSONSchema(body); err != nil {
		t.Fatalf("normalized redirect must remain schema-valid: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	routes := document["routes"].([]any)
	if got := int(routes[0].(map[string]any)["statusCode"].(float64)); got != RouteRedirectStatusDefault {
		t.Fatalf("canonical statusCode = %d", got)
	}
}

func TestLegacyManifestRejectsRedirectStatusCode(t *testing.T) {
	for _, version := range []int{0, ManifestVersionV1, ManifestVersionV2} {
		manifest := versionedTestManifest(version)
		manifest.Routes = []ManifestRoute{{
			Path: "/legacy", Methods: []string{"GET"}, Access: RouteAccessPublic, StatusCode: 301,
		}}
		if err := Validate(manifest); err == nil {
			t.Fatalf("manifest version %d accepted V3 redirect status", version)
		}
	}
}

func redirectStatusManifest() Manifest {
	manifest := completeV3Manifest()
	route := &manifest.Routes[0]
	route.Action = RouteActionRedirect
	route.Guard = GuardCorePublic
	route.Access = ""
	route.Handler = ""
	route.RequestSchema = ""
	route.ResponseSchema = ""
	route.Destination = "/api/v3/demo-new"
	route.StatusCode = 0
	return manifest
}
