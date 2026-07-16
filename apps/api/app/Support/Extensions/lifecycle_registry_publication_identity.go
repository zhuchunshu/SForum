package extensionsruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

func (b *PostgresLifecycleBoundaryRegistries) validateIdentityTransition(
	source, target *lifecycleRegistryMaterial,
) error {
	if !lifecycleMaterialsPublishIdentity(source, target) {
		return nil
	}
	if b == nil || b.identity == nil || b.identityStore == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	extensionID := lifecycleComponentExtensionID(source, target)
	snapshot := b.identity.Snapshot()
	for _, desired := range []*lifecycleRegistryMaterial{source, target} {
		var publication *identityregistry.Publication
		if desired != nil {
			publication = desired.identityPublication
		}
		graph, err := lifecycleIdentityGraph(
			snapshot, extensionID, publication,
			lifecycleIdentityPublication(source), lifecycleIdentityPublication(target),
		)
		if err != nil {
			return err
		}
		probe := identityregistry.New()
		if _, err := probe.ReplaceAllIfRevision(0, graph, snapshot.Tombstones, snapshot.SafeMode); err != nil {
			return wrapLifecycleIdentityError("validate identity registry publication", err)
		}
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) reconcileIdentity(
	ctx context.Context,
	request LifecycleBoundaryRequest,
	source, target, desired *lifecycleRegistryMaterial,
) error {
	if !lifecycleMaterialsPublishIdentity(source, target) {
		return nil
	}
	if b == nil || b.identity == nil || b.identityStore == nil || ctx == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	desiredPublication := lifecycleIdentityPublication(desired)
	durable, err := b.identityStore.Reconcile(ctx, identityregistry.ReconcilePublicationInput{
		ExtensionID:   request.TargetExtension.ID,
		AllowedSource: lifecycleIdentityArtifact(source),
		AllowedTarget: lifecycleIdentityArtifact(target),
		Desired:       desiredPublication, ActorUserID: request.ActorUserID, AuditEventID: request.AuditEventID,
	})
	if err != nil {
		return wrapLifecycleIdentityError("reconcile durable identity registry", err)
	}
	if desiredPublication != nil {
		if err := identityregistry.ValidateDurablePublication(durable, *desiredPublication); err != nil {
			return wrapLifecycleIdentityError("validate reconciled identity publication", err)
		}
	} else if err := identityregistry.ValidateDurableRetirement(
		durable, request.TargetExtension.ID,
	); err != nil {
		return wrapLifecycleIdentityError("validate retired identity publication", err)
	}
	tombstones, err := identityregistry.DurableStateToTombstones(durable)
	if err != nil {
		return wrapLifecycleIdentityError("restore durable identity ownership", err)
	}
	for attempts := 0; attempts < 16; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot := b.identity.Snapshot()
		graph, graphErr := lifecycleIdentityGraph(
			snapshot, request.TargetExtension.ID, desiredPublication,
			lifecycleIdentityPublication(source), lifecycleIdentityPublication(target),
		)
		if graphErr != nil {
			return graphErr
		}
		if _, replaceErr := b.identity.ReplaceAllIfRevision(
			snapshot.Revision, graph, tombstones, snapshot.SafeMode,
		); replaceErr == nil {
			return nil
		} else if !errors.Is(replaceErr, identityregistry.ErrRevisionConflict) {
			return wrapLifecycleIdentityError("publish identity registry graph", replaceErr)
		}
	}
	return identityregistry.ErrRevisionConflict
}

