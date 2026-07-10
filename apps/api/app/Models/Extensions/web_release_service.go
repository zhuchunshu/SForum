package extensions

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type WebReleaseCompositionPlanner interface {
	Plan(context.Context, PlanWebReleaseInput) (PlannedWebRelease, error)
}

type WebReleaseTxBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type WebReleaseTransactionalStore interface {
	ActiveWebReleaseTx(context.Context, pgx.Tx) (WebRelease, error)
	LiveWebReleasesByCompositionTx(context.Context, pgx.Tx, string) ([]WebReleaseDetail, error)
	CreateWebReleaseTx(context.Context, pgx.Tx, WebReleaseCreateInput) (WebRelease, error)
}

type WebReleaseBuildEnqueuer interface {
	EnqueueWebReleaseBuildTx(context.Context, pgx.Tx, int64) error
}

type QueueWebReleaseInput struct {
	Plan    PlanWebReleaseInput
	Effects []WebReleaseEffectInput
}

type WebReleaseQueueResult struct {
	Release WebRelease
	Created bool
}

type WebReleaseService struct {
	planner  WebReleaseCompositionPlanner
	tx       WebReleaseTxBeginner
	store    WebReleaseTransactionalStore
	enqueuer WebReleaseBuildEnqueuer
}

func NewWebReleaseService(
	planner WebReleaseCompositionPlanner,
	tx WebReleaseTxBeginner,
	store WebReleaseTransactionalStore,
	enqueuer WebReleaseBuildEnqueuer,
) *WebReleaseService {
	return &WebReleaseService{planner: planner, tx: tx, store: store, enqueuer: enqueuer}
}

func (s *WebReleaseService) PlanAndQueue(ctx context.Context, input QueueWebReleaseInput) (WebReleaseQueueResult, error) {
	if s == nil || s.planner == nil || s.tx == nil || s.store == nil || s.enqueuer == nil {
		return WebReleaseQueueResult{}, fmt.Errorf("web release service dependencies are incomplete")
	}
	planned, err := s.planner.Plan(ctx, input.Plan)
	if err != nil {
		return WebReleaseQueueResult{}, err
	}
	effects, err := canonicalWebReleaseEffects(input.Effects)
	if err != nil {
		return WebReleaseQueueResult{}, err
	}
	reloadMode := normalizeWebReleaseReloadMode(input.Plan.ReloadMode)

	tx, err := s.tx.Begin(ctx)
	if err != nil {
		return WebReleaseQueueResult{}, fmt.Errorf("begin web release plan: %w", err)
	}
	defer tx.Rollback(ctx)
	lockKey, err := webReleaseAdvisoryKey(planned.Hash)
	if err != nil {
		return WebReleaseQueueResult{}, err
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey); err != nil {
		return WebReleaseQueueResult{}, fmt.Errorf("lock web release composition: %w", err)
	}

	var previousReleaseID *int64
	active, err := s.store.ActiveWebReleaseTx(ctx, tx)
	if err == nil {
		previousReleaseID = &active.ID
	} else if !errors.Is(err, ErrWebReleaseNotFound) {
		return WebReleaseQueueResult{}, err
	}
	live, err := s.store.LiveWebReleasesByCompositionTx(ctx, tx, planned.Hash)
	if err != nil {
		return WebReleaseQueueResult{}, err
	}
	for _, candidate := range live {
		if compatibleWebReleaseIntent(candidate, input.Plan, reloadMode, previousReleaseID, effects) {
			if err := tx.Commit(ctx); err != nil {
				return WebReleaseQueueResult{}, fmt.Errorf("commit reused web release plan: %w", err)
			}
			return WebReleaseQueueResult{Release: candidate.WebRelease}, nil
		}
	}

	createInput := webReleaseCreateInput(planned, input.Plan, reloadMode, previousReleaseID, effects)
	release, err := s.store.CreateWebReleaseTx(ctx, tx, createInput)
	if err != nil {
		return WebReleaseQueueResult{}, err
	}
	if err := s.enqueuer.EnqueueWebReleaseBuildTx(ctx, tx, release.ID); err != nil {
		return WebReleaseQueueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WebReleaseQueueResult{}, fmt.Errorf("commit web release plan: %w", err)
	}
	return WebReleaseQueueResult{Release: release, Created: true}, nil
}

