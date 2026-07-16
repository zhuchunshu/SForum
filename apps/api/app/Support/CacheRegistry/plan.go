package cacheregistry

import (
	"reflect"
	"strings"

	"golang.org/x/text/language"
)

// Plan validates Host-derived isolation input against the frozen declaration.
// It returns metadata only and never contacts Redis or any declared provider.
func (r *Registry) Plan(request PlanRequest) (Plan, error) {
	if r == nil {
		return Plan{}, ErrInvalid
	}
	cacheID := strings.ToLower(strings.TrimSpace(request.CacheID))
	namespace := strings.ToLower(strings.TrimSpace(request.Namespace))
	if (cacheID == "") == (namespace == "") || cacheID != "" && !idPattern.MatchString(cacheID) ||
		namespace != "" && !idPattern.MatchString(namespace) {
		return Plan{}, ErrInvalid
	}
	state := r.load()
	var contribution Contribution
	var found bool
	if cacheID != "" {
		contribution, found = state.caches[cacheID]
	} else {
		contribution, found = state.namespaces[namespace]
	}
	if !found {
		return Plan{}, ErrNotFound
	}
	if err := r.requireStableAdmission(state, contribution.Artifact); err != nil {
		return Plan{}, err
	}
	actor, err := normalizeFingerprint(request.ActorFingerprint)
	if err != nil {
		return Plan{}, err
	}
	permission, err := normalizeFingerprint(request.PermissionFingerprint)
	if err != nil {
		return Plan{}, err
	}
	locale, err := normalizeLocale(request.LocaleFingerprint)
	if err != nil {
		return Plan{}, err
	}
	// Only the policy-relevant Host projection participates in isolation.
	switch contribution.Policy {
	case PolicyActor:
		if actor == "" {
			return Plan{}, ErrIsolationRequired
		}
		permission = ""
	case PolicyPermission:
		if permission == "" {
			return Plan{}, ErrIsolationRequired
		}
		actor = ""
	case PolicyPrivate, PolicyPublic:
		actor, permission = "", ""
	default:
		return Plan{}, ErrInvalid
	}
	if err := r.requireStableAdmission(state, contribution.Artifact); err != nil {
		return Plan{}, err
	}
	isolation := IsolationMetadata{
		CacheID: contribution.ID, Namespace: contribution.Namespace, Policy: contribution.Policy,
		Artifact: contribution.Artifact, ActorFingerprint: actor, PermissionFingerprint: permission,
		LocaleFingerprint: locale,
	}
	isolation.SegmentDigest = isolationDigest(contribution, actor, permission, locale)
	plan := Plan{
		SchemaVersion: SchemaVersion, Revision: state.revision, Digest: state.digest,
		SafeMode: state.safeMode, Cache: cloneContribution(contribution), Isolation: isolation,
	}
	if err := r.ValidatePlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// ValidatePlan is the post-provider fence for every cache operation. It rejects
// registry replacement, Safe Mode transitions, declaration drift, forged
// isolation metadata, and a runtime that drained while the backend was active.
func (r *Registry) ValidatePlan(plan Plan) error {
	state, contribution, err := r.validatePlanSnapshot(plan)
	if err != nil {
		return err
	}
	if err := r.requireStableAdmission(state, contribution.Artifact); err != nil {
		return err
	}
	return nil
}

// ValidateLeasedPlan validates the immutable declaration/snapshot portion of a
// plan while the caller holds the Host runtime's exact admission lease. A
// graceful drain closes new boolean admission but must not turn an already
// leased, successfully committed operation into a reported stale failure.
func (r *Registry) ValidateLeasedPlan(plan Plan) error {
	_, _, err := r.validatePlanSnapshot(plan)
	return err
}

func (r *Registry) validatePlanSnapshot(plan Plan) (*registryState, Contribution, error) {
	if r == nil || plan.SchemaVersion != SchemaVersion {
		return nil, Contribution{}, ErrPlanStale
	}
	state := r.load()
	if plan.Revision != state.revision || plan.Digest != state.digest || plan.SafeMode != state.safeMode {
		return nil, Contribution{}, ErrPlanStale
	}
	contribution, found := state.caches[plan.Cache.ID]
	if !found || !reflect.DeepEqual(plan.Cache, contribution) {
		return nil, Contribution{}, ErrPlanStale
	}
	actor, err := normalizeFingerprint(plan.Isolation.ActorFingerprint)
	if err != nil {
		return nil, Contribution{}, ErrPlanStale
	}
	permission, err := normalizeFingerprint(plan.Isolation.PermissionFingerprint)
	if err != nil {
		return nil, Contribution{}, ErrPlanStale
	}
	locale, err := normalizeLocale(plan.Isolation.LocaleFingerprint)
	if err != nil {
		return nil, Contribution{}, ErrPlanStale
	}
	switch contribution.Policy {
	case PolicyActor:
		if actor == "" || permission != "" {
			return nil, Contribution{}, ErrPlanStale
		}
	case PolicyPermission:
		if permission == "" || actor != "" {
			return nil, Contribution{}, ErrPlanStale
		}
	case PolicyPrivate, PolicyPublic:
		if actor != "" || permission != "" {
			return nil, Contribution{}, ErrPlanStale
		}
	default:
		return nil, Contribution{}, ErrPlanStale
	}
	expected := IsolationMetadata{
		CacheID: contribution.ID, Namespace: contribution.Namespace, Policy: contribution.Policy,
		Artifact: contribution.Artifact, ActorFingerprint: actor, PermissionFingerprint: permission,
		LocaleFingerprint: locale,
	}
	expected.SegmentDigest = isolationDigest(contribution, actor, permission, locale)
	if plan.Isolation != expected {
		return nil, Contribution{}, ErrPlanStale
	}
	return state, contribution, nil
}

func normalizeFingerprint(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) > maxFingerprintLength {
		return "", ErrInvalid
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return "", ErrInvalid
		}
	}
	return value, nil
}

func normalizeLocale(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return language.Und.String(), nil
	}
	if len(value) > maxLocaleLength {
		return "", ErrInvalid
	}
	tag, err := language.Parse(value)
	if err != nil {
		return "", ErrInvalid
	}
	return tag.String(), nil
}
