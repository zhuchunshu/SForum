package extensionsruntime

import (
	"context"
	"errors"
	"fmt"

	extensionopenapi "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionOpenAPI"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

type lifecycleRoutePublicationCAS interface {
	PublicationSnapshot() routes.PublicationSnapshot
	PublishIfRevision(routes.Publication, uint64) (routes.Snapshot, error)
}

type lifecycleRoutePublicationCandidate func(routes.Publication) (routes.Publication, error)

// restoreRoutePolicyPublications first proves that the Route and OpenAPI
// candidates compose, then publishes schemas before the bound Route snapshot.
// Every Route CAS retry resolves from the live schema revision so a concurrent
// unrelated Route writer cannot retain stale policy bindings.
func (b *PostgresLifecycleBoundaryRegistries) restoreRoutePolicyPublications(
	ctx context.Context,
	pluginRoutes []routes.PluginRouteSet,
	schemaArtifacts []extensionopenapi.Artifact,
	safeMode bool,
) error {
	var schemaChange lifecycleRouteSchemaChange
	for attempts := 0; attempts < 16; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		before := b.routeSchemas.PublicationSnapshot()
		preparedSchemas, err := b.routeSchemas.Prepare(schemaArtifacts)
		if err != nil {
			return fmt.Errorf("prepare startup route schema publication: %w", err)
		}
		publication := startupRoutePublication(
			b.routes.PublicationSnapshot().Publication, pluginRoutes, safeMode,
		)
		if _, err := routes.BindRouteExecutionPolicies(publication, preparedSchemas); err != nil {
			return fmt.Errorf("bind startup route policies: %w", err)
		}
		// Prepare reads the current owner snapshot internally. If another writer
		// advanced it between our revision read and Prepare, retry the full pair.
		if b.routeSchemas.Revision() != before.Revision {
			continue
		}
		if published, err := b.routeSchemas.PublishPrepared(preparedSchemas, before.Revision); err == nil {
			schemaChange = lifecycleRouteSchemaChange{before: before, published: published, changed: true}
			break
		} else if !errors.Is(err, extensionopenapi.ErrRouteSchemaRevisionConflict) {
			return fmt.Errorf("restore route schema publication: %w", err)
		}
	}
	if !schemaChange.changed {
		return extensionopenapi.ErrRouteSchemaRevisionConflict
	}

	err := b.publishBoundRoutePublication(ctx, func(publication routes.Publication) (routes.Publication, error) {
		return startupRoutePublication(publication, pluginRoutes, safeMode), nil
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, routes.ErrRevisionConflict) {
		err = fmt.Errorf("restore route registry publication: %w", err)
	}
	return b.restoreRouteSchemasAfterRouteFailure(schemaChange, err)
}

type lifecycleRouteSchemaChange struct {
	before    extensionopenapi.RouteSchemaPublicationSnapshot
	published extensionopenapi.RouteSchemaPublicationSnapshot
	changed   bool
}

// restoreRouteSchemasAfterRouteFailure prevents a successful schema CAS from
// surviving a failed Route publication. The owner-bound snapshot and expected
// revision ensure compensation cannot overwrite a newer legitimate writer.
func (b *PostgresLifecycleBoundaryRegistries) restoreRouteSchemasAfterRouteFailure(
	change lifecycleRouteSchemaChange,
	cause error,
) error {
	if !change.changed {
		return cause
	}
	if _, err := b.routeSchemas.Restore(change.before, change.published.Revision); err != nil {
		return errors.Join(cause, fmt.Errorf("restore route schemas after route publication failure: %w", err))
	}
	return cause
}

func (b *PostgresLifecycleBoundaryRegistries) publishBoundRoutePublication(
	ctx context.Context,
	build lifecycleRoutePublicationCandidate,
) error {
	if b == nil || ctx == nil || b.routePublisher == nil || b.routeSchemas == nil || build == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	for attempts := 0; attempts < 16; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot := b.routePublisher.PublicationSnapshot()
		publication, err := build(snapshot.Publication)
		if err != nil {
			return err
		}
		bound, err := routes.BindRouteExecutionPolicies(publication, b.routeSchemas)
		if err != nil {
			return fmt.Errorf("bind lifecycle route policies: %w", err)
		}
		if _, err := b.routePublisher.PublishIfRevision(bound, snapshot.Revision); err == nil {
			return nil
		} else if !errors.Is(err, routes.ErrRevisionConflict) {
			return err
		}
	}
	return routes.ErrRevisionConflict
}

