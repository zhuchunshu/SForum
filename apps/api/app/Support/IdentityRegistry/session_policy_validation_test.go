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

	bound, provider := boundSessionPolicyTestPublication(t)
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

func TestRegistrySessionPolicyLeaseClaimsOnlyExactPublicBinding(t *testing.T) {
	publication, provider := boundSessionPolicyTestPublication(t)
	unbound := provider
	unbound.ID = "fixture.identity.session.unbound"
	unbound.ContractVersion = "fixture.identity.session.unbound@1"
	unbound.Operations = nil
	publication.Identity.Providers = append(publication.Identity.Providers, unbound)

	registry := New()
	if _, err := registry.Publish(publication); err != nil {
		t.Fatal(err)
	}
	if err := registry.RunWithSessionPolicyLease(provider.ID, func(claim SessionPolicyLeaseClaim) error {
		if claim.Revision != 1 || claim.SafeMode || claim.Provider == nil ||
			claim.Provider.Artifact != publication.Artifact || claim.Provider.ID != provider.ID {
			t.Fatalf("exact claim = %#v", claim)
		}
		operation := &claim.Provider.Operations[0]
		if operation.boundInputSchema != nil || operation.boundOutputSchema != nil {
			t.Fatal("session policy lease exposed private compiled Schema material")
		}
		operation.Name = "mutated"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	resolved, err := registry.ResolveProvider(provider.ID)
	if err != nil || resolved.Operations[0].Name != "session.evaluate" {
		t.Fatalf("leased claim mutated Registry provider: %#v err=%v", resolved, err)
	}
	for _, policyID := range []string{unbound.ID, "fixture.identity.session.missing", "core.session.default"} {
		if err := registry.RunWithSessionPolicyLease(policyID, func(claim SessionPolicyLeaseClaim) error {
			if claim.Provider != nil || claim.SafeMode {
				t.Fatalf("policy %q claim = %#v", policyID, claim)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	snapshot := registry.Snapshot()
	if _, err := registry.ReplaceAllIfRevision(
		snapshot.Revision,
		snapshot.Publications,
		snapshot.Tombstones,
		true,
	); err != nil {
		t.Fatal(err)
	}
	if err := registry.RunWithSessionPolicyLease(provider.ID, func(claim SessionPolicyLeaseClaim) error {
		if !claim.SafeMode || claim.Provider != nil {
			t.Fatalf("Safe Mode claim = %#v", claim)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func boundSessionPolicyTestPublication(t *testing.T) (Publication, Provider) {
	t.Helper()
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
	return bound, provider
}

func sessionPolicyTestPublication() Publication {
	publication := testPublication(1)
	publication.Identity.SessionPolicy = "fixture.identity.session"
	return publication
}
