package identityregistry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	maximumIdentitySchemaBytes = 1 << 20
	maximumIdentitySchemaDepth = 64
	identitySchemaDraft2020URI = "https://json-schema.org/draft/2020-12/schema"
)

type JSONSchemaMaterial struct {
	Reference     string
	WireReference string
	Digest        string
	Schema        []byte
}

type UserFieldSchemaBinding struct {
	FieldID         string
	ContractVersion string
	Artifact        Artifact
	Schema          JSONSchemaMaterial
}

type ProviderOperationSchemaBinding struct {
	ProviderID      string
	ContractVersion string
	Operation       string
	Artifact        Artifact
	Input           JSONSchemaMaterial
	Output          JSONSchemaMaterial
}

type UserFieldSchemaClaim struct {
	FieldID         string
	ContractVersion string
	Artifact        Artifact
}

type ProviderOperationSchemaClaim struct {
	ProviderID      string
	ContractVersion string
	Operation       string
	Artifact        Artifact
}

type compiledIdentitySchema struct {
	reference     string
	wireReference string
	digest        string
	schema        []byte
	validator     *jsonschema.Schema
}

// BindJSONSchemas attaches exact package bytes and compiled validators as
// private publication material. Host-derived wire references and digests remain
// inspectable, while raw bytes and validator pointers never enter JSON.
func BindJSONSchemas(
	publication Publication,
	fields []UserFieldSchemaBinding,
	operations []ProviderOperationSchemaBinding,
) (Publication, error) {
	rebindFields := make(map[string]struct{})
	if publication.Identity != nil {
		for _, field := range publication.Identity.UserFields {
			if field.SchemaWireReference != "" || field.SchemaDigest != "" || field.boundSchema != nil {
				rebindFields[strings.ToLower(strings.TrimSpace(field.ID))] = struct{}{}
			}
		}
	}
	prepared := clearDerivedSchemaMaterial(publication)
	normalized, err := normalizePublication(prepared)
	if err != nil {
		return Publication{}, err
	}
	result := clonePublication(normalized)
	if result.Identity == nil && (len(fields) > 0 || len(operations) > 0) {
		return Publication{}, ErrInvalid
	}

	fieldIndex := make(map[string]int)
	providerIndex := make(map[string]int)
	operationIndex := make(map[string]map[string]int)
	if result.Identity != nil {
		for index := range result.Identity.UserFields {
			fieldIndex[result.Identity.UserFields[index].ID] = index
		}
		for index := range result.Identity.Providers {
			provider := &result.Identity.Providers[index]
			providerIndex[provider.ID] = index
			operationsByName := make(map[string]int, len(provider.Operations))
			for operation := range provider.Operations {
				operationsByName[provider.Operations[operation].Name] = operation
			}
			operationIndex[provider.ID] = operationsByName
		}
	}

	boundFields := make(map[string]struct{}, len(fields))
	for _, raw := range fields {
		raw.FieldID = strings.ToLower(strings.TrimSpace(raw.FieldID))
		raw.ContractVersion = strings.TrimSpace(raw.ContractVersion)
		artifact, artifactErr := normalizeArtifact(raw.Artifact)
		if artifactErr != nil || artifact != result.Artifact {
			return Publication{}, ErrInvalid
		}
		index, found := fieldIndex[raw.FieldID]
		if !found {
			return Publication{}, ErrInvalid
		}
		if _, duplicate := boundFields[raw.FieldID]; duplicate {
			return Publication{}, ErrConflict
		}
		field := &result.Identity.UserFields[index]
		if field.ContractVersion != raw.ContractVersion || field.Schema != strings.TrimSpace(raw.Schema.Reference) {
			return Publication{}, ErrInvalid
		}
		compiled, compileErr := compileIdentitySchema(raw.Schema)
		if compileErr != nil {
			return Publication{}, compileErr
		}
		field.SchemaWireReference = compiled.wireReference
		field.SchemaDigest = compiled.digest
		field.boundSchema = &compiled
		boundFields[raw.FieldID] = struct{}{}
	}
	for fieldID := range rebindFields {
		if _, rebound := boundFields[fieldID]; !rebound {
			return Publication{}, ErrInvalid
		}
	}

	boundOperations := make(map[string]struct{}, len(operations))
	for _, raw := range operations {
		raw.ProviderID = strings.ToLower(strings.TrimSpace(raw.ProviderID))
		raw.ContractVersion = strings.TrimSpace(raw.ContractVersion)
		raw.Operation = strings.ToLower(strings.TrimSpace(raw.Operation))
		artifact, artifactErr := normalizeArtifact(raw.Artifact)
		if artifactErr != nil || artifact != result.Artifact {
			return Publication{}, ErrInvalid
		}
		providerAt, found := providerIndex[raw.ProviderID]
		if !found {
			return Publication{}, ErrInvalid
		}
		operationAt, found := operationIndex[raw.ProviderID][raw.Operation]
		if !found {
			return Publication{}, ErrInvalid
		}
		key := raw.ProviderID + "\x00" + raw.Operation
		if _, duplicate := boundOperations[key]; duplicate {
			return Publication{}, ErrConflict
		}
		provider := &result.Identity.Providers[providerAt]
		operation := &provider.Operations[operationAt]
		if provider.ContractVersion != raw.ContractVersion ||
			operation.InputSchema != strings.TrimSpace(raw.Input.Reference) ||
			operation.OutputSchema != strings.TrimSpace(raw.Output.Reference) {
			return Publication{}, ErrInvalid
		}
		input, inputErr := compileIdentitySchema(raw.Input)
		if inputErr != nil {
			return Publication{}, inputErr
		}
		output, outputErr := compileIdentitySchema(raw.Output)
		if outputErr != nil {
			return Publication{}, outputErr
		}
		operation.InputSchemaWireReference = input.wireReference
		operation.InputSchemaDigest = input.digest
		operation.OutputSchemaWireReference = output.wireReference
		operation.OutputSchemaDigest = output.digest
		operation.boundInputSchema = &input
		operation.boundOutputSchema = &output
		boundOperations[key] = struct{}{}
	}

	normalized, err = normalizePublication(result)
	if err != nil {
		return Publication{}, err
	}
	if err := validateExecutableBindings(normalized); err != nil {
		return Publication{}, err
	}
	return normalized, nil
}

