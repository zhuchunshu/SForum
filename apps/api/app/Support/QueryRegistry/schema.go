package queryregistry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	maximumResultSchemaBytes = 1 << 20
	maximumSchemaDepth       = 64
)

type JSONResultSchemaBinding struct {
	QueryID         string
	ContractVersion string
	PlanVersion     string
	ResultSchema    string
	Artifact        Artifact
	SchemaDigest    string
	Schema          []byte
}

type compiledResultSchema struct {
	binding   JSONResultSchemaBinding
	validator *jsonschema.Schema
}

// BindResultSchemas compiles exact result Schemas and attaches them to their
// declarations as private publication material. Package-external callers can
// inspect the derived digest but cannot construct the validator or raw-byte
// binding that Registry requires before publication.
func BindResultSchemas(publication Publication, bindings []JSONResultSchemaBinding) (Publication, error) {
	artifact, err := normalizeArtifact(publication.Artifact)
	if err != nil || len(bindings) > len(publication.Queries) {
		return Publication{}, ErrExecutionInvalid
	}
	result := clonePublication(publication)
	result.Artifact = artifact
	queries := make(map[string]int, len(result.Queries))
	for index, raw := range result.Queries {
		declaration, declarationErr := normalizeQueryDeclaration(artifact, raw)
		if declarationErr != nil {
			return Publication{}, errors.Join(ErrExecutionInvalid, declarationErr)
		}
		if _, duplicate := queries[declaration.ID]; duplicate {
			return Publication{}, fmt.Errorf("%w: duplicate query %s", ErrExecutionInvalid, declaration.ID)
		}
		declaration.ResultSchemaDigest = ""
		declaration.boundResultSchema = nil
		result.Queries[index] = declaration
		queries[declaration.ID] = index
	}
	for _, raw := range bindings {
		compiled, compileErr := compileJSONResultSchema(raw)
		if compileErr != nil {
			return Publication{}, compileErr
		}
		index, found := queries[compiled.binding.QueryID]
		if !found || compiled.binding.Artifact != artifact {
			return Publication{}, fmt.Errorf("%w: result schema has no exact query declaration", ErrExecutionInvalid)
		}
		declaration := &result.Queries[index]
		if declaration.boundResultSchema != nil ||
			declaration.ContractVersion != compiled.binding.ContractVersion ||
			declaration.PlanVersion != compiled.binding.PlanVersion ||
			declaration.ResultSchema != compiled.binding.ResultSchema {
			return Publication{}, fmt.Errorf("%w: result schema does not match query %s", ErrExecutionInvalid, declaration.ID)
		}
		material := cloneCompiledResultSchema(compiled)
		declaration.ResultSchemaDigest = material.binding.SchemaDigest
		declaration.boundResultSchema = &material
	}
	return result, nil
}

// JSONResultSchemaCatalog compiles a complete exact-artifact schema snapshot
// once. It uses the project's established Draft 2020-12 validator and rejects
// external references so result validation never performs network or package
// reads on the execution path.
type JSONResultSchemaCatalog struct {
	bindings map[string]compiledResultSchema
}

func NewJSONResultSchemaCatalog(bindings []JSONResultSchemaBinding) (*JSONResultSchemaCatalog, error) {
	catalog := &JSONResultSchemaCatalog{bindings: make(map[string]compiledResultSchema, len(bindings))}
	for _, raw := range bindings {
		compiled, err := compileJSONResultSchema(raw)
		if err != nil {
			return nil, err
		}
		if _, exists := catalog.bindings[compiled.binding.QueryID]; exists {
			return nil, fmt.Errorf("%w: duplicate result schema for %s", ErrExecutionInvalid, compiled.binding.QueryID)
		}
		catalog.bindings[compiled.binding.QueryID] = compiled
	}
	return catalog, nil
}

func (c *JSONResultSchemaCatalog) ValidateQueryResult(_ context.Context, claim ResultSchemaClaim, row QueryRow) error {
	if c == nil {
		return ErrResultInvalid
	}
	compiled, ok := c.bindings[claim.QueryID]
	if !ok {
		return ErrResultInvalid
	}
	return validateCompiledResultSchema(compiled, claim, row)
}

// ValidateQueryResult resolves a validator from the same immutable Registry
// state that owns the query declaration. Lifecycle publication therefore cannot
// expose a new declaration with an old Schema sidecar.
func (r *Registry) ValidateQueryResult(_ context.Context, claim ResultSchemaClaim, row QueryRow) error {
	if r == nil {
		return ErrResultInvalid
	}
	compiled, ok := r.load().schemas[claim.QueryID]
	if !ok {
		return ErrResultInvalid
	}
	return validateCompiledResultSchema(compiled, claim, row)
}

