package extensionsruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	componentcatalog "github.com/zhuchunshu/sforum/apps/api/app/Support/ComponentCatalog"
	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

var (
	ErrComponentRegistryInvalid          = errors.New("component registry declaration is invalid")
	ErrComponentRegistryConflict         = errors.New("component registry contract conflicts with the active snapshot")
	ErrComponentRegistryTargetNotFound   = errors.New("component registry target is not found")
	ErrComponentRegistryProviderNotFound = errors.New("component registry replace provider is not found")
	ErrComponentRegistryRevisionConflict = errors.New("component registry revision conflict")
)

type ComponentContribution struct {
	ID                    string       `json:"id"`
	ContractVersion       string       `json:"contractVersion"`
	Action                string       `json:"action"`
	TargetID              string       `json:"targetId,omitempty"`
	TargetContractVersion string       `json:"targetContractVersion,omitempty"`
	Priority              int          `json:"priority"`
	SSRTemplate           string       `json:"ssrTemplate,omitempty"`
	L2Component           string       `json:"l2Component,omitempty"`
	PropsSchema           string       `json:"propsSchema,omitempty"`
	PropsSchemaDigest     string       `json:"propsSchemaDigest,omitempty"`
	ResultSchema          string       `json:"resultSchema,omitempty"`
	ResultSchemaDigest    string       `json:"resultSchemaDigest,omitempty"`
	ThemeOverrideKey      string       `json:"themeOverrideKey,omitempty"`
	Artifact              HookArtifact `json:"artifact"`

	manifest        extensions.ManifestComponent
	propsValidator  providerDocumentValidator
	resultValidator providerDocumentValidator
}

type ComponentTarget struct {
	ID              string                   `json:"id"`
	ContractVersion string                   `json:"contractVersion"`
	Core            bool                     `json:"core"`
	Kind            componentcatalog.Kind    `json:"kind,omitempty"`
	Owners          []componentcatalog.Owner `json:"owners,omitempty"`
	Route           string                   `json:"route,omitempty"`
	Source          string                   `json:"source,omitempty"`
	Provider        *ComponentContribution   `json:"provider,omitempty"`
}

type ComponentProviderSelection struct {
	TargetID              string       `json:"targetId"`
	TargetContractVersion string       `json:"targetContractVersion"`
	ContributionID        string       `json:"contributionId"`
	Artifact              HookArtifact `json:"artifact"`
	SelectedAtRevision    uint64       `json:"selectedAtRevision"`
}

type ComponentProviderConflict struct {
	TargetID              string                  `json:"targetId"`
	TargetContractVersion string                  `json:"targetContractVersion"`
	Candidates            []ComponentContribution `json:"candidates"`
	Winner                *ComponentContribution  `json:"winner,omitempty"`
	ExplicitSelection     bool                    `json:"explicitSelection"`
}

type ComponentRegistrySnapshot struct {
	Revision      uint64                       `json:"revision"`
	Targets       []ComponentTarget            `json:"targets"`
	Contributions []ComponentContribution      `json:"contributions"`
	Conflicts     []ComponentProviderConflict  `json:"conflicts"`
	Selections    []ComponentProviderSelection `json:"selections"`
}

type ComponentResolvePlan struct {
	Revision          uint64                      `json:"revision"`
	Target            ComponentTarget             `json:"target"`
	Contributions     []ComponentContribution     `json:"contributions"`
	ReplaceCandidates []ComponentContribution     `json:"replaceCandidates,omitempty"`
	ReplaceWinner     *ComponentContribution      `json:"replaceWinner,omitempty"`
	Conflict          *ComponentProviderConflict  `json:"conflict,omitempty"`
	Selection         *ComponentProviderSelection `json:"selection,omitempty"`
}

type SelectComponentProviderRequest struct {
	TargetID              string
	TargetContractVersion string
	ContributionID        string
	ExpectedRevision      uint64
}

type ResetComponentProviderRequest struct {
	TargetID              string
	TargetContractVersion string
	ExpectedRevision      uint64
}

