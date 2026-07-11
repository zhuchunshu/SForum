package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"
)

var ErrFrontendTrustUnavailable = errors.New("extensions: frontend trust is unavailable")

type FrontendExtensionReader interface {
	Get(context.Context, string) (Extension, error)
}

type FrontendTrustLifecycleStore interface {
	FrontendTrustStore
}

type FrontendReleaseManager interface {
	PlanAndQueue(context.Context, QueueWebReleaseInput) (WebReleaseQueueResult, error)
}

type FrontendActiveReleaseReader interface {
	ActiveWebRelease(context.Context) (WebRelease, error)
	WebRelease(context.Context, int64) (WebReleaseDetail, error)
}

type FrontendService struct {
	extensions FrontendExtensionReader
	trust      FrontendTrustLifecycleStore
	releases   FrontendReleaseManager
	active     FrontendActiveReleaseReader
	host       WebCompositionHost
	mu         sync.Mutex
}

func NewFrontendService(
	extensions FrontendExtensionReader,
	trust FrontendTrustLifecycleStore,
	releases FrontendReleaseManager,
	active FrontendActiveReleaseReader,
	host WebCompositionHost,
) *FrontendService {
	return &FrontendService{extensions: extensions, trust: trust, releases: releases, active: active, host: host}
}

func (s *FrontendService) Frontend(ctx context.Context, actor identity.Actor, extensionID string) (FrontendStatus, error) {
	// 前端信任状态：插件/主题管理或只读查看均可读取。
	if !canViewExtensions(actor) && !canManagePlugins(actor) && !canManageThemes(actor) {
		return FrontendStatus{}, identity.ErrPermissionDenied
	}
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return FrontendStatus{}, err
	}
	return s.frontendStatus(ctx, extension)
}

func (s *FrontendService) Grant(
	ctx context.Context,
	actor identity.Actor,
	extensionID string,
	input GrantFrontendInput,
) (ExtensionOperation, error) {
	if !actor.IsSuperAdmin() {
		return ExtensionOperation{}, identity.ErrPermissionDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return ExtensionOperation{}, err
	}
	admin, err := interactiveAdminFrontend(extension)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if input.PackageDigest != extension.PackageDigest {
		return ExtensionOperation{}, fmt.Errorf("%w: requested digest does not match installed extension", ErrWebReleasePackageChanged)
	}
	if err := verifyPlannedPackage(extension); err != nil {
		return ExtensionOperation{}, err
	}
	dependencies, err := s.inspectFrontend(extension, admin)
	if err != nil {
		return ExtensionOperation{}, err
	}
	contributions, err := trustedComponentContributions(extension.Manifest, admin.Components)
	if err != nil {
		return ExtensionOperation{}, err
	}
	points := make([]string, 0, len(contributions))
	componentIDs := make([]string, 0, len(contributions))
	for _, contribution := range contributions {
		points = append(points, contribution.Point)
		var payload AdminComponentContributionPayload
		if err := jsonUnmarshal(contribution.Payload, &payload); err != nil {
			return ExtensionOperation{}, err
		}
		componentIDs = append(componentIDs, payload.Component)
	}
	points = sortedUniqueStrings(points)
	componentIDs = sortedUniqueStrings(componentIDs)
	if _, err := s.trust.CreateFrontendGrant(ctx, FrontendTrustGrantInput{
		ExtensionID:        extension.ID,
		ExtensionVersion:   extension.Version,
		PackageDigest:      extension.PackageDigest,
		APIVersion:         admin.APIVersion,
		ContributionPoints: points,
		ComponentIDs:       componentIDs,
		GrantedByUserID:    actor.ID,
	}); err != nil {
		return ExtensionOperation{}, err
	}
	status := frontendStatusForGrant(extension, admin, FrontendTrustTrusted, dependencies)
	operation := ExtensionOperation{Extension: extension, Frontend: &status}
	if extension.Status != StatusEnabled {
		return operation, nil
	}
	queued, err := s.releases.PlanAndQueue(ctx, QueueWebReleaseInput{Plan: PlanWebReleaseInput{
		TriggerKind:        WebReleaseTriggerTrustGrant,
		TriggerExtensionID: extension.ID,
		RequestedBy:        actor.ID,
		ReloadMode:         WebReleaseReloadPrompt,
	}})
	if err != nil {
		return ExtensionOperation{}, err
	}
	operation.Queued = true
	operation.WebRelease = webReleaseSummary(queued.Release)
	return operation, nil
}