func (b *PostgresLifecycleBoundaryRegistries) restoreIdentityPublications(
	ctx context.Context,
	items []extensions.Extension,
	safeMode bool,
) error {
	if !lifecycleIdentityRestoreRequired(b, items) {
		return nil
	}
	if b == nil || b.identity == nil || b.identityStore == nil || ctx == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	durable, err := b.identityStore.LoadDurableState(ctx)
	if err != nil {
		return wrapLifecycleIdentityError("load durable identity registry", err)
	}

	pluginPublications := make([]identityregistry.Publication, 0, len(items))
	validatedPublications := make([]identityregistry.Publication, 0, len(items))
	if !safeMode {
		// Build every expected enabled publication first so adoption never
		// publishes a partial in-memory graph while other owners are still missing.
		type expectedIdentityPublication struct {
			extensionID string
			publication identityregistry.Publication
			suppressed  bool
		}
		expected := make([]expectedIdentityPublication, 0, len(items))
		for _, item := range items {
			if err := ctx.Err(); err != nil {
				return err
			}
			if item.Type != extensions.TypePlugin || item.Status != extensions.StatusEnabled ||
				!manifestPublishesIdentity(item.Manifest) {
				continue
			}
			binding := extensions.LifecycleRuntimeBinding{
				ExtensionID: item.ID, ExtensionVersion: item.Version,
				PackageDigest: item.PackageDigest, VersionID: item.ActiveVersionID,
			}
			suppressed := false
			if manifestIdentityRequiresRuntime(item.Manifest.Identity) {
				runtime, runtimeErr := b.manager.ActiveRuntimeInstance(item.ID)
				if runtimeErr != nil {
					// Validate the durable declaration even while boot-loop suppression
					// keeps its executable process surface closed.
					binding.RuntimeInstanceID = "durable-restore-validation"
					suppressed = true
				} else if !runtimeInstanceMatchesExtension(runtime, item) ||
					!b.manager.RuntimeInstanceAvailable(runtime.Identity) {
					return fmt.Errorf("%w: startup identity runtime for %s is not exact and available",
						ErrLifecycleRegistryPublicationConflict, item.ID)
				} else {
					binding.RuntimeInstanceID = runtime.Identity.InstanceID
				}
			}
			publication, buildErr := buildLifecycleIdentityPublication(item, binding)
			if buildErr != nil || publication == nil {
				return fmt.Errorf("restore identity registry for %s: %w", item.ID, buildErr)
			}
			expected = append(expected, expectedIdentityPublication{
				extensionID: item.ID, publication: *publication, suppressed: suppressed,
			})
		}
		// Stable order matches the adopter batch lock order (extension id).
		sort.Slice(expected, func(i, j int) bool {
			return expected[i].extensionID < expected[j].extensionID
		})

		// Safe Mode never reaches this branch. Normal mode may adopt only on
		// ErrNotFound (missing root/history), never on conflict/stale/partial shapes.
		// Collect every missing publication first, then invoke the adopter ONCE.
		adopter, hasAdopter := b.identityStore.(identityregistry.LegacyPublicationAdopter)
		missing := make([]identityregistry.Publication, 0)
		for _, item := range expected {
			if err := ctx.Err(); err != nil {
				return err
			}
			validateErr := identityregistry.ValidateDurablePublication(durable, item.publication)
			if validateErr != nil {
				if !errors.Is(validateErr, identityregistry.ErrNotFound) {
					action := "validate durable identity publication for " + item.extensionID
					if item.suppressed {
						action = "validate suppressed durable identity publication for " + item.extensionID
					}
					// Keep the owner id in the wrapped error so pre-upgrade enabled
					// plugins without durable identity history are operator-actionable.
					return wrapLifecycleIdentityError(action, validateErr)
				}
				if !hasAdopter {
					action := "validate durable identity publication for " + item.extensionID
					if item.suppressed {
						action = "validate suppressed durable identity publication for " + item.extensionID
					}
					return wrapLifecycleIdentityError(action, validateErr)
				}
				missing = append(missing, item.publication)
			}
			validatedPublications = append(validatedPublications, item.publication)
			if !item.suppressed {
				pluginPublications = append(pluginPublications, item.publication)
			}
		}
		if len(missing) > 0 {
			if _, adoptErr := adopter.AdoptLegacyPublications(ctx, missing); adoptErr != nil {
				owners := make([]string, 0, len(missing))
				for _, publication := range missing {
					owners = append(owners, publication.Artifact.ExtensionID)
				}
				return wrapLifecycleIdentityError(
					"adopt legacy durable identity publications for "+strings.Join(owners, ","), adoptErr,
				)
			}
			// Prefer a fresh read after the adopter commits so concurrent
			// startup nodes observe the same durable tip set.
			reloaded, reloadErr := b.identityStore.LoadDurableState(ctx)
			if reloadErr != nil {
				return wrapLifecycleIdentityError("reload durable identity registry after adoption", reloadErr)
			}
			durable = reloaded
			// Re-validate the full expected set before any in-memory publish.
			if revalidateErr := identityregistry.ValidateDurablePublicationSet(durable, validatedPublications); revalidateErr != nil {
				return wrapLifecycleIdentityError("validate adopted durable identity publication set", revalidateErr)
			}
		} else if err := identityregistry.ValidateDurablePublicationSet(durable, validatedPublications); err != nil {
			return wrapLifecycleIdentityError("validate durable identity publication set", err)
		}
	}

	tombstones, err := identityregistry.DurableStateToTombstones(durable)
	if err != nil {
		return wrapLifecycleIdentityError("restore durable identity ownership", err)
	}

	for attempts := 0; attempts < 16; attempts++ {
		snapshot := b.identity.Snapshot()
		publications := coreLifecycleIdentityPublications(snapshot.Publications)
		publications = append(publications, pluginPublications...)
		if _, err := b.identity.ReplaceAllIfRevision(
			snapshot.Revision, publications, tombstones, safeMode,
		); err == nil {
			return nil
		} else if !errors.Is(err, identityregistry.ErrRevisionConflict) {
			return wrapLifecycleIdentityError("restore identity registry publication", err)
		}
	}
	return identityregistry.ErrRevisionConflict
}

