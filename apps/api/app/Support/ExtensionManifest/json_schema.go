package extensionmanifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const manifestV3SchemaURL = "https://sforum.dev/schemas/extensions/manifest-v3.schema.json"

//go:embed schemas/manifest-v3.schema.json
var manifestV3SchemaBody []byte

var (
	manifestSchemaOnce sync.Once
	manifestSchemas    map[string]*jsonschema.Schema
	manifestSchemaErr  error
)

func ManifestV3JSONSchema() []byte {
	return append([]byte(nil), manifestV3SchemaBody...)
}

func ValidateV3JSONSchema(body []byte) error {
	return validateV3JSONSchemaFragment(body, "")
}

func validateV3JSONSchemaFragment(body []byte, fragment string) error {
	schema, err := compiledManifestSchema(fragment)
	if err != nil {
		return fmt.Errorf("%w: compile V3 JSON Schema: %v", ErrInvalidManifest, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: decode V3 JSON: %v", ErrInvalidManifest, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%w: trailing V3 JSON value", ErrInvalidManifest)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("%w: V3 JSON Schema: %v", ErrInvalidManifest, err)
	}
	return nil
}

func compiledManifestSchema(fragment string) (*jsonschema.Schema, error) {
	manifestSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		compiler.AssertFormat()
		var resource any
		if err := json.Unmarshal(manifestV3SchemaBody, &resource); err != nil {
			manifestSchemaErr = err
			return
		}
		if err := compiler.AddResource(manifestV3SchemaURL, resource); err != nil {
			manifestSchemaErr = err
			return
		}
		manifestSchemas = map[string]*jsonschema.Schema{}
		fragments := []string{"", "guards", "schedules", "components", "templates", "assets", "content", "database", "cache", "seo", "services", "commands", "adminSurfaces", "queries", "identity", "permissionDefinitions", "media", "navigation", "regions", "dependencies", "lifecycle", "openapi", "packageFiles"}
		for _, name := range fragments {
			location := manifestV3SchemaURL
			if name != "" {
				location += "#/$defs/" + name
			}
			schema, err := compiler.Compile(location)
			if err != nil {
				manifestSchemaErr = err
				return
			}
			manifestSchemas[name] = schema
		}
	})
	if manifestSchemaErr != nil {
		return nil, manifestSchemaErr
	}
	schema, ok := manifestSchemas[fragment]
	if !ok {
		return nil, fmt.Errorf("unknown manifest schema fragment %q", fragment)
	}
	return schema, nil
}