func validateCompiledResultSchema(compiled compiledResultSchema, claim ResultSchemaClaim, row QueryRow) error {
	if compiled.binding.QueryID != claim.QueryID || compiled.binding.ContractVersion != claim.ContractVersion ||
		compiled.binding.PlanVersion != claim.PlanVersion || compiled.binding.ResultSchema != claim.ResultSchema ||
		compiled.binding.Artifact != claim.Artifact || compiled.validator == nil {
		return ErrResultInvalid
	}
	if err := compiled.validator.Validate(map[string]any(row)); err != nil {
		return fmt.Errorf("%w: %v", ErrResultInvalid, err)
	}
	return nil
}

func publicationResultSchema(
	artifact Artifact,
	declaration QueryDeclaration,
) (compiledResultSchema, bool, error) {
	if declaration.ResultSchemaDigest == "" && declaration.boundResultSchema == nil {
		return compiledResultSchema{}, false, nil
	}
	if !digestPattern.MatchString(declaration.ResultSchemaDigest) || declaration.boundResultSchema == nil {
		return compiledResultSchema{}, false, ErrInvalid
	}
	compiled := *declaration.boundResultSchema
	binding := compiled.binding
	if compiled.validator == nil || binding.QueryID != declaration.ID ||
		binding.ContractVersion != declaration.ContractVersion || binding.PlanVersion != declaration.PlanVersion ||
		binding.ResultSchema != declaration.ResultSchema || binding.Artifact != artifact ||
		binding.SchemaDigest != declaration.ResultSchemaDigest || len(binding.Schema) == 0 ||
		len(binding.Schema) > maximumResultSchemaBytes {
		return compiledResultSchema{}, false, ErrInvalid
	}
	digest := sha256.Sum256(binding.Schema)
	if hex.EncodeToString(digest[:]) != binding.SchemaDigest {
		return compiledResultSchema{}, false, ErrInvalid
	}
	return cloneCompiledResultSchema(compiled), true, nil
}

func cloneCompiledResultSchema(value compiledResultSchema) compiledResultSchema {
	value.binding.Schema = append([]byte(nil), value.binding.Schema...)
	return value
}

func compileJSONResultSchema(raw JSONResultSchemaBinding) (compiledResultSchema, error) {
	raw.QueryID = strings.ToLower(strings.TrimSpace(raw.QueryID))
	raw.ContractVersion = strings.TrimSpace(raw.ContractVersion)
	raw.PlanVersion = strings.TrimSpace(raw.PlanVersion)
	raw.ResultSchema = strings.TrimSpace(raw.ResultSchema)
	raw.SchemaDigest = normalizeDigest(raw.SchemaDigest)
	artifact, err := normalizeArtifact(raw.Artifact)
	if err != nil || !validContributionIdentity(artifact, raw.QueryID, raw.ContractVersion) ||
		!contractPattern.MatchString(raw.PlanVersion) || !validSchemaRef(raw.ResultSchema) ||
		!digestPattern.MatchString(raw.SchemaDigest) || len(raw.Schema) == 0 || len(raw.Schema) > maximumResultSchemaBytes {
		return compiledResultSchema{}, ErrExecutionInvalid
	}
	raw.Artifact = artifact
	digest := sha256.Sum256(raw.Schema)
	if hex.EncodeToString(digest[:]) != raw.SchemaDigest {
		return compiledResultSchema{}, fmt.Errorf("%w: result schema digest mismatch", ErrExecutionInvalid)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw.Schema))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return compiledResultSchema{}, fmt.Errorf("%w: invalid result schema JSON", ErrExecutionInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return compiledResultSchema{}, fmt.Errorf("%w: result schema contains trailing JSON", ErrExecutionInvalid)
	}
	if err := rejectExternalResultSchemaRefs(document, 0); err != nil {
		return compiledResultSchema{}, err
	}
	resource := "https://sforum.invalid/query-result-schema/" + raw.SchemaDigest + ".json"
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource(resource, document); err != nil {
		return compiledResultSchema{}, fmt.Errorf("%w: add result schema", ErrExecutionInvalid)
	}
	validator, err := compiler.Compile(resource)
	if err != nil {
		return compiledResultSchema{}, fmt.Errorf("%w: compile result schema: %v", ErrExecutionInvalid, err)
	}
	raw.Schema = append([]byte(nil), raw.Schema...)
	return compiledResultSchema{binding: raw, validator: validator}, nil
}

func rejectExternalResultSchemaRefs(value any, depth int) error {
	if depth > maximumSchemaDepth {
		return fmt.Errorf("%w: result schema nesting exceeds Host bounds", ErrExecutionInvalid)
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "$ref" || key == "$dynamicRef" || key == "$recursiveRef" {
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#") {
					return fmt.Errorf("%w: external result schema references are not allowed", ErrExecutionInvalid)
				}
			}
			if err := rejectExternalResultSchemaRefs(child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectExternalResultSchemaRefs(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

var _ ResultSchemaValidator = (*Registry)(nil)
