package extensions

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestWebReleaseServiceReusesOnlyCompatibleLiveIntent(t *testing.T) {
	plan := webReleaseServicePlanFixture()
	active := WebRelease{ID: 10, Status: WebReleaseActive}
	previousID := active.ID
	existing := WebRelease{
		ID:                 11,
		Status:             WebReleaseBuilding,
		TriggerKind:        WebReleaseTriggerPluginEnable,
		TriggerExtensionID: "demo.plugin",
		CompositionHash:    plan.Hash,
		ReloadMode:         WebReleaseReloadPrompt,
		PreviousReleaseID:  &previousID,
	}
	effects := []WebReleaseEffectInput{
		{ExtensionID: "zeta.plugin", PreviousStatus: StatusDisabled, TargetStatus: StatusEnabled},
		{ExtensionID: "alpha.plugin", PreviousStatus: StatusDisabled, TargetStatus: StatusEnabled},
	}
	store := &fakeWebReleaseTransactionalStore{
		active: active,
		live: []WebReleaseDetail{{
			WebRelease: existing,
			Effects: []WebReleaseExtensionEffect{
				{ExtensionID: "alpha.plugin", PreviousStatus: StatusDisabled, TargetStatus: StatusEnabled},
				{ExtensionID: "zeta.plugin", PreviousStatus: StatusDisabled, TargetStatus: StatusEnabled},
			},
		}},
	}
	tx := &fakeWebReleaseServiceTx{calls: &store.calls}
	enqueuer := &fakeWebReleaseBuildEnqueuer{calls: &store.calls}
	service := NewWebReleaseService(
		&fakeWebReleaseCompositionPlanner{planned: plan},
		&fakeWebReleaseTxBeginner{tx: tx},
		store,
		enqueuer,
	)

	result, err := service.PlanAndQueue(context.Background(), QueueWebReleaseInput{
		Plan: PlanWebReleaseInput{
			TriggerKind:        WebReleaseTriggerPluginEnable,
			TriggerExtensionID: "demo.plugin",
			RequestedBy:        7,
			ReloadMode:         WebReleaseReloadPrompt,
		},
		Effects: effects,
	})
	if err != nil {
		t.Fatalf("plan and queue: %v", err)
	}
	if result.Created || result.Release.ID != existing.ID {
		t.Fatalf("compatible release was not reused: %#v", result)
	}
	if store.createCalls != 0 || enqueuer.count != 0 {
		t.Fatalf("compatible intent created duplicate work: create=%d enqueue=%d", store.createCalls, enqueuer.count)
	}
	if !tx.committed || tx.rolledBack {
		t.Fatalf("reuse transaction did not commit cleanly: %#v", tx)
	}
	wantOrder := []string{"lock", "active", "live", "commit"}
	if !slices.Equal(store.calls, wantOrder) {
		t.Fatalf("unexpected transaction order: got=%#v want=%#v", store.calls, wantOrder)
	}
}

func TestWebReleaseServiceDoesNotMergeDifferentSafetyOrEffects(t *testing.T) {
	for _, test := range []struct {
		name     string
		reload   string
		effects  []WebReleaseEffectInput
		existing WebReleaseDetail
	}{
		{
			name:    "force does not reuse prompt",
			reload:  WebReleaseReloadForce,
			effects: []WebReleaseEffectInput{{ExtensionID: "demo.plugin", PreviousStatus: StatusEnabled, TargetStatus: StatusDisabled}},
			existing: webReleaseLiveIntentFixture(WebReleaseReloadPrompt, []WebReleaseExtensionEffect{{
				ExtensionID: "demo.plugin", PreviousStatus: StatusEnabled, TargetStatus: StatusDisabled,
			}}),
		},
		{
			name:    "different effects are distinct",
			reload:  WebReleaseReloadPrompt,
			effects: []WebReleaseEffectInput{{ExtensionID: "demo.plugin", PreviousStatus: StatusDisabled, TargetStatus: StatusEnabled}},
			existing: webReleaseLiveIntentFixture(WebReleaseReloadPrompt, []WebReleaseExtensionEffect{{
				ExtensionID: "demo.plugin", PreviousStatus: StatusEnabled, TargetStatus: StatusDisabled,
			}}),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := webReleaseServicePlanFixture()
			test.existing.CompositionHash = plan.Hash
			active := WebRelease{ID: 10, Status: WebReleaseActive}
			previousID := active.ID
			test.existing.PreviousReleaseID = &previousID
			store := &fakeWebReleaseTransactionalStore{
				active:  active,
				live:    []WebReleaseDetail{test.existing},
				created: WebRelease{ID: 12, Status: WebReleaseQueued},
			}
			tx := &fakeWebReleaseServiceTx{calls: &store.calls}
			enqueuer := &fakeWebReleaseBuildEnqueuer{calls: &store.calls}
			service := NewWebReleaseService(
				&fakeWebReleaseCompositionPlanner{planned: plan},
				&fakeWebReleaseTxBeginner{tx: tx},
				store,
				enqueuer,
			)

			result, err := service.PlanAndQueue(context.Background(), QueueWebReleaseInput{
				Plan: PlanWebReleaseInput{
					TriggerKind:        WebReleaseTriggerPluginDisable,
					TriggerExtensionID: "demo.plugin",
					ReloadMode:         test.reload,
				},
				Effects: test.effects,
			})
			if err != nil {
				t.Fatalf("plan and queue: %v", err)
			}
			if !result.Created || result.Release.ID != 12 || store.createCalls != 1 || enqueuer.count != 1 {
				t.Fatalf("distinct intent was not queued: result=%#v store=%#v", result, store)
			}
			wantOrder := []string{"lock", "active", "live", "create", "enqueue", "commit"}
			if !slices.Equal(store.calls, wantOrder) {
				t.Fatalf("unexpected transaction order: got=%#v want=%#v", store.calls, wantOrder)
			}
		})
	}
}

