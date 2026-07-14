package extensionopenapi

import (
	"fmt"
	"strings"
)

func validatePathParameters(operation map[string]any, pathValue string, artifact *loadedArtifact, sourcePath string) error {
	expected := pathTemplateParameters(pathValue)
	parametersValue, exists := operation["parameters"]
	if !exists {
		if len(expected) != 0 {
			return fmt.Errorf("path template parameters are undocumented")
		}
		return nil
	}
	parameters, ok := parametersValue.([]any)
	if !ok {
		return fmt.Errorf("parameters must be an array")
	}
	seen := make(map[string]struct{}, len(expected))
	for _, value := range parameters {
		parameter, parameterSource, err := resolveObject(value, artifact, sourcePath, 0)
		if err != nil {
			return fmt.Errorf("invalid parameter: %w", err)
		}
		location, locationOK := canonicalStringField(parameter, "in")
		name, nameOK := canonicalStringField(parameter, "name")
		if !locationOK || !nameOK {
			return fmt.Errorf("parameter name and in must be canonical strings")
		}
		schema, hasSchema := parameter["schema"]
		content, hasContent := parameter["content"]
		if hasSchema == hasContent {
			return fmt.Errorf("parameter %s must declare exactly one of schema or content", name)
		}
		if hasSchema {
			if err := validateSchemaValue(schema, artifact, parameterSource, 0, map[string]bool{}); err != nil {
				return fmt.Errorf("parameter %s schema: %w", name, err)
			}
		} else if contentHasSchema, err := validateContentSchemas(content, artifact, parameterSource); err != nil || !contentHasSchema {
			if err != nil {
				return fmt.Errorf("parameter %s content: %w", name, err)
			}
			return fmt.Errorf("parameter %s content requires a schema", name)
		}
		if required, exists := parameter["required"]; exists {
			if _, ok := required.(bool); !ok {
				return fmt.Errorf("parameter %s required must be boolean", name)
			}
		}
		if location != "path" {
			continue
		}
		if _, declared := expected[name]; !declared {
			return fmt.Errorf("extra path parameter %q", name)
		}
		if required, ok := parameter["required"].(bool); !ok || !required {
			return fmt.Errorf("path parameter %q must set required=true", name)
		}
		if !hasSchema {
			return fmt.Errorf("path parameter %q requires a schema", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("duplicate path parameter %q", name)
		}
		seen[name] = struct{}{}
	}
	for name := range expected {
		if _, documented := seen[name]; !documented {
			return fmt.Errorf("path parameter %q is missing", name)
		}
	}
	return nil
}

func pathTemplateParameters(pathValue string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, segment := range strings.Split(pathValue, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			result[strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")] = struct{}{}
		}
	}
	return result
}

func validateRequestBody(value any, exists bool, artifact *loadedArtifact, sourcePath string) (hasSchema, present bool, err error) {
	if !exists {
		return false, false, nil
	}
	if value == nil {
		return false, true, fmt.Errorf("requestBody cannot be null")
	}
	body, bodySource, err := resolveObject(value, artifact, sourcePath, 0)
	if err != nil {
		return false, true, err
	}
	if required, exists := body["required"]; exists {
		if _, ok := required.(bool); !ok {
			return false, true, fmt.Errorf("requestBody.required must be boolean")
		}
	}
	hasSchema, err = validateContentSchemas(body["content"], artifact, bodySource)
	return hasSchema, true, err
}

func validateResponses(responses map[string]any, artifact *loadedArtifact, sourcePath string) (bool, error) {
	hasSuccessfulSchema := false
	for status, value := range responses {
		if !responseStatusPattern.MatchString(status) {
			return false, fmt.Errorf("invalid response key %q", status)
		}
		response, responseSource, err := resolveObject(value, artifact, sourcePath, 0)
		if err != nil {
			return false, fmt.Errorf("response %s: %w", status, err)
		}
		if _, referenced := response["$ref"]; !referenced {
			if _, ok := canonicalStringField(response, "description"); !ok {
				return false, fmt.Errorf("response %s requires a canonical description", status)
			}
		}
		hasSchema, err := validateContentSchemas(response["content"], artifact, responseSource)
		if err != nil {
			return false, fmt.Errorf("response %s: %w", status, err)
		}
		if len(status) == 3 && status[0] == '2' {
			hasSuccessfulSchema = hasSuccessfulSchema || hasSchema
		}
	}
	return hasSuccessfulSchema, nil
}

func validateContentSchemas(value any, artifact *loadedArtifact, sourcePath string) (bool, error) {
	if value == nil {
		return false, nil
	}
	content, ok := value.(map[string]any)
	if !ok || len(content) == 0 {
		return false, fmt.Errorf("content must be a non-empty object")
	}
	hasSchema := false
	for mediaType, mediaValue := range content {
		if mediaType == "" || mediaType != strings.TrimSpace(mediaType) {
			return false, fmt.Errorf("invalid media type key")
		}
		media, ok := mediaValue.(map[string]any)
		if !ok {
			return false, fmt.Errorf("media type %s must be an object", mediaType)
		}
		schema, exists := media["schema"]
		if !exists {
			continue
		}
		if err := validateSchemaValue(schema, artifact, sourcePath, 0, map[string]bool{}); err != nil {
			return false, fmt.Errorf("media type %s schema: %w", mediaType, err)
		}
		hasSchema = true
	}
	return hasSchema, nil
}

func validateSchemaValue(value any, artifact *loadedArtifact, sourcePath string, depth int, visited map[string]bool) error {
	if depth > maxDocumentDepth {
		return fmt.Errorf("%w: schema nesting exceeds %d", ErrResourceBudget, maxDocumentDepth)
	}
	switch schema := value.(type) {
	case bool:
		return nil
	case map[string]any:
		if len(schema) == 0 {
			return fmt.Errorf("schema object cannot be empty")
		}
		if reference, exists := schema["$ref"]; exists {
			text, ok := reference.(string)
			if !ok || text == "" || text != strings.TrimSpace(text) {
				return fmt.Errorf("schema $ref must be canonical")
			}
			targetPath, pointer, err := resolvePackageReference(sourcePath, text)
			if err != nil {
				return err
			}
			location := targetPath + "#" + pointer
			if !visited[location] {
				visited[location] = true
				target, exists := artifact.documents[targetPath]
				if !exists {
					return fmt.Errorf("schema reference target is not loaded")
				}
				targetValue, err := resolveJSONPointer(target, pointer)
				if err != nil {
					return err
				}
				if err := validateSchemaValue(targetValue, artifact, targetPath, depth+1, visited); err != nil {
					return err
				}
			}
		}
		if schemaType, exists := schema["type"]; exists {
			if err := validateSchemaType(schemaType); err != nil {
				return err
			}
		}
		for _, key := range []string{"items", "contains", "not", "if", "then", "else", "propertyNames", "additionalProperties", "unevaluatedProperties", "unevaluatedItems"} {
			if child, exists := schema[key]; exists {
				if err := validateSchemaValue(child, artifact, sourcePath, depth+1, visited); err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
			}
		}
		for _, key := range []string{"allOf", "anyOf", "oneOf", "prefixItems"} {
			if children, exists := schema[key]; exists {
				array, ok := children.([]any)
				if !ok || len(array) == 0 {
					return fmt.Errorf("%s must be a non-empty schema array", key)
				}
				for _, child := range array {
					if err := validateSchemaValue(child, artifact, sourcePath, depth+1, visited); err != nil {
						return fmt.Errorf("%s: %w", key, err)
					}
				}
			}
		}
		for _, key := range []string{"properties", "patternProperties", "dependentSchemas", "$defs"} {
			if children, exists := schema[key]; exists {
				object, ok := children.(map[string]any)
				if !ok {
					return fmt.Errorf("%s must be a schema object map", key)
				}
				for name, child := range object {
					if name == "" {
						return fmt.Errorf("%s contains an empty key", key)
					}
					if err := validateSchemaValue(child, artifact, sourcePath, depth+1, visited); err != nil {
						return fmt.Errorf("%s.%s: %w", key, name, err)
					}
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("schema must be a non-empty object or boolean")
	}
}

func validateSchemaType(value any) error {
	valid := map[string]bool{"null": true, "boolean": true, "object": true, "array": true, "number": true, "string": true, "integer": true}
	switch typed := value.(type) {
	case string:
		if !valid[typed] {
			return fmt.Errorf("invalid schema type %q", typed)
		}
	case []any:
		if len(typed) == 0 {
			return fmt.Errorf("schema type array cannot be empty")
		}
		seen := map[string]bool{}
		for _, item := range typed {
			text, ok := item.(string)
			if !ok || !valid[text] || seen[text] {
				return fmt.Errorf("invalid schema type array")
			}
			seen[text] = true
		}
	default:
		return fmt.Errorf("schema type must be a string or non-empty string array")
	}
	return nil
}

func resolveObject(value any, artifact *loadedArtifact, sourcePath string, depth int) (map[string]any, string, error) {
	if depth > maxDocumentDepth {
		return nil, sourcePath, fmt.Errorf("%w: reference resolution exceeds depth %d", ErrResourceBudget, maxDocumentDepth)
	}
	object, ok := value.(map[string]any)
	if !ok || len(object) == 0 {
		return nil, sourcePath, fmt.Errorf("value must be a non-empty object")
	}
	reference, hasReference := object["$ref"]
	if !hasReference {
		return object, sourcePath, nil
	}
	text, ok := reference.(string)
	if !ok || text == "" || text != strings.TrimSpace(text) {
		return nil, sourcePath, fmt.Errorf("$ref must be canonical")
	}
	targetPath, pointer, err := resolvePackageReference(sourcePath, text)
	if err != nil {
		return nil, sourcePath, err
	}
	target, exists := artifact.documents[targetPath]
	if !exists {
		return nil, sourcePath, fmt.Errorf("reference target is not loaded")
	}
	targetValue, err := resolveJSONPointer(target, pointer)
	if err != nil {
		return nil, sourcePath, err
	}
	return resolveObject(targetValue, artifact, targetPath, depth+1)
}
