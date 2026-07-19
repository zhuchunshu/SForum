package identityregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBindJSONSchemasPublishesPrivateExactMaterial(t *testing.T) {
	raw, fieldBinding, operationBinding := identitySchemaTestPublication()
	if _, err := New().Publish(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unbound executable provider publication = %v", err)
	}
	bound, err := BindJSONSchemas(raw, []UserFieldSchemaBinding{fieldBinding}, []ProviderOperationSchemaBinding{operationBinding})
	if err != nil {
		t.Fatal(err)
	}
	field := bound.Identity.UserFields[0]
	operation := bound.Identity.Providers[0].Operations[0]
	if field.SchemaDigest != fieldBinding.Schema.Digest || field.SchemaWireReference != fieldBinding.Schema.WireReference ||
		operation.InputSchemaDigest != operationBinding.Input.Digest ||
		operation.OutputSchemaDigest != operationBinding.Output.Digest ||
		field.boundSchema == nil || operation.boundInputSchema == nil || operation.boundOutputSchema == nil {
		t.Fatalf("bound Identity Schemas = %#v / %#v", field, operation)
	}
	originalValidator := field.boundSchema.validator

	registry := New()
	revision, err := registry.Publish(bound)
	if err != nil || revision != 1 {
		t.Fatalf("publish bound Identity Schemas = %d, %v", revision, err)
	}
	// Registry owns a deep copy of raw bytes and compiled material.
	bound.Identity.UserFields[0].boundSchema.schema[0] = 'x'
	bound.Identity.UserFields[0].boundSchema.validator = nil
	bound.Identity.Providers[0].Operations[0].boundInputSchema.schema[0] = 'x'
	bound.Identity.Providers[0].Operations[0].boundOutputSchema.validator = nil

	fieldClaim := UserFieldSchemaClaim{
		FieldID: field.ID, ContractVersion: field.ContractVersion, Artifact: bound.Artifact,
	}
	if err := registry.ValidateUserFieldSchemaClaim(fieldClaim); err != nil {
		t.Fatalf("available user-field Schema claim = %v", err)
	}
	staleFieldClaim := fieldClaim
	staleFieldClaim.Artifact.PackageDigest = strings.Repeat("f", 64)
	if err := registry.ValidateUserFieldSchemaClaim(staleFieldClaim); !errors.Is(err, ErrSchemaUnavailable) {
		t.Fatalf("stale user-field Schema claim = %v", err)
	}
	var nilRegistry *Registry
	if err := nilRegistry.ValidateUserFieldSchemaClaim(fieldClaim); !errors.Is(err, ErrSchemaUnavailable) {
		t.Fatalf("nil Registry user-field Schema claim = %v", err)
	}
	if err := registry.ValidateUserFieldValue(fieldClaim, "member-code"); err != nil {
		t.Fatalf("valid user field = %v", err)
	}
	if err := registry.ValidateUserFieldValue(fieldClaim, "x"); !errors.Is(err, ErrSchemaValueInvalid) {
		t.Fatalf("invalid user field = %v", err)
	}
	providerClaim := ProviderOperationSchemaClaim{
		ProviderID:      bound.Identity.Providers[0].ID,
		ContractVersion: bound.Identity.Providers[0].ContractVersion,
		Operation:       operation.Name, Artifact: bound.Artifact,
	}
	if err := registry.ValidateProviderOperationInput(providerClaim, map[string]any{"risk": true}); err != nil {
		t.Fatalf("valid provider input = %v", err)
	}
	if err := registry.ValidateProviderOperationInput(providerClaim, map[string]any{"risk": "yes"}); !errors.Is(err, ErrSchemaValueInvalid) {
		t.Fatalf("invalid provider input = %v", err)
	}
	if err := registry.ValidateProviderOperationOutput(providerClaim, map[string]any{"disposition": "allow"}); err != nil {
		t.Fatalf("valid provider output = %v", err)
	}
	if err := registry.ValidateProviderOperationOutput(providerClaim, map[string]any{"disposition": "unknown"}); !errors.Is(err, ErrSchemaValueInvalid) {
		t.Fatalf("invalid provider output = %v", err)
	}

	resolvedField, err := registry.ResolveUserField(field.ID)
	if err != nil || resolvedField.boundSchema != nil || resolvedField.SchemaDigest == "" {
		t.Fatalf("public field projection = %#v, %v", resolvedField, err)
	}
	resolvedProvider, err := registry.ResolveProvider(bound.Identity.Providers[0].ID)
	if err != nil || len(resolvedProvider.Operations) != 1 ||
		resolvedProvider.Operations[0].boundInputSchema != nil || resolvedProvider.Operations[0].boundOutputSchema != nil {
		t.Fatalf("public provider projection = %#v, %v", resolvedProvider, err)
	}
	encoded, err := json.Marshal(registry.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("private_schema_marker")) ||
		!bytes.Contains(encoded, []byte(field.SchemaDigest)) ||
		!bytes.Contains(encoded, []byte(operation.InputSchemaDigest)) {
		t.Fatalf("snapshot private/public Schema projection = %s", encoded)
	}

	// Recompiling identical bytes creates different validator pointers but is
	// still the same exact public contract and does not advance the revision.
	reboundRaw, reboundField, reboundOperation := identitySchemaTestPublication()
	rebound, err := BindJSONSchemas(
		reboundRaw, []UserFieldSchemaBinding{reboundField}, []ProviderOperationSchemaBinding{reboundOperation},
	)
	if err != nil {
		t.Fatal(err)
	}
	if rebound.Identity.UserFields[0].boundSchema.validator == originalValidator {
		t.Fatal("test did not recompile an independent validator")
	}
	if replay, err := registry.Publish(rebound); err != nil || replay != revision {
		t.Fatalf("exact Schema replay = %d, %v", replay, err)
	}

	forged := publicationContract(rebound)
	if _, err := New().Publish(forged); !errors.Is(err, ErrInvalid) {
		t.Fatalf("public digest forged private Schema material = %v", err)
	}
	safe := New()
	if _, err := safe.ReplaceAll([]Publication{forged}, true); err != nil {
		t.Fatalf("Safe Mode parsed filtered private Schema forgery = %v", err)
	}
	if snapshot := safe.Snapshot(); !snapshot.SafeMode || len(snapshot.Publications) != 0 {
		t.Fatalf("Safe Mode retained private Schema forgery = %#v", snapshot)
	}
	publicJSON, err := json.Marshal(rebound)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Publication
	if err := json.Unmarshal(publicJSON, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Publish(decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("JSON round-trip forged private Schema material = %v", err)
	}
	if _, err := registry.ReplaceAll(registry.Snapshot().Publications, false); err != nil {
		t.Fatalf("in-memory immutable snapshot lost private material = %v", err)
	}
	if _, err := BindJSONSchemas(rebound, nil, []ProviderOperationSchemaBinding{reboundOperation}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("partial rebind downgraded exact user field = %v", err)
	}
}

func TestIdentitySchemaLegacyFieldStaysCatalogOnly(t *testing.T) {
	legacy := testPublication(1)
	registry := New()
	if _, err := registry.Publish(legacy); err != nil {
		t.Fatalf("legacy catalog-only field = %v", err)
	}
	claim := UserFieldSchemaClaim{
		FieldID:         legacy.Identity.UserFields[0].ID,
		ContractVersion: legacy.Identity.UserFields[0].ContractVersion,
		Artifact:        legacy.Artifact,
	}
	if err := registry.ValidateUserFieldValue(claim, "value"); !errors.Is(err, ErrSchemaUnavailable) {
		t.Fatalf("legacy field became writable = %v", err)
	}
	pathOnly := testPublication(2)
	pathOnly.Identity.UserFields[0].Schema = "schemas/field.json"
	if _, err := New().Publish(pathOnly); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unbound package-path field = %v", err)
	}
}

