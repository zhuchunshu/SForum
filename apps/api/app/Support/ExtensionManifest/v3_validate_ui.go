package extensionmanifest

import (
	"strings"

	componentcatalog "github.com/zhuchunshu/sforum/apps/api/app/Support/ComponentCatalog"
)

func (v *v3Validator) validateUIAndPackage() error {
	packageFiles := map[string]ManifestPackageFile{}
	packagePaths := map[string]ManifestPackageFile{}
	targetedSchemas := map[string]string{}
	for _, provider := range v.manifest.Providers {
		if provider.TargetID == "" {
			continue
		}
		for _, reference := range []string{provider.RequestSchema, provider.ResponseSchema} {
			separator := strings.LastIndex(reference, "@")
			if separator > 0 && separator < len(reference)-1 {
				targetedSchemas[reference[:separator]] = reference[separator+1:]
			}
		}
	}
	for _, file := range v.manifest.PackageFiles {
		foreignTargetedSchema := file.Kind == "schema" && targetedSchemas[file.ID] == file.Version
		if !manifestIDPattern.MatchString(file.ID) || !strings.HasPrefix(file.ID, v.manifest.ID+".") && !foreignTargetedSchema {
			return ErrInvalidManifest
		}
		if _, duplicate := packageFiles[file.ID]; duplicate {
			return ErrInvalidManifest
		}
		if _, duplicate := packagePaths[file.Path]; duplicate {
			return ErrInvalidManifest
		}
		if file.Path == ManifestFileName || !validPackagePath(file.Path) || !validDigest(file.Digest) {
			return ErrInvalidManifest
		}
		switch file.Kind {
		case "executable", "frontend", "locale", "schema", "migration", "template", "asset", "openapi", "database_operation":
		default:
			return ErrInvalidManifest
		}
		if file.Kind == "locale" && file.Locale == "" {
			return ErrInvalidManifest
		}
		packageFiles[file.ID] = file
		packagePaths[file.Path] = file
	}
	if backend := v.manifest.Backend; backend.Entry != "" && !matchingPackageFile(packagePaths, backend.Entry, "executable", backend.Digest) {
		return ErrInvalidManifest
	}
	for _, guard := range v.manifest.Guards {
		if !matchingPackageFile(packagePaths, guard.Entry, "executable", guard.Digest) {
			return ErrInvalidManifest
		}
	}
	for _, migration := range v.manifest.Migrations {
		if !matchingPackageFile(packagePaths, migration.Path, "migration", migration.Digest) {
			return ErrInvalidManifest
		}
	}
	if database := v.manifest.Database; database != nil {
		for _, operation := range database.Operations {
			if !matchingPackageFile(packagePaths, operation.Path, "database_operation", operation.Digest) {
				return ErrInvalidManifest
			}
		}
	}
	for _, provider := range v.manifest.Providers {
		if provider.RequestSchema == "" && provider.ResponseSchema == "" {
			continue
		}
		if !matchingVersionedSchemaFile(packageFiles, provider.RequestSchema) ||
			!matchingVersionedSchemaFile(packageFiles, provider.ResponseSchema) {
			return ErrInvalidManifest
		}
	}

	templates := map[string]bool{}
	for _, template := range v.manifest.Templates {
		if err := v.versionedID(template.ID, template.ContractVersion, "template"); err != nil {
			return err
		}
		if template.Action != "add" && template.Action != "replace" || template.Action == "replace" && template.TargetID == "" {
			return ErrInvalidManifest
		}
		if !validPackagePath(template.Path) || !validDigest(template.Digest) || !validSchemaRef(template.ViewModelSchema) {
			return ErrInvalidManifest
		}
		file, declared := packagePaths[template.Path]
		if !declared || file.Kind != "template" || file.Digest != template.Digest {
			return ErrInvalidManifest
		}
		templates[template.ID] = true
	}

	for _, asset := range v.manifest.Assets {
		if err := v.versionedID(asset.Handle, asset.ContractVersion, "asset"); err != nil {
			return err
		}
		if asset.Type != "script" && asset.Type != "style" || !validPackagePath(asset.Path) || !validDigest(asset.Digest) {
			return ErrInvalidManifest
		}
		file, declared := packagePaths[asset.Path]
		if !declared || file.Kind != "asset" || file.Digest != asset.Digest {
			return ErrInvalidManifest
		}
		switch asset.Loading {
		case "", "blocking", "defer", "async", "preload":
		default:
			return ErrInvalidManifest
		}
		for _, dependency := range asset.Dependencies {
			if !manifestIDPattern.MatchString(dependency) {
				return ErrInvalidManifest
			}
		}
	}
	for _, asset := range v.manifest.Assets {
		for _, dependency := range asset.Dependencies {
			if family, declared := v.ids[dependency]; !declared && !strings.HasPrefix(dependency, "core.asset.") || declared && family != "asset" {
				return ErrInvalidManifest
			}
		}
	}

	for _, component := range v.manifest.Components {
		if err := v.versionedID(component.ID, component.ContractVersion, "component"); err != nil {
			return err
		}
		if !validComponentAction(component.Action) || component.Action != ComponentActionAdd && component.TargetID == "" {
			return ErrInvalidManifest
		}
		if !validComponentTarget(component.TargetID, component.TargetContractVersion, v.manifest.Type) {
			return ErrInvalidManifest
		}
		if component.Action != ComponentActionHide && !validSchemaRef(component.PropsSchema) {
			return ErrInvalidManifest
		}
		if (component.Action == ComponentActionFilterResult || component.Action == ComponentActionWrap || component.Action == ComponentActionReplace) && !validSchemaRef(component.ResultSchema) {
			return ErrInvalidManifest
		}
		if component.Action != ComponentActionHide && component.SSRTemplate == "" && component.L2Component == "" {
			return ErrInvalidManifest
		}
		if component.SSRTemplate != "" && !templates[component.SSRTemplate] {
			return ErrInvalidManifest
		}
		if component.L2Component != "" {
			file, exists := packageFiles[component.L2Component]
			if !exists || file.Kind != "frontend" {
				return ErrInvalidManifest
			}
		}
	}

	for _, content := range v.manifest.Content {
		if err := v.versionedID(content.ID, content.ContractVersion, "content"); err != nil {
			return err
		}
		switch content.Kind {
		case "block", "shortcode", "embed", "node", "mark", "render_filter", "sanitizer":
		default:
			return ErrInvalidManifest
		}
		if !validSchemaRef(content.Schema) || content.Handler == "" && content.Renderer == "" {
			return ErrInvalidManifest
		}
		if content.Renderer != "" && !templates[content.Renderer] || content.Migration != "" && v.ids[content.Migration] != "migration" {
			return ErrInvalidManifest
		}
	}

	for _, fragment := range v.manifest.OpenAPI {
		if err := v.versionedID(fragment.ID, fragment.ContractVersion, "openapi"); err != nil {
			return err
		}
		if !validPackagePath(fragment.Path) || !validDigest(fragment.Digest) || !manifestIDPattern.MatchString(fragment.Namespace) {
			return ErrInvalidManifest
		}
		file, declared := packagePaths[fragment.Path]
		if !declared || file.Kind != "openapi" || file.Digest != fragment.Digest {
			return ErrInvalidManifest
		}
	}

	if component := v.manifest.SettingsDocument.UI.Component; component != nil && component.Entry != "" {
		entry, declared := packagePaths[component.Entry]
		if !declared || entry.Kind != "frontend" {
			return ErrInvalidManifest
		}
		if component.CSS != "" {
			css, declared := packagePaths[component.CSS]
			if !declared || css.Kind != "asset" {
				return ErrInvalidManifest
			}
		}
	}
	if !v.allLocalSchemasDeclared(packagePaths) {
		return ErrInvalidManifest
	}
	return nil
}

