package extensionmanifest

import "strings"

const (
	manifestDatabaseMaximumParameters    = 32
	manifestDatabaseMaximumColumns       = 64
	manifestDatabaseMaximumParameterSize = 64 << 10
	manifestDatabaseMaximumRows          = 1000
	manifestDatabaseMaximumAffectedRows  = 10_000
	manifestDatabaseMaximumTimeoutMS     = 5000
)

func (v *v3Validator) validatePlatform() error {
	if err := v.validateDatabaseAndCache(); err != nil {
		return err
	}
	if err := v.validateServicesCommandsAdminAndQueries(); err != nil {
		return err
	}
	if err := v.validateIdentityAndPermissions(); err != nil {
		return err
	}
	if err := v.validateMediaNavigationAndRegions(); err != nil {
		return err
	}
	if err := v.validateDependenciesAndLifecycle(); err != nil {
		return err
	}
	return nil
}

func (v *v3Validator) validateDatabaseAndCache() error {
	if database := v.manifest.Database; database != nil {
		if !contractVersionPattern.MatchString(database.ContractVersion) {
			return ErrInvalidManifest
		}
		grants := DatabaseGrants(database)
		if database.Authority != "" || len(grants) == 0 || len(grants) != len(database.Grants) {
			return ErrInvalidManifest
		}
		seenGrants := make(map[string]struct{}, len(grants))
		for _, grant := range grants {
			switch grant {
			case DatabaseGrantOwnSchema, DatabaseGrantCoreViews, DatabaseGrantHostCommands, DatabaseGrantRawCore, DatabaseGrantKernel:
			default:
				return ErrInvalidManifest
			}
			if _, duplicate := seenGrants[grant]; duplicate {
				return ErrInvalidManifest
			}
			seenGrants[grant] = struct{}{}
		}
		hasOwnSchema := HasDatabaseGrant(database, DatabaseGrantOwnSchema)
		if hasOwnSchema {
			// Host owns physical names. Logical schema/role remain required for
			// the historical own_schema-only contract and optional for cumulative
			// legacy tiers, but they must always be supplied as a valid pair.
			if (database.Schema == "") != (database.Role == "") ||
				(database.Schema != "" && (!databaseNamePattern.MatchString(database.Schema) || !databaseNamePattern.MatchString(database.Role))) ||
				len(grants) == 1 && database.Schema == "" {
				return ErrInvalidManifest
			}
		} else if database.Schema != "" || database.Role != "" {
			return ErrInvalidManifest
		}
		if (HasDatabaseGrant(database, DatabaseGrantRawCore) || HasDatabaseGrant(database, DatabaseGrantKernel)) && !validSemverConstraint(database.CoreCompatibility) {
			return ErrInvalidManifest
		}
		if database.Backup.Required && database.Backup.Strategy == "" || database.Retention.Days < 0 {
			return ErrInvalidManifest
		}
		if database.Retention.OnDisable != "retain" {
			return ErrInvalidManifest
		}
		switch database.Retention.OnUninstall {
		case "retain", "delete", "export":
		default:
			return ErrInvalidManifest
		}
		if (len(database.Operations) > 0 || len(v.manifest.Migrations) > 0) && !hasOwnSchema {
			return ErrInvalidManifest
		}
		for _, operation := range database.Operations {
			if err := v.validateDatabaseOperation(operation); err != nil {
				return err
			}
		}
	}
	for _, cache := range v.manifest.Cache {
		if err := v.versionedID(cache.ID, cache.ContractVersion, "cache"); err != nil {
			return err
		}
		if !strings.HasPrefix(cache.Namespace, v.manifest.ID+".") {
			return ErrInvalidManifest
		}
		switch cache.Policy {
		case "private", "actor", "permission", "public":
		default:
			return ErrInvalidManifest
		}
		for _, tag := range append(append([]string{}, cache.Tags...), cache.Invalidators...) {
			if !manifestIDPattern.MatchString(tag) {
				return ErrInvalidManifest
			}
		}
	}
	return nil
}

