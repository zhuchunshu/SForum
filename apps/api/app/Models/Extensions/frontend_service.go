package extensions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	audit "github.com/zhuchunshu/sforum/apps/api/app/Support/Audit"
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
	auditor    audit.Writer
	challenges map[string]frontendTrustChallengeState
	mu         sync.Mutex
}

type frontendTrustChallengeState struct {
	ActorUserID int64
	Challenge   FrontendTrustChallenge
}

func (s *FrontendService) WithAuditor(writer audit.Writer) *FrontendService {
	s.auditor = writer
	return s
}

func (s *FrontendService) appendAudit(ctx context.Context, actor identity.Actor, action string, extension Extension, kind string) {
	if s == nil || s.auditor == nil {
		return
	}
	_ = s.auditor.Append(ctx, audit.Event{ActorUserID: actor.ID, Action: action, Metadata: map[string]any{
		"extensionId": extension.ID, "version": extension.Version, "adminFrontendDigest": extension.AdminFrontendDigest, "kind": kind,
	}})
}

func NewFrontendService(
	extensions FrontendExtensionReader,
	trust FrontendTrustLifecycleStore,
	releases FrontendReleaseManager,
	active FrontendActiveReleaseReader,
	host WebCompositionHost,
) *FrontendService {
	return &FrontendService{extensions: extensions, trust: trust, releases: releases, active: active, host: host, challenges: map[string]frontendTrustChallengeState{}}
}

func (s *FrontendService) Frontend(ctx context.Context, actor identity.Actor, extensionID string) (FrontendStatus, error) {
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return FrontendStatus{}, err
	}
	// 前端信任状态随设置页读取；具体授权/撤销仍只允许 super_admin。
	if !canViewExtensions(actor) && !canManagePlugins(actor) && !canManageThemes(actor) && !canManageExtensionSettings(actor, extension) {
		return FrontendStatus{}, identity.ErrPermissionDenied
	}
	return s.frontendStatus(ctx, extension)
}

