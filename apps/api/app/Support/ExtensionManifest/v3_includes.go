package extensionmanifest

import (
	"encoding/json"
	"fmt"
)

func applyV3Includes(manifest *Manifest, includes ManifestIncludes, pkg PackageFS) error {
	if err := applyListInclude(includes.Guards, pkg, "guards", &manifest.Guards); err != nil {
		return err
	}
	if err := applyListInclude(includes.Schedules, pkg, "schedules", &manifest.Schedules); err != nil {
		return err
	}
	if err := applyListInclude(includes.Components, pkg, "components", &manifest.Components); err != nil {
		return err
	}
	if err := applyListInclude(includes.Templates, pkg, "templates", &manifest.Templates); err != nil {
		return err
	}
	if err := applyListInclude(includes.Assets, pkg, "assets", &manifest.Assets); err != nil {
		return err
	}
	if err := applyListInclude(includes.Content, pkg, "content", &manifest.Content); err != nil {
		return err
	}
	if err := applyObjectInclude(includes.Database, pkg, "database", manifest.Database != nil, &manifest.Database); err != nil {
		return err
	}
	if err := applyListInclude(includes.Cache, pkg, "cache", &manifest.Cache); err != nil {
		return err
	}
	if err := applyListInclude(includes.Services, pkg, "services", &manifest.Services); err != nil {
		return err
	}
	if err := applyListInclude(includes.Commands, pkg, "commands", &manifest.Commands); err != nil {
		return err
	}
	if err := applyListInclude(includes.AdminSurfaces, pkg, "adminSurfaces", &manifest.AdminSurfaces); err != nil {
		return err
	}
	if err := applyListInclude(includes.Queries, pkg, "queries", &manifest.Queries); err != nil {
		return err
	}
	if err := applyObjectInclude(includes.Identity, pkg, "identity", manifest.Identity != nil, &manifest.Identity); err != nil {
		return err
	}
	if err := applyListInclude(includes.PermissionDefinitions, pkg, "permissionDefinitions", &manifest.PermissionDefinitions); err != nil {
		return err
	}
	if err := applyListInclude(includes.Media, pkg, "media", &manifest.Media); err != nil {
		return err
	}
	if err := applyListInclude(includes.Navigation, pkg, "navigation", &manifest.Navigation); err != nil {
		return err
	}
	if err := applyListInclude(includes.Regions, pkg, "regions", &manifest.Regions); err != nil {
		return err
	}
	if err := applyListInclude(includes.Dependencies, pkg, "dependencies", &manifest.Dependencies); err != nil {
		return err
	}
	if err := applyObjectInclude(includes.Lifecycle, pkg, "lifecycle", manifest.Lifecycle != nil, &manifest.Lifecycle); err != nil {
		return err
	}
	if err := applyListInclude(includes.OpenAPI, pkg, "openapi", &manifest.OpenAPI); err != nil {
		return err
	}
	return applyListInclude(includes.PackageFiles, pkg, "packageFiles", &manifest.PackageFiles)
}

func applyListInclude[T any](raw json.RawMessage, pkg PackageFS, label string, destination *[]T) error {
	if len(raw) == 0 || isJSONNull(raw) {
		return nil
	}
	if len(*destination) > 0 {
		return fmt.Errorf("%w: dual source for %s (root and includes)", ErrInvalidManifest, label)
	}
	if err := validateV3IncludeFiles(raw, pkg, label, false); err != nil {
		return err
	}
	items, err := loadJSONShardList[T](raw, pkg, label)
	if err != nil {
		return fmt.Errorf("%w: includes.%s: %v", ErrInvalidManifest, label, err)
	}
	*destination = items
	return nil
}

func applyObjectInclude[T any](raw json.RawMessage, pkg PackageFS, label string, filled bool, destination **T) error {
	if len(raw) == 0 || isJSONNull(raw) {
		return nil
	}
	if filled {
		return fmt.Errorf("%w: dual source for %s (root and includes)", ErrInvalidManifest, label)
	}
	if err := validateV3IncludeFiles(raw, pkg, label, true); err != nil {
		return err
	}
	var value T
	if err := decodeIncludeObject(raw, pkg, &value); err != nil {
		return fmt.Errorf("%w: includes.%s: %v", ErrInvalidManifest, label, err)
	}
	*destination = &value
	return nil
}

func validateV3IncludeFiles(raw json.RawMessage, pkg PackageFS, label string, object bool) error {
	paths, err := includePaths(raw, pkg, label)
	if err != nil {
		return fmt.Errorf("%w: includes.%s: %v", ErrInvalidManifest, label, err)
	}
	if object && len(paths) != 1 {
		return fmt.Errorf("%w: includes.%s must reference one object", ErrInvalidManifest, label)
	}
	for _, includePath := range paths {
		body, err := pkg.ReadFile(includePath)
		if err != nil {
			return fmt.Errorf("%w: read includes.%s %s: %v", ErrInvalidManifest, label, includePath, err)
		}
		if err := validateV3JSONSchemaFragment(body, label); err != nil {
			return fmt.Errorf("%w: includes.%s %s: %v", ErrInvalidManifest, label, includePath, err)
		}
	}
	return nil
}