func matchingVersionedSchemaFile(files map[string]ManifestPackageFile, reference string) bool {
	separator := strings.LastIndex(reference, "@")
	if separator <= 0 || separator == len(reference)-1 {
		return false
	}
	file, ok := files[reference[:separator]]
	return ok && file.Kind == "schema" && file.Version == reference[separator+1:]
}

func matchingPackageFile(files map[string]ManifestPackageFile, path string, kind string, digest string) bool {
	file, exists := files[path]
	return exists && file.Kind == kind && file.Digest == digest
}

func (v *v3Validator) allLocalSchemasDeclared(files map[string]ManifestPackageFile) bool {
	refs := make([]string, 0)
	for _, route := range v.manifest.Routes {
		refs = append(refs, route.RequestSchema, route.ResponseSchema)
	}
	for _, hook := range v.manifest.Hooks {
		refs = append(refs, hook.InputSchema, hook.ResultSchema)
	}
	for _, event := range v.manifest.Events {
		refs = append(refs, event.InputSchema, event.ResultSchema)
	}
	for _, job := range v.manifest.Jobs {
		refs = append(refs, job.PayloadSchema)
	}
	for _, provider := range v.manifest.Providers {
		refs = append(refs, provider.RequestSchema, provider.ResponseSchema)
	}
	for _, component := range v.manifest.Components {
		refs = append(refs, component.PropsSchema, component.ResultSchema)
	}
	for _, template := range v.manifest.Templates {
		refs = append(refs, template.ViewModelSchema)
	}
	for _, content := range v.manifest.Content {
		refs = append(refs, content.Schema)
	}
	for _, service := range v.manifest.Services {
		refs = append(refs, service.RequestSchema, service.ResponseSchema)
	}
	for _, command := range v.manifest.Commands {
		refs = append(refs, command.InputSchema, command.ResultSchema)
	}
	for _, surface := range v.manifest.AdminSurfaces {
		refs = append(refs, surface.Schema)
	}
	for _, query := range v.manifest.Queries {
		refs = append(refs, query.ResultSchema)
	}
	if v.manifest.Database != nil {
		for _, operation := range v.manifest.Database.Operations {
			refs = append(refs, operation.ResultSchema)
			for _, parameter := range operation.Parameters {
				refs = append(refs, parameter.Schema)
			}
		}
	}
	if v.manifest.Identity != nil {
		for _, field := range v.manifest.Identity.UserFields {
			refs = append(refs, field.Schema)
		}
	}
	if v.manifest.Lifecycle != nil {
		for _, operation := range []*ManifestLifecycleOperation{v.manifest.Lifecycle.Install, v.manifest.Lifecycle.Enable, v.manifest.Lifecycle.Disable, v.manifest.Lifecycle.Upgrade, v.manifest.Lifecycle.Rollback, v.manifest.Lifecycle.Uninstall} {
			if operation != nil {
				refs = append(refs, operation.ProgressSchema, operation.CheckpointSchema)
			}
		}
	}
	for _, ref := range refs {
		if ref == "" || contractVersionPattern.MatchString(ref) {
			continue
		}
		if file, exists := files[ref]; !exists || file.Kind != "schema" {
			return false
		}
	}
	return true
}

