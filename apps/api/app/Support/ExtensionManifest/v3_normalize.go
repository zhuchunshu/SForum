package extensionmanifest

import "strings"

func normalizeV3Manifest(manifest *Manifest) {
	manifest.Backend.Digest = normalizeDigest(manifest.Backend.Digest)
	manifest.Backend.HostAPIVersion = strings.TrimSpace(manifest.Backend.HostAPIVersion)
	for index := range manifest.Migrations {
		item := &manifest.Migrations[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Path = strings.TrimSpace(item.Path)
		item.Digest = normalizeDigest(item.Digest)
		item.Transaction = strings.ToLower(strings.TrimSpace(item.Transaction))
	}
	for index := range manifest.Routes {
		item := &manifest.Routes[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		if item.Action == "" {
			item.Action = RouteActionAdd
		}
		item.TargetID = NormalizeID(item.TargetID)
		item.Guard = NormalizeID(item.Guard)
		if item.Guard == "" {
			if guard, ok := coreGuardForRouteAccess(item.Access); ok {
				item.Guard = guard
			} else if item.Access == "" && routeTargetsExisting(item.Action) {
				item.Guard = GuardCoreInherit
			}
		}
		item.Fallback = strings.ToLower(strings.TrimSpace(item.Fallback))
		item.Mode = strings.ToLower(strings.TrimSpace(item.Mode))
		if item.Mode == "" {
			item.Mode = RouteModeHTTP
		}
		if item.Fallback == "" {
			item.Fallback = "closed"
		}
		item.Destination = strings.TrimSpace(item.Destination)
		item.Handler = strings.TrimSpace(item.Handler)
		item.RequestSchema = strings.TrimSpace(item.RequestSchema)
		item.ResponseSchema = strings.TrimSpace(item.ResponseSchema)
	}
	for index := range manifest.Hooks {
		item := &manifest.Hooks[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Name = NormalizeID(item.Name)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.TargetID = NormalizeID(item.TargetID)
		item.Handler = strings.TrimSpace(item.Handler)
		item.InputSchema = strings.TrimSpace(item.InputSchema)
		item.ResultSchema = strings.TrimSpace(item.ResultSchema)
		item.Execution = strings.ToLower(strings.TrimSpace(item.Execution))
		if item.Execution == "" {
			if item.Kind == "observe" {
				item.Execution = "async"
			} else {
				item.Execution = "sync"
			}
		}
		item.FailurePolicy = strings.ToLower(strings.TrimSpace(item.FailurePolicy))
		if item.FailurePolicy == "" {
			if item.Execution == "async" {
				item.FailurePolicy = "fail_open"
			} else {
				item.FailurePolicy = "fail_closed"
			}
		}
		if item.TimeoutMS == 0 {
			if item.Execution == "async" {
				item.TimeoutMS = 5000
			} else {
				item.TimeoutMS = 2000
			}
		}
		for fieldIndex := range item.MutableFields {
			item.MutableFields[fieldIndex] = strings.TrimSpace(item.MutableFields[fieldIndex])
		}
	}
	for index := range manifest.Jobs {
		item := &manifest.Jobs[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Name = NormalizeID(item.Name)
		item.Handler = strings.TrimSpace(item.Handler)
		item.PayloadSchema = strings.TrimSpace(item.PayloadSchema)
		item.RetryPolicy = strings.ToLower(strings.TrimSpace(item.RetryPolicy))
		if item.ConcurrencyLimit == 0 {
			item.ConcurrencyLimit = PluginJobDefaultConcurrencyLimit
		}
		if item.MaxAttempts == 0 {
			switch item.RetryPolicy {
			case "none":
				item.MaxAttempts = 1
			case "bounded":
				item.MaxAttempts = PluginJobDefaultBoundedAttempts
			case "exponential":
				item.MaxAttempts = PluginJobDefaultExponentialAttempts
			}
		}
		if item.RetryPolicy == "bounded" && item.RetryDelaySeconds == 0 {
			item.RetryDelaySeconds = PluginJobDefaultRetryDelaySeconds
		}
	}
	for index := range manifest.Providers {
		item := &manifest.Providers[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Slot = NormalizeID(item.Slot)
		item.TargetID = NormalizeID(item.TargetID)
		item.Handler = strings.TrimSpace(item.Handler)
		item.RequestSchema = strings.TrimSpace(item.RequestSchema)
		item.ResponseSchema = strings.TrimSpace(item.ResponseSchema)
		item.Fallback = strings.ToLower(strings.TrimSpace(item.Fallback))
		if item.RequestSchema != "" || item.ResponseSchema != "" || item.TargetID != "" {
			if item.Fallback == "" {
				item.Fallback = "next"
			}
			if item.TimeoutMS == 0 {
				item.TimeoutMS = ProviderSlotMaximumTimeoutMS
			}
		}
	}

	for index := range manifest.Guards {
		item := &manifest.Guards[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.Entry = strings.TrimSpace(item.Entry)
		item.Digest = normalizeDigest(item.Digest)
	}
	for index := range manifest.Schedules {
		item := &manifest.Schedules[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.JobID = NormalizeID(item.JobID)
		item.Cron = strings.TrimSpace(item.Cron)
		item.Timezone = strings.TrimSpace(item.Timezone)
	}
	for index := range manifest.Components {
		item := &manifest.Components[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.TargetContractVersion = strings.TrimSpace(item.TargetContractVersion)
		item.SSRTemplate = NormalizeID(item.SSRTemplate)
		item.L2Component = NormalizeID(item.L2Component)
		item.PropsSchema = strings.TrimSpace(item.PropsSchema)
		item.ResultSchema = strings.TrimSpace(item.ResultSchema)
		item.ThemeOverrideKey = NormalizeID(item.ThemeOverrideKey)
	}
	for index := range manifest.Templates {
		item := &manifest.Templates[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Action = strings.ToLower(strings.TrimSpace(item.Action))
		item.TargetID = NormalizeID(item.TargetID)
		item.Path = strings.TrimSpace(item.Path)
		item.Digest = normalizeDigest(item.Digest)
		item.ViewModelSchema = strings.TrimSpace(item.ViewModelSchema)
		item.ThemeOverrideKey = NormalizeID(item.ThemeOverrideKey)
	}
	for index := range manifest.Assets {
		item := &manifest.Assets[index]
		item.Handle = NormalizeID(item.Handle)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Type = strings.ToLower(strings.TrimSpace(item.Type))
		item.Path = strings.TrimSpace(item.Path)
		item.Digest = normalizeDigest(item.Digest)
		item.Loading = strings.ToLower(strings.TrimSpace(item.Loading))
		item.Integrity = strings.TrimSpace(item.Integrity)
		for dependencyIndex := range item.Dependencies {
			item.Dependencies[dependencyIndex] = NormalizeID(item.Dependencies[dependencyIndex])
		}
		for scopeIndex := range item.Scope {
			item.Scope[scopeIndex] = NormalizeID(item.Scope[scopeIndex])
		}
		for cspIndex := range item.CSP {
			item.CSP[cspIndex] = strings.Join(strings.Fields(item.CSP[cspIndex]), " ")
		}
	}
	for index := range manifest.Content {
		item := &manifest.Content[index]
		item.ID = NormalizeID(item.ID)
		item.ContractVersion = strings.TrimSpace(item.ContractVersion)
		item.Kind = strings.ToLower(strings.TrimSpace(item.Kind))
		item.Handler = strings.TrimSpace(item.Handler)
		item.Schema = strings.TrimSpace(item.Schema)
		item.Renderer = NormalizeID(item.Renderer)
		item.Migration = NormalizeID(item.Migration)
	}
	normalizeV3Platform(manifest)
}

func coreGuardForRouteAccess(access string) (string, bool) {
	switch access {
	case RouteAccessPublic:
		return GuardCorePublic, true
	case RouteAccessLogin:
		return GuardCoreLogin, true
	case RouteAccessPermission:
		return GuardCorePermission, true
	default:
		return "", false
	}
}

func normalizeDigest(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.TrimSpace(strings.TrimPrefix(value, "sha256:"))
}
