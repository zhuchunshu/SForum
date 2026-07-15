package extensionmanifest

import "strings"

func normalizeV3Platform(manifest *Manifest) {
	if manifest.Database != nil {
		manifest.Database.ContractVersion = strings.TrimSpace(manifest.Database.ContractVersion)
		normalizeDatabaseGrants(manifest.Database)
		manifest.Database.Schema = NormalizeID(manifest.Database.Schema)
		manifest.Database.Role = NormalizeID(manifest.Database.Role)
		manifest.Database.CoreCompatibility = strings.TrimSpace(manifest.Database.CoreCompatibility)
		manifest.Database.Backup.Strategy = strings.ToLower(strings.TrimSpace(manifest.Database.Backup.Strategy))
		manifest.Database.Retention.OnDisable = strings.ToLower(strings.TrimSpace(manifest.Database.Retention.OnDisable))
		manifest.Database.Retention.OnUninstall = strings.ToLower(strings.TrimSpace(manifest.Database.Retention.OnUninstall))
		for index := range manifest.Database.Operations {
			operation := &manifest.Database.Operations[index]
			operation.ID = NormalizeID(operation.ID)
			operation.StatementVersion = strings.TrimSpace(operation.StatementVersion)
			operation.Kind = strings.ToLower(strings.TrimSpace(operation.Kind))
			operation.Path = strings.TrimSpace(operation.Path)
			operation.Digest = normalizeDigest(operation.Digest)
			operation.ResultSchema = strings.TrimSpace(operation.ResultSchema)
			for parameterIndex := range operation.Parameters {
				parameter := &operation.Parameters[parameterIndex]
				parameter.Schema = strings.TrimSpace(parameter.Schema)
				parameter.Field = NormalizeID(parameter.Field)
				parameter.Kind = strings.ToLower(strings.TrimSpace(parameter.Kind))
			}
			for columnIndex := range operation.Columns {
				operation.Columns[columnIndex].Name = NormalizeID(operation.Columns[columnIndex].Name)
			}
		}
	}
	for index := range manifest.Cache {
		item := &manifest.Cache[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Namespace = NormalizeID(item.Namespace)
		item.Policy = strings.ToLower(strings.TrimSpace(item.Policy))
		item.Provider = NormalizeID(item.Provider)
	}
	for index := range manifest.Services {
		item := &manifest.Services[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.Handler = strings.TrimSpace(item.Handler)
		item.RequestSchema = strings.TrimSpace(item.RequestSchema)
		item.ResponseSchema = strings.TrimSpace(item.ResponseSchema)
	}
	for index := range manifest.Commands {
		item := &manifest.Commands[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Handler = strings.TrimSpace(item.Handler)
		item.Permission = NormalizeID(item.Permission)
		item.InputSchema = strings.TrimSpace(item.InputSchema)
		item.ResultSchema = strings.TrimSpace(item.ResultSchema)
		item.Description = strings.TrimSpace(item.Description)
		if item.TimeoutMS == 0 {
			item.TimeoutMS = PluginCommandMaximumTimeoutMS
		}
	}
	for index := range manifest.AdminSurfaces {
		item := &manifest.AdminSurfaces[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.PlacementID = NormalizeID(item.PlacementID)
		item.PlacementContractVersion = strings.TrimSpace(item.PlacementContractVersion)
		item.Label = strings.TrimSpace(item.Label)
		item.Handler = strings.TrimSpace(item.Handler)
		item.Schema = strings.TrimSpace(item.Schema)
		item.PropsSchema = strings.TrimSpace(item.PropsSchema)
		item.ResultSchema = strings.TrimSpace(item.ResultSchema)
		if item.Schema != "" {
			if item.PropsSchema == "" {
				item.PropsSchema = item.Schema
			}
			if item.ResultSchema == "" {
				item.ResultSchema = item.Schema
			}
		}
		item.Operation = strings.ToLower(strings.TrimSpace(item.Operation))
		if item.Operation == "" {
			item.Operation = defaultAdminSurfaceOperation(item.Kind)
		}
		item.Permission = NormalizeID(item.Permission)
	}
	for index := range manifest.Queries {
		item := &manifest.Queries[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Entity = NormalizeID(item.Entity)
		item.PlanVersion = strings.TrimSpace(item.PlanVersion)
		item.Pagination = strings.ToLower(strings.TrimSpace(item.Pagination))
		item.ResultSchema = strings.TrimSpace(item.ResultSchema)
		item.PermissionPolicy = NormalizeID(item.PermissionPolicy)
	}
	if manifest.Identity != nil {
		manifest.Identity.ContractVersion = strings.TrimSpace(manifest.Identity.ContractVersion)
		manifest.Identity.SessionPolicy = NormalizeID(manifest.Identity.SessionPolicy)
		for index := range manifest.Identity.UserFields {
			item := &manifest.Identity.UserFields[index]
			item.ID = NormalizeID(item.ID)
			item.ContractVersion = strings.TrimSpace(item.ContractVersion)
			item.Type = strings.ToLower(strings.TrimSpace(item.Type))
			item.Schema = strings.TrimSpace(item.Schema)
			item.ReadPermission = NormalizeID(item.ReadPermission)
			item.WritePermission = NormalizeID(item.WritePermission)
		}
		for index := range manifest.Identity.Providers {
			item := &manifest.Identity.Providers[index]
			item.ID = NormalizeID(item.ID)
			item.ContractVersion = strings.TrimSpace(item.ContractVersion)
			item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
			item.Handler = strings.TrimSpace(item.Handler)
		}
	}
	for index := range manifest.PermissionDefinitions {
		item := &manifest.PermissionDefinitions[index]
		item.Key = NormalizeID(item.Key)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Label = strings.TrimSpace(item.Label)
		item.Description = strings.TrimSpace(item.Description)
		item.AssignmentPolicy = strings.ToLower(strings.TrimSpace(item.AssignmentPolicy))
	}
	for index := range manifest.Media {
		item := &manifest.Media[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.Handler = strings.TrimSpace(item.Handler)
		item.Permission = NormalizeID(item.Permission)
	}
	for index := range manifest.Navigation {
		item := &manifest.Navigation[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.Label = strings.TrimSpace(item.Label)
		item.Href = strings.TrimSpace(item.Href)
		item.Permission = NormalizeID(item.Permission)
	}
	for index := range manifest.Regions {
		item := &manifest.Regions[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.Label = strings.TrimSpace(item.Label)
	}
	for index := range manifest.Dependencies {
		item := &manifest.Dependencies[index]
		item.ID = NormalizeID(item.ID)
		item.Capability = NormalizeID(item.Capability)
		item.Version = strings.TrimSpace(item.Version)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
	}
	if manifest.Lifecycle != nil {
		manifest.Lifecycle.ContractVersion = strings.TrimSpace(manifest.Lifecycle.ContractVersion)
		normalizeLifecycleOperation(manifest.Lifecycle.Install)
		normalizeLifecycleOperation(manifest.Lifecycle.Enable)
		normalizeLifecycleOperation(manifest.Lifecycle.Disable)
		normalizeLifecycleOperation(manifest.Lifecycle.Upgrade)
		normalizeLifecycleOperation(manifest.Lifecycle.Rollback)
		normalizeLifecycleOperation(manifest.Lifecycle.Uninstall)
	}
	for index := range manifest.OpenAPI {
		item := &manifest.OpenAPI[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Path = strings.TrimSpace(item.Path)
		item.Digest = normalizeDigest(item.Digest)
		item.Namespace = NormalizeID(item.Namespace)
	}
	for index := range manifest.PackageFiles {
		item := &manifest.PackageFiles[index]
		item.ID = NormalizeID(item.ID)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.Path = strings.TrimSpace(item.Path)
		item.Digest = normalizeDigest(item.Digest)
		item.Locale = normalizeLocaleKey(item.Locale)
		item.Version = strings.TrimSpace(item.Version)
	}
}

func defaultAdminSurfaceOperation(kind string) string {
	switch kind {
	case "row_action", "bulk_action", "form", "importer":
		return AdminSurfaceOperationCommand
	default:
		return AdminSurfaceOperationQuery
	}
}

func normalizeLifecycleOperation(operation *ManifestLifecycleOperation) {
	if operation == nil {
		return
	}
	operation.Plan = strings.TrimSpace(operation.Plan)
	operation.Execute = strings.TrimSpace(operation.Execute)
	operation.ProgressSchema = strings.TrimSpace(operation.ProgressSchema)
	operation.CheckpointSchema = strings.TrimSpace(operation.CheckpointSchema)
}
