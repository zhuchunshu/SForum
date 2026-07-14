package extensionsruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	routes "github.com/zhuchunshu/sforum/apps/api/app/Support/Routes"
)

var (
	ErrLifecycleRegistryPublicationInvalid     = errors.New("extension lifecycle registry publication input is invalid")
	ErrLifecycleRegistryPublicationConflict    = errors.New("extension lifecycle registry publication exact fence conflict")
	ErrLifecycleRegistryPublicationNotPrepared = errors.New("extension lifecycle registry publication is not prepared")
	ErrLifecycleRegistryPublicationCommitted   = errors.New("extension lifecycle registry publication marker is committed")
	ErrLifecycleRegistryPublicationUnavailable = errors.New("extension lifecycle registry publication dependency is unavailable")
)

type LifecycleRegistryPublicationPhase string

const (
	LifecycleRegistryPublicationSource LifecycleRegistryPublicationPhase = "source"
	LifecycleRegistryPublicationTarget LifecycleRegistryPublicationPhase = "target"
)

type LifecycleRegistryPublicationRef struct {
	OperationID int64
	StepID      string
	Mode        LifecycleBoundaryPublicationMode
	Attempt     int
}

type PrepareLifecycleRegistryPublicationInput struct {
	Fence        lifecyclePublicationFence
	SourceDigest string
	TargetDigest string
}

// LifecycleRegistryPublicationRepository owns the durable aggregate phase.
// Move keeps the PostgreSQL fence locked while one node reconciles its local
// immutable snapshots, so an opposite or stale attempt cannot commit midway.
type LifecycleRegistryPublicationRepository interface {
	PrepareLifecycleRegistryPublication(context.Context, PrepareLifecycleRegistryPublicationInput) (LifecycleRegistryPublicationRef, error)
	InspectLifecycleRegistryPublication(context.Context, LifecycleRegistryPublicationRef) (LifecycleRegistryPublicationPhase, error)
	MoveLifecycleRegistryPublication(
		context.Context,
		LifecycleRegistryPublicationRef,
		LifecycleRegistryPublicationPhase,
		func() error,
	) error
}

type LifecycleRegistryBoundaryConfig struct {
	Repository LifecycleRegistryPublicationRepository
	Manager    *Manager
	Pages      *pages.Registry
	Routes     *routes.Registry
	Services   *hostapi.ServiceRegistry
}

// PostgresLifecycleBoundaryRegistries composes the production HookBus,
// Protocol-v2 Service Registry and Page Registry. The Route Registry member is
// the P6 immutable-snapshot foundation only; this adapter does not claim that
// P6 route execution or later P7 registry families are complete.
type PostgresLifecycleBoundaryRegistries struct {
	repository LifecycleRegistryPublicationRepository
	manager    *Manager
	hooks      *HookBus
	pages      *pages.Registry
	routes     *routes.Registry
	services   *hostapi.ServiceRegistry
}

func NewPostgresLifecycleBoundaryRegistries(config LifecycleRegistryBoundaryConfig) *PostgresLifecycleBoundaryRegistries {
	boundary := &PostgresLifecycleBoundaryRegistries{
		repository: config.Repository,
		manager:    config.Manager,
		pages:      config.Pages,
		routes:     config.Routes,
		services:   config.Services,
	}
	if config.Manager != nil {
		boundary.hooks = config.Manager.HookBus()
	}
	if boundary.pages != nil && boundary.manager != nil {
		boundary.pages.WithRuntimeAdmission(func(artifact pages.RuntimeArtifact) bool {
			identity := RuntimeInstanceIdentity{ExtensionID: artifact.ExtensionID, InstanceID: artifact.RuntimeInstanceID}
			snapshot, err := boundary.manager.InspectRuntimeInstance(identity)
			return err == nil && snapshot.ExtensionVersion == artifact.ExtensionVersion &&
				snapshot.ArtifactDigest == artifact.PackageDigest && boundary.manager.RuntimeInstanceAvailable(identity)
		})
	}
	if boundary.routes != nil && boundary.manager != nil {
		boundary.routes.WithPluginAdmission(func(artifact routes.PluginArtifact) bool {
			identity := RuntimeInstanceIdentity{ExtensionID: artifact.ExtensionID, InstanceID: artifact.RuntimeInstanceID}
			snapshot, err := boundary.manager.InspectRuntimeInstance(identity)
			return err == nil && snapshot.ExtensionVersion == artifact.ExtensionVersion &&
				snapshot.ArtifactDigest == artifact.PackageDigest && boundary.manager.RuntimeInstanceAvailable(identity)
		})
	}
	return boundary
}