func (s *FrontendService) Revoke(ctx context.Context, actor identity.Actor, extensionID string) (ExtensionOperation, error) {
	if !actor.IsSuperAdmin() {
		return ExtensionOperation{}, identity.ErrPermissionDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return ExtensionOperation{}, err
	}
	admin, err := interactiveAdminFrontend(extension)
	if err != nil {
		return ExtensionOperation{}, err
	}
	grant, err := s.trust.FrontendGrant(ctx, extension.ID, extension.Version, extension.PackageDigest)
	if err != nil {
		return ExtensionOperation{}, err
	}
	grant, err = s.trust.RequestFrontendRevocation(ctx, FrontendRevocationInput{
		ExtensionID:       grant.ExtensionID,
		ExtensionVersion:  grant.ExtensionVersion,
		PackageDigest:     grant.PackageDigest,
		RequestedByUserID: actor.ID,
	})
	if err != nil {
		return ExtensionOperation{}, err
	}
	status := frontendStatusForGrant(extension, admin, FrontendTrustRevocationPending, DependencySummary{})
	operation := ExtensionOperation{Extension: extension, Frontend: &status}
	contains, err := s.activeContainsGrant(ctx, grant)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if !contains {
		if _, err := s.trust.FinalizeFrontendRevocation(ctx, FrontendFinalizeInput{
			ExtensionID: grant.ExtensionID, ExtensionVersion: grant.ExtensionVersion, PackageDigest: grant.PackageDigest,
		}); err != nil {
			return ExtensionOperation{}, err
		}
		operation.Frontend.TrustState = FrontendTrustRevoked
		return operation, nil
	}
	queued, err := s.releases.PlanAndQueue(ctx, QueueWebReleaseInput{Plan: PlanWebReleaseInput{
		TriggerKind:        WebReleaseTriggerTrustRevoke,
		TriggerExtensionID: extension.ID,
		RequestedBy:        actor.ID,
		ReloadMode:         WebReleaseReloadForce,
	}})
	if err != nil {
		return ExtensionOperation{}, err
	}
	operation.Queued = true
	operation.WebRelease = webReleaseSummary(queued.Release)
	return operation, nil
}

