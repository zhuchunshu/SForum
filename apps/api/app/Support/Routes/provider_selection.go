package routes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

var (
	ErrProviderSelectionInvalid          = errors.New("routes: invalid provider selection")
	ErrProviderSelectionNotFound         = errors.New("routes: provider selection not found")
	ErrProviderSelectionRevisionConflict = errors.New("routes: provider selection revision conflict")
	ErrProviderSelectionStale            = errors.New("routes: provider selection is stale")
)

type ProviderSelectionKey struct {
	TargetRouteID         string `json:"targetRouteId"`
	TargetContractVersion string `json:"targetContractVersion"`
	Method                string `json:"method"`
	PathSignature         string `json:"pathSignature"`
}

type ProviderSelection struct {
	Key                        ProviderSelectionKey `json:"key"`
	ProviderRouteID            string               `json:"routeId"`
	ProviderContractVersion    string               `json:"contractVersion"`
	ProviderExtensionID        string               `json:"extensionId"`
	ProviderExtensionVersionID int64                `json:"extensionVersionId"`
	ProviderExtensionVersion   string               `json:"extensionVersion"`
	ProviderPackageDigest      string               `json:"packageDigest"`
	SelectedByUserID           int64                `json:"selectedByUserId"`
	SelectionAuditEventID      int64                `json:"selectionAuditEventId"`
	Revision                   int64                `json:"revision"`
	SelectedAt                 time.Time            `json:"selectedAt"`
	UpdatedAt                  time.Time            `json:"updatedAt"`
}

type SelectProviderRequest struct {
	Key                     ProviderSelectionKey
	ProviderRouteID         string
	ProviderContractVersion string
	ProviderArtifact        PluginArtifact
	ExpectedRevision        int64
	ActorUserID             int64
	AuditEventID            int64
}

type ResetProviderRequest struct {
	Key              ProviderSelectionKey
	ExpectedRevision int64
	ActorUserID      int64
	AuditEventID     int64
	ReasonCode       string
}

type InvalidateProviderRequest struct {
	ExtensionID  string
	ActorUserID  int64
	AuditEventID int64
	ReasonCode   string
}

type ProviderSelectionEvent struct {
	ID                int64
	Key               ProviderSelectionKey
	Action            string
	PreviousProvider  *ProviderSelection
	SelectedProvider  *ProviderSelection
	ActorUserID       int64
	AuditEventID      int64
	ReasonCode        string
	SelectionRevision int64
	CreatedAt         time.Time
}

type ProviderSelectionStore interface {
	Selected(context.Context, ProviderSelectionKey) (ProviderSelection, error)
	Select(context.Context, SelectProviderRequest) (ProviderSelection, error)
	Reset(context.Context, ResetProviderRequest) error
	InvalidateExtension(context.Context, InvalidateProviderRequest) (int64, error)
	ListEvents(context.Context, ProviderSelectionKey, int) ([]ProviderSelectionEvent, error)
}

// ProviderSelectionAPI resolves ambiguous replace candidates only through a
// durable, exact-artifact administrator choice. Priority never chooses a
// replacement implicitly.
type ProviderSelectionAPI struct {
	registry *Registry
	store    ProviderSelectionStore
}

func NewProviderSelectionAPI(registry *Registry, store ProviderSelectionStore) *ProviderSelectionAPI {
	return &ProviderSelectionAPI{registry: registry, store: store}
}

func (a *ProviderSelectionAPI) Select(ctx context.Context, request SelectProviderRequest) (ProviderSelection, error) {
	if a == nil || a.registry == nil || a.store == nil {
		return ProviderSelection{}, fmt.Errorf("%w: provider selection API is not configured", ErrProviderSelectionInvalid)
	}
	if request.ExpectedRevision < 0 || request.ActorUserID <= 0 || request.AuditEventID <= 0 ||
		validateProviderSelectionKey(request.Key) != nil {
		return ProviderSelection{}, ErrProviderSelectionInvalid
	}
	snapshot := a.registry.Snapshot()
	candidate, err := exactReplacementCandidate(snapshot, request)
	if err != nil {
		return ProviderSelection{}, err
	}
	request.ProviderRouteID = candidate.ID
	request.ProviderContractVersion = candidate.ContractVersion
	request.ProviderArtifact = candidate.Provider.Artifact
	selected, err := a.store.Select(ctx, request)
	if err != nil {
		return ProviderSelection{}, err
	}
	return selected, nil
}

