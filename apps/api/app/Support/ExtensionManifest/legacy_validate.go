package extensionmanifest

import (
	"strings"

	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

func validateLegacyRuntimeDeclarations(manifest Manifest) error {
	if hasV3Declarations(manifest) {
		return ErrInvalidManifest
	}
	if manifest.Backend.Entry != "" {
		if _, ok := SafeArchivePath(manifest.Backend.Entry); !ok {
			return ErrInvalidManifest
		}
	}
	if manifest.Backend.RPC != "" && manifest.Backend.RPC != "hashicorp-go-plugin" {
		return ErrInvalidManifest
	}
	if manifest.Backend.ProtocolVersion < 0 || manifest.Backend.ProtocolVersion > 1 {
		return ErrInvalidManifest
	}
	for _, migration := range manifest.Migrations {
		if _, ok := SafeArchivePath(migration.Path); !ok || !strings.HasSuffix(migration.Path, ".sql") {
			return ErrInvalidManifest
		}
	}
	for _, route := range manifest.Routes {
		if route.Path == "" || !strings.HasPrefix(route.Path, "/") || strings.Contains(route.Path, "..") {
			return ErrInvalidManifest
		}
		access := route.Access
		if access == "" {
			access = RouteAccessLogin
		}
		if access != RouteAccessPublic && access != RouteAccessLogin && access != RouteAccessPermission {
			return ErrInvalidManifest
		}
		if len(route.Methods) == 0 {
			return ErrInvalidManifest
		}
		for _, method := range route.Methods {
			switch method {
			case "GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE":
			default:
				return ErrInvalidManifest
			}
			if access == RouteAccessPublic && method != "GET" && method != "HEAD" && method != "OPTIONS" {
				return ErrInvalidManifest
			}
		}
		if access == RouteAccessPermission && (route.Permission == "" || !manifestHasPermission(manifest, route.Permission)) {
			return ErrInvalidManifest
		}
		if route.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	for _, hook := range manifest.Hooks {
		if !appevents.Known(hook.Name) {
			return ErrInvalidManifest
		}
	}
	seenEvents := map[string]bool{}
	for _, event := range DeclaredEvents(manifest) {
		definition, ok := appevents.FindDefinition(event.Name)
		if !ok {
			return ErrInvalidManifest
		}
		kind := event.Kind
		if kind == "" {
			kind = definition.Kind
		}
		if kind != definition.Kind || event.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
		key := event.Name + ":" + kind
		if seenEvents[key] {
			return ErrInvalidManifest
		}
		seenEvents[key] = true
	}
	for _, provider := range manifest.Providers {
		if provider.Label == "" || !knownProviderSlot(provider.Slot) || provider.TimeoutMS < 0 {
			return ErrInvalidManifest
		}
	}
	return nil
}
