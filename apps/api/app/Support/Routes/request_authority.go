package routes

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
	"strings"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

type RequestAuthorityMode string

const (
	RequestAuthorityFiltered RequestAuthorityMode = "filtered"
	RequestAuthorityRaw      RequestAuthorityMode = "raw"
)

type RequestGuardKind string

const (
	RequestGuardHost       RequestGuardKind = "host"
	RequestGuardCustom     RequestGuardKind = "custom"
	RequestGuardRawRequest RequestGuardKind = "raw_request"
)

// RouteGuardAuthorization is an opaque proof returned by the Host guard
// authorizer. Other packages may relay it but cannot manufacture a valid proof.
type RouteGuardAuthorization struct {
	mode          RequestAuthorityMode
	guardKind     RequestGuardKind
	planRevision  uint64
	stepIndex     int
	step          RouteExecutionStep
	requestDigest [sha256.Size]byte
	issued        bool
}

// ResolvedRequestAuthority is the transport-safe view of a private invocation
// stamp after its exact plan, step, and request bindings have been rechecked.
type ResolvedRequestAuthority struct {
	Mode      RequestAuthorityMode
	GuardKind RequestGuardKind
}

type routeInvocationAuthority struct {
	authorization RouteGuardAuthorization
	planRevision  uint64
	stepIndex     int
	step          RouteExecutionStep
	stage         InvocationStage
	commit        *RouteCommitObserver
	requestDigest [sha256.Size]byte
	issued        bool
}

func (i RouteInvocation) RequestAuthority() (ResolvedRequestAuthority, bool) {
	stamp := i.authority
	if !stamp.issued || !stamp.authorization.issued || stamp.planRevision != i.PlanRevision ||
		stamp.stepIndex != i.StepIndex || stamp.authorization.planRevision != i.PlanRevision ||
		stamp.authorization.stepIndex != i.StepIndex || stamp.stage != i.Stage || stamp.commit != i.Commit ||
		!equalCoreGuardExecutionStep(stamp.step, i.Step) ||
		!equalCoreGuardExecutionStep(stamp.authorization.step, i.Step) {
		return ResolvedRequestAuthority{}, false
	}
	digest := routeAuthorityRequestDigest(i.Request)
	if digest != stamp.requestDigest || digest != stamp.authorization.requestDigest {
		return ResolvedRequestAuthority{}, false
	}
	expectedKind, ok := routeRequestGuardKind(i.Step)
	if !ok || stamp.authorization.guardKind != expectedKind ||
		stamp.authorization.mode == RequestAuthorityRaw && expectedKind != RequestGuardRawRequest ||
		stamp.authorization.mode != RequestAuthorityRaw && expectedKind == RequestGuardRawRequest {
		return ResolvedRequestAuthority{}, false
	}
	return ResolvedRequestAuthority{
		Mode: stamp.authorization.mode, GuardKind: stamp.authorization.guardKind,
	}, true
}

func (i RouteInvocation) RawRequestAuthorized() bool {
	authority, ok := i.RequestAuthority()
	return ok && authority.Mode == RequestAuthorityRaw && authority.GuardKind == RequestGuardRawRequest
}

func authorizedRouteGuardAuthorization(
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	request DispatchRequest,
) (RouteGuardAuthorization, bool) {
	guardKind, ok := routeRequestGuardKind(step)
	if !ok || !exactRouteExecutionStepAt(plan, stepIndex, step) {
		return RouteGuardAuthorization{}, false
	}
	mode := RequestAuthorityFiltered
	if guardKind == RequestGuardRawRequest {
		mode = RequestAuthorityRaw
	}
	return RouteGuardAuthorization{
		mode: mode, guardKind: guardKind, planRevision: plan.Revision(), stepIndex: stepIndex,
		step:          cloneRouteExecutionSteps([]RouteExecutionStep{step})[0],
		requestDigest: routeAuthorityRequestDigest(request), issued: true,
	}, true
}

func legacyFilteredRouteGuardAuthorization(
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	request DispatchRequest,
) (RouteGuardAuthorization, bool) {
	guardKind, ok := routeRequestGuardKind(step)
	if !ok || guardKind != RequestGuardHost || !exactRouteExecutionStepAt(plan, stepIndex, step) {
		return RouteGuardAuthorization{}, false
	}
	return RouteGuardAuthorization{
		mode: RequestAuthorityFiltered, guardKind: guardKind, planRevision: plan.Revision(), stepIndex: stepIndex,
		step:          cloneRouteExecutionSteps([]RouteExecutionStep{step})[0],
		requestDigest: routeAuthorityRequestDigest(request), issued: true,
	}, true
}