func TestIdentitySchemaDurableProjectionRequiresExactRebind(t *testing.T) {
	raw, fieldBinding, operationBinding := identitySchemaTestPublication()
	bound, err := BindJSONSchemas(raw, []UserFieldSchemaBinding{fieldBinding}, []ProviderOperationSchemaBinding{operationBinding})
	if err != nil {
		t.Fatal(err)
	}
	state := durableStateForPublication(t, bound, 41, 81)
	if err := ValidateDurablePublication(state, bound); err != nil {
		t.Fatalf("validate exact durable publication = %v", err)
	}
	root := state.RootTips[0]
	if strings.Contains(string(root.PublicationJSON), "private_schema_marker") ||
		strings.Contains(string(root.PublicationJSON), "runtimeInstanceId") ||
		!strings.Contains(string(root.PublicationJSON), fieldBinding.Schema.Digest) ||
		!strings.Contains(string(root.PublicationJSON), operationBinding.Input.Digest) {
		t.Fatalf("durable public projection = %s", root.PublicationJSON)
	}
	decoded, canonical, digest, err := decodeDurableRootPublication(root.PublicationJSON)
	if err != nil || digest != root.PublicationDigest || !bytes.Equal(canonical, root.PublicationJSON) {
		t.Fatalf("decode durable executable root = %#v, %x, %s, %v", decoded, canonical, digest, err)
	}
	decoded.Artifact.RuntimeInstanceID = bound.Artifact.RuntimeInstanceID
	if _, err := New().Publish(decoded); !errors.Is(err, ErrInvalid) {
		t.Fatalf("durable public JSON became executable without rebind = %v", err)
	}
	if _, err := New().ReplaceAll([]Publication{decoded}, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ReplaceAll accepted durable JSON without rebind = %v", err)
	}
	if _, err := New().ReplaceAllIfRevision(0, []Publication{decoded}, nil, false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ReplaceAllIfRevision accepted durable JSON without rebind = %v", err)
	}
	allowedTarget := decoded.Artifact
	if _, err := normalizeReconcilePublicationInput(ReconcilePublicationInput{
		ExtensionID: decoded.Artifact.ExtensionID, AllowedTarget: &allowedTarget,
		Desired: &decoded, ActorUserID: 41, AuditEventID: 81,
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("durable reconcile accepted JSON without rebind = %v", err)
	}
	if _, err := prepareLegacyAdoptionBatch([]Publication{decoded}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("legacy adoption accepted JSON without rebind = %v", err)
	}

	restartRaw, restartField, restartOperation := identitySchemaTestPublication()
	restartRaw.Artifact.RuntimeInstanceID = "runtime-restarted"
	restartField.Artifact = restartRaw.Artifact
	restartOperation.Artifact = restartRaw.Artifact
	restarted, err := BindJSONSchemas(
		restartRaw, []UserFieldSchemaBinding{restartField},
		[]ProviderOperationSchemaBinding{restartOperation},
	)
	if err != nil {
		t.Fatalf("recompile restarted exact package = %v", err)
	}
	if err := ValidateDurablePublication(state, restarted); err != nil {
		t.Fatalf("runtime rotation changed public durable contract = %v", err)
	}
}

func TestIdentitySchemaTrustImpactUsesManifestProjection(t *testing.T) {
	raw, fieldBinding, operationBinding := identitySchemaTestPublication()
	bound, err := BindJSONSchemas(raw, []UserFieldSchemaBinding{fieldBinding}, []ProviderOperationSchemaBinding{operationBinding})
	if err != nil {
		t.Fatal(err)
	}
	body, digest := mustCanonicalTrustImpactForPublication(t, bound, nil)
	if bytes.Contains(body, []byte("schemaWireReference")) || bytes.Contains(body, []byte("schemaDigest")) {
		t.Fatalf("TrustImpact leaked Host-derived Registry metadata: %s", body)
	}
	if err := trustImpactAuthorizesDesiredPublication(testValidateStoredTrustImpact, body, digest, bound); err != nil {
		t.Fatalf("exact Manifest-facing Identity impact = %v", err)
	}
	drifted := raw
	drifted.Identity = clonePublication(raw).Identity
	drifted.Identity.Providers[0].Operations[0].InputSchema = "fixture.identity.changed@1"
	driftedBody, driftedDigest := mustCanonicalTrustImpactForPublication(t, drifted, nil)
	if err := trustImpactAuthorizesDesiredPublication(
		testValidateStoredTrustImpact, driftedBody, driftedDigest, bound,
	); !errors.Is(err, ErrInvalid) {
		t.Fatalf("drifted Manifest Identity impact = %v", err)
	}
}

func TestIdentitySchemaRejectsDigestExternalReferenceAndContractDrift(t *testing.T) {
	raw, fieldBinding, operationBinding := identitySchemaTestPublication()
	tests := []struct {
		name   string
		mutate func(*UserFieldSchemaBinding, *ProviderOperationSchemaBinding)
	}{
		{name: "field digest", mutate: func(field *UserFieldSchemaBinding, _ *ProviderOperationSchemaBinding) {
			field.Schema.Digest = strings.Repeat("f", 64)
		}},
		{name: "external ref", mutate: func(field *UserFieldSchemaBinding, _ *ProviderOperationSchemaBinding) {
			field.Schema = schemaTestMaterial(field.Schema.Reference, field.Schema.WireReference, `{"$ref":"https://example.test/schema.json"}`)
		}},
		{name: "operation wire ref", mutate: func(_ *UserFieldSchemaBinding, operation *ProviderOperationSchemaBinding) {
			operation.Input.WireReference = "schemas/input.json"
		}},
		{name: "operation mismatch", mutate: func(_ *UserFieldSchemaBinding, operation *ProviderOperationSchemaBinding) {
			operation.Operation = "session.evaluate"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := fieldBinding
			operation := operationBinding
			test.mutate(&field, &operation)
			if _, err := BindJSONSchemas(raw, []UserFieldSchemaBinding{field}, []ProviderOperationSchemaBinding{operation}); !errors.Is(err, ErrInvalid) {
				t.Fatalf("unsafe Schema binding = %v", err)
			}
		})
	}
}

func identitySchemaTestPublication() (Publication, UserFieldSchemaBinding, ProviderOperationSchemaBinding) {
	publication := testPublication(1)
	provider := &publication.Identity.Providers[0]
	provider.Operations = []ProviderOperation{{
		Name: "risk.evaluate", InputSchema: "schemas/risk-input.json",
		OutputSchema: "fixture.identity.risk.output@1", TimeoutMS: 1_000,
		FailurePolicy: ProviderFailureFailClosed,
	}}
	field := UserFieldSchemaBinding{
		FieldID:         publication.Identity.UserFields[0].ID,
		ContractVersion: publication.Identity.UserFields[0].ContractVersion,
		Artifact:        publication.Artifact,
		Schema: schemaTestMaterial(
			publication.Identity.UserFields[0].Schema,
			"fixture.identity.field.code.schema@1",
			`{"type":"string","minLength":2,"private_schema_marker":true}`,
		),
	}
	operation := ProviderOperationSchemaBinding{
		ProviderID: provider.ID, ContractVersion: provider.ContractVersion,
		Operation: provider.Operations[0].Name, Artifact: publication.Artifact,
		Input: schemaTestMaterial(
			provider.Operations[0].InputSchema, "fixture.identity.risk.input@1",
			`{"type":"object","required":["risk"],"properties":{"risk":{"type":"boolean"}},"additionalProperties":false,"private_schema_marker":true}`,
		),
		Output: schemaTestMaterial(
			provider.Operations[0].OutputSchema, "fixture.identity.risk.output@1",
			`{"type":"object","required":["disposition"],"properties":{"disposition":{"enum":["allow","deny","step_up"]}},"additionalProperties":false,"private_schema_marker":true}`,
		),
	}
	return publication, field, operation
}

func schemaTestMaterial(reference, wireReference, schema string) JSONSchemaMaterial {
	body := []byte(schema)
	digest := sha256.Sum256(body)
	return JSONSchemaMaterial{
		Reference: reference, WireReference: wireReference,
		Digest: hex.EncodeToString(digest[:]), Schema: body,
	}
}
