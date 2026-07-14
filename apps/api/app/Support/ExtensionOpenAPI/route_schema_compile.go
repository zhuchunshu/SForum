package extensionopenapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func compileRouteSchemas(
	aggregate map[string]any,
	candidates []routeSchemaCandidate,
	revision string,
) (map[string]*jsonschema.Schema, error) {
	aggregateURL := "https://sforum.invalid/route-schema-catalog/" + strings.TrimPrefix(revision, "sha256:") + "/aggregate.json"
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	if err := compiler.AddResource(aggregateURL, aggregate); err != nil {
		return nil, fmt.Errorf("%w: add aggregate resource: %v", ErrRouteSchemaCatalogInvalid, err)
	}
	values := make(map[string]any)
	for _, candidate := range candidates {
		if _, exists := values[candidate.binding.SchemaDigest]; exists {
			continue
		}
		rebased, err := rebaseAggregateSchemaReferences(candidate.value, aggregateURL, 0)
		if err != nil {
			return nil, err
		}
		values[candidate.binding.SchemaDigest] = rebased
	}
	digests := sortedMapKeys(values)
	for _, digest := range digests {
		if err := compiler.AddResource(routeSchemaResourceURL(digest), values[digest]); err != nil {
			return nil, fmt.Errorf("%w: add schema %s: %v", ErrRouteSchemaCatalogInvalid, digest, err)
		}
	}
	compiled := make(map[string]*jsonschema.Schema, len(values))
	for _, digest := range digests {
		schema, err := compiler.Compile(routeSchemaResourceURL(digest))
		if err != nil {
			return nil, fmt.Errorf("%w: compile schema %s: %v", ErrRouteSchemaCatalogInvalid, digest, err)
		}
		compiled[digest] = schema
	}
	return compiled, nil
}

func routeSchemaClosureBytes(aggregate map[string]any, candidates []routeSchemaCandidate, limit int) (int, error) {
	if limit <= 0 {
		return 0, ErrResourceBudget
	}
	total := 0
	seenDigests := make(map[string]struct{})
	seenReferences := make(map[string]struct{})
	addValue := func(value any, canonical []byte) error {
		if canonical == nil {
			var err error
			canonical, err = json.Marshal(value)
			if err != nil {
				return ErrRouteSchemaCatalogInvalid
			}
		}
		digest := digestRouteSchema(canonical)
		if _, exists := seenDigests[digest]; exists {
			return nil
		}
		seenDigests[digest] = struct{}{}
		total += len(canonical)
		if total > limit {
			return fmt.Errorf("%w: route schema closure exceeds %d bytes", ErrResourceBudget, limit)
		}
		return nil
	}
	var walk func(any, int) error
	walk = func(value any, depth int) error {
		if depth > maxDocumentDepth {
			return ErrResourceBudget
		}
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if key == "$dynamicRef" || key == "$recursiveRef" {
					return fmt.Errorf("%w: %s is not supported in route schemas", ErrUnsafeReference, key)
				}
				if key == "$ref" {
					text, ok := child.(string)
					if !ok || !strings.HasPrefix(text, "#/") {
						return ErrUnsafeReference
					}
					if _, seen := seenReferences[text]; seen {
						continue
					}
					seenReferences[text] = struct{}{}
					target, err := resolveJSONPointer(aggregate, strings.TrimPrefix(text, "#"))
					if err != nil {
						return err
					}
					if err := addValue(target, nil); err != nil {
						return err
					}
					if err := walk(target, depth+1); err != nil {
						return err
					}
					continue
				}
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, candidate := range candidates {
		if err := addValue(candidate.value, candidate.canonical); err != nil {
			return 0, err
		}
		if err := walk(candidate.value, 0); err != nil {
			return 0, err
		}
	}
	return total, nil
}

func enforceRouteSchemaComplexity(
	aggregate map[string]any,
	candidates []routeSchemaCandidate,
	limits routeSchemaComplexityLimits,
) error {
	if limits.nodes <= 0 || limits.depth <= 0 || limits.branches <= 0 || limits.refExpansions <= 0 {
		return ErrResourceBudget
	}
	var nodes, branches, refExpansions int
	seenDigests := make(map[string]struct{})
	var walk func(any, int, map[string]bool) error
	walk = func(value any, depth int, refStack map[string]bool) error {
		if depth > limits.depth {
			return fmt.Errorf("%w: route schema depth exceeds %d", ErrResourceBudget, limits.depth)
		}
		nodes++
		if nodes > limits.nodes {
			return fmt.Errorf("%w: route schema nodes exceed %d", ErrResourceBudget, limits.nodes)
		}
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if key == "$ref" {
					text, ok := child.(string)
					if !ok || !strings.HasPrefix(text, "#/") {
						return ErrUnsafeReference
					}
					refExpansions++
					if refExpansions > limits.refExpansions {
						return fmt.Errorf("%w: route schema ref expansions exceed %d", ErrResourceBudget, limits.refExpansions)
					}
					if refStack[text] {
						continue
					}
					target, err := resolveJSONPointer(aggregate, strings.TrimPrefix(text, "#"))
					if err != nil {
						return err
					}
					refStack[text] = true
					err = walk(target, depth+1, refStack)
					delete(refStack, text)
					if err != nil {
						return err
					}
					continue
				}
				if key == "allOf" || key == "anyOf" || key == "oneOf" || key == "prefixItems" {
					if values, ok := child.([]any); ok {
						branches += len(values)
						if branches > limits.branches {
							return fmt.Errorf("%w: route schema branches exceed %d", ErrResourceBudget, limits.branches)
						}
					}
				}
				if err := walk(child, depth+1, refStack); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := walk(child, depth+1, refStack); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, candidate := range candidates {
		digest := digestRouteSchema(candidate.canonical)
		if _, exists := seenDigests[digest]; exists {
			continue
		}
		seenDigests[digest] = struct{}{}
		if err := walk(candidate.value, 0, make(map[string]bool)); err != nil {
			return err
		}
	}
	return nil
}

func rebaseAggregateSchemaReferences(value any, aggregateURL string, depth int) (any, error) {
	if depth > maxDocumentDepth {
		return nil, ErrResourceBudget
	}
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if key == "$dynamicRef" || key == "$recursiveRef" {
				return nil, fmt.Errorf("%w: %s is not supported in route schemas", ErrUnsafeReference, key)
			}
			if key == "$ref" {
				text, ok := child.(string)
				if !ok || !strings.HasPrefix(text, "#/") {
					return nil, ErrUnsafeReference
				}
				result[key] = aggregateURL + text
				continue
			}
			cloned, err := rebaseAggregateSchemaReferences(child, aggregateURL, depth+1)
			if err != nil {
				return nil, err
			}
			result[key] = cloned
		}
		return result, nil
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			cloned, err := rebaseAggregateSchemaReferences(child, aggregateURL, depth+1)
			if err != nil {
				return nil, err
			}
			result[index] = cloned
		}
		return result, nil
	default:
		return typed, nil
	}
}

func routeSchemaResourceURL(digest string) string {
	return "https://sforum.invalid/route-schema-catalog/schema/" + digest + ".json"
}

func digestRouteSchema(canonical []byte) string {
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}
