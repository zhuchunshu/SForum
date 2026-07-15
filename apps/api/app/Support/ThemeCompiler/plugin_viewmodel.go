package themecompiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

const MaxPluginViewModelSchemaBytes = 1 << 20

// PluginPageViewModelSchema identifies one plugin-owned business-data
// contract. Template identity and schema version are kept separate so a theme
// can replace presentation without claiming ownership of the payload contract.
type PluginPageViewModelSchema struct {
	ViewModelID   string `json:"viewModelId"`
	SchemaVersion string `json:"schemaVersion"`
	SchemaDigest  string `json:"schemaDigest"`
}

// PluginPageViewModelContract is immutable after compilation. Only Host-loaded
// exact package schema bytes can construct it; request data must validate before
// it can become a BoundPageViewModel.
type PluginPageViewModelContract struct {
	descriptor PluginPageViewModelSchema
	validator  *jsonschema.Schema
}

func CompilePluginPageViewModelContract(
	viewModelID, schemaVersion, schemaDigest string,
	schemaBody []byte,
) (*PluginPageViewModelContract, error) {
	viewModelID = strings.TrimSpace(viewModelID)
	schemaVersion = strings.TrimSpace(schemaVersion)
	schemaDigest = strings.ToLower(strings.TrimSpace(schemaDigest))
	if !viewModelIDPattern.MatchString(viewModelID) || !schemaVersionPattern.MatchString(schemaVersion) ||
		!canonicalDigestPattern.MatchString(schemaDigest) || len(schemaBody) == 0 || len(schemaBody) > MaxPluginViewModelSchemaBytes {
		return nil, ErrPluginViewModelSchema
	}
	digest := sha256.Sum256(schemaBody)
	if hex.EncodeToString(digest[:]) != schemaDigest {
		return nil, fmt.Errorf("%w: exact schema digest mismatch", ErrPluginViewModelSchema)
	}

	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBody))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPluginViewModelSchema, err)
	}
	if err := rejectExternalPluginSchemaReferences(document, 0); err != nil {
		return nil, err
	}

	resource := "https://sforum.invalid/plugin-page-viewmodel/" + schemaDigest + ".json"
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource(resource, document); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPluginViewModelSchema, err)
	}
	compiled, err := compiler.Compile(resource)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPluginViewModelSchema, err)
	}
	return &PluginPageViewModelContract{
		descriptor: PluginPageViewModelSchema{
			ViewModelID: viewModelID, SchemaVersion: schemaVersion, SchemaDigest: schemaDigest,
		},
		validator: compiled,
	}, nil
}

func (c *PluginPageViewModelContract) Schema() PluginPageViewModelSchema {
	if c == nil {
		return PluginPageViewModelSchema{}
	}
	return c.descriptor
}

// Bind validates and clones one JSON document into a passive DTO.
func (c *PluginPageViewModelContract) Bind(
	themePackageDigest string,
	payload json.RawMessage,
	seo PageSEOView,
) (BoundPageViewModel, error) {
	if c == nil || c.validator == nil || !canonicalDigestPattern.MatchString(themePackageDigest) {
		return BoundPageViewModel{}, ErrViewModelTheme
	}
	if len(payload) == 0 {
		return BoundPageViewModel{}, ErrInvalidViewModel
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return BoundPageViewModel{}, ErrInvalidViewModel
	}
	switch document.(type) {
	case map[string]any, []any:
	default:
		return BoundPageViewModel{}, ErrInvalidViewModel
	}
	if err := c.validator.Validate(document); err != nil {
		return BoundPageViewModel{}, fmt.Errorf("%w: %v", ErrViewModelSchema, err)
	}
	if err := validatePassiveViewModel(document); err != nil {
		return BoundPageViewModel{}, err
	}
	sealedBody, err := json.Marshal(document)
	if err != nil {
		return BoundPageViewModel{}, ErrInvalidViewModel
	}
	sealed, err := jsonschema.UnmarshalJSON(bytes.NewReader(sealedBody))
	if err != nil {
		return BoundPageViewModel{}, ErrInvalidViewModel
	}
	return BoundPageViewModel{
		pageID: c.descriptor.ViewModelID, schemaVersion: c.descriptor.SchemaVersion, pluginSchemaDigest: c.descriptor.SchemaDigest,
		themePackageDigest: themePackageDigest, value: sealed, seo: cloneSEOView(seo),
	}, nil
}

// Rebind keeps the sealed DTO unchanged while granting one compatible compiled
// snapshot permission to render it. BoundPageViewModel does not expose the DTO,
// so templates cannot replace or mutate plugin business data between fallbacks.
func (c *PluginPageViewModelContract) Rebind(
	model BoundPageViewModel,
	themePackageDigest string,
) (BoundPageViewModel, error) {
	if c == nil || c.validator == nil || !canonicalDigestPattern.MatchString(themePackageDigest) {
		return BoundPageViewModel{}, ErrViewModelTheme
	}
	if model.pageID != c.descriptor.ViewModelID || model.schemaVersion != c.descriptor.SchemaVersion ||
		model.pluginSchemaDigest != c.descriptor.SchemaDigest || model.value == nil {
		return BoundPageViewModel{}, ErrViewModelSchema
	}
	return BoundPageViewModel{
		pageID: c.descriptor.ViewModelID, schemaVersion: c.descriptor.SchemaVersion, pluginSchemaDigest: c.descriptor.SchemaDigest,
		themePackageDigest: themePackageDigest, value: model.value, seo: cloneSEOView(model.seo),
	}, nil
}

func rejectExternalPluginSchemaReferences(value any, depth int) error {
	if depth > DefaultMaxCallDepth*2 {
		return fmt.Errorf("%w: schema nesting exceeds the Host limit", ErrPluginViewModelSchema)
	}
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			switch key {
			case "$dynamicRef", "$recursiveRef":
				return fmt.Errorf("%w: %s is not supported", ErrPluginViewModelSchema, key)
			case "$ref":
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#") {
					return fmt.Errorf("%w: external references are not allowed", ErrPluginViewModelSchema)
				}
			}
			if err := rejectExternalPluginSchemaReferences(child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range item {
			if err := rejectExternalPluginSchemaReferences(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
