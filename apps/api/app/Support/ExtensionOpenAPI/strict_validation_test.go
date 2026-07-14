package extensionopenapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBuildRejectsNonCanonicalOpenAPI31Versions(t *testing.T) {
	for _, version := range []string{"3.1", "3.1.x", "3.1.00", "3.1.0+build", " 3.1.0", "3.1.0 "} {
		t.Run(strings.ReplaceAll(version, " ", "space"), func(t *testing.T) {
			options := defaultFixtureOptions("version.strict")
			options.document = strings.Replace(fixtureDocument(options), "openapi: 3.1.0", fmt.Sprintf("openapi: %q", version), 1)
			if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, options)}}); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("version %q error = %v", version, err)
			}
		})
	}
}

func TestBuildRequiresExactPathTemplateParameters(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
	}{
		{"required false", func(document string) string { return strings.Replace(document, "required: true", "required: false", 1) }},
		{"missing", func(document string) string {
			return strings.Replace(document, "      parameters:\n        - name: id\n          in: path\n          required: true\n          schema:\n            type: string\n", "", 1)
		}},
		{"extra", func(document string) string { return strings.Replace(document, "name: id", "name: extra", 1) }},
		{"duplicate", func(document string) string {
			parameter := "        - name: id\n          in: path\n          required: true\n          schema:\n            type: string\n"
			return strings.Replace(document, parameter, parameter+parameter, 1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := defaultFixtureOptions("parameters." + strings.ReplaceAll(test.name, " ", "-"))
			options.document = test.mutate(fixtureDocument(options))
			if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, options)}}); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBuildAcceptsBooleanSchemasAndRejectsPseudoSchemas(t *testing.T) {
	valid := defaultFixtureOptions("schema.boolean")
	valid.document = strings.Replace(fixtureDocument(valid), "schema:\n            type: string", "schema: true", 1)
	if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, valid)}}); err != nil {
		t.Fatalf("boolean schema failed: %v", err)
	}

	for name, replacement := range map[string]string{
		"string": "schema: bogus",
		"number": "schema: 42",
		"null":   "schema: null",
		"empty":  "schema: {}",
	} {
		t.Run(name, func(t *testing.T) {
			options := defaultFixtureOptions("schema." + name)
			options.document = strings.Replace(fixtureDocument(options), "schema:\n            type: string", replacement, 1)
			if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, options)}}); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestBuildRejectsInvalidResponseKeys(t *testing.T) {
	for _, status := range []string{"099", "600", "20X", "2xx", "2000", "success"} {
		t.Run(status, func(t *testing.T) {
			options := defaultFixtureOptions("response.key")
			options.document = strings.Replace(fixtureDocument(options), `"200":`, fmt.Sprintf("%q:", status), 1)
			if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, options)}}); !errors.Is(err, ErrInvalidDocument) {
				t.Fatalf("status %q error = %v", status, err)
			}
		})
	}
}

func TestBuildRejectsNonCanonicalOperationAndPolicyFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*fixtureOptions, *Artifact)
	}{
		{"operation id", func(options *fixtureOptions, _ *Artifact) {
			options.document = strings.Replace(fixtureDocument(*options), "operationId: "+options.operationID, "operationId: ' "+options.operationID+"'", 1)
		}},
		{"route id", func(options *fixtureOptions, _ *Artifact) {
			options.document = strings.Replace(fixtureDocument(*options), "x-sforum-route-id: "+options.extensionID+".catalog", "x-sforum-route-id: '"+options.extensionID+".catalog '", 1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := defaultFixtureOptions("canonical." + strings.ReplaceAll(test.name, " ", "-"))
			test.mutate(&options, nil)
			if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, options)}}); !errors.Is(err, ErrContractMismatch) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	policy := buildFixture(t, defaultFixtureOptions("canonical.policy"))
	policy.Policies[0].Method = "get"
	if _, err := Build(BuildInput{Artifacts: []Artifact{policy}}); !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("non-canonical policy error = %v", err)
	}

	info := defaultFixtureOptions("canonical.info")
	info.document = strings.Replace(fixtureDocument(info), "title: Fixture", "title: ' Fixture'", 1)
	if _, err := Build(BuildInput{Artifacts: []Artifact{buildFixture(t, info)}}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("non-canonical info error = %v", err)
	}
}

func TestBuildRequiresExactBidirectionalRoutePolicies(t *testing.T) {
	missing := buildFixture(t, defaultFixtureOptions("policy.missing"))
	missing.Policies = nil
	if _, err := Build(BuildInput{Artifacts: []Artifact{missing}}); !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("missing policy error = %v", err)
	}

	extra := buildFixture(t, defaultFixtureOptions("policy.extra"))
	extra.Policies = append(extra.Policies, RoutePolicy{
		RouteID: "policy.extra.stale", Method: "GET", RateLimit: "public.read@1",
		Idempotency: "disabled", Security: SecurityPublic,
	})
	if _, err := Build(BuildInput{Artifacts: []Artifact{extra}}); !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("extra policy error = %v", err)
	}

	contradiction := buildFixture(t, defaultFixtureOptions("policy.security"))
	contradiction.Policies[0].Security = SecurityAuthenticated
	if _, err := Build(BuildInput{Artifacts: []Artifact{contradiction}}); !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("security policy contradiction error = %v", err)
	}
}

func TestBuildCanonicalFieldsRemainUnmodifiedInAggregate(t *testing.T) {
	fixture := buildFixture(t, defaultFixtureOptions("canonical.output"))
	snapshot, err := Build(BuildInput{Artifacts: []Artifact{fixture}})
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(snapshot.Document(), &document); err != nil {
		t.Fatal(err)
	}
	operation := document["paths"].(map[string]any)["/api/catalog/{id}"].(map[string]any)["get"].(map[string]any)
	if operation["operationId"] != "canonical.output.api.getCatalog" {
		t.Fatalf("operation id changed: %#v", operation)
	}
}
