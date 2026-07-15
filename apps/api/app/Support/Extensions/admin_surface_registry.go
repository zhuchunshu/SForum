package extensionsruntime

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

var (
	ErrAdminSurfaceRegistryInvalid  = errors.New("admin surface registry declaration is invalid")
	ErrAdminSurfaceRegistryConflict = errors.New("admin surface registry contract conflicts with the active snapshot")
	ErrAdminSurfaceNotFound         = errors.New("admin surface is not found")
)

type AdminSurfaceContract struct {
	ID               string `json:"id"`
	ContractVersion  string `json:"contractVersion"`
	ExtensionID      string `json:"extensionId"`
	ExtensionVersion string `json:"extensionVersion"`
	ArtifactDigest   string `json:"artifactDigest"`
	InstanceID       string `json:"runtimeInstanceId"`
	Kind             string `json:"kind"`
	Action           string `json:"action"`
	TargetID         string `json:"targetId,omitempty"`
	Label            string `json:"label"`
	Handler          string `json:"handler,omitempty"`
	Schema           string `json:"schema,omitempty"`
	SchemaDigest     string `json:"schemaDigest,omitempty"`
	Permission       string `json:"permission,omitempty"`
	Priority         int    `json:"priority"`

	validator providerDocumentValidator
}

type AdminSurfaceRegistrySnapshot struct {
	Revision uint64                 `json:"revision"`
	Surfaces []AdminSurfaceContract `json:"surfaces"`
}

type adminSurfaceRuntimeRegistration struct {
	extension  extensions.Extension
	instanceID string
}

type adminSurfaceRegistryState struct {
	revision      uint64
	registrations map[string]adminSurfaceRuntimeRegistration
	surfaces      map[string]AdminSurfaceContract
}

// AdminSurfaceRegistry freezes the complete operator UI contribution catalog
// to exact runtime instances. Rendering and handler invocation consume only a
// snapshot, so an upgrade cannot mutate contracts underneath an admin request.
type AdminSurfaceRegistry struct {
	mu    sync.Mutex
	state atomic.Pointer[adminSurfaceRegistryState]
}

func NewAdminSurfaceRegistry() *AdminSurfaceRegistry {
	registry := &AdminSurfaceRegistry{}
	registry.state.Store(emptyAdminSurfaceRegistryState())
	return registry
}

func emptyAdminSurfaceRegistryState() *adminSurfaceRegistryState {
	return &adminSurfaceRegistryState{
		registrations: make(map[string]adminSurfaceRuntimeRegistration),
		surfaces:      make(map[string]AdminSurfaceContract),
	}
}