// ComponentRuntimeSnapshot identifies one exact Host-published package graph.
// Component publication is declarative and therefore does not borrow a plugin
// process identity, including for packages that have no backend process.
type ComponentRuntimeSnapshot struct {
	Extension  extensions.Extension `json:"extension"`
	InstanceID string               `json:"instanceId"`
}

type componentRuntimeRegistration struct {
	extension  extensions.Extension
	instanceID string
}

type componentRegistryState struct {
	revision              uint64
	registrations         map[string]componentRuntimeRegistration
	targetsByID           map[string]ComponentTarget
	contributionsByID     map[string]ComponentContribution
	contributionsByTarget map[string][]ComponentContribution
	replaceByTarget       map[string][]ComponentContribution
	replaceWinnerByTarget map[string]ComponentContribution
	conflictsByTarget     map[string]ComponentProviderConflict
	selectionsByTarget    map[string]ComponentProviderSelection
}

// ComponentRegistry publishes one complete component graph. Writers rebuild
// off to the side and readers observe either the old or the new exact-runtime
// snapshot, never a partially removed target or modifier chain.
type ComponentRegistry struct {
	mu    sync.Mutex
	state atomic.Pointer[componentRegistryState]
}

func NewComponentRegistry() *ComponentRegistry {
	registry := &ComponentRegistry{}
	registry.state.Store(emptyComponentRegistryState())
	return registry
}

func (r *ComponentRegistry) ReplaceRuntime(extension extensions.Extension, instanceID string) error {
	return r.replaceRuntime(extension, instanceID, true, false, nil)
}

