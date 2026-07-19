package identityregistry

import (
	"errors"
	"testing"
)

func TestIdentitySessionPolicyRequiresExecutableSamePublicationProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
	}{
		{name: "missing provider"},
		{
			name: "wrong provider kind",
			provider: Provider{
				ID: "fixture.identity.session", ContractVersion: "fixture.identity.session@1",
				Kind: ProviderKindAuth, Handler: "identity.auth",
			},
		},
		{
			name: "inspect-only session provider",
			provider: Provider{
				ID: "fixture.identity.session", ContractVersion: "fixture.identity.session@1",
				Kind: ProviderKindSession, Handler: "identity.session",
			},
		},
		{
			name: "different executable session provider",
			provider: Provider{
				ID: "fixture.identity.other_session", ContractVersion: "fixture.identity.other_session@1",
				Kind: ProviderKindSession, Handler: "identity.session",
				Operations: []ProviderOperation{{
					Name: "session.evaluate", InputSchema: "schemas/session-input.json",
					OutputSchema: "schemas/session-output.json", TimeoutMS: 500,
					FailurePolicy: ProviderFailureFailClosed,
				}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := sessionPolicyTestPublication()
			if test.provider.ID != "" {
				publication.Identity.Providers = append(publication.Identity.Providers, test.provider)
			}
			if _, err := normalizePublication(publication); !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid session policy publication error=%v", err)
			}
		})
	}

	publication := sessionPolicyTestPublication()
	provider := Provider{
		ID: "fixture.identity.session", ContractVersion: "fixture.identity.session@1",
		Kind: ProviderKindSession, Handler: "identity.session",
		Operations: []ProviderOperation{{
			Name: "session.evaluate", InputSchema: "schemas/session-input.json",
			OutputSchema: "schemas/session-output.json", TimeoutMS: 500,
			FailurePolicy: ProviderFailureFailClosed,
		}},
	}
	publication.Identity.Providers = append(publication.Identity.Providers, provider)
	binding := ProviderOperationSchemaBinding{
		ProviderID: provider.ID, ContractVersion: provider.ContractVersion,
		Operation: provider.Operations[0].Name, Artifact: publication.Artifact,
		Input: schemaTestMaterial(
			provider.Operations[0].InputSchema, "fixture.identity.session.input@1",
			`{"type":"object","additionalProperties":false}`,
		),
		Output: schemaTestMaterial(
			provider.Operations[0].OutputSchema, "fixture.identity.session.output@1",
			`{"type":"object","required":["disposition"],"properties":{"disposition":{"enum":["allow","deny","step_up"]}},"additionalProperties":false}`,
		),
	}
	bound, err := BindJSONSchemas(publication, nil, []ProviderOperationSchemaBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	registry := New()
	if _, err := registry.Publish(bound); err != nil {
		t.Fatalf("publish exact session policy provider: %v", err)
	}
	snapshot := registry.Snapshot()
	if len(snapshot.Publications) != 1 || snapshot.Publications[0].Identity == nil ||
		snapshot.Publications[0].Identity.SessionPolicy != provider.ID {
		t.Fatalf("session policy snapshot=%#v", snapshot)
	}
}

func sessionPolicyTestPublication() Publication {
	publication := testPublication(1)
	publication.Identity.SessionPolicy = "fixture.identity.session"
	return publication
}