func (r *AdminSurfaceRegistry) ReplaceRuntime(extension extensions.Extension, instanceID string) error {
	if r == nil {
		return ErrAdminSurfaceRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	registrations := cloneAdminSurfaceRegistrations(current.registrations)
	if previous, ok := registrations[extension.ID]; ok {
		if err := validateAdminSurfaceUpgrade(previous.extension, extension); err != nil {
			return err
		}
	}
	registrations[extension.ID] = adminSurfaceRuntimeRegistration{
		extension: cloneAdminSurfaceExtension(extension), instanceID: strings.TrimSpace(instanceID),
	}
	next, err := buildAdminSurfaceRegistryState(current.revision+1, registrations)
	if err != nil {
		return err
	}
	r.state.Store(next)
	return nil
}

func (r *AdminSurfaceRegistry) ValidateReplaceRuntime(extension extensions.Extension, instanceID string) error {
	if r == nil {
		return ErrAdminSurfaceRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	registrations := cloneAdminSurfaceRegistrations(current.registrations)
	if previous, ok := registrations[extension.ID]; ok {
		if err := validateAdminSurfaceUpgrade(previous.extension, extension); err != nil {
			return err
		}
	}
	registrations[extension.ID] = adminSurfaceRuntimeRegistration{
		extension: cloneAdminSurfaceExtension(extension), instanceID: strings.TrimSpace(instanceID),
	}
	_, err := buildAdminSurfaceRegistryState(current.revision+1, registrations)
	return err
}

func (r *AdminSurfaceRegistry) ValidateRemoveRuntime(extensionID, instanceID string) error {
	current := r.load()
	registration, ok := current.registrations[strings.TrimSpace(extensionID)]
	if !ok {
		return nil
	}
	if registration.instanceID != strings.TrimSpace(instanceID) {
		return ErrAdminSurfaceRegistryConflict
	}
	registrations := cloneAdminSurfaceRegistrations(current.registrations)
	delete(registrations, extensionID)
	_, err := buildAdminSurfaceRegistryState(current.revision+1, registrations)
	return err
}

func (r *AdminSurfaceRegistry) RemoveRuntime(extensionID, instanceID string) (bool, error) {
	if r == nil {
		return false, ErrAdminSurfaceRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	registration, ok := current.registrations[strings.TrimSpace(extensionID)]
	if !ok {
		return false, nil
	}
	if registration.instanceID != strings.TrimSpace(instanceID) {
		return false, ErrAdminSurfaceRegistryConflict
	}
	registrations := cloneAdminSurfaceRegistrations(current.registrations)
	delete(registrations, extensionID)
	next, err := buildAdminSurfaceRegistryState(current.revision+1, registrations)
	if err != nil {
		return false, err
	}
	r.state.Store(next)
	return true, nil
}

func (r *AdminSurfaceRegistry) Resolve(id string) (AdminSurfaceContract, error) {
	contract, err := r.resolveRuntimeContract(id)
	if err != nil {
		return AdminSurfaceContract{}, err
	}
	return cloneAdminSurfaceContract(contract), nil
}

// resolveRuntimeContract retains the compiled validator from one immutable
// snapshot so an admitted call never re-reads a replacement publication while
// validating its result.
func (r *AdminSurfaceRegistry) resolveRuntimeContract(id string) (AdminSurfaceContract, error) {
	contract, ok := r.load().surfaces[strings.TrimSpace(id)]
	if !ok {
		return AdminSurfaceContract{}, ErrAdminSurfaceNotFound
	}
	return contract, nil
}

func (r *AdminSurfaceRegistry) ValidateDocument(contract AdminSurfaceContract, document map[string]any) error {
	if contract.validator == nil {
		stored, ok := r.load().surfaces[contract.ID]
		if !ok || !sameAdminSurfaceRuntimeContract(stored, contract) {
			return ErrAdminSurfaceNotFound
		}
		contract = stored
	}
	if contract.validator == nil {
		return nil
	}
	if err := contract.validator.Validate(document); err != nil {
		return fmt.Errorf("%w: %v", ErrAdminSurfaceRegistryInvalid, err)
	}
	return nil
}

func sameAdminSurfaceRuntimeContract(left, right AdminSurfaceContract) bool {
	return left.ID == right.ID && left.ContractVersion == right.ContractVersion &&
		left.ExtensionID == right.ExtensionID && left.ExtensionVersion == right.ExtensionVersion &&
		left.ArtifactDigest == right.ArtifactDigest && left.InstanceID == right.InstanceID &&
		left.Kind == right.Kind && left.Action == right.Action && left.TargetID == right.TargetID &&
		left.Label == right.Label && left.Handler == right.Handler && left.Schema == right.Schema &&
		left.SchemaDigest == right.SchemaDigest && left.Permission == right.Permission && left.Priority == right.Priority
}

func (r *AdminSurfaceRegistry) Snapshot(kind string) AdminSurfaceRegistrySnapshot {
	state := r.load()
	result := AdminSurfaceRegistrySnapshot{Revision: state.revision}
	for _, surface := range state.surfaces {
		if kind != "" && surface.Kind != kind {
			continue
		}
		result.Surfaces = append(result.Surfaces, cloneAdminSurfaceContract(surface))
	}
	sort.Slice(result.Surfaces, func(i, j int) bool {
		return adminSurfaceBefore(result.Surfaces[i], result.Surfaces[j])
	})
	return result
}

func (r *AdminSurfaceRegistry) load() *adminSurfaceRegistryState {
	if r == nil {
		return emptyAdminSurfaceRegistryState()
	}
	if state := r.state.Load(); state != nil {
		return state
	}
	return emptyAdminSurfaceRegistryState()
}

func buildAdminSurfaceRegistryState(
	revision uint64,
	registrations map[string]adminSurfaceRuntimeRegistration,
) (*adminSurfaceRegistryState, error) {
	state := emptyAdminSurfaceRegistryState()
	state.revision = revision
	state.registrations = cloneAdminSurfaceRegistrations(registrations)
	for extensionID, registration := range state.registrations {
		extension := registration.extension
		if extension.ID != extensionID || extension.Type != extensions.TypePlugin ||
			extension.Manifest.Backend.ProtocolVersion != 2 || extension.PackageDigest == "" || registration.instanceID == "" {
			return nil, ErrAdminSurfaceRegistryInvalid
		}
		for _, declaration := range extension.Manifest.AdminSurfaces {
			if err := validateAdminSurfaceDeclaration(extension, declaration); err != nil {
				return nil, err
			}
			if _, duplicate := state.surfaces[declaration.ID]; duplicate {
				return nil, fmt.Errorf("%w: duplicate surface %s", ErrAdminSurfaceRegistryConflict, declaration.ID)
			}
			validator, digest, err := compileAdminSurfaceSchema(extension, declaration.Schema)
			if err != nil {
				return nil, fmt.Errorf("%w: surface %s schema: %v", ErrAdminSurfaceRegistryInvalid, declaration.ID, err)
			}
			state.surfaces[declaration.ID] = AdminSurfaceContract{
				ID: declaration.ID, ContractVersion: declaration.ContractVersion,
				ExtensionID: extension.ID, ExtensionVersion: extension.Version,
				ArtifactDigest: extension.PackageDigest, InstanceID: registration.instanceID,
				Kind: declaration.Kind, Action: declaration.Action, TargetID: declaration.TargetID,
				Label: declaration.Label, Handler: declaration.Handler, Schema: declaration.Schema,
				SchemaDigest: digest, Permission: declaration.Permission, Priority: declaration.Priority,
				validator: validator,
			}
		}
	}
	for _, contract := range state.surfaces {
		if contract.Action == "add" {
			continue
		}
		target, ok := state.surfaces[contract.TargetID]
		if !ok {
			registration := state.registrations[contract.ExtensionID]
			dependency, declared := hookDependencyForTarget(registration.extension, contract.TargetID)
			if declared && dependency.Kind == "optional" {
				delete(state.surfaces, contract.ID)
				continue
			}
			return nil, fmt.Errorf("%w: surface %s target %s", ErrAdminSurfaceRegistryConflict, contract.ID, contract.TargetID)
		}
		if target.Action != "add" || target.Kind != contract.Kind {
			return nil, fmt.Errorf("%w: surface %s target %s", ErrAdminSurfaceRegistryConflict, contract.ID, contract.TargetID)
		}
		if target.ExtensionID == contract.ExtensionID {
			continue
		}
		registration := state.registrations[contract.ExtensionID]
		dependency, declared, compatible := hookDependency(registration.extension, target.ExtensionID, target.ExtensionVersion)
		if !declared || !compatible {
			if declared && dependency.Kind == "optional" {
				delete(state.surfaces, contract.ID)
				continue
			}
			return nil, fmt.Errorf("%w: surface %s dependency", ErrAdminSurfaceRegistryConflict, contract.ID)
		}
	}
	return state, nil
}

func validateAdminSurfaceDeclaration(extension extensions.Extension, declaration extensions.ManifestAdminSurface) error {
	if !strings.HasPrefix(declaration.ID, extension.ID+".") || declaration.ContractVersion == "" ||
		!validRuntimeAdminSurfaceKind(declaration.Kind) || !validRuntimeAdminSurfaceAction(declaration.Action) ||
		declaration.Label == "" || declaration.Handler == "" && declaration.Schema == "" ||
		declaration.Action != "add" && declaration.TargetID == "" {
		return fmt.Errorf("%w: surface %s", ErrAdminSurfaceRegistryInvalid, declaration.ID)
	}
	return nil
}

func compileAdminSurfaceSchema(
	extension extensions.Extension,
	reference string,
) (providerDocumentValidator, string, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, "", nil
	}
	if strings.TrimSpace(extension.PackagePath) == "" {
		digest, _ := providerSchemaDigest(extension, reference)
		return nil, digest, nil
	}
	return compileExactProviderSchema(extension, reference)
}

func validateAdminSurfaceUpgrade(previous, next extensions.Extension) error {
	oldSurfaces := make(map[string]extensions.ManifestAdminSurface, len(previous.Manifest.AdminSurfaces))
	for _, surface := range previous.Manifest.AdminSurfaces {
		oldSurfaces[surface.ID] = surface
	}
	for _, surface := range next.Manifest.AdminSurfaces {
		old, ok := oldSurfaces[surface.ID]
		if !ok || old.ContractVersion != surface.ContractVersion {
			continue
		}
		oldDigest, oldDigestOK := providerSchemaDigest(previous, old.Schema)
		newDigest, newDigestOK := providerSchemaDigest(next, surface.Schema)
		if old != surface || oldDigestOK != newDigestOK || oldDigestOK && oldDigest != newDigest {
			return fmt.Errorf("%w: surface %s changed without contract version", ErrAdminSurfaceRegistryConflict, surface.ID)
		}
	}
	return nil
}

func validRuntimeAdminSurfaceKind(value string) bool {
	switch value {
	case "navigation", "dashboard", "list_column", "list_filter", "row_action", "bulk_action",
		"form", "notice", "editor_panel", "detail_region", "importer", "exporter":
		return true
	default:
		return false
	}
}

func validRuntimeAdminSurfaceAction(value string) bool {
	switch value {
	case "add", "before", "after", "wrap", "replace", "hide", "filter":
		return true
	default:
		return false
	}
}

func adminSurfaceBefore(left, right AdminSurfaceContract) bool {
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	if left.ExtensionID != right.ExtensionID {
		return left.ExtensionID < right.ExtensionID
	}
	return left.ID < right.ID
}

func cloneAdminSurfaceContract(contract AdminSurfaceContract) AdminSurfaceContract {
	contract.validator = nil
	return contract
}

func cloneAdminSurfaceExtension(extension extensions.Extension) extensions.Extension {
	extension.Manifest.AdminSurfaces = append([]extensions.ManifestAdminSurface(nil), extension.Manifest.AdminSurfaces...)
	extension.Manifest.Dependencies = append([]extensions.ManifestDependency(nil), extension.Manifest.Dependencies...)
	extension.Manifest.PackageFiles = append([]extensions.ManifestPackageFile(nil), extension.Manifest.PackageFiles...)
	return extension
}

func cloneAdminSurfaceRegistrations(
	values map[string]adminSurfaceRuntimeRegistration,
) map[string]adminSurfaceRuntimeRegistration {
	result := make(map[string]adminSurfaceRuntimeRegistration, len(values))
	for id, registration := range values {
		result[id] = adminSurfaceRuntimeRegistration{
			extension: cloneAdminSurfaceExtension(registration.extension), instanceID: registration.instanceID,
		}
	}
	return result
}