func (r *Registry) ValidateUserFieldValue(claim UserFieldSchemaClaim, value any) error {
	claim.FieldID = strings.ToLower(strings.TrimSpace(claim.FieldID))
	claim.ContractVersion = strings.TrimSpace(claim.ContractVersion)
	artifact, err := normalizeArtifact(claim.Artifact)
	if r == nil || err != nil {
		return ErrSchemaUnavailable
	}
	field, found := r.load().userFields[claim.FieldID]
	if !found || field.Artifact != artifact || field.ContractVersion != claim.ContractVersion ||
		field.boundSchema == nil {
		return ErrSchemaUnavailable
	}
	return validateIdentitySchemaValue(*field.boundSchema, value)
}

func (r *Registry) ValidateProviderOperationInput(claim ProviderOperationSchemaClaim, value any) error {
	return r.validateProviderOperationValue(claim, value, true)
}

func (r *Registry) ValidateProviderOperationOutput(claim ProviderOperationSchemaClaim, value any) error {
	return r.validateProviderOperationValue(claim, value, false)
}

func (r *Registry) validateProviderOperationValue(
	claim ProviderOperationSchemaClaim,
	value any,
	input bool,
) error {
	claim.ProviderID = strings.ToLower(strings.TrimSpace(claim.ProviderID))
	claim.ContractVersion = strings.TrimSpace(claim.ContractVersion)
	claim.Operation = strings.ToLower(strings.TrimSpace(claim.Operation))
	artifact, err := normalizeArtifact(claim.Artifact)
	if r == nil || err != nil {
		return ErrSchemaUnavailable
	}
	provider, found := r.load().providers[claim.ProviderID]
	if !found || provider.Artifact != artifact || provider.ContractVersion != claim.ContractVersion {
		return ErrSchemaUnavailable
	}
	for _, operation := range provider.Operations {
		if operation.Name != claim.Operation {
			continue
		}
		compiled := operation.boundOutputSchema
		if input {
			compiled = operation.boundInputSchema
		}
		if compiled == nil {
			return ErrSchemaUnavailable
		}
		return validateIdentitySchemaValue(*compiled, value)
	}
	return ErrSchemaUnavailable
}

func clearDerivedSchemaMaterial(publication Publication) Publication {
	result := clonePublication(publication)
	if result.Identity == nil {
		return result
	}
	for index := range result.Identity.UserFields {
		field := &result.Identity.UserFields[index]
		field.SchemaWireReference = ""
		field.SchemaDigest = ""
		field.boundSchema = nil
	}
	for providerIndex := range result.Identity.Providers {
		for operationIndex := range result.Identity.Providers[providerIndex].Operations {
			operation := &result.Identity.Providers[providerIndex].Operations[operationIndex]
			operation.InputSchemaWireReference = ""
			operation.InputSchemaDigest = ""
			operation.OutputSchemaWireReference = ""
			operation.OutputSchemaDigest = ""
			operation.boundInputSchema = nil
			operation.boundOutputSchema = nil
		}
	}
	return result
}