func validComponentAction(value string) bool {
	switch value {
	case ComponentActionAdd, ComponentActionBefore, ComponentActionAfter, ComponentActionWrap, ComponentActionReplace, ComponentActionHide, ComponentActionFilterProps, ComponentActionFilterResult:
		return true
	default:
		return false
	}
}

func validComponentTarget(targetID string, targetContractVersion string, manifestType string) bool {
	if targetID == "" {
		return targetContractVersion == ""
	}
	if !manifestIDPattern.MatchString(targetID) || !contractVersionPattern.MatchString(targetContractVersion) {
		return false
	}
	if !strings.HasPrefix(targetID, "core.") {
		return true
	}
	if !strings.HasPrefix(targetID, "core.component.") {
		return false
	}
	target, found := componentcatalog.FindCoreComponent(targetID)
	return found && target.ContractVersion == targetContractVersion && (manifestType != TypeTheme || target.OwnedBy(componentcatalog.OwnerPublic))
}

func validAdminSurfacePlacement(targetID string, targetContractVersion string) bool {
	if !manifestIDPattern.MatchString(targetID) || !contractVersionPattern.MatchString(targetContractVersion) {
		return false
	}
	target, found := componentcatalog.FindCoreComponent(targetID)
	return found && target.ContractVersion == targetContractVersion && target.Kind == componentcatalog.KindPage && target.OwnedBy(componentcatalog.OwnerAdmin)
}