func (a *ProviderSelectionAPI) Reset(ctx context.Context, request ResetProviderRequest) error {
	if a == nil || a.store == nil || request.ExpectedRevision <= 0 || request.ActorUserID <= 0 ||
		request.AuditEventID <= 0 || validateProviderSelectionKey(request.Key) != nil ||
		!validReasonCode(request.ReasonCode, false) {
		return ErrProviderSelectionInvalid
	}
	return a.store.Reset(ctx, request)
}

func (a *ProviderSelectionAPI) InvalidateExtension(ctx context.Context, request InvalidateProviderRequest) (int64, error) {
	if a == nil || a.store == nil || !routeIDPattern.MatchString(request.ExtensionID) ||
		request.ActorUserID <= 0 || request.AuditEventID <= 0 || !validReasonCode(request.ReasonCode, true) {
		return 0, ErrProviderSelectionInvalid
	}
	return a.store.InvalidateExtension(ctx, request)
}

// InvalidateRouteProviderSelections is the narrow lifecycle adapter. It keeps
// Models/Extensions independent from Registry implementation types while
// preserving actor/audit evidence for every invalidated binding.
func (a *ProviderSelectionAPI) InvalidateRouteProviderSelections(
	ctx context.Context,
	extensionID string,
	actorUserID int64,
	auditEventID int64,
	reasonCode string,
) error {
	_, err := a.InvalidateExtension(ctx, InvalidateProviderRequest{
		ExtensionID: extensionID, ActorUserID: actorUserID,
		AuditEventID: auditEventID, ReasonCode: reasonCode,
	})
	return err
}

func (a *ProviderSelectionAPI) Resolve(ctx context.Context, method, requestPath string) (Match, error) {
	if a == nil || a.registry == nil || a.store == nil {
		return Match{}, fmt.Errorf("%w: provider selection API is not configured", ErrProviderSelectionInvalid)
	}
	match, err := a.registry.Resolve(method, requestPath)
	if err == nil || !errors.Is(err, ErrAmbiguousRoute) {
		return match, err
	}
	snapshot := a.registry.Snapshot()
	if snapshot.Revision != match.Revision {
		return Match{}, ErrRevisionConflict
	}
	key, err := providerSelectionKeyForMatch(snapshot, match, method, requestPath)
	if err != nil {
		return match, err
	}
	selection, err := a.store.Selected(ctx, key)
	if err != nil {
		if errors.Is(err, ErrProviderSelectionNotFound) {
			return match, ErrAmbiguousRoute
		}
		return Match{}, err
	}
	if selection.Key != key {
		return Match{}, ErrProviderSelectionStale
	}
	return selectedMatch(snapshot, match, selection, method, requestPath)
}

func (a *ProviderSelectionAPI) BuildExecutionPlan(ctx context.Context, method, requestPath string) (RouteExecutionPlan, error) {
	for attempt := 0; attempt < 4; attempt++ {
		match, err := a.Resolve(ctx, method, requestPath)
		if err != nil {
			return RouteExecutionPlan{}, err
		}
		snapshot := a.registry.Snapshot()
		if snapshot.Revision != match.Revision {
			continue
		}
		return buildRouteExecutionPlan(snapshot, match, method, requestPath)
	}
	return RouteExecutionPlan{}, ErrRevisionConflict
}

func exactReplacementCandidate(snapshot Snapshot, request SelectProviderRequest) (Route, error) {
	var target *Route
	var candidate *Route
	for index := range snapshot.Routes {
		route := snapshot.Routes[index]
		if route.ID == request.Key.TargetRouteID && addressableAction(route.Action) &&
			route.ContractVersion == request.Key.TargetContractVersion && route.PathSignature == request.Key.PathSignature &&
			methodsOverlap(route.Method, request.Key.Method) {
			copy := route
			target = &copy
		}
		if route.Action == extensionmanifest.RouteActionReplace && route.TargetID == request.Key.TargetRouteID &&
			route.ID == request.ProviderRouteID && route.ContractVersion == request.ProviderContractVersion &&
			route.PathSignature == request.Key.PathSignature && methodsOverlap(route.Method, request.Key.Method) &&
			route.Provider.Kind == ProviderPlugin && route.Provider.Artifact == request.ProviderArtifact {
			copy := route
			candidate = &copy
		}
	}
	if target == nil || candidate == nil || snapshot.SafeMode {
		return Route{}, ErrProviderSelectionStale
	}
	return *candidate, nil
}