func (v *v3Validator) validateDatabaseOperation(operation ManifestDatabaseOperation) error {
	if !manifestIDPattern.MatchString(operation.ID) || !strings.HasPrefix(operation.ID, v.manifest.ID+".") ||
		!positiveIntegerPattern.MatchString(operation.StatementVersion) ||
		!validPackagePath(operation.Path) || !validDigest(operation.Digest) ||
		operation.Parameters == nil || operation.Columns == nil ||
		len(operation.Parameters) > manifestDatabaseMaximumParameters ||
		len(operation.Columns) > manifestDatabaseMaximumColumns ||
		operation.TimeoutMS < 0 || operation.TimeoutMS > manifestDatabaseMaximumTimeoutMS {
		return ErrInvalidManifest
	}
	if _, duplicate := v.ids[operation.ID]; duplicate {
		return ErrInvalidManifest
	}
	v.ids[operation.ID] = "database_operation"

	for _, parameter := range operation.Parameters {
		if !validSchemaRef(parameter.Schema) || !databaseOperationNamePattern.MatchString(parameter.Field) ||
			parameter.MaxBytes < 0 || parameter.MaxBytes > manifestDatabaseMaximumParameterSize {
			return ErrInvalidManifest
		}
		switch parameter.Kind {
		case "string", "int64", "number", "bool":
		default:
			return ErrInvalidManifest
		}
	}
	seenColumns := make(map[string]struct{}, len(operation.Columns))
	for _, column := range operation.Columns {
		if !databaseOperationNamePattern.MatchString(column.Name) {
			return ErrInvalidManifest
		}
		if _, duplicate := seenColumns[column.Name]; duplicate {
			return ErrInvalidManifest
		}
		seenColumns[column.Name] = struct{}{}
	}

	switch operation.Kind {
	case "query":
		if !validSchemaRef(operation.ResultSchema) || len(operation.Columns) == 0 ||
			operation.MaxRows <= 0 || operation.MaxRows > manifestDatabaseMaximumRows ||
			operation.MaxAffectedRows != 0 {
			return ErrInvalidManifest
		}
	case "execute":
		if operation.MaxRows != 0 || operation.MaxAffectedRows == 0 || operation.MaxAffectedRows > manifestDatabaseMaximumAffectedRows {
			return ErrInvalidManifest
		}
		if len(operation.Columns) == 0 && operation.ResultSchema != "" || len(operation.Columns) > 0 && !validSchemaRef(operation.ResultSchema) {
			return ErrInvalidManifest
		}
	default:
		return ErrInvalidManifest
	}
	return nil
}

func (v *v3Validator) validateServicesCommandsAdminAndQueries() error {
	for _, service := range v.manifest.Services {
		if err := v.versionedID(service.ID, service.ContractVersion, "service"); err != nil {
			return err
		}
		if !validComposableAction(service.Action) || service.Action != "add" && service.TargetID == "" || !validHandler(service.Handler) || !validSchemaRef(service.RequestSchema) || !validSchemaRef(service.ResponseSchema) {
			return ErrInvalidManifest
		}
	}
	for _, command := range v.manifest.Commands {
		if err := v.versionedID(command.ID, command.ContractVersion, "command"); err != nil {
			return err
		}
		if !validHandler(command.Handler) || !validSchemaRef(command.InputSchema) || !validSchemaRef(command.ResultSchema) {
			return ErrInvalidManifest
		}
		if command.Permission != "" && !manifestHasPermission(v.manifest, command.Permission) {
			return ErrInvalidManifest
		}
	}
	for _, surface := range v.manifest.AdminSurfaces {
		if err := v.versionedID(surface.ID, surface.ContractVersion, "admin_surface"); err != nil {
			return err
		}
		if !validAdminSurfaceKind(surface.Kind) || !validSurfaceAction(surface.Action) || surface.Action != "add" && surface.TargetID == "" || surface.Label == "" {
			return ErrInvalidManifest
		}
		if surface.Handler == "" && surface.Schema == "" {
			return ErrInvalidManifest
		}
		if surface.Schema != "" && !validSchemaRef(surface.Schema) || surface.Handler != "" && !validHandler(surface.Handler) {
			return ErrInvalidManifest
		}
		if surface.Permission != "" && !manifestHasPermission(v.manifest, surface.Permission) {
			return ErrInvalidManifest
		}
	}
	for _, query := range v.manifest.Queries {
		if err := v.versionedID(query.ID, query.ContractVersion, "query"); err != nil {
			return err
		}
		if query.Entity == "" || !contractVersionPattern.MatchString(query.PlanVersion) || len(query.Fields) == 0 || !validSchemaRef(query.ResultSchema) || query.PermissionPolicy == "" {
			return ErrInvalidManifest
		}
		switch query.Pagination {
		case "none", "offset", "cursor":
		default:
			return ErrInvalidManifest
		}
		if query.PermissionPolicy != "public" && query.PermissionPolicy != "login" && !manifestHasPermission(v.manifest, query.PermissionPolicy) {
			return ErrInvalidManifest
		}
	}
	return nil
}

func (v *v3Validator) validateIdentityAndPermissions() error {
	for _, permission := range v.manifest.PermissionDefinitions {
		if err := v.versionedID(permission.Key, permission.ContractVersion, "permission"); err != nil {
			return err
		}
		if permission.Label == "" || permission.Description == "" || permission.AssignmentPolicy != "host" {
			return ErrInvalidManifest
		}
		seenRoles := map[string]bool{}
		for _, role := range permission.RecommendedRoles {
			role = NormalizeID(role)
			if role == "" || seenRoles[role] {
				return ErrInvalidManifest
			}
			seenRoles[role] = true
		}
	}
	identity := v.manifest.Identity
	if identity == nil {
		return nil
	}
	if !contractVersionPattern.MatchString(identity.ContractVersion) {
		return ErrInvalidManifest
	}
	for _, field := range identity.UserFields {
		if err := v.versionedID(field.ID, field.ContractVersion, "identity_field"); err != nil {
			return err
		}
		if !validSchemaRef(field.Schema) {
			return ErrInvalidManifest
		}
		switch field.Type {
		case "string", "number", "boolean", "object", "array":
		default:
			return ErrInvalidManifest
		}
		for _, permission := range []string{field.ReadPermission, field.WritePermission} {
			if permission != "" && !manifestHasPermission(v.manifest, permission) {
				return ErrInvalidManifest
			}
		}
	}
	for _, provider := range identity.Providers {
		if err := v.versionedID(provider.ID, provider.ContractVersion, "identity_provider"); err != nil {
			return err
		}
		if !validHandler(provider.Handler) {
			return ErrInvalidManifest
		}
		switch provider.Kind {
		case "auth", "profile", "recovery", "session", "risk":
		default:
			return ErrInvalidManifest
		}
	}
	return nil
}