func (r *ComponentRegistry) replaceRuntime(
	extension extensions.Extension,
	instanceID string,
	publish bool,
	idempotent bool,
	currentAllowed func(componentRuntimeRegistration) bool,
) error {
	if r == nil || validateComponentRuntime(extension, instanceID) != nil {
		return ErrComponentRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	registrations := cloneComponentRegistrations(current.registrations)
	if previous, found := registrations[extension.ID]; found {
		if currentAllowed != nil && !currentAllowed(previous) {
			return ErrComponentRegistryConflict
		}
		if idempotent && componentRuntimeRegistrationMatches(previous, extension, instanceID) {
			return nil
		}
		if err := validateComponentUpgrade(previous.extension, extension); err != nil {
			return err
		}
	}
	registrations[extension.ID] = componentRuntimeRegistration{
		extension: cloneComponentExtension(extension), instanceID: strings.TrimSpace(instanceID),
	}
	next, err := buildComponentRegistryState(current.revision+1, registrations, current.selectionsByTarget)
	if err != nil {
		return err
	}
	if publish {
		r.state.Store(next)
	}
	return nil
}

func (r *ComponentRegistry) ValidateReplaceRuntime(extension extensions.Extension, instanceID string) error {
	return r.replaceRuntime(extension, instanceID, false, false, nil)
}

// ReplaceAll validates and publishes one complete component graph. The graph is
// built once, so required cross-plugin targets do not depend on input order.
func (r *ComponentRegistry) ReplaceAll(snapshots []ComponentRuntimeSnapshot) error {
	if r == nil {
		return ErrComponentRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	registrations := make(map[string]componentRuntimeRegistration, len(snapshots))
	for _, snapshot := range snapshots {
		extension := snapshot.Extension
		instanceID := strings.TrimSpace(snapshot.InstanceID)
		if validateComponentRuntime(extension, instanceID) != nil {
			return ErrComponentRegistryInvalid
		}
		if _, duplicate := registrations[extension.ID]; duplicate {
			return fmt.Errorf("%w: duplicate runtime %s", ErrComponentRegistryConflict, extension.ID)
		}
		if previous, found := current.registrations[extension.ID]; found {
			if err := validateComponentUpgrade(previous.extension, extension); err != nil {
				return err
			}
		}
		registrations[extension.ID] = componentRuntimeRegistration{
			extension: cloneComponentExtension(extension), instanceID: instanceID,
		}
	}
	if componentRuntimeRegistrationsMatch(current.registrations, registrations) {
		return nil
	}
	next, err := buildComponentRegistryState(current.revision+1, registrations, current.selectionsByTarget)
	if err != nil {
		return err
	}
	r.state.Store(next)
	return nil
}

// RestoreRuntimes reconstructs enabled Manifest V3 component packages without
// requiring an executable backend. Safe Mode atomically removes all extension
// registrations while retaining the Host-owned Core target catalog.
func (r *ComponentRegistry) RestoreRuntimes(items []extensions.Extension, safeMode bool) error {
	snapshots := make([]ComponentRuntimeSnapshot, 0, len(items))
	if !safeMode {
		for _, item := range items {
			if item.Status != extensions.StatusEnabled || item.Manifest.ManifestVersion != 3 ||
				(item.Type != extensions.TypePlugin && item.Type != extensions.TypeTheme) ||
				len(item.Manifest.Components) == 0 {
				continue
			}
			snapshots = append(snapshots, ComponentRuntimeSnapshot{
				Extension: item, InstanceID: componentPackageRuntimeInstanceID(item),
			})
		}
	}
	return r.ReplaceAll(snapshots)
}

func (r *ComponentRegistry) RuntimeSnapshot(extensionID string) (ComponentRuntimeSnapshot, bool) {
	if r == nil || strings.TrimSpace(extensionID) == "" {
		return ComponentRuntimeSnapshot{}, false
	}
	registration, found := r.load().registrations[strings.TrimSpace(extensionID)]
	if !found {
		return ComponentRuntimeSnapshot{}, false
	}
	return ComponentRuntimeSnapshot{
		Extension: cloneComponentExtension(registration.extension), InstanceID: registration.instanceID,
	}, true
}

func (r *ComponentRegistry) ValidateRemoveRuntime(extensionID, instanceID string) error {
	if r == nil {
		return ErrComponentRegistryInvalid
	}
	current := r.load()
	registration, found := current.registrations[strings.TrimSpace(extensionID)]
	if !found {
		return nil
	}
	if registration.instanceID != strings.TrimSpace(instanceID) {
		return ErrComponentRegistryConflict
	}
	registrations := cloneComponentRegistrations(current.registrations)
	delete(registrations, strings.TrimSpace(extensionID))
	_, err := buildComponentRegistryState(current.revision+1, registrations, current.selectionsByTarget)
	return err
}

func (r *ComponentRegistry) RemoveRuntime(extensionID, instanceID string) (bool, error) {
	if r == nil || strings.TrimSpace(extensionID) == "" || strings.TrimSpace(instanceID) == "" {
		return false, ErrComponentRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	registration, found := current.registrations[strings.TrimSpace(extensionID)]
	if !found {
		return false, nil
	}
	if registration.instanceID != strings.TrimSpace(instanceID) {
		return false, ErrComponentRegistryConflict
	}
	registrations := cloneComponentRegistrations(current.registrations)
	delete(registrations, strings.TrimSpace(extensionID))
	next, err := buildComponentRegistryState(current.revision+1, registrations, current.selectionsByTarget)
	if err != nil {
		return false, err
	}
	r.state.Store(next)
	return true, nil
}

func (r *ComponentRegistry) SelectReplaceProvider(
	request SelectComponentProviderRequest,
) (ComponentProviderSelection, error) {
	if r == nil {
		return ComponentProviderSelection{}, ErrComponentRegistryInvalid
	}
	request.TargetID = strings.TrimSpace(request.TargetID)
	request.TargetContractVersion = strings.TrimSpace(request.TargetContractVersion)
	request.ContributionID = strings.TrimSpace(request.ContributionID)
	if request.TargetID == "" || request.TargetContractVersion == "" || request.ContributionID == "" {
		return ComponentProviderSelection{}, ErrComponentRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != request.ExpectedRevision {
		return ComponentProviderSelection{}, ErrComponentRegistryRevisionConflict
	}
	target, found := current.targetsByID[request.TargetID]
	if !found || target.ContractVersion != request.TargetContractVersion {
		return ComponentProviderSelection{}, ErrComponentRegistryTargetNotFound
	}
	var candidate ComponentContribution
	for _, item := range current.replaceByTarget[request.TargetID] {
		if item.ID == request.ContributionID {
			candidate = item
			break
		}
	}
	if candidate.ID == "" {
		return ComponentProviderSelection{}, ErrComponentRegistryProviderNotFound
	}
	selection := ComponentProviderSelection{
		TargetID: request.TargetID, TargetContractVersion: request.TargetContractVersion,
		ContributionID: candidate.ID, Artifact: candidate.Artifact,
		SelectedAtRevision: current.revision + 1,
	}
	selections := cloneComponentSelections(current.selectionsByTarget)
	selections[request.TargetID] = selection
	next := rebuildComponentProviderState(current, current.revision+1, selections)
	r.state.Store(next)
	return selection, nil
}

func (r *ComponentRegistry) ResetReplaceProvider(request ResetComponentProviderRequest) (bool, error) {
	if r == nil {
		return false, ErrComponentRegistryInvalid
	}
	request.TargetID = strings.TrimSpace(request.TargetID)
	request.TargetContractVersion = strings.TrimSpace(request.TargetContractVersion)
	if request.TargetID == "" || request.TargetContractVersion == "" {
		return false, ErrComponentRegistryInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.load()
	if current.revision != request.ExpectedRevision {
		return false, ErrComponentRegistryRevisionConflict
	}
	target, found := current.targetsByID[request.TargetID]
	if !found || target.ContractVersion != request.TargetContractVersion {
		return false, ErrComponentRegistryTargetNotFound
	}
	if _, selected := current.selectionsByTarget[request.TargetID]; !selected {
		return false, nil
	}
	selections := cloneComponentSelections(current.selectionsByTarget)
	delete(selections, request.TargetID)
	next := rebuildComponentProviderState(current, current.revision+1, selections)
	r.state.Store(next)
	return true, nil
}

func (r *ComponentRegistry) ResolvePlan(targetID, contractVersion string) (ComponentResolvePlan, error) {
	plan, err := r.resolveRuntimePlan(targetID, contractVersion)
	if err != nil {
		return ComponentResolvePlan{}, err
	}
	return cloneComponentResolvePlan(plan), nil
}

func (r *ComponentRegistry) resolveRuntimePlan(targetID, contractVersion string) (ComponentResolvePlan, error) {
	state := r.load()
	target, found := state.targetsByID[strings.TrimSpace(targetID)]
	if !found || target.ContractVersion != strings.TrimSpace(contractVersion) {
		return ComponentResolvePlan{}, ErrComponentRegistryTargetNotFound
	}
	plan := ComponentResolvePlan{
		Revision: state.revision, Target: target,
		ReplaceCandidates: append([]ComponentContribution(nil), state.replaceByTarget[target.ID]...),
	}
	if winner, ok := state.replaceWinnerByTarget[target.ID]; ok {
		value := winner
		plan.ReplaceWinner = &value
	}
	for _, contribution := range state.contributionsByTarget[target.ID] {
		if contribution.Action == extensionmanifest.ComponentActionReplace &&
			(plan.ReplaceWinner == nil || !sameComponentRuntimeContribution(contribution, *plan.ReplaceWinner)) {
			continue
		}
		plan.Contributions = append(plan.Contributions, contribution)
	}
	if conflict, ok := state.conflictsByTarget[target.ID]; ok {
		value := conflict
		plan.Conflict = &value
	}
	if selection, ok := state.selectionsByTarget[target.ID]; ok {
		value := selection
		plan.Selection = &value
	}
	return plan, nil
}

func (r *ComponentRegistry) Snapshot() ComponentRegistrySnapshot {
	state := r.load()
	snapshot := ComponentRegistrySnapshot{Revision: state.revision}
	for _, target := range state.targetsByID {
		snapshot.Targets = append(snapshot.Targets, cloneComponentTarget(target))
	}
	for _, contribution := range state.contributionsByID {
		snapshot.Contributions = append(snapshot.Contributions, cloneComponentContribution(contribution))
	}
	for _, conflict := range state.conflictsByTarget {
		snapshot.Conflicts = append(snapshot.Conflicts, cloneComponentProviderConflict(conflict))
	}
	for _, selection := range state.selectionsByTarget {
		snapshot.Selections = append(snapshot.Selections, selection)
	}
	sort.Slice(snapshot.Targets, func(i, j int) bool { return snapshot.Targets[i].ID < snapshot.Targets[j].ID })
	sort.Slice(snapshot.Contributions, func(i, j int) bool {
		if snapshot.Contributions[i].TargetID != snapshot.Contributions[j].TargetID {
			return snapshot.Contributions[i].TargetID < snapshot.Contributions[j].TargetID
		}
		return componentContributionBefore(snapshot.Contributions[i], snapshot.Contributions[j])
	})
	sort.Slice(snapshot.Conflicts, func(i, j int) bool { return snapshot.Conflicts[i].TargetID < snapshot.Conflicts[j].TargetID })
	sort.Slice(snapshot.Selections, func(i, j int) bool { return snapshot.Selections[i].TargetID < snapshot.Selections[j].TargetID })
	return snapshot
}

func (r *ComponentRegistry) ValidateProps(contribution ComponentContribution, document any) error {
	return r.validateComponentDocument(contribution, document, true)
}

func (r *ComponentRegistry) ValidateResult(contribution ComponentContribution, document any) error {
	return r.validateComponentDocument(contribution, document, false)
}

func (r *ComponentRegistry) validateComponentDocument(
	contribution ComponentContribution,
	document any,
	props bool,
) error {
	if contribution.propsValidator == nil && contribution.resultValidator == nil {
		stored, found := r.load().contributionsByID[contribution.ID]
		if !found || !sameComponentRuntimeContribution(stored, contribution) {
			return ErrComponentRegistryTargetNotFound
		}
		contribution = stored
	}
	validator := contribution.resultValidator
	if props {
		validator = contribution.propsValidator
	}
	if validator == nil {
		return nil
	}
	if err := validator.Validate(document); err != nil {
		return fmt.Errorf("%w: %v", ErrComponentRegistryInvalid, err)
	}
	return nil
}

func (r *ComponentRegistry) load() *componentRegistryState {
	if r == nil {
		return emptyComponentRegistryState()
	}
	if state := r.state.Load(); state != nil {
		return state
	}
	return emptyComponentRegistryState()
}

// This format is a restart-stable Host identity, not a plugin process lease.
// The domain-separated NUL-delimited tuple is frozen by a hardcoded test vector.
func componentPackageRuntimeInstanceID(extension extensions.Extension) string {
	document := strings.Join([]string{
		"sforum.component.package-publication@1",
		extension.Type,
		extension.ID,
		extension.Version,
		strings.ToLower(strings.TrimSpace(extension.PackageDigest)),
	}, "\x00")
	sum := sha256.Sum256([]byte(document))
	return "host-component-package:" + hex.EncodeToString(sum[:])
}

func componentRuntimeRegistrationMatches(
	registration componentRuntimeRegistration,
	extension extensions.Extension,
	instanceID string,
) bool {
	return registration.instanceID == strings.TrimSpace(instanceID) &&
		registration.extension.ID == extension.ID && registration.extension.Version == extension.Version &&
		registration.extension.Type == extension.Type && registration.extension.PackageDigest == extension.PackageDigest
}

func componentRuntimeRegistrationsMatch(
	left, right map[string]componentRuntimeRegistration,
) bool {
	if len(left) != len(right) {
		return false
	}
	for extensionID, registration := range left {
		candidate, found := right[extensionID]
		if !found || !componentRuntimeRegistrationMatches(
			candidate,
			registration.extension,
			registration.instanceID,
		) {
			return false
		}
	}
	return true
}
