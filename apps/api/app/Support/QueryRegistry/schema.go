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
	if !ok || compiled.binding.ContractVersion != claim.ContractVersion ||
		compiled.binding.PlanVersion != claim.PlanVersion || compiled.binding.ResultSchema != claim.ResultSchema ||
		compiled.binding.Artifact != claim.Artifact || compiled.validator == nil {
		return ErrResultInvalid
	}
	if err := compiled.validator.Validate(map[string]any(row)); err != nil {
		return fmt.Errorf("%w: %v", ErrResultInvalid, err)
	}
	return nil
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