func (b *PostgresLifecycleBoundaryRegistries) reconcileRoutePolicyPublications(
	ctx context.Context,
	extensionID string,
	source, target, desired *lifecycleRegistryMaterial,
) error {
	schemaChange, err := b.reconcileRouteSchemas(ctx, extensionID, source, target, desired)
	if err != nil {
		return err
	}
	if err := b.reconcileRoutes(ctx, extensionID, source, target, desired); err != nil {
		return b.restoreRouteSchemasAfterRouteFailure(schemaChange, err)
	}
	return nil
}

func startupRoutePublication(
	publication routes.Publication,
	pluginRoutes []routes.PluginRouteSet,
	safeMode bool,
) routes.Publication {
	publication.SafeMode = safeMode
	publication.Plugins = append([]routes.PluginRouteSet(nil), pluginRoutes...)
	return publication
}

// validateLifecycleRoutePolicyStates proves both durable phases before the
// lifecycle marker is prepared. This includes rollback to source and removal
// to an absent source/target state.
func (b *PostgresLifecycleBoundaryRegistries) validateLifecycleRoutePolicyStates(
	source, target *lifecycleRegistryMaterial,
) error {
	extensionID := lifecycleRoutePolicyExtensionID(source, target)
	if extensionID == "" {
		return ErrLifecycleRegistryPublicationInvalid
	}
	if err := b.validateLifecycleRoutePolicyState(extensionID, source, source, target); err != nil {
		return err
	}
	return b.validateLifecycleRoutePolicyState(extensionID, target, source, target)
}

func lifecycleRoutePolicyExtensionID(materials ...*lifecycleRegistryMaterial) string {
	for _, material := range materials {
		if material != nil {
			return material.extension.ID
		}
	}
	return ""
}

// validateLifecycleRoutePolicyState compiles one complete candidate without
// mutating either live owner. The same replacement fence and policy binder are
// reused by actual publication, avoiding a validation-only composition rule.
func (b *PostgresLifecycleBoundaryRegistries) validateLifecycleRoutePolicyState(
	extensionID string,
	desired, source, target *lifecycleRegistryMaterial,
) error {
	var desiredRoutes *routes.PluginRouteSet
	var desiredSchema *extensionopenapi.Artifact
	if desired != nil {
		routeValue := desired.routes
		desiredRoutes = &routeValue
		schemaValue := desired.routeSchema
		desiredSchema = &schemaValue
	}
	publication, err := replaceLifecycleRouteSet(
		b.routes.PublicationSnapshot().Publication,
		extensionID,
		desiredRoutes,
		source,
		target,
	)
	if err != nil {
		return err
	}
	preparedSchemas, err := b.routeSchemas.PrepareExtensionReplacement(
		extensionID,
		desiredSchema,
		lifecycleRouteSchemaAllowedArtifacts(source, target),
	)
	if err != nil {
		if errors.Is(err, extensionopenapi.ErrRouteSchemaArtifactConflict) {
			return fmt.Errorf("%w: route schema exact artifact", ErrLifecycleRegistryPublicationConflict)
		}
		return fmt.Errorf("prepare lifecycle route schemas: %w", err)
	}
	if _, err := routes.BindRouteExecutionPolicies(publication, preparedSchemas); err != nil {
		return fmt.Errorf("bind lifecycle route policies: %w", err)
	}
	return nil
}