func lifecycleIdentityRestoreRequired(b *PostgresLifecycleBoundaryRegistries, items []extensions.Extension) bool {
	if b != nil && b.identitySet {
		return true
	}
	for _, item := range items {
		if item.Type == extensions.TypePlugin && manifestPublishesIdentity(item.Manifest) {
			return true
		}
	}
	return false
}

func coreLifecycleIdentityPublications(input []identityregistry.Publication) []identityregistry.Publication {
	result := make([]identityregistry.Publication, 0, len(input))
	for _, publication := range input {
		if publication.Artifact.Core {
			result = append(result, publication)
		}
	}
	return result
}

func lifecycleMaterialsPublishIdentity(materials ...*lifecycleRegistryMaterial) bool {
	for _, material := range materials {
		if material != nil && material.identityPublication != nil {
			return true
		}
	}
	return false
}

func lifecycleIdentityPublication(material *lifecycleRegistryMaterial) *identityregistry.Publication {
	if material == nil {
		return nil
	}
	return material.identityPublication
}

func lifecycleIdentityArtifact(material *lifecycleRegistryMaterial) *identityregistry.Artifact {
	publication := lifecycleIdentityPublication(material)
	if publication == nil {
		return nil
	}
	artifact := publication.Artifact
	return &artifact
}

func wrapLifecycleIdentityError(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, identityregistry.ErrConflict) || errors.Is(err, identityregistry.ErrArtifactConflict) ||
		errors.Is(err, identityregistry.ErrRevisionConflict) || errors.Is(err, identityregistry.ErrSafeMode) ||
		errors.Is(err, identityregistry.ErrStale) {
		return fmt.Errorf("%w: %s: %v", ErrLifecycleRegistryPublicationConflict, action, err)
	}
	return fmt.Errorf("%s: %w", action, err)
}

