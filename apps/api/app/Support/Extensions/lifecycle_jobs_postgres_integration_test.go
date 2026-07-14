package extensionsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

func TestPostgresLifecycleBoundaryJobsCrashRestartCommitAndConcurrentReconcile(t *testing.T) {
	ctx, pool, journal, request, extensionID := newLifecycleJobPublicationIntegration(t)
	attachLifecycleJobManifest(&request.TargetExtension)
	manager := NewManager(ManagerConfig{})
	if err := manager.Start(ctx, request.TargetExtension); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background(), request.TargetExtension) })
	runtime, err := manager.ActiveRuntimeInstance(extensionID)
	if err != nil {
		t.Fatal(err)
	}
	request.TargetBinding = lifecycleHostBindingForTest(request.TargetExtension, runtime.Identity.InstanceID)
	if _, err := manager.BeginDrain(runtime.Identity); err != nil {
		t.Fatal(err)
	}
	if err := manager.WaitDrain(ctx, runtime.Identity); err != nil {
		t.Fatal(err)
	}
	schedules := supportjobs.NewPluginScheduleAdmissionRegistry()

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	stale := hostapi.PluginJobArgs{
		EnvelopeVersion: supportjobs.PluginJobEnvelopeVersion,
		ExtensionID:     extensionID, ExtensionVersion: "0.9.0",
		ArtifactDigest: strings.Repeat("c", 64), TrustGrantID: "grant:0.9.0",
		JobName: "demo.sync", JobContractVersion: "demo.job@1",
		PayloadSchemaID: "demo.payload", PayloadSchemaVersion: "0",
		Payload:    map[string]any{"privateCursor": "must-not-enter-evidence"},
		EnqueuedAt: time.Now().UTC(),
	}
	inserted, err := riverClient.Insert(ctx, stale, &river.InsertOpts{Queue: "default"})
	if err != nil || inserted == nil || inserted.Job == nil {
		t.Fatalf("insert stale River job: %#v, %v", inserted, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM river_job WHERE id = $1`, inserted.Job.ID)
	})

	coordinator := &hostapi.PluginJobLifecycleCoordinator{
		Store: hostapi.NewPostgresPluginJobLifecycleStore(pool, riverClient),
	}
	newBoundary := func() *PostgresLifecycleBoundaryJobs {
		return NewPostgresLifecycleBoundaryJobs(PostgresLifecycleBoundaryJobsConfig{
			Pool: pool, Runtime: manager, Schedules: schedules,
			Coordinator: coordinator, Trust: lifecycleBoundaryJobTrust{}, Journal: journal,
		})
	}
	boundary := newBoundary()
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	transaction, err := boundary.PrepareLifecycleJobPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleJobTransactionState(t, ctx, transaction, LifecycleBoundaryTransactionSource)
	if err := transaction.Publish(ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleJobTransactionState(t, ctx, transaction, LifecycleBoundaryTransactionTarget)
	if err := transaction.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	assertLifecycleJobTransactionState(t, ctx, transaction, LifecycleBoundaryTransactionSource)
	if err := transaction.Publish(ctx); err != nil {
		t.Fatal(err)
	}

	// Simulate a process restart before the shared marker commits. The new
	// transaction reconstructs target state from PostgreSQL, not a closure.
	restarted := newBoundary()
	recovered, err := restarted.PrepareLifecycleJobPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleJobTransactionState(t, ctx, recovered, LifecycleBoundaryTransactionTarget)
	if err := journal.CommitLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	if err := recovered.Restore(ctx); !errors.Is(err, ErrLifecycleBoundaryJobsCommitted) {
		t.Fatalf("post-marker restore error = %v", err)
	}
	// Simulate the sharp crash window: River committed the prepared cancel, but
	// the Host died before writing reconciliation_result. Recovery must prove the
	// old cancelled row against the durable expected plan instead of requiring a
	// newly computed plan with the same shape.
	record, err := loadLifecycleJobPublication(
		ctx, pool, request.OperationID, request.StepID, LifecycleBoundaryActivate, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	material, err := restarted.lifecycleJobPublicationMaterial(
		ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
	)
	if err != nil {
		t.Fatal(err)
	}
	crashedResult, err := coordinator.ReconcileExpected(ctx, material.Input, lifecycleJobExpectedPlan(record.Plan))
	if err != nil || !crashedResult.Committed {
		t.Fatalf("simulate River commit before evidence: %#v, %v", crashedResult, err)
	}

	const workers = 16
	start := make(chan struct{})
	errorsCh := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			errorsCh <- restarted.ReconcileCommittedLifecycleJobs(
				ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
			)
		}()
	}
	close(start)
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	job, err := riverClient.JobGet(ctx, inserted.Job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.State != rivertype.JobStateCancelled {
		t.Fatalf("stale job state = %q", job.State)
	}
	var publicationState, reconciliationState string
	var reconcileAttempt int
	var planDocument, resultDocument []byte
	if err := pool.QueryRow(ctx, `
		SELECT publication_state, reconciliation_state, reconciliation_attempt,
		       reconciliation_plan, reconciliation_result
		FROM extension_lifecycle_job_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID).Scan(
		&publicationState, &reconciliationState, &reconcileAttempt, &planDocument, &resultDocument,
	); err != nil {
		t.Fatal(err)
	}
	if publicationState != "target" || reconciliationState != "succeeded" || reconcileAttempt != 1 {
		t.Fatalf("publication evidence = %s/%s attempt %d", publicationState, reconciliationState, reconcileAttempt)
	}
	if !json.Valid(planDocument) || !json.Valid(resultDocument) {
		t.Fatal("reconciliation evidence is not JSON")
	}
	if strings.Contains(string(planDocument), "privateCursor") || strings.Contains(string(resultDocument), "privateCursor") {
		t.Fatal("durable lifecycle evidence leaked a River payload")
	}
	if err := newBoundary().ReconcileCommittedLifecycleJobs(
		ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
	); err != nil {
		t.Fatalf("restart reconciliation replay: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_job_publications
		SET reconciliation_result = jsonb_set(reconciliation_result, '{executions}', '[]'::jsonb)
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID); err != nil {
		t.Fatal(err)
	}
	if err := newBoundary().ReconcileCommittedLifecycleJobs(
		ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
	); !errors.Is(err, ErrLifecycleBoundaryJobsConflict) {
		t.Fatalf("corrupt retained evidence error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_job_publications
		SET reconciliation_result = $3::jsonb
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID, resultDocument); err != nil {
		t.Fatal(err)
	}
	if err := restarted.ResumeLifecycleJobs(
		ctx, request, LifecycleBoundaryJobsEnable, extensions.LifecycleRuntimeTarget,
	); err != nil {
		t.Fatalf("resume committed target admissions: %v", err)
	}
	jobLease, err := manager.AcquireRuntimeCall(ctx, runtime.Identity, RuntimeCallJob)
	if err != nil {
		t.Fatalf("committed direct enqueue admission: %v", err)
	}
	jobLease.Release()
	scheduleIdentity := supportjobs.PluginScheduleRuntimeIdentity{
		ExtensionID: extensionID, ExtensionVersion: request.TargetExtension.Version,
		ArtifactDigest: request.TargetExtension.PackageDigest, InstanceID: runtime.Identity.InstanceID,
	}
	_, triggerLease, err := schedules.AcquireTrigger(ctx, scheduleIdentity, "demo.schedule")
	if err != nil {
		t.Fatalf("committed schedule trigger admission: %v", err)
	}
	triggerLease.Release()
}

func TestPostgresLifecycleBoundaryJobsRejectsReconcileBeforeMarker(t *testing.T) {
	ctx, pool, journal, request, _ := newLifecycleJobPublicationIntegration(t)
	attachLifecycleJobManifest(&request.TargetExtension)
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	boundary := NewPostgresLifecycleBoundaryJobs(PostgresLifecycleBoundaryJobsConfig{
		Pool: pool,
		Coordinator: &hostapi.PluginJobLifecycleCoordinator{
			Store: hostapi.NewPostgresPluginJobLifecycleStore(pool, riverClient),
		},
		Trust: lifecycleBoundaryJobTrust{}, Journal: journal,
	})
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	transaction, err := boundary.PrepareLifecycleJobPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := boundary.ReconcileCommittedLifecycleJobs(
		ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
	); !errors.Is(err, ErrLifecycleBoundaryJobsUncommitted) {
		t.Fatalf("pre-marker reconciliation error = %v", err)
	}
	var state string
	if err := pool.QueryRow(ctx, `
		SELECT reconciliation_state
		FROM extension_lifecycle_job_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "pending" {
		t.Fatalf("pre-marker reconciliation state = %q", state)
	}
}

func TestPostgresLifecycleBoundaryJobsRejectsPostPrepareRowDriftWithoutMutation(t *testing.T) {
	ctx, pool, journal, request, extensionID := newLifecycleJobPublicationIntegration(t)
	attachLifecycleJobManifest(&request.TargetExtension)
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	stale := lifecycleBoundaryIntegrationArgs(
		extensionID, "0.9.0", strings.Repeat("c", 64), "grant:0.9.0", "0", "stale",
	)
	staleInsert, err := riverClient.Insert(ctx, stale, &river.InsertOpts{Queue: "default"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM river_job WHERE id = ANY($1)`, []int64{staleInsert.Job.ID})
	})
	coordinator := &hostapi.PluginJobLifecycleCoordinator{
		Store: hostapi.NewPostgresPluginJobLifecycleStore(pool, riverClient),
	}
	boundary := NewPostgresLifecycleBoundaryJobs(PostgresLifecycleBoundaryJobsConfig{
		Pool: pool, Coordinator: coordinator, Trust: lifecycleBoundaryJobTrust{}, Journal: journal,
	})
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	transaction, err := boundary.PrepareLifecycleJobPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	target := lifecycleBoundaryIntegrationArgs(
		extensionID, request.TargetExtension.Version, request.TargetExtension.PackageDigest,
		"grant:"+request.TargetExtension.Version, "1", "new-after-prepare",
	)
	driftInsert, err := riverClient.Insert(ctx, target, &river.InsertOpts{Queue: "default"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM river_job WHERE id = $1`, driftInsert.Job.ID)
	})

	err = boundary.ReconcileCommittedLifecycleJobs(
		ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
	)
	if !errors.Is(err, hostapi.ErrPluginJobLifecyclePlanDrift) {
		t.Fatalf("row drift error = %v", err)
	}
	for _, jobID := range []int64{staleInsert.Job.ID, driftInsert.Job.ID} {
		job, getErr := riverClient.JobGet(ctx, jobID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if job.State != rivertype.JobStateAvailable {
			t.Fatalf("job %d mutated to %q before drift rejection", jobID, job.State)
		}
	}
	var state, code string
	if err := pool.QueryRow(ctx, `
		SELECT reconciliation_state, reconciliation_error
		FROM extension_lifecycle_job_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID).Scan(&state, &code); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || code != "plugin_job.plan_drift" {
		t.Fatalf("drift evidence = %q/%q", state, code)
	}
}

func TestPostgresLifecycleBoundaryJobsDoesNotPersistReconcileErrorSecrets(t *testing.T) {
	ctx, pool, journal, request, _ := newLifecycleJobPublicationIntegration(t)
	attachLifecycleJobManifest(&request.TargetExtension)
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	const sentinel = "secret-payload-token-must-not-persist"
	store := &lifecycleBoundaryFailNthStore{
		delegate: hostapi.NewPostgresPluginJobLifecycleStore(pool, riverClient),
		failAt:   3,
		failure:  errors.New(sentinel),
	}
	boundary := NewPostgresLifecycleBoundaryJobs(PostgresLifecycleBoundaryJobsConfig{
		Pool:        pool,
		Coordinator: &hostapi.PluginJobLifecycleCoordinator{Store: store},
		Trust:       lifecycleBoundaryJobTrust{}, Journal: journal,
	})
	if err := journal.PrepareLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	transaction, err := boundary.PrepareLifecycleJobPublication(ctx, request, LifecycleBoundaryActivate)
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Publish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := journal.CommitLifecyclePublication(ctx, request, LifecycleBoundaryActivate); err != nil {
		t.Fatal(err)
	}
	err = boundary.ReconcileCommittedLifecycleJobs(
		ctx, request, LifecycleBoundaryJobsEnable, LifecycleBoundaryActivate,
	)
	if err == nil || !strings.Contains(err.Error(), sentinel) {
		t.Fatalf("reconcile error = %v", err)
	}
	var state, code, durableText string
	if err := pool.QueryRow(ctx, `
		SELECT reconciliation_state, reconciliation_error,
		       source_snapshot::text || target_snapshot::text || reconciliation_plan::text ||
		       COALESCE(reconciliation_result::text, '') || reconciliation_error
		FROM extension_lifecycle_job_publications
		WHERE operation_id = $1 AND step_id = $2 AND publication_mode = 'activate'
	`, request.OperationID, request.StepID).Scan(&state, &code, &durableText); err != nil {
		t.Fatal(err)
	}
	if state != "failed" || code != "plugin_job.reconciliation_failed" {
		t.Fatalf("failure evidence = %q/%q", state, code)
	}
	if strings.Contains(durableText, sentinel) {
		t.Fatal("durable lifecycle job row contains the secret sentinel")
	}
}

func newLifecycleJobPublicationIntegration(
	t *testing.T,
) (context.Context, *pgxpool.Pool, *PostgresLifecycleBoundaryPublicationJournal, LifecycleBoundaryRequest, string) {
	t.Helper()
	ctx, pool, journal, request, extensionID := newLifecyclePublicationIntegration(t)
	unique := fmt.Sprintf("lifecycle-job-fence-%d", time.Now().UnixNano())
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (username, username_lower, email, email_lower, display_name)
		VALUES ($1, $1, $2, $2, 'Lifecycle Job Fence')
		RETURNING id
	`, unique, unique+"@example.com").Scan(&request.ActorUserID); err != nil {
		t.Fatal(err)
	}
	request.AuditEventID = time.Now().UnixNano()
	path, err := extensions.RecommendedLifecyclePath(request.Operation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE extension_lifecycle_operations
		SET state = $2, current_step_id = $3,
		    requested_by_user_id = $4, audit_event_id = $5,
		    revision = revision + 1, updated_at = statement_timestamp()
		WHERE id = $1
	`, request.OperationID, path[request.Position].State, request.StepID,
		request.ActorUserID, request.AuditEventID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO extension_lifecycle_steps (
			operation_id, step_id, lifecycle_action, plan_version, attempt,
			status, actor_user_id, audit_event_id, started_at
		) VALUES ($1, $2, 'host.gate', 'publication.integration@1', $3,
		          'running', $4, $5, statement_timestamp())
	`, request.OperationID, request.StepID, request.Attempt,
		request.ActorUserID, request.AuditEventID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, request.ActorUserID)
	})
	return ctx, pool, journal, request, extensionID
}

func lifecycleBoundaryIntegrationArgs(
	extensionID string,
	version string,
	digest string,
	trustGrantID string,
	schemaVersion string,
	value string,
) hostapi.PluginJobArgs {
	return hostapi.PluginJobArgs{
		EnvelopeVersion: supportjobs.PluginJobEnvelopeVersion,
		ExtensionID:     extensionID, ExtensionVersion: version, ArtifactDigest: digest,
		TrustGrantID: trustGrantID, JobName: "demo.sync", JobContractVersion: "demo.job@1",
		PayloadSchemaID: "demo.payload", PayloadSchemaVersion: schemaVersion,
		Payload: map[string]any{"value": value}, EnqueuedAt: time.Now().UTC(),
	}
}

type lifecycleBoundaryFailNthStore struct {
	mu       sync.Mutex
	delegate hostapi.PluginJobLifecycleStore
	calls    int
	failAt   int
	failure  error
}

func (s *lifecycleBoundaryFailNthStore) WithPluginJobLifecycleTx(
	ctx context.Context,
	fn func(hostapi.PluginJobLifecycleTx) error,
) error {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	if call == s.failAt {
		return s.failure
	}
	return s.delegate.WithPluginJobLifecycleTx(ctx, fn)
}

func assertLifecycleJobTransactionState(
	t *testing.T,
	ctx context.Context,
	transaction LifecycleBoundaryTransaction,
	want LifecycleBoundaryTransactionState,
) {
	t.Helper()
	got, err := transaction.Inspect(ctx)
	if err != nil || got != want {
		t.Fatal(fmt.Errorf("transaction state = %q, %v; want %q", got, err, want))
	}
}

var _ river.JobArgs = hostapi.PluginJobArgs{}