func webReleaseCreateInput(
	planned PlannedWebRelease,
	plan PlanWebReleaseInput,
	reloadMode string,
	previousReleaseID *int64,
	effects []WebReleaseEffectInput,
) WebReleaseCreateInput {
	extensions := make([]WebReleaseExtensionInput, len(planned.Composition.Extensions))
	for index, item := range planned.Composition.Extensions {
		extensions[index] = WebReleaseExtensionInput{
			ExtensionID:       item.ExtensionID,
			ExtensionVersion:  item.Version,
			PackageDigest:     item.PackageDigest,
			FrontendRoot:      item.FrontendRoot,
			ComponentMap:      cloneStringMap(item.ComponentMap),
			APIVersion:        item.APIVersion,
			TrustedComponents: append([]ManifestContribution(nil), item.Contributions...),
			LocaleMap:         cloneStringMap(item.LocaleMap),
			LocaleMapDigest:   item.LocaleMapDigest,
			LockfileDigest:    item.Dependencies.LockDigest,
			SortOrder:         item.SortOrder,
		}
	}
	var requestedByUserID *int64
	if plan.RequestedBy > 0 {
		actorID := plan.RequestedBy
		requestedByUserID = &actorID
	}
	return WebReleaseCreateInput{
		TriggerKind:         plan.TriggerKind,
		TriggerExtensionID:  plan.TriggerExtensionID,
		CompositionHash:     planned.Hash,
		CompositionSnapshot: append([]byte(nil), planned.Snapshot...),
		ActiveThemeID:       planned.Composition.Theme.ExtensionID,
		ThemeVersion:        planned.Composition.Theme.Version,
		ThemeLayerPath:      planned.Composition.Theme.LayerPath,
		ThemePackageDigest:  planned.Composition.Theme.PackageDigest,
		ReloadMode:          reloadMode,
		PreviousReleaseID:   previousReleaseID,
		RequestedByUserID:   requestedByUserID,
		Extensions:          extensions,
		Effects:             append([]WebReleaseEffectInput(nil), effects...),
		Reason:              "web_release.queued",
	}
}

func compatibleWebReleaseIntent(
	candidate WebReleaseDetail,
	plan PlanWebReleaseInput,
	reloadMode string,
	previousReleaseID *int64,
	effects []WebReleaseEffectInput,
) bool {
	if candidate.TriggerKind != plan.TriggerKind ||
		candidate.TriggerExtensionID != plan.TriggerExtensionID ||
		!sameOptionalInt64(candidate.PreviousReleaseID, previousReleaseID) ||
		!compatibleReloadMode(candidate.ReloadMode, reloadMode) {
		return false
	}
	candidateEffects := make([]WebReleaseEffectInput, len(candidate.Effects))
	for index, item := range candidate.Effects {
		candidateEffects[index] = WebReleaseEffectInput{
			ExtensionID:    item.ExtensionID,
			PreviousStatus: item.PreviousStatus,
			TargetStatus:   item.TargetStatus,
		}
	}
	canonical, err := canonicalWebReleaseEffects(candidateEffects)
	return err == nil && slices.Equal(canonical, effects)
}

func canonicalWebReleaseEffects(effects []WebReleaseEffectInput) ([]WebReleaseEffectInput, error) {
	result := append([]WebReleaseEffectInput(nil), effects...)
	sort.Slice(result, func(i, j int) bool { return result[i].ExtensionID < result[j].ExtensionID })
	for index, effect := range result {
		if strings.TrimSpace(effect.ExtensionID) == "" ||
			!knownExtensionLifecycleStatus(effect.PreviousStatus) ||
			!knownExtensionLifecycleStatus(effect.TargetStatus) {
			return nil, fmt.Errorf("%w: invalid lifecycle effect", ErrWebReleaseInvalidComposition)
		}
		if index > 0 && result[index-1].ExtensionID == effect.ExtensionID {
			return nil, fmt.Errorf("%w: duplicate lifecycle effect for %s", ErrWebReleaseInvalidComposition, effect.ExtensionID)
		}
	}
	return result, nil
}

func knownExtensionLifecycleStatus(status string) bool {
	return status == StatusInstalled || status == StatusEnabled || status == StatusDisabled
}

func normalizeWebReleaseReloadMode(mode string) string {
	if strings.TrimSpace(mode) == WebReleaseReloadForce {
		return WebReleaseReloadForce
	}
	return WebReleaseReloadPrompt
}

func compatibleReloadMode(existing string, requested string) bool {
	return existing == requested || (existing == WebReleaseReloadForce && requested == WebReleaseReloadPrompt)
}

func sameOptionalInt64(left *int64, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func webReleaseAdvisoryKey(compositionHash string) (int64, error) {
	if len(compositionHash) != 64 {
		return 0, fmt.Errorf("%w: invalid composition hash", ErrWebReleaseInvalidComposition)
	}
	value, err := strconv.ParseUint(compositionHash[:16], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid composition hash", ErrWebReleaseInvalidComposition)
	}
	return int64(value), nil
}