func (v *v3Validator) validateMediaNavigationAndRegions() error {
	for _, media := range v.manifest.Media {
		if err := v.versionedID(media.ID, media.ContractVersion, "media"); err != nil {
			return err
		}
		if !validComposableAction(media.Action) || media.Action != "add" && media.TargetID == "" || len(media.MIMEs) == 0 || !validHandler(media.Handler) {
			return ErrInvalidManifest
		}
		if media.Permission != "" && !manifestHasPermission(v.manifest, media.Permission) {
			return ErrInvalidManifest
		}
		seenTransforms := map[string]bool{}
		for _, transform := range media.Transforms {
			if transform.ID == "" || transform.Variant == "" || transform.Width < 0 || transform.Height < 0 || seenTransforms[transform.ID] {
				return ErrInvalidManifest
			}
			seenTransforms[transform.ID] = true
		}
	}
	for _, navigation := range v.manifest.Navigation {
		if err := v.versionedID(navigation.ID, navigation.ContractVersion, "navigation"); err != nil {
			return err
		}
		if !validSurfaceAction(navigation.Action) || navigation.Action != "add" && navigation.TargetID == "" || navigation.Label == "" || navigation.Order < 0 {
			return ErrInvalidManifest
		}
		switch navigation.Kind {
		case "menu", "item", "breadcrumb", "header", "footer", "sidebar":
		default:
			return ErrInvalidManifest
		}
		if navigation.Href != "" && !safeHostLinkPath(navigation.Href) {
			return ErrInvalidManifest
		}
		if navigation.Permission != "" && !manifestHasPermission(v.manifest, navigation.Permission) {
			return ErrInvalidManifest
		}
	}
	for _, region := range v.manifest.Regions {
		if err := v.versionedID(region.ID, region.ContractVersion, "region"); err != nil {
			return err
		}
		if !validSurfaceAction(region.Action) || region.Action != "add" && region.TargetID == "" || region.Label == "" {
			return ErrInvalidManifest
		}
		switch region.Kind {
		case "menu", "widget", "header", "footer", "sidebar", "content":
		default:
			return ErrInvalidManifest
		}
	}
	return nil
}

func (v *v3Validator) validateDependenciesAndLifecycle() error {
	seen := map[string]bool{}
	for _, dependency := range v.manifest.Dependencies {
		switch dependency.Kind {
		case "required", "optional", "conflict":
			if !validSemverConstraint(dependency.Version) || (dependency.ID == "") == (dependency.Capability == "") || dependency.ID == v.manifest.ID {
				return ErrInvalidManifest
			}
		case "provides":
			if !validSemverVersion(dependency.Version) || dependency.ID != "" || !manifestIDPattern.MatchString(dependency.Capability) {
				return ErrInvalidManifest
			}
		default:
			return ErrInvalidManifest
		}
		key := dependency.Kind + "\x00" + dependency.ID + "\x00" + dependency.Capability
		if seen[key] {
			return ErrInvalidManifest
		}
		seen[key] = true
	}
	lifecycle := v.manifest.Lifecycle
	if lifecycle == nil {
		return nil
	}
	if v.manifest.Backend.Entry == "" || !contractVersionPattern.MatchString(lifecycle.ContractVersion) {
		return ErrInvalidManifest
	}
	for _, operation := range []*ManifestLifecycleOperation{lifecycle.Install, lifecycle.Enable, lifecycle.Disable, lifecycle.Upgrade, lifecycle.Rollback, lifecycle.Uninstall} {
		if operation == nil {
			continue
		}
		if !validHandler(operation.Plan) || !validHandler(operation.Execute) || !validSchemaRef(operation.ProgressSchema) || !validSchemaRef(operation.CheckpointSchema) {
			return ErrInvalidManifest
		}
	}
	return nil
}

func validComposableAction(value string) bool {
	return value == "add" || value == "before" || value == "after" || value == "wrap" || value == "replace"
}

func validSurfaceAction(value string) bool {
	return validComposableAction(value) || value == "hide" || value == "filter"
}

func validAdminSurfaceKind(value string) bool {
	switch value {
	case "navigation", "dashboard", "list_column", "list_filter", "row_action", "bulk_action", "form", "notice", "editor_panel", "detail_region", "importer", "exporter":
		return true
	default:
		return false
	}
}