func (s *FrontendService) Challenge(ctx context.Context, actor identity.Actor, extensionID string) (FrontendTrustChallenge, error) {
	if !actor.IsSuperAdmin() {
		return FrontendTrustChallenge{}, identity.ErrPermissionDenied
	}
	extension, err := s.extension(ctx, extensionID)
	if err != nil {
		return FrontendTrustChallenge{}, err
	}
	component := prebuiltSettingsComponent(extension)
	if component == nil || extension.Source != SourceUploaded {
		return FrontendTrustChallenge{}, ErrFrontendTrustUnavailable
	}
	random := make([]byte, 20)
	if _, err := rand.Read(random); err != nil {
		return FrontendTrustChallenge{}, err
	}
	challenge := FrontendTrustChallenge{
		ChallengeID: hex.EncodeToString(random[:16]),
		Code:        fmt.Sprintf("%06d", (uint32(random[16])<<16|uint32(random[17])<<8|uint32(random[18]))%1000000),
		ExtensionID: extension.ID, Version: extension.Version, Digest: extension.AdminFrontendDigest,
		APIVersion: component.APIVersion, ComponentID: component.ID, ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, item := range s.challenges {
		if time.Now().UTC().After(item.Challenge.ExpiresAt) {
			delete(s.challenges, id)
		}
	}
	s.challenges[challenge.ChallengeID] = frontendTrustChallengeState{ActorUserID: actor.ID, Challenge: challenge}
	return challenge, nil
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
	if component := prebuiltSettingsComponent(extension); component != nil {
		if extension.Source != SourceUploaded {
			return ExtensionOperation{}, ErrFrontendTrustUnavailable
		}
		if input.PackageDigest != extension.AdminFrontendDigest {
			return ExtensionOperation{}, fmt.Errorf("%w: requested digest does not match installed extension", ErrWebReleasePackageChanged)
		}
		if err := s.consumePrebuiltConfirmation(actor, extension, *component, input.Confirmation); err != nil {
			return ExtensionOperation{}, err
		}
		if err := verifyPlannedPackage(extension); err != nil {
			return ExtensionOperation{}, err
		}
		if _, err := s.trust.CreateFrontendGrant(ctx, FrontendTrustGrantInput{
			ExtensionID: extension.ID, ExtensionVersion: extension.Version,
			PackageDigest: extension.PackageDigest, AdminFrontendDigest: extension.AdminFrontendDigest,
			APIVersion: component.APIVersion, ContributionPoints: []string{"admin.extension.settings.component"},
			ComponentIDs: []string{component.ID}, GrantedByUserID: actor.ID,
		}); err != nil {
			return ExtensionOperation{}, err
		}
		s.appendAudit(ctx, actor, audit.ActionExtensionFrontendGrant, extension, AdminFrontendKindPrebuiltComponent)
		status := prebuiltFrontendStatus(extension, *component, FrontendTrustTrusted)
		return ExtensionOperation{Extension: extension, Frontend: &status}, nil
	}
	admin, err := interactiveAdminFrontend(extension)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if input.PackageDigest != extension.AdminFrontendDigest {
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
		ExtensionID:         extension.ID,
		ExtensionVersion:    extension.Version,
		PackageDigest:       extension.PackageDigest,
		AdminFrontendDigest: extension.AdminFrontendDigest,
		APIVersion:          admin.APIVersion,
		ContributionPoints:  points,
		ComponentIDs:        componentIDs,
		GrantedByUserID:     actor.ID,
	}); err != nil {
		return ExtensionOperation{}, err
	}
	s.appendAudit(ctx, actor, audit.ActionExtensionFrontendGrant, extension, AdminFrontendKindLegacyWebRelease)
	status := frontendStatusForGrant(extension, admin, FrontendTrustTrusted, dependencies)
	operation := ExtensionOperation{Extension: extension, Frontend: &status}
	if extension.Status != StatusEnabled {
		return operation, nil
	}
	active, err := s.activeContainsFrontendDigest(ctx, extension.ID, extension.AdminFrontendDigest)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if active {
		operation.Frontend.ArtifactActive = true
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
	component := prebuiltSettingsComponent(extension)
	admin := extension.Manifest.Frontend.Admin
	if component == nil {
		var err error
		admin, err = interactiveAdminFrontend(extension)
		if err != nil {
			return ExtensionOperation{}, err
		}
	}
	grant, err := s.trust.FrontendGrant(ctx, extension.ID, extension.Version, extension.AdminFrontendDigest)
	if err != nil {
		return ExtensionOperation{}, err
	}
	grant, err = s.trust.RequestFrontendRevocation(ctx, FrontendRevocationInput{
		ExtensionID:         grant.ExtensionID,
		ExtensionVersion:    grant.ExtensionVersion,
		PackageDigest:       grant.PackageDigest,
		AdminFrontendDigest: grant.AdminFrontendDigest,
		RequestedByUserID:   actor.ID,
	})
	if err != nil {
		return ExtensionOperation{}, err
	}
	kind := AdminFrontendKindLegacyWebRelease
	if component != nil {
		kind = AdminFrontendKindPrebuiltComponent
	}
	s.appendAudit(ctx, actor, audit.ActionExtensionFrontendRevoke, extension, kind)
	status := FrontendStatus{}
	if component != nil {
		status = prebuiltFrontendStatus(extension, *component, FrontendTrustRevocationPending)
	} else {
		status = frontendStatusForGrant(extension, admin, FrontendTrustRevocationPending, DependencySummary{})
	}
	operation := ExtensionOperation{Extension: extension, Frontend: &status}
	contains, err := s.activeContainsGrant(ctx, grant)
	if err != nil {
		return ExtensionOperation{}, err
	}
	if !contains {
		if _, err := s.trust.FinalizeFrontendRevocation(ctx, FrontendFinalizeInput{
			ExtensionID: grant.ExtensionID, ExtensionVersion: grant.ExtensionVersion, PackageDigest: grant.PackageDigest, AdminFrontendDigest: grant.AdminFrontendDigest,
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

// RevokeAllForExtension 吊销该扩展全部 live 前端信任（F2.4 升级 digest 变化时重审批）。
// 实现 TrustRevoker；不排队 Web Release——升级后状态回到 installed，运营重新启用/授权时再构图。
func (s *FrontendService) RevokeAllForExtension(ctx context.Context, extensionID string, actorUserID int64) error {
	if s == nil || s.trust == nil {
		return nil
	}
	grants, err := s.trust.LiveFrontendGrants(ctx, strings.TrimSpace(extensionID))
	if err != nil {
		return err
	}
	for _, grant := range grants {
		if grant.RevocationRequestedAt != nil {
			continue
		}
		if _, err := s.trust.RequestFrontendRevocation(ctx, FrontendRevocationInput{
			ExtensionID:         grant.ExtensionID,
			ExtensionVersion:    grant.ExtensionVersion,
			PackageDigest:       grant.PackageDigest,
			AdminFrontendDigest: grant.AdminFrontendDigest,
			RequestedByUserID:   actorUserID,
		}); err != nil {
			// 并发已吊销或无 live 行时忽略；其它错误向上抛。
			if errors.Is(err, ErrFrontendGrantNotFound) || errors.Is(err, ErrFrontendGrantStateConflict) {
				continue
			}
			return err
		}
	}
	return nil
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
				ExtensionID: grant.ExtensionID, ExtensionVersion: grant.ExtensionVersion, PackageDigest: grant.PackageDigest, AdminFrontendDigest: grant.AdminFrontendDigest,
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
		grant, err := s.trust.FrontendGrant(ctx, snapshot.ExtensionID, snapshot.ExtensionVersion, snapshot.AdminFrontendDigest)
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
	if component := prebuiltSettingsComponent(extension); component != nil {
		if extension.Source == SourceBuiltin && extension.IsSystem && !extension.IsDeletable {
			return prebuiltFrontendStatus(extension, *component, FrontendTrustSourceTrusted), nil
		}
		grants, err := s.trust.LiveFrontendGrants(ctx, extension.ID)
		if err != nil {
			return FrontendStatus{}, err
		}
		state := FrontendTrustRequired
		for _, grant := range grants {
			if grant.ExtensionVersion == extension.Version && grant.AdminFrontendDigest == extension.AdminFrontendDigest &&
				grant.APIVersion == component.APIVersion && slices.Contains(grant.ComponentIDs, component.ID) {
				state = FrontendTrustTrusted
				if grant.RevocationRequestedAt != nil {
					state = FrontendTrustRevocationPending
				}
				return prebuiltFrontendStatus(extension, *component, state), nil
			}
		}
		if len(grants) > 0 {
			state = FrontendTrustInvalidated
		}
		return prebuiltFrontendStatus(extension, *component, state), nil
	}
	admin := extension.Manifest.Frontend.Admin
	if admin == nil {
		return FrontendStatus{ExtensionID: extension.ID, Kind: AdminFrontendKindNone, TrustState: FrontendTrustNone}, nil
	}
	dependencies, err := s.inspectFrontend(extension, admin)
	if err != nil {
		return FrontendStatus{}, err
	}
	if extension.Source == SourceBuiltin && extension.IsSystem && !extension.IsDeletable {
		status := frontendStatusForGrant(extension, admin, FrontendTrustSourceTrusted, dependencies)
		status.ArtifactActive, _ = s.activeContainsFrontendDigest(ctx, extension.ID, extension.AdminFrontendDigest)
		return status, nil
	}
	grants, err := s.trust.LiveFrontendGrants(ctx, extension.ID)
	if err != nil {
		return FrontendStatus{}, err
	}
	state := FrontendTrustRequired
	for _, grant := range grants {
		if grant.ExtensionVersion == extension.Version && grant.AdminFrontendDigest == extension.AdminFrontendDigest {
			state = FrontendTrustTrusted
			if grant.RevocationRequestedAt != nil {
				state = FrontendTrustRevocationPending
			}
			status := frontendStatusForGrant(extension, admin, state, dependencies)
			status.ArtifactActive, _ = s.activeContainsFrontendDigest(ctx, extension.ID, extension.AdminFrontendDigest)
			return status, nil
		}
	}
	if len(grants) > 0 {
		state = FrontendTrustInvalidated
	}
	status := frontendStatusForGrant(extension, admin, state, dependencies)
	status.ArtifactActive, _ = s.activeContainsFrontendDigest(ctx, extension.ID, extension.AdminFrontendDigest)
	return status, nil
}

func (s *FrontendService) activeContainsFrontendDigest(ctx context.Context, extensionID, digest string) (bool, error) {
	if digest == "" {
		return false, nil
	}
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
	for _, item := range detail.Extensions {
		if item.ExtensionID == extensionID && item.AdminFrontendDigest == digest {
			return true, nil
		}
	}
	return false, nil
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
				extension.AdminFrontendDigest == grant.AdminFrontendDigest {
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

func prebuiltSettingsComponent(extension Extension) *SettingsComponent {
	component := extension.Manifest.SettingsDocument.UI.Component
	if extension.Manifest.SettingsDocument.UI.Mode != "component" || component == nil || component.Entry == "" {
		return nil
	}
	return component
}

func (s *FrontendService) consumePrebuiltConfirmation(actor identity.Actor, extension Extension, component SettingsComponent, confirmation *FrontendTrustConfirmation) error {
	if confirmation == nil || !confirmation.Acknowledged ||
		confirmation.ExtensionID != extension.ID || confirmation.Version != extension.Version ||
		confirmation.Digest != extension.AdminFrontendDigest || confirmation.APIVersion != component.APIVersion ||
		confirmation.ComponentID != component.ID || strings.TrimSpace(confirmation.Phrase) != extension.ID {
		return fmt.Errorf("%w: explicit component trust confirmation is required", ErrFrontendTrustUnavailable)
	}
	state, exists := s.challenges[confirmation.ChallengeID]
	delete(s.challenges, confirmation.ChallengeID)
	if !exists || state.ActorUserID != actor.ID || time.Now().UTC().After(state.Challenge.ExpiresAt) ||
		state.Challenge.Code != confirmation.Code || state.Challenge.ExtensionID != extension.ID ||
		state.Challenge.Version != extension.Version || state.Challenge.Digest != extension.AdminFrontendDigest ||
		state.Challenge.APIVersion != component.APIVersion || state.Challenge.ComponentID != component.ID {
		return fmt.Errorf("%w: confirmation challenge is invalid or expired", ErrFrontendTrustUnavailable)
	}
	return nil
}

func prebuiltFrontendStatus(extension Extension, component SettingsComponent, state string) FrontendStatus {
	copy := component
	return FrontendStatus{
		ExtensionID: extension.ID, Kind: AdminFrontendKindPrebuiltComponent,
		Component: &copy, TrustState: state, Digest: extension.AdminFrontendDigest,
		BuildRequired: false, Dependencies: DependencySummary{},
	}
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
		ExtensionID: extension.ID, Kind: AdminFrontendKindLegacyWebRelease,
		Declaration: &declaration, TrustState: state, Digest: extension.AdminFrontendDigest,
		BuildRequired: true, Dependencies: dependencies,
	}
}

// Asset 仅返回 manifest settings component 明确声明的 entry/css，绝不暴露任意包文件。
func (s *FrontendService) Asset(ctx context.Context, actor identity.Actor, extensionID, digest, assetName string) (FrontendAsset, error) {
	if s == nil || s.extensions == nil || s.trust == nil {
		return FrontendAsset{}, ErrFrontendTrustUnavailable
	}
	extension, err := s.extensions.Get(ctx, normalizeID(extensionID))
	if err != nil {
		return FrontendAsset{}, err
	}
	if !canManageExtensionSettings(actor, extension) {
		return FrontendAsset{}, identity.ErrPermissionDenied
	}
	component := prebuiltSettingsComponent(extension)
	if component == nil || digest == "" || digest != extension.AdminFrontendDigest {
		return FrontendAsset{}, ErrFrontendTrustUnavailable
	}
	if !(extension.Source == SourceBuiltin && extension.IsSystem && !extension.IsDeletable) {
		grant, err := s.trust.FrontendGrant(ctx, extension.ID, extension.Version, extension.AdminFrontendDigest)
		if err != nil || grant.RevocationRequestedAt != nil || grant.RevokedAt != nil ||
			grant.APIVersion != component.APIVersion || !slices.Contains(grant.ComponentIDs, component.ID) {
			return FrontendAsset{}, ErrFrontendTrustUnavailable
		}
	}
	currentDigest, err := ComputeAdminFrontendDigest(extension.Manifest, extension.PackagePath)
	if err != nil || currentDigest != extension.AdminFrontendDigest {
		return FrontendAsset{}, fmt.Errorf("%w: component bytes changed", ErrWebReleasePackageChanged)
	}
	relative := component.Entry
	contentType := "application/javascript; charset=utf-8"
	if assetName == "style" {
		relative = component.CSS
		contentType = "text/css; charset=utf-8"
	} else if assetName != "entry" {
		return FrontendAsset{}, ErrFrontendTrustUnavailable
	}
	if relative == "" {
		return FrontendAsset{}, ErrFrontendTrustUnavailable
	}
	target, info, err := resolveAdminAsset(extension.PackagePath, relative)
	if err != nil {
		return FrontendAsset{}, ErrFrontendTrustUnavailable
	}
	limit := int64(maxPrebuiltAdminModuleBytes)
	if assetName == "style" {
		limit = maxPrebuiltAdminCSSBytes
	}
	if info.Size() > limit {
		return FrontendAsset{}, ErrFrontendTrustUnavailable
	}
	body, err := os.ReadFile(target)
	if err != nil {
		return FrontendAsset{}, err
	}
	etag := sha256.Sum256(body)
	return FrontendAsset{Body: body, ContentType: contentType, ETag: fmt.Sprintf("\"%x\"", etag[:])}, nil
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