func (b *PostgresLifecycleBoundaryRegistries) ValidateLifecycleRegistries(
	ctx context.Context,
	request LifecycleBoundaryRequest,
) error {
	if err := b.validateDependencies(ctx); err != nil {
		return err
	}
	_, source, target, err := b.prepareMaterial(request)
	if err != nil {
		return err
	}
	for _, material := range []*lifecycleRegistryMaterial{source, target} {
		if material == nil {
			continue
		}
		if err := b.validateRuntimeMaterial(*material); err != nil {
			return err
		}
		if err := b.pages.PreflightContributionsReplacing(material.extension.ID, material.pages, material.extension.ID); err != nil {
			return fmt.Errorf("validate page registry: %w", err)
		}
		if err := b.validateRouteMaterial(*material); err != nil {
			return err
		}
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) PrepareLifecycleRegistryPublication(
	ctx context.Context,
	request LifecycleBoundaryRequest,
	mode LifecycleBoundaryPublicationMode,
) (LifecycleBoundaryTransaction, error) {
	if err := b.validateDependencies(ctx); err != nil {
		return nil, err
	}
	fence, source, target, err := b.prepareMaterial(request)
	if err != nil {
		return nil, err
	}
	if fence.Mode != mode {
		return nil, ErrLifecycleRegistryPublicationInvalid
	}
	if err := b.ValidateLifecycleRegistries(ctx, request); err != nil {
		return nil, err
	}
	ref, err := b.repository.PrepareLifecycleRegistryPublication(ctx, PrepareLifecycleRegistryPublicationInput{
		Fence: fence, SourceDigest: registryMaterialDigest(source, request.TargetExtension.ID),
		TargetDigest: registryMaterialDigest(target, request.TargetExtension.ID),
	})
	if err != nil {
		return nil, err
	}
	return &postgresLifecycleRegistryTransaction{
		boundary: b, ref: ref, request: cloneLifecycleBoundaryRequest(request), source: source, target: target,
	}, nil
}

func (b *PostgresLifecycleBoundaryRegistries) validateDependencies(ctx context.Context) error {
	if b == nil || ctx == nil || b.repository == nil || b.manager == nil || b.hooks == nil ||
		b.pages == nil || b.routes == nil || b.services == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	return ctx.Err()
}

type lifecycleRegistryMaterial struct {
	extension extensions.Extension
	binding   extensions.LifecycleRuntimeBinding
	pages     []pages.PageContribution
	routes    routes.PluginRouteSet
	digest    string
}

type lifecycleRegistryDigestDocument struct {
	Schema             string                       `json:"schema"`
	ExtensionID        string                       `json:"extensionId"`
	ExtensionVersion   string                       `json:"extensionVersion"`
	PackageDigest      string                       `json:"packageDigest"`
	VersionID          int64                        `json:"versionId"`
	RuntimeInstanceID  string                       `json:"runtimeInstanceId"`
	Hooks              []extensions.ManifestEvent   `json:"hooks"`
	Services           []extensions.ManifestService `json:"services"`
	Pages              []pages.PageContribution     `json:"pages"`
	Routes             routes.PluginRouteSet        `json:"routes"`
	ProductionFamilies []string                     `json:"productionFamilies"`
	FoundationFamilies []string                     `json:"foundationFamilies"`
}

func (b *PostgresLifecycleBoundaryRegistries) prepareMaterial(
	request LifecycleBoundaryRequest,
) (lifecyclePublicationFence, *lifecycleRegistryMaterial, *lifecycleRegistryMaterial, error) {
	mode := LifecycleBoundaryActivate
	if request.Operation == extensions.LifecycleMachineDisable || request.Operation == extensions.LifecycleMachineUninstall {
		mode = LifecycleBoundaryDeactivate
	}
	fence, err := lifecyclePublicationFenceFor(request, mode)
	if err != nil {
		return lifecyclePublicationFence{}, nil, nil, err
	}
	var source *lifecycleRegistryMaterial
	if request.Operation != extensions.LifecycleMachineInstall && request.Operation != extensions.LifecycleMachineEnable {
		if request.SourceExtension == nil {
			return lifecyclePublicationFence{}, nil, nil, ErrLifecycleRegistryPublicationInvalid
		}
		value, materialErr := buildLifecycleRegistryMaterial(*request.SourceExtension, request.SourceBinding)
		if materialErr != nil {
			return lifecyclePublicationFence{}, nil, nil, materialErr
		}
		source = &value
	}
	var target *lifecycleRegistryMaterial
	if mode == LifecycleBoundaryActivate {
		value, materialErr := buildLifecycleRegistryMaterial(request.TargetExtension, request.TargetBinding)
		if materialErr != nil {
			return lifecyclePublicationFence{}, nil, nil, materialErr
		}
		target = &value
	}
	return fence, source, target, nil
}

func buildLifecycleRegistryMaterial(
	extension extensions.Extension,
	binding extensions.LifecycleRuntimeBinding,
) (lifecycleRegistryMaterial, error) {
	if err := validateExactCoordinatorArtifact("registry", extension); err != nil ||
		validateExactCoordinatorBinding("registry", binding, extension, true) != nil {
		return lifecycleRegistryMaterial{}, ErrLifecycleRegistryPublicationInvalid
	}
	pkg, err := pages.LoadThemePackage(extensions.PackageContentRoot(extension))
	if err != nil {
		return lifecycleRegistryMaterial{}, fmt.Errorf("load plugin page package: %w", err)
	}
	pageContributions := pages.ContributionsFromTheme(extension.ID, extension.Version, extension.PackageDigest, pkg)
	for index := range pageContributions {
		pageContributions[index].RuntimeInstanceID = binding.RuntimeInstanceID
		if template := strings.TrimSpace(pageContributions[index].Template); template != "" {
			if _, err := pages.LoadTemplate(extensions.PackageContentRoot(extension), template); err != nil {
				return lifecycleRegistryMaterial{}, fmt.Errorf("validate plugin page template %s: %w", template, err)
			}
		}
	}
	material := lifecycleRegistryMaterial{
		extension: extension, binding: binding, pages: pageContributions,
		routes: routes.PluginRouteSet{Artifact: routes.PluginArtifact{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, RuntimeInstanceID: binding.RuntimeInstanceID,
		}, Routes: append([]extensions.ManifestRoute(nil), extension.Manifest.Routes...)},
	}
	document := lifecycleRegistryDigestDocument{
		Schema: "sforum.lifecycle.registry-plan@1", ExtensionID: extension.ID,
		ExtensionVersion: extension.Version, PackageDigest: extension.PackageDigest,
		VersionID: extension.ActiveVersionID, RuntimeInstanceID: binding.RuntimeInstanceID,
		Hooks:    append([]extensions.ManifestEvent(nil), extensions.DeclaredManifestEvents(extension.Manifest)...),
		Services: append([]extensions.ManifestService(nil), extension.Manifest.Services...),
		Pages:    append([]pages.PageContribution(nil), pageContributions...), Routes: material.routes,
		ProductionFamilies: []string{"hooks.v1", "pages.runtime", "services.v2"},
		FoundationFamilies: []string{"routes.v1-foundation"},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return lifecycleRegistryMaterial{}, fmt.Errorf("encode lifecycle registry plan: %w", err)
	}
	sum := sha256.Sum256(raw)
	material.digest = hex.EncodeToString(sum[:])
	return material, nil
}

func registryMaterialDigest(material *lifecycleRegistryMaterial, extensionID string) string {
	if material != nil {
		return material.digest
	}
	raw, _ := json.Marshal(struct {
		Schema      string `json:"schema"`
		ExtensionID string `json:"extensionId"`
		State       string `json:"state"`
	}{"sforum.lifecycle.registry-plan@1", extensionID, "absent"})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (b *PostgresLifecycleBoundaryRegistries) validateRuntimeMaterial(material lifecycleRegistryMaterial) error {
	identity := RuntimeInstanceIdentity{
		ExtensionID: material.binding.ExtensionID, InstanceID: material.binding.RuntimeInstanceID,
	}
	snapshot, err := b.manager.InspectRuntimeInstance(identity)
	if err != nil || !runtimeInstanceMatchesExtension(snapshot, material.extension) {
		return fmt.Errorf("%w: exact registry runtime is unavailable", ErrLifecycleRegistryPublicationConflict)
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) validateRouteMaterial(material lifecycleRegistryMaterial) error {
	snapshot := b.routes.PublicationSnapshot()
	plugins := make([]routes.PluginRouteSet, 0, len(snapshot.Publication.Plugins)+1)
	for _, plugin := range snapshot.Publication.Plugins {
		if plugin.Artifact.ExtensionID != material.extension.ID {
			plugins = append(plugins, plugin)
		}
	}
	snapshot.Publication.Plugins = append(plugins, material.routes)
	publication := snapshot.Publication
	_, err := routes.NewRegistry().Publish(publication)
	if err != nil {
		return fmt.Errorf("validate route registry foundation: %w", err)
	}
	return nil
}

type postgresLifecycleRegistryTransaction struct {
	boundary *PostgresLifecycleBoundaryRegistries
	ref      LifecycleRegistryPublicationRef
	request  LifecycleBoundaryRequest
	source   *lifecycleRegistryMaterial
	target   *lifecycleRegistryMaterial
}

func (t *postgresLifecycleRegistryTransaction) Inspect(ctx context.Context) (LifecycleBoundaryTransactionState, error) {
	if t == nil || t.boundary == nil || ctx == nil {
		return "", ErrLifecycleRegistryPublicationUnavailable
	}
	phase, err := t.boundary.repository.InspectLifecycleRegistryPublication(ctx, t.ref)
	if err != nil {
		return "", err
	}
	switch phase {
	case LifecycleRegistryPublicationSource:
		return LifecycleBoundaryTransactionSource, nil
	case LifecycleRegistryPublicationTarget:
		return LifecycleBoundaryTransactionTarget, nil
	default:
		return "", ErrLifecycleRegistryPublicationConflict
	}
}

func (t *postgresLifecycleRegistryTransaction) Publish(ctx context.Context) error {
	return t.move(ctx, LifecycleRegistryPublicationTarget, t.target)
}

func (t *postgresLifecycleRegistryTransaction) Restore(ctx context.Context) error {
	return t.move(ctx, LifecycleRegistryPublicationSource, t.source)
}

func (t *postgresLifecycleRegistryTransaction) move(
	ctx context.Context,
	phase LifecycleRegistryPublicationPhase,
	desired *lifecycleRegistryMaterial,
) error {
	if t == nil || t.boundary == nil || ctx == nil {
		return ErrLifecycleRegistryPublicationUnavailable
	}
	return t.boundary.repository.MoveLifecycleRegistryPublication(ctx, t.ref, phase, func() error {
		return t.boundary.reconcileLocalRegistries(ctx, t.request, t.source, t.target, desired, phase)
	})
}

func (b *PostgresLifecycleBoundaryRegistries) reconcileLocalRegistries(
	ctx context.Context,
	request LifecycleBoundaryRequest,
	source, target, desired *lifecycleRegistryMaterial,
	phase LifecycleRegistryPublicationPhase,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if phase == LifecycleRegistryPublicationTarget && desired != nil {
		identity := RuntimeInstanceIdentity{ExtensionID: desired.extension.ID, InstanceID: desired.binding.RuntimeInstanceID}
		snapshot, err := b.manager.ActiveRuntimeInstance(desired.extension.ID)
		if err != nil || snapshot.Identity != identity || !runtimeInstanceMatchesExtension(snapshot, desired.extension) ||
			!snapshot.Admission.Draining || snapshot.Admission.ActiveTotal != 0 {
			return fmt.Errorf("%w: target runtime is not published and drained", ErrLifecycleRegistryPublicationConflict)
		}
	}
	if err := b.reconcileServices(request.TargetExtension.ID, source, target, desired, phase); err != nil {
		return err
	}
	if err := b.reconcileRoutes(ctx, request.TargetExtension.ID, source, target, desired); err != nil {
		return err
	}
	if err := b.reconcilePages(ctx, request.TargetExtension.ID, source, target, desired); err != nil {
		return err
	}
	if desired == nil {
		b.unregisterAllowedHookRuntime(request.TargetExtension.ID, source, target)
	} else {
		b.hooks.RegisterRuntime(desired.extension, desired.binding.RuntimeInstanceID)
	}
	return nil
}

func (b *PostgresLifecycleBoundaryRegistries) reconcileServices(
	extensionID string,
	source, target, desired *lifecycleRegistryMaterial,
	phase LifecycleRegistryPublicationPhase,
) error {
	snapshot, exists, err := b.services.ExtensionSnapshot(extensionID)
	if err != nil {
		return err
	}
	if desired == nil {
		if !exists {
			return nil
		}
		if !registryInstanceAllowed(snapshot.InstanceID, source, target) ||
			!b.services.UnregisterProtocolV2ServiceInstance(extensionID, snapshot.InstanceID) {
			return ErrLifecycleRegistryPublicationConflict
		}
		return nil
	}
	if len(desired.extension.Manifest.Services) == 0 {
		if exists && snapshot.InstanceID != desired.binding.RuntimeInstanceID {
			return ErrLifecycleRegistryPublicationConflict
		}
		return nil
	}
	if exists && snapshot.InstanceID == desired.binding.RuntimeInstanceID &&
		len(snapshot.Registrations) == len(desired.extension.Manifest.Services) {
		return nil
	}
	// Restore runs before Manager republishes a retained source. An empty
	// Service Registry is a deliberate closed state; PublishInstance rebuilds
	// the frozen service set before admission can reopen.
	if phase == LifecycleRegistryPublicationSource {
		if !exists {
			return nil
		}
		if registryInstanceAllowed(snapshot.InstanceID, source, target) &&
			b.services.UnregisterProtocolV2ServiceInstance(extensionID, snapshot.InstanceID) {
			return nil
		}
	}
	return fmt.Errorf("%w: exact service set is unavailable", ErrLifecycleRegistryPublicationConflict)
}

func (b *PostgresLifecycleBoundaryRegistries) unregisterAllowedHookRuntime(
	extensionID string,
	source, target *lifecycleRegistryMaterial,
) {
	current, ok := b.hooks.RuntimeSnapshot(extensionID)
	if !ok {
		return
	}
	if registryInstanceAllowed(current.InstanceID, source, target) {
		b.hooks.UnregisterRuntime(extensionID, current.InstanceID)
	}
}

func registryInstanceAllowed(instanceID string, materials ...*lifecycleRegistryMaterial) bool {
	for _, material := range materials {
		if material != nil && material.binding.RuntimeInstanceID == instanceID {
			return true
		}
	}
	return false
}

func (b *PostgresLifecycleBoundaryRegistries) reconcileRoutes(
	ctx context.Context,
	extensionID string,
	source, target, desired *lifecycleRegistryMaterial,
) error {
	for attempts := 0; attempts < 16; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot := b.routes.PublicationSnapshot()
		var desiredSet *routes.PluginRouteSet
		if desired != nil {
			value := desired.routes
			desiredSet = &value
		}
		publication, err := replaceLifecycleRouteSet(snapshot.Publication, extensionID, desiredSet, source, target)
		if err != nil {
			return err
		}
		if _, err := b.routes.PublishIfRevision(publication, snapshot.Revision); err == nil {
			return nil
		} else if !errors.Is(err, routes.ErrRevisionConflict) {
			return err
		}
	}
	return routes.ErrRevisionConflict
}

func replaceLifecycleRouteSet(
	publication routes.Publication,
	extensionID string,
	desired *routes.PluginRouteSet,
	source, target *lifecycleRegistryMaterial,
) (routes.Publication, error) {
	plugins := make([]routes.PluginRouteSet, 0, len(publication.Plugins)+1)
	for _, plugin := range publication.Plugins {
		if plugin.Artifact.ExtensionID != extensionID {
			plugins = append(plugins, plugin)
			continue
		}
		if source != nil && sameRouteArtifact(plugin.Artifact, source.routes.Artifact) ||
			target != nil && sameRouteArtifact(plugin.Artifact, target.routes.Artifact) {
			continue
		}
		return routes.Publication{}, ErrLifecycleRegistryPublicationConflict
	}
	if desired != nil {
		plugins = append(plugins, *desired)
	}
	publication.Plugins = plugins
	return publication, nil
}

func sameRouteArtifact(left, right routes.PluginArtifact) bool {
	return left.ExtensionID == right.ExtensionID && left.ExtensionVersion == right.ExtensionVersion &&
		left.PackageDigest == right.PackageDigest && left.RuntimeInstanceID == right.RuntimeInstanceID
}

func (b *PostgresLifecycleBoundaryRegistries) reconcilePages(
	ctx context.Context,
	extensionID string,
	source, target, desired *lifecycleRegistryMaterial,
) error {
	for attempts := 0; attempts < 16; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot, exists := b.pages.ExtensionSnapshot(extensionID)
		if exists && !pageArtifactAllowed(snapshot.Artifact, source, target) {
			return ErrLifecycleRegistryPublicationConflict
		}
		if desired != nil {
			artifact := pages.RuntimeArtifact{
				ExtensionID: desired.extension.ID, ExtensionVersion: desired.extension.Version,
				PackageDigest: desired.extension.PackageDigest, RuntimeInstanceID: desired.binding.RuntimeInstanceID,
			}
			if _, err := b.pages.PublishExtensionIfRevision(artifact, desired.pages, snapshot.Revision); err == nil {
				return nil
			} else if !errors.Is(err, pages.ErrRevisionConflict) {
				return err
			}
			continue
		}
		if !exists {
			return nil
		}
		if _, err := b.pages.RemoveExtensionIfRevision(extensionID, snapshot.Artifact, snapshot.Revision); err == nil {
			return nil
		} else if !errors.Is(err, pages.ErrRevisionConflict) {
			return err
		}
	}
	return pages.ErrRevisionConflict
}

func pageArtifactAllowed(artifact pages.RuntimeArtifact, materials ...*lifecycleRegistryMaterial) bool {
	for _, material := range materials {
		if material == nil {
			continue
		}
		if artifact.ExtensionID == material.extension.ID &&
			artifact.ExtensionVersion == material.extension.Version &&
			artifact.PackageDigest == material.extension.PackageDigest &&
			artifact.RuntimeInstanceID == material.binding.RuntimeInstanceID {
			return true
		}
	}
	return false
}

var _ LifecycleBoundaryRegistries = (*PostgresLifecycleBoundaryRegistries)(nil)
var _ LifecycleBoundaryTransaction = (*postgresLifecycleRegistryTransaction)(nil)