func buildLifecycleIdentityPublication(
	extension extensions.Extension,
	binding extensions.LifecycleRuntimeBinding,
) (*identityregistry.Publication, error) {
	if !manifestPublishesIdentity(extension.Manifest) {
		return nil, nil
	}
	if extension.ID == "" || extension.Version == "" || extension.ActiveVersionID <= 0 ||
		extension.ID != strings.TrimSpace(extension.ID) || extension.Version != strings.TrimSpace(extension.Version) ||
		!validLifecycleCleanupDigest(extension.PackageDigest) || extension.Type != extensions.TypePlugin ||
		extension.Manifest.ID != extension.ID || extension.Manifest.Version != extension.Version ||
		extension.Manifest.Type != extensions.TypePlugin {
		return nil, ErrLifecycleRegistryPublicationInvalid
	}

	artifact := identityregistry.Artifact{
		ExtensionID: extension.ID, ExtensionVersion: extension.Version,
		PackageDigest: extension.PackageDigest, VersionID: extension.ActiveVersionID,
	}
	if manifestIdentityRequiresRuntime(extension.Manifest.Identity) {
		if validateExactCoordinatorBinding("identity registry", binding, extension, true) != nil {
			return nil, ErrLifecycleRegistryPublicationInvalid
		}
		artifact.RuntimeInstanceID = binding.RuntimeInstanceID
	}

	publication := identityregistry.Publication{Artifact: artifact}
	publication.Permissions = make([]identityregistry.PermissionDefinition, 0, len(extension.Manifest.PermissionDefinitions))
	for _, permission := range extension.Manifest.PermissionDefinitions {
		publication.Permissions = append(publication.Permissions, identityregistry.PermissionDefinition{
			Key: permission.Key, ContractVersion: permission.ContractVersion,
			Label: permission.Label, Description: permission.Description,
			RecommendedRoles: append([]string(nil), permission.RecommendedRoles...),
			AssignmentPolicy: permission.AssignmentPolicy,
		})
	}
	if declared := extension.Manifest.Identity; declared != nil {
		identity := identityregistry.IdentityDeclaration{
			ContractVersion: declared.ContractVersion,
			SessionPolicy:   declared.SessionPolicy,
			RiskHooks:       append([]string(nil), declared.RiskHooks...),
			UserFields:      make([]identityregistry.UserField, 0, len(declared.UserFields)),
			Providers:       make([]identityregistry.Provider, 0, len(declared.Providers)),
		}
		for _, field := range declared.UserFields {
			identity.UserFields = append(identity.UserFields, identityregistry.UserField{
				ID: field.ID, ContractVersion: field.ContractVersion, Type: field.Type,
				Schema: field.Schema, ReadPermission: field.ReadPermission,
				WritePermission: field.WritePermission,
			})
		}
		for _, provider := range declared.Providers {
			identity.Providers = append(identity.Providers, identityregistry.Provider{
				ID: provider.ID, ContractVersion: provider.ContractVersion,
				Kind: provider.Kind, Handler: provider.Handler, Priority: provider.Priority,
			})
		}
		publication.Identity = &identity
	}

	// Freeze canonical Registry ordering into lifecycle material and its digest.
	probe := identityregistry.New()
	if _, err := probe.Publish(publication); err != nil {
		return nil, ErrLifecycleRegistryPublicationInvalid
	}
	frozen, found := probe.SnapshotPublication(extension.ID)
	if !found {
		return nil, ErrLifecycleRegistryPublicationInvalid
	}
	return &frozen, nil
}

func manifestPublishesIdentity(manifest extensions.Manifest) bool {
	return manifest.Identity != nil || len(manifest.PermissionDefinitions) > 0
}

func manifestIdentityRequiresRuntime(identity *extensions.ManifestIdentity) bool {
	if identity == nil {
		return false
	}
	return len(identity.Providers) > 0 || len(identity.RiskHooks) > 0 ||
		identity.SessionPolicy != "" && identity.SessionPolicy != "core.session.default"
}

func lifecycleIdentityGraph(
	snapshot identityregistry.Snapshot,
	extensionID string,
	desired *identityregistry.Publication,
	allowed ...*identityregistry.Publication,
) ([]identityregistry.Publication, error) {
	allowedPublications := make(map[identityregistry.Artifact]identityregistry.Publication, len(allowed))
	for _, publication := range allowed {
		if publication == nil {
			continue
		}
		if existing, found := allowedPublications[publication.Artifact]; found && !reflect.DeepEqual(existing, *publication) {
			return nil, ErrLifecycleRegistryPublicationConflict
		}
		allowedPublications[publication.Artifact] = *publication
	}

	publications := make([]identityregistry.Publication, 0, len(snapshot.Publications)+1)
	for _, publication := range snapshot.Publications {
		if publication.Artifact.ExtensionID != extensionID {
			publications = append(publications, publication)
			continue
		}
		frozen, ok := allowedPublications[publication.Artifact]
		if !ok || !reflect.DeepEqual(frozen, publication) {
			return nil, ErrLifecycleRegistryPublicationConflict
		}
	}
	if desired != nil {
		if snapshot.SafeMode {
			return nil, ErrLifecycleRegistryPublicationConflict
		}
		publications = append(publications, *desired)
	}
	return publications, nil
}