func TestWebReleaseServiceRollsBackReleaseWhenEnqueueFails(t *testing.T) {
	plan := webReleaseServicePlanFixture()
	store := &fakeWebReleaseTransactionalStore{activeErr: ErrWebReleaseNotFound, created: WebRelease{ID: 12, Status: WebReleaseQueued}}
	tx := &fakeWebReleaseServiceTx{calls: &store.calls}
	enqueueErr := errors.New("river unavailable")
	service := NewWebReleaseService(
		&fakeWebReleaseCompositionPlanner{planned: plan},
		&fakeWebReleaseTxBeginner{tx: tx},
		store,
		&fakeWebReleaseBuildEnqueuer{calls: &store.calls, err: enqueueErr},
	)

	_, err := service.PlanAndQueue(context.Background(), QueueWebReleaseInput{
		Plan: PlanWebReleaseInput{TriggerKind: WebReleaseTriggerRebuild},
	})
	if !errors.Is(err, enqueueErr) {
		t.Fatalf("expected enqueue error, got %v", err)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("enqueue failure did not roll back: %#v", tx)
	}
}

func webReleaseServicePlanFixture() PlannedWebRelease {
	composition := WebComposition{
		Theme: WebThemeSnapshot{
			ExtensionID:   DefaultThemeID,
			Version:       "1.0.0",
			LayerPath:     "/tmp/layer",
			PackageDigest: fmt.Sprintf("%064d", 1),
		},
		Extensions: []WebExtensionSnapshot{},
		WebSource:  "source",
		WebLock:    "lock",
		SDKVersion: 1,
		BunVersion: "1.3.0",
		Contract:   1,
	}
	snapshot := []byte(`{"theme":{"extensionId":"sforum.default-theme","version":"1.0.0","layerPath":"/tmp/layer","packageDigest":"0000000000000000000000000000000000000000000000000000000000000001"},"extensions":[],"webSource":"source","webLock":"lock","sdkVersion":1,"bunVersion":"1.3.0","contract":1}`)
	digest := sha256.Sum256(snapshot)
	return PlannedWebRelease{Composition: composition, Snapshot: snapshot, Hash: fmt.Sprintf("%x", digest)}
}

func webReleaseLiveIntentFixture(reload string, effects []WebReleaseExtensionEffect) WebReleaseDetail {
	return WebReleaseDetail{
		WebRelease: WebRelease{
			ID:                 11,
			Status:             WebReleaseBuilding,
			TriggerKind:        WebReleaseTriggerPluginDisable,
			TriggerExtensionID: "demo.plugin",
			ReloadMode:         reload,
		},
		Effects: effects,
	}
}

type fakeWebReleaseCompositionPlanner struct {
	planned PlannedWebRelease
	err     error
}

func (p *fakeWebReleaseCompositionPlanner) Plan(context.Context, PlanWebReleaseInput) (PlannedWebRelease, error) {
	return p.planned, p.err
}

type fakeWebReleaseTxBeginner struct {
	tx  pgx.Tx
	err error
}

func (b *fakeWebReleaseTxBeginner) Begin(context.Context) (pgx.Tx, error) {
	return b.tx, b.err
}

type fakeWebReleaseTransactionalStore struct {
	calls       []string
	active      WebRelease
	activeErr   error
	live        []WebReleaseDetail
	liveErr     error
	created     WebRelease
	createErr   error
	createCalls int
}

func (s *fakeWebReleaseTransactionalStore) ActiveWebReleaseTx(context.Context, pgx.Tx) (WebRelease, error) {
	s.calls = append(s.calls, "active")
	return s.active, s.activeErr
}

func (s *fakeWebReleaseTransactionalStore) LiveWebReleasesByCompositionTx(context.Context, pgx.Tx, string) ([]WebReleaseDetail, error) {
	s.calls = append(s.calls, "live")
	return append([]WebReleaseDetail(nil), s.live...), s.liveErr
}

func (s *fakeWebReleaseTransactionalStore) CreateWebReleaseTx(_ context.Context, _ pgx.Tx, _ WebReleaseCreateInput) (WebRelease, error) {
	s.calls = append(s.calls, "create")
	s.createCalls++
	return s.created, s.createErr
}

type fakeWebReleaseBuildEnqueuer struct {
	calls *[]string
	count int
	err   error
}

func (e *fakeWebReleaseBuildEnqueuer) EnqueueWebReleaseBuildTx(_ context.Context, _ pgx.Tx, _ int64) error {
	*e.calls = append(*e.calls, "enqueue")
	e.count++
	return e.err
}

type fakeWebReleaseServiceTx struct {
	pgx.Tx
	calls      *[]string
	committed  bool
	rolledBack bool
}

func (tx *fakeWebReleaseServiceTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if sql == "SELECT pg_advisory_xact_lock($1)" {
		*tx.calls = append(*tx.calls, "lock")
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (tx *fakeWebReleaseServiceTx) Commit(context.Context) error {
	tx.committed = true
	*tx.calls = append(*tx.calls, "commit")
	return nil
}

func (tx *fakeWebReleaseServiceTx) Rollback(context.Context) error {
	if !tx.committed {
		tx.rolledBack = true
	}
	return nil
}