func newRouteInvocationAuthority(
	plan RouteExecutionPlan,
	stepIndex int,
	step RouteExecutionStep,
	request DispatchRequest,
	authorization RouteGuardAuthorization,
	stage InvocationStage,
	commit *RouteCommitObserver,
) (routeInvocationAuthority, bool) {
	if !authorization.issued || authorization.planRevision != plan.Revision() ||
		authorization.stepIndex != stepIndex || !exactRouteExecutionStepAt(plan, stepIndex, step) ||
		!equalCoreGuardExecutionStep(authorization.step, step) ||
		authorization.requestDigest != routeAuthorityRequestDigest(request) {
		return routeInvocationAuthority{}, false
	}
	guardKind, ok := routeRequestGuardKind(step)
	if !ok || authorization.guardKind != guardKind ||
		authorization.mode == RequestAuthorityRaw && guardKind != RequestGuardRawRequest ||
		authorization.mode != RequestAuthorityRaw && guardKind == RequestGuardRawRequest {
		return routeInvocationAuthority{}, false
	}
	return routeInvocationAuthority{
		authorization: authorization, planRevision: plan.Revision(), stepIndex: stepIndex,
		step:  cloneRouteExecutionSteps([]RouteExecutionStep{step})[0],
		stage: stage, commit: commit,
		requestDigest: routeAuthorityRequestDigest(request), issued: true,
	}, true
}

func exactRouteExecutionStepAt(plan RouteExecutionPlan, index int, step RouteExecutionStep) bool {
	chain := plan.Chain()
	return index >= 0 && index < len(chain) && equalCoreGuardExecutionStep(chain[index], step)
}

func uniqueRouteExecutionStepIndex(plan RouteExecutionPlan, step RouteExecutionStep) (int, bool) {
	found := -1
	for index, candidate := range plan.Chain() {
		if !equalCoreGuardExecutionStep(candidate, step) {
			continue
		}
		if found >= 0 {
			return -1, false
		}
		found = index
	}
	return found, found >= 0
}

func routeRequestGuardKind(step RouteExecutionStep) (RequestGuardKind, bool) {
	if step.Provider.Kind != ProviderPlugin {
		return "", false
	}
	switch step.Guard {
	case extensionmanifest.GuardCorePublic, extensionmanifest.GuardCoreLogin,
		extensionmanifest.GuardCorePermission, extensionmanifest.GuardCoreGuest,
		extensionmanifest.GuardCoreInherit:
		return RequestGuardHost, equalPluginGuardBinding(step.PluginGuard, PluginGuardBinding{})
	case extensionmanifest.GuardCoreRaw:
		return RequestGuardRawRequest, equalPluginGuardBinding(step.PluginGuard, PluginGuardBinding{})
	}
	if !validPluginGuardBinding(step) {
		return "", false
	}
	if step.PluginGuard.Kind == "raw_request" {
		return RequestGuardRawRequest, true
	}
	if step.PluginGuard.Kind == "custom" {
		return RequestGuardCustom, true
	}
	return "", false
}

type routeAuthorityHeader struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type routeAuthorityEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type routeAuthorityPermission struct {
	Key     string `json:"key"`
	Allowed bool   `json:"allowed"`
}

func routeAuthorityRequestDigest(request DispatchRequest) [sha256.Size]byte {
	headerNames := make([]string, 0, len(request.Headers))
	for name := range request.Headers {
		headerNames = append(headerNames, name)
	}
	sort.Slice(headerNames, func(left, right int) bool {
		leftCanonical, rightCanonical := strings.ToLower(headerNames[left]), strings.ToLower(headerNames[right])
		if leftCanonical == rightCanonical {
			return headerNames[left] < headerNames[right]
		}
		return leftCanonical < rightCanonical
	})
	headers := make([]routeAuthorityHeader, 0, len(headerNames))
	for _, name := range headerNames {
		headers = append(headers, routeAuthorityHeader{Name: name, Values: append([]string(nil), request.Headers[name]...)})
	}
	params := sortedRouteAuthorityEntries(request.Params)
	permissionKeys := make([]string, 0, len(request.Permissions))
	for key := range request.Permissions {
		permissionKeys = append(permissionKeys, key)
	}
	sort.Strings(permissionKeys)
	permissions := make([]routeAuthorityPermission, 0, len(permissionKeys))
	for _, key := range permissionKeys {
		permissions = append(permissions, routeAuthorityPermission{Key: key, Allowed: request.Permissions[key]})
	}
	payload, _ := json.Marshal(struct {
		Method           string                     `json:"method"`
		Path             string                     `json:"path"`
		Query            string                     `json:"query"`
		Headers          []routeAuthorityHeader     `json:"headers"`
		Body             []byte                     `json:"body"`
		Params           []routeAuthorityEntry      `json:"params"`
		ActorID          int64                      `json:"actorId"`
		Authenticated    bool                       `json:"authenticated"`
		CredentialSource DispatchCredentialSource   `json:"credentialSource"`
		Permissions      []routeAuthorityPermission `json:"permissions"`
		ClientIP         string                     `json:"clientIp"`
	}{
		Method: request.Method, Path: request.Path, Query: request.Query, Headers: headers,
		Body: append([]byte(nil), request.Body...), Params: params, ActorID: request.ActorID,
		Authenticated: request.Authenticated, CredentialSource: request.CredentialSource,
		Permissions: permissions, ClientIP: request.ClientIP,
	})
	return sha256.Sum256(payload)
}

func sortedRouteAuthorityEntries(values map[string]string) []routeAuthorityEntry {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]routeAuthorityEntry, 0, len(keys))
	for _, key := range keys {
		result = append(result, routeAuthorityEntry{Key: key, Value: values[key]})
	}
	return result
}