func validateExecutableBindings(publication Publication) error {
	if err := validatePublicSchemaMetadata(publication); err != nil {
		return err
	}
	if publication.Identity == nil {
		return nil
	}
	for _, field := range publication.Identity.UserFields {
		hasMetadata := field.SchemaWireReference != "" || field.SchemaDigest != "" || field.boundSchema != nil
		if !hasMetadata {
			continue
		}
		if err := validateBoundIdentitySchema(
			field.Schema, field.SchemaWireReference, field.SchemaDigest, field.boundSchema,
		); err != nil {
			return err
		}
	}
	for _, provider := range publication.Identity.Providers {
		for _, operation := range provider.Operations {
			if err := validateBoundIdentitySchema(
				operation.InputSchema, operation.InputSchemaWireReference,
				operation.InputSchemaDigest, operation.boundInputSchema,
			); err != nil {
				return err
			}
			if err := validateBoundIdentitySchema(
				operation.OutputSchema, operation.OutputSchemaWireReference,
				operation.OutputSchemaDigest, operation.boundOutputSchema,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePublicSchemaMetadata(publication Publication) error {
	if publication.Identity == nil {
		return nil
	}
	for _, field := range publication.Identity.UserFields {
		hasMetadata := field.SchemaWireReference != "" || field.SchemaDigest != ""
		if !hasMetadata {
			// Digestless legacy fields remain inspectable, but package-path fields
			// never had a valid legacy Registry representation.
			if !contractPattern.MatchString(field.Schema) {
				return ErrInvalid
			}
			continue
		}
		if !validSchemaReference(field.Schema) || !contractPattern.MatchString(field.SchemaWireReference) ||
			!digestPattern.MatchString(field.SchemaDigest) {
			return ErrInvalid
		}
	}
	for _, provider := range publication.Identity.Providers {
		for _, operation := range provider.Operations {
			if !validSchemaReference(operation.InputSchema) ||
				!contractPattern.MatchString(operation.InputSchemaWireReference) ||
				!digestPattern.MatchString(operation.InputSchemaDigest) ||
				!validSchemaReference(operation.OutputSchema) ||
				!contractPattern.MatchString(operation.OutputSchemaWireReference) ||
				!digestPattern.MatchString(operation.OutputSchemaDigest) {
				return ErrInvalid
			}
		}
	}
	return nil
}

func validateBoundIdentitySchema(
	reference, wireReference, digest string,
	compiled *compiledIdentitySchema,
) error {
	if !validSchemaReference(reference) || !contractPattern.MatchString(wireReference) ||
		!digestPattern.MatchString(digest) || compiled == nil || compiled.validator == nil ||
		compiled.reference != reference || compiled.wireReference != wireReference ||
		compiled.digest != digest || len(compiled.schema) == 0 || len(compiled.schema) > maximumIdentitySchemaBytes {
		return ErrInvalid
	}
	sum := sha256.Sum256(compiled.schema)
	if hex.EncodeToString(sum[:]) != digest {
		return ErrInvalid
	}
	return nil
}

func cloneCompiledIdentitySchema(value compiledIdentitySchema) compiledIdentitySchema {
	value.schema = append([]byte(nil), value.schema...)
	return value
}

func compileIdentitySchema(raw JSONSchemaMaterial) (compiledIdentitySchema, error) {
	raw.Reference = strings.TrimSpace(raw.Reference)
	raw.WireReference = strings.TrimSpace(raw.WireReference)
	raw.Digest = strings.ToLower(strings.TrimSpace(raw.Digest))
	if !validSchemaReference(raw.Reference) || !contractPattern.MatchString(raw.WireReference) ||
		!digestPattern.MatchString(raw.Digest) || len(raw.Schema) == 0 || len(raw.Schema) > maximumIdentitySchemaBytes {
		return compiledIdentitySchema{}, ErrInvalid
	}
	sum := sha256.Sum256(raw.Schema)
	if hex.EncodeToString(sum[:]) != raw.Digest {
		return compiledIdentitySchema{}, ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw.Schema))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return compiledIdentitySchema{}, ErrInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return compiledIdentitySchema{}, ErrInvalid
	}
	if err := validateIdentitySchemaDocument(document, 0); err != nil {
		return compiledIdentitySchema{}, err
	}
	resource := "https://sforum.invalid/identity-schema/" + raw.Digest + ".json"
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	if err := compiler.AddResource(resource, document); err != nil {
		return compiledIdentitySchema{}, ErrInvalid
	}
	validator, err := compiler.Compile(resource)
	if err != nil {
		return compiledIdentitySchema{}, ErrInvalid
	}
	return compiledIdentitySchema{
		reference: raw.Reference, wireReference: raw.WireReference, digest: raw.Digest,
		schema: append([]byte(nil), raw.Schema...), validator: validator,
	}, nil
}

func validateIdentitySchemaDocument(value any, depth int) error {
	if depth > maximumIdentitySchemaDepth {
		return ErrInvalid
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch key {
			case "$schema":
				dialect, ok := child.(string)
				if !ok || dialect != identitySchemaDraft2020URI {
					return ErrInvalid
				}
			case "$ref", "$dynamicRef", "$recursiveRef":
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#") {
					return ErrInvalid
				}
			}
			if err := validateIdentitySchemaDocument(child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := validateIdentitySchemaDocument(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateIdentitySchemaValue(compiled compiledIdentitySchema, value any) error {
	if compiled.validator == nil {
		return ErrSchemaUnavailable
	}
	if err := compiled.validator.Validate(value); err != nil {
		// Identity values may contain personal data. Validator diagnostics can
		// echo rejected values, so callers receive only the stable failure class.
		return ErrSchemaValueInvalid
	}
	return nil
}
