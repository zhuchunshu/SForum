package extensionmanifest

import (
	"encoding/base64"
	"encoding/hex"
	"path/filepath"
	"strings"

	componentcatalog "github.com/zhuchunshu/sforum/apps/api/app/Support/ComponentCatalog"
)

func (v *v3Validator) validateUIAndPackage() error {
	if len(v.manifest.Content) > ContentDeclarationsMaximum {
		return ErrInvalidManifest
	}
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

	// 模板按规范化 ID 唯一登记；后续组件 SSR 与 content renderer 都只解析同包声明。
	templates := map[string]ManifestTemplate{}
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
		templates[template.ID] = template
	}

	for _, asset := range v.manifest.Assets {
		if err := v.versionedID(asset.Handle, asset.ContractVersion, "asset"); err != nil {
			return err
		}
		if asset.Type != "script" && asset.Type != "style" || !validPackagePath(asset.Path) || !validDigest(asset.Digest) ||
			asset.Type == "style" && asset.Module || !validPrebuiltAssetPath(asset.Type, asset.Path) ||
			!validAssetIntegrity(asset.Digest, asset.Integrity) || !validAssetCSP(asset.CSP) {
			return ErrInvalidManifest
		}
		file, declared := packagePaths[asset.Path]
		if !declared || file.Kind != "asset" || file.Digest != asset.Digest {
			return ErrInvalidManifest
		}
		switch asset.Loading {
		case "", "blocking", "defer", "async", "preload", "lazy":
		default:
			return ErrInvalidManifest
		}
		seenDependencies := map[string]struct{}{}
		for _, dependency := range asset.Dependencies {
			if !manifestIDPattern.MatchString(dependency) || dependency == asset.Handle {
				return ErrInvalidManifest
			}
			if _, duplicate := seenDependencies[dependency]; duplicate {
				return ErrInvalidManifest
			}
			seenDependencies[dependency] = struct{}{}
		}
		seenScopes := map[string]struct{}{}
		for _, scope := range asset.Scope {
			if !manifestIDPattern.MatchString(scope) {
				return ErrInvalidManifest
			}
			if _, duplicate := seenScopes[scope]; duplicate {
				return ErrInvalidManifest
			}
			seenScopes[scope] = struct{}{}
		}
	}
	for _, asset := range v.manifest.Assets {
		for _, dependency := range asset.Dependencies {
			family, declared := v.ids[dependency]
			if declared && family != "asset" {
				return ErrInvalidManifest
			}
			if !declared && !strings.HasPrefix(dependency, "core.asset.") &&
				!requiredExtensionOwnsAsset(v.manifest.Dependencies, dependency) {
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
		// 非空 ssrTemplate 必须解析到同包唯一模板，并与 props/override 契约精确一致。
		if component.SSRTemplate != "" {
			template, declared := templates[component.SSRTemplate]
			if !declared || !componentSSRTemplateConsistent(component, template) {
				return ErrInvalidManifest
			}
		}
		if component.L2Component != "" {
			file, exists := packageFiles[component.L2Component]
			if !exists || file.Kind != "frontend" || !validPrebuiltAssetPath("script", file.Path) {
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
		if !validSchemaRef(content.Schema) ||
			(content.Handler != "" && !validHandler(content.Handler)) ||
			content.Handler == "" && content.Renderer == "" {
			return ErrInvalidManifest
		}
		if content.Renderer != "" {
			if _, declared := templates[content.Renderer]; !declared {
				return ErrInvalidManifest
			}
		}
		if content.Migration != "" && v.ids[content.Migration] != "migration" {
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

func requiredExtensionOwnsAsset(dependencies []ManifestDependency, handle string) bool {
	for _, dependency := range dependencies {
		if dependency.Kind == "required" && dependency.ID != "" &&
			strings.HasPrefix(handle, dependency.ID+".") {
			return true
		}
	}
	return false
}

func validPrebuiltAssetPath(assetType, value string) bool {
	extension := strings.ToLower(filepath.Ext(value))
	if assetType == "style" {
		return extension == ".css"
	}
	return extension == ".js" || extension == ".mjs"
}

func validAssetIntegrity(digest, integrity string) bool {
	if integrity == "" {
		return true
	}
	raw, err := hex.DecodeString(strings.TrimSpace(digest))
	if err != nil || len(raw) != 32 {
		return false
	}
	return integrity == "sha256-"+base64.StdEncoding.EncodeToString(raw)
}

func validAssetCSP(declarations []string) bool {
	for _, declaration := range declarations {
		if declaration == "" || len(declaration) > 512 || strings.ContainsAny(declaration, ";\r\n\x00") {
			return false
		}
		fields := strings.Fields(declaration)
		if len(fields) < 2 {
			return false
		}
		switch fields[0] {
		case "connect-src", "font-src", "img-src", "media-src", "script-src", "style-src", "worker-src":
		default:
			return false
		}
		for _, source := range fields[1:] {
			if len(source) > 240 || strings.ContainsAny(source, "\"<>\\") {
				return false
			}
		}
	}
	return true
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

// componentSSRTemplateConsistent 校验组件与其 SSR 模板的生产准入契约。
// 仅使用已冻结字段：不发明 fragment 输入/wrap/filter 语义，也不合成 theme override target version。
func componentSSRTemplateConsistent(component ManifestComponent, template ManifestTemplate) bool {
	// 组件 SSR 片段只绑定同包 action=add 且无 target 的模板；replace/target 留给主题覆盖声明。
	if template.Action != "add" || template.TargetID != "" {
		return false
	}
	// PropsSchema 与 ViewModelSchema 必须是规范化后的精确同一引用，禁止隐式拓宽或缺失。
	if component.PropsSchema != template.ViewModelSchema {
		return false
	}
	// 双方皆空：明确表示无 theme override 绑定；任一方非空时必须逐字一致。
	if component.ThemeOverrideKey != template.ThemeOverrideKey {
		return false
	}
	return true
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
