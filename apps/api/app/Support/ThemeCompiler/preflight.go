package themecompiler

import (
	"fmt"
	htmltemplate "html/template"
	"io/fs"
)

// PreflightFS parses every installable L1 page against the shared
// layouts/partials before an inert theme package is persisted. It deliberately
// has no runtime bindings: activation still builds the exact immutable snapshot.
func (c *Compiler) PreflightFS(source fs.FS) error {
	if c == nil || source == nil {
		return fmt.Errorf("%w: compiler and source filesystem are required", ErrInvalidInput)
	}
	sources, err := loadSources(source, c.limits)
	if err != nil {
		return err
	}

	funcs := restrictedFuncMap(Bindings{})
	shared := htmltemplate.New(internalRootTemplate).Option("missingkey=error").Funcs(funcs)
	sharedNames := map[string]string{}
	for _, item := range sources {
		if item.kind == KindPage {
			continue
		}
		if err := mergeSharedSource(shared, sharedNames, item, funcs); err != nil {
			return err
		}
	}

	for _, item := range sources {
		if item.kind != KindPage {
			continue
		}
		if err := inspectStaticHTML(item.name, item.body); err != nil {
			return err
		}
		compiled, err := shared.Clone()
		if err != nil {
			return fmt.Errorf("%w: clone shared templates: %v", ErrInvalidTemplate, err)
		}
		compiled, err = compiled.New(item.name).Parse(string(item.body))
		if err != nil {
			return classifyParseError(item.name, err)
		}
		if err := validateTemplateSet(compiled, item.name, c.limits.MaxCallDepth); err != nil {
			return err
		}
		if err := validateContextEscaping(compiled, item.name); err != nil {
			return err
		}
	}
	return nil
}