func providerSelectionKeyForMatch(snapshot Snapshot, match Match, method, requestPath string) (ProviderSelectionKey, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	normalizedPath, err := normalizeRequestPath(requestPath)
	if err != nil {
		return ProviderSelectionKey{}, err
	}
	for _, conflict := range snapshot.Conflicts {
		if conflict.Kind != ConflictProviderSelection || !methodsOverlap(conflict.Method, method) ||
			!sameRouteCandidateSet(conflict.Candidates, match.Candidates) {
			continue
		}
		for _, route := range conflict.Candidates {
			if route.ID != conflict.RouteID || !addressableAction(route.Action) {
				continue
			}
			compiled, compileErr := compileRoutePath(route.Path)
			if compileErr == nil {
				if _, ok := compiled.match(normalizedPath); ok {
					selectionMethod := conflict.Method
					if selectionMethod != "*" {
						selectionMethod = method
					}
					return ProviderSelectionKey{
						TargetRouteID: route.ID, TargetContractVersion: route.ContractVersion,
						Method: selectionMethod, PathSignature: route.PathSignature,
					}, nil
				}
			}
		}
	}
	return ProviderSelectionKey{}, ErrAmbiguousRoute
}

func selectedMatch(snapshot Snapshot, ambiguous Match, selection ProviderSelection, method, requestPath string) (Match, error) {
	currentKey, err := providerSelectionKeyForMatch(snapshot, ambiguous, method, requestPath)
	if err != nil || selection.Key != currentKey || selection.Revision <= 0 {
		return Match{}, ErrProviderSelectionStale
	}
	normalizedPath, err := normalizeRequestPath(requestPath)
	if err != nil {
		return Match{}, err
	}
	var selected Route
	for _, route := range ambiguous.Candidates {
		if route.Action == extensionmanifest.RouteActionReplace && route.ID == selection.ProviderRouteID &&
			route.ContractVersion == selection.ProviderContractVersion &&
			route.Provider.Artifact.ExtensionID == selection.ProviderExtensionID &&
			route.Provider.Artifact.ExtensionVersion == selection.ProviderExtensionVersion &&
			route.Provider.Artifact.PackageDigest == selection.ProviderPackageDigest {
			selected = route
			break
		}
	}
	if selected.ID == "" {
		return Match{}, ErrProviderSelectionStale
	}
	compiled, err := compileRoutePath(selected.Path)
	if err != nil {
		return Match{}, ErrProviderSelectionStale
	}
	params, ok := compiled.match(normalizedPath)
	if !ok {
		return Match{}, ErrProviderSelectionStale
	}
	contributions := matchingContributions(snapshot, selected.TargetID, method, normalizedPath)
	return Match{Revision: snapshot.Revision, Route: selected, Params: params, Contributions: contributions}, nil
}

func matchingContributions(snapshot Snapshot, targetID, method, requestPath string) []Route {
	result := make([]Route, 0)
	for _, route := range snapshot.Routes {
		if route.Provider.Kind != ProviderPlugin {
			continue
		}
		if route.Action == extensionmanifest.RouteActionGlobalMiddleware {
			result = append(result, route)
			continue
		}
		if !compositionAction(route.Action) || route.TargetID != targetID || !routeMethodMatches(route, method) {
			continue
		}
		compiled, err := compileRoutePath(route.Path)
		if err == nil {
			if _, ok := compiled.match(requestPath); ok {
				result = append(result, route)
			}
		}
	}
	sortRoutes(result)
	return result
}

func sameRouteCandidateSet(left, right []Route) bool {
	if len(left) != len(right) {
		return false
	}
	remaining := append([]Route(nil), right...)
	for _, candidate := range left {
		found := -1
		for index := range remaining {
			if remaining[index] == candidate {
				found = index
				break
			}
		}
		if found < 0 {
			return false
		}
		remaining = append(remaining[:found], remaining[found+1:]...)
	}
	return true
}

func validateProviderSelectionKey(key ProviderSelectionKey) error {
	if !routeIDPattern.MatchString(key.TargetRouteID) || !contractPattern.MatchString(key.TargetContractVersion) ||
		!validMethod(key.Method) || key.Method != strings.ToUpper(key.Method) || key.PathSignature == "" ||
		len(key.PathSignature) > 1024 || key.PathSignature != strings.TrimSpace(key.PathSignature) {
		return ErrProviderSelectionInvalid
	}
	return nil
}

func validReasonCode(value string, required bool) bool {
	if value == "" {
		return !required
	}
	return len(value) <= 128 && routeIDPattern.MatchString(value)
}