func (s *FrontendService) RestoreDefaults(ctx context.Context, actor identity.Actor) (ExtensionOperation, error) {
	if !actor.IsSuperAdmin() {
		return ExtensionOperation{}, identity.ErrPermissionDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	grants, err := s.trust.RequestAllFrontendRevocations(ctx, actor.ID)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if len(grants) == 0 {
		return ExtensionOperation{}, nil
	}
	contains, err := s.activeContainsAnyGrant(ctx, grants)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if !contains {
		for _, grant := range grants {
			if _, err := s.trust.FinalizeFrontendRevocation(ctx, FrontendFinalizeInput{
				ExtensionID: grant.ExtensionID, ExtensionVersion: grant.ExtensionVersion, PackageDigest: grant.PackageDigest,
			}); err != nil {
				return ExtensionOperation{}, err
			}
		}
		return ExtensionOperation{}, nil
	}
	queued, err := s.releases.PlanAndQueue(ctx, QueueWebReleaseInput{Plan: PlanWebReleaseInput{
		TriggerKind: WebReleaseTriggerRestore,
		RequestedBy: actor.ID,
		ReloadMode:  WebReleaseReloadForce,
	}})
	if err != nil {
		return ExtensionOperation{}, err
	}
	return ExtensionOperation{Queued: true, WebRelease: webReleaseSummary(queued.Release)}, nil
}

func (s *FrontendService) ValidateRollback(ctx context.Context, target WebReleaseDetail) error {
	if s == nil || s.extensions == nil || s.trust == nil {
		return ErrWebReleaseRollbackIneligible
	}
	for _, snapshot := range target.Extensions {
		extension, err := s.extensions.Get(ctx, snapshot.ExtensionID)
		if err != nil {
			return fmt.Errorf("%w: extension %s is unavailable", ErrWebReleaseRollbackIneligible, snapshot.ExtensionID)
		}
		if extension.Source == SourceBuiltin && extension.IsSystem && !extension.IsDeletable {
			continue
		}
		grant, err := s.trust.FrontendGrant(ctx, snapshot.ExtensionID, snapshot.ExtensionVersion, snapshot.PackageDigest)
		if err != nil || grant.RevocationRequestedAt != nil || grant.RevokedAt != nil {
			return fmt.Errorf("%w: extension %s no longer has exact frontend trust", ErrWebReleaseRollbackIneligible, snapshot.ExtensionID)
		}
	}
	return nil
}

func (s *FrontendService) extension(ctx context.Context, extensionID string) (Extension, error) {
	if s == nil || s.extensions == nil || s.trust == nil || s.releases == nil || s.active == nil {
		return Extension{}, ErrFrontendTrustUnavailable
	}
	return s.extensions.Get(ctx, normalizeID(extensionID))
}

func (s *FrontendService) frontendStatus(ctx context.Context, extension Extension) (FrontendStatus, error) {
	admin := extension.Manifest.Frontend.Admin
	if admin == nil {
		return FrontendStatus{ExtensionID: extension.ID, TrustState: FrontendTrustNone}, nil
	}
	dependencies, err := s.inspectFrontend(extension, admin)
	if err != nil {
		return FrontendStatus{}, err
	}
	if extension.Source == SourceBuiltin && extension.IsSystem && !extension.IsDeletable {
		return frontendStatusForGrant(extension, admin, FrontendTrustSourceTrusted, dependencies), nil
	}
	grants, err := s.trust.LiveFrontendGrants(ctx, extension.ID)
	if err != nil {
		return FrontendStatus{}, err
	}
	state := FrontendTrustRequired
	for _, grant := range grants {
		if grant.ExtensionVersion == extension.Version && grant.PackageDigest == extension.PackageDigest {
			state = FrontendTrustTrusted
			if grant.RevocationRequestedAt != nil {
				state = FrontendTrustRevocationPending
			}
			return frontendStatusForGrant(extension, admin, state, dependencies), nil
		}
	}
	if len(grants) > 0 {
		state = FrontendTrustInvalidated
	}
	return frontendStatusForGrant(extension, admin, state, dependencies), nil
}

func (s *FrontendService) inspectFrontend(extension Extension, admin *ManifestAdminFrontend) (DependencySummary, error) {
	return extensionpackage.InspectAdminFrontend(extensionpackage.FrontendInspectInput{
		PackageRoot: extension.PackagePath,
		Root:        admin.Root,
		Components:  admin.Components,
		Locales:     admin.Locales,
		HostPeers:   s.host.HostPeers,
	})
}

func (s *FrontendService) activeContainsGrant(ctx context.Context, grant FrontendTrustGrant) (bool, error) {
	return s.activeContainsAnyGrant(ctx, []FrontendTrustGrant{grant})
}

func (s *FrontendService) activeContainsAnyGrant(ctx context.Context, grants []FrontendTrustGrant) (bool, error) {
	active, err := s.active.ActiveWebRelease(ctx)
	if errors.Is(err, ErrWebReleaseNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	detail, err := s.active.WebRelease(ctx, active.ID)
	if err != nil {
		return false, err
	}
	for _, extension := range detail.Extensions {
		for _, grant := range grants {
			if extension.ExtensionID == grant.ExtensionID &&
				extension.ExtensionVersion == grant.ExtensionVersion &&
				extension.PackageDigest == grant.PackageDigest {
				return true, nil
			}
		}
	}
	return false, nil
}

func interactiveAdminFrontend(extension Extension) (*ManifestAdminFrontend, error) {
	if extension.Type != TypePlugin || extension.Source != SourceUploaded || extension.Manifest.Frontend.Admin == nil {
		return nil, ErrFrontendTrustUnavailable
	}
	return extension.Manifest.Frontend.Admin, nil
}

func frontendStatusForGrant(
	extension Extension,
	admin *ManifestAdminFrontend,
	state string,
	dependencies DependencySummary,
) FrontendStatus {
	declaration := *admin
	declaration.Components = cloneStringMap(admin.Components)
	declaration.Locales = cloneStringMap(admin.Locales)
	return FrontendStatus{
		ExtensionID:  extension.ID,
		Declaration:  &declaration,
		TrustState:   state,
		Digest:       extension.PackageDigest,
		Dependencies: dependencies,
	}
}

func webReleaseSummary(release WebRelease) *WebReleaseSummary {
	return &WebReleaseSummary{
		ID:                 release.ID,
		Status:             release.Status,
		CompositionHash:    release.CompositionHash,
		ReloadMode:         release.ReloadMode,
		TriggerKind:        release.TriggerKind,
		TriggerExtensionID: release.TriggerExtensionID,
		PublicReason:       release.PublicReason,
		PublicMessage:      release.PublicMessage,
		BuildLog:           release.BuildLog,
	}
}

func sortedUniqueStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return slices.Compact(result)
}

func jsonUnmarshal(body []byte, target any) error {
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode trusted frontend contribution: %w", err)
	}
	return nil
}
