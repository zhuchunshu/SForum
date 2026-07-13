package hostapi

import (
	"slices"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestPostgresPluginJobLifecycleLocksOnlyExactScopeIncludingFinalized(t *testing.T) {
	harness := newPluginJobLifecycleIntegrationHarness(t)
	exactID := "sforum.lifecycle.scope"
	runtime := integrationPluginJobRuntime(exactID, "scope", "1.0.0", "sha256:scope", "1", "grant-scope")
	valid := harness.insertJob(t, integrationPluginJobArgs(runtime, map[string]any{"value": "valid"}), &river.InsertOpts{
		MaxAttempts: 9, Queue: "plugin-scope", Priority: 4, Tags: []string{"scope-valid"},
	})
	malformedArgs := integrationPluginJobArgs(runtime, map[string]any{"value": "malformed"})
	malformedArgs.ArtifactDigest = ""
	malformed := harness.insertJob(t, malformedArgs, nil)
	finalized := harness.insertJob(t, integrationPluginJobArgs(runtime, map[string]any{"value": "finalized"}), nil)
	if _, err := harness.river.JobCancel(harness.ctx, finalized.ID); err != nil {
		t.Fatalf("finalize exact-scope job: %v", err)
	}

	foreignRuntime := integrationPluginJobRuntime("sforum.lifecycle.foreign", "scope", "1.0.0", "sha256:foreign", "1", "grant-foreign")
	foreign := harness.insertJob(t, integrationPluginJobArgs(foreignRuntime, nil), nil)
	wrongKind := harness.insertJob(t, integrationWrongKindArgs{ExtensionID: exactID}, nil)
	nonStringScope := harness.insertJob(t, integrationForeignJobArgs{ExtensionID: []string{exactID}}, nil)

	var locked []PluginJobLifecycleRow
	if err := harness.store.WithPluginJobLifecycleTx(harness.ctx, func(tx PluginJobLifecycleTx) error {
		var err error
		locked, err = tx.LockPluginJobs(harness.ctx, exactID)
		return err
	}); err != nil {
		t.Fatalf("lock exact plugin job scope: %v", err)
	}
	if len(locked) != 3 {
		t.Fatalf("locked %d rows, want all three exact-scope rows: %#v", len(locked), locked)
	}
	lockedIDs := []int64{locked[0].JobID, locked[1].JobID, locked[2].JobID}
	if !slices.Equal(lockedIDs, []int64{valid.ID, malformed.ID, finalized.ID}) {
		t.Fatalf("locked ids = %v", lockedIDs)
	}
	if locked[0].Kind != PluginJobKind || locked[0].State != rivertype.JobStateAvailable ||
		locked[0].Attempt != 0 || locked[0].MaxAttempts != 9 || locked[0].Queue != "plugin-scope" ||
		locked[0].Priority != 4 || !slices.Equal(locked[0].Tags, []string{"scope-valid"}) ||
		locked[0].ScheduledAt.IsZero() {
		t.Fatalf("locked River row mapping = %#v", locked[0])
	}
	if locked[2].State != rivertype.JobStateCancelled {
		t.Fatalf("finalized row was omitted or remapped: %#v", locked[2])
	}

	input := PluginJobLifecycleInput{
		ExtensionID: exactID, SourceRuntimeAvailable: true,
		SourceContracts: map[string]PluginJobRuntimeContract{"scope": runtime},
		TargetContracts: map[string]PluginJobRuntimeContract{"scope": runtime},
	}
	result, err := (&PluginJobLifecycleCoordinator{Store: harness.store}).Reconcile(harness.ctx, input)
	if err != nil {
		t.Fatalf("reconcile exact scope: %v", err)
	}
	if !result.Committed || result.Plan.IgnoredFinalized != 1 || len(result.Plan.Entries) != 2 {
		t.Fatalf("scope reconcile result = %#v", result)
	}
	if harness.job(t, valid.ID).State != rivertype.JobStateAvailable ||
		harness.job(t, malformed.ID).State != rivertype.JobStateCancelled ||
		harness.job(t, finalized.ID).State != rivertype.JobStateCancelled {
		t.Fatal("exact-scope execute/cancel/finalized policy was not applied")
	}
	for name, jobID := range map[string]int64{
		"foreign extension": foreign.ID,
		"foreign kind":      wrongKind.ID,
		"non-string scope":  nonStringScope.ID,
	} {
		if state := harness.job(t, jobID).State; state != rivertype.JobStateAvailable {
			t.Fatalf("%s job %d was touched: %s", name, jobID, state)
		}
	}
}

func TestPostgresPluginJobLifecycleLockDoesNotSilentlyPaginate(t *testing.T) {
	harness := newPluginJobLifecycleIntegrationHarness(t)
	const count = 257
	extensionID := "sforum.lifecycle.bulk"
	runtime := integrationPluginJobRuntime(extensionID, "bulk", "1.0.0", "sha256:bulk", "1", "grant-bulk")
	params := make([]river.InsertManyParams, 0, count)
	for index := range count {
		params = append(params, river.InsertManyParams{
			Args: integrationPluginJobArgs(runtime, map[string]any{"index": index}),
			InsertOpts: &river.InsertOpts{
				MaxAttempts: 3, Queue: "plugin-bulk", Priority: 2, Tags: []string{"bulk-row"},
			},
		})
	}
	inserted, err := harness.river.InsertMany(harness.ctx, params)
	if err != nil {
		t.Fatalf("insert plugin job batch: %v", err)
	}
	if len(inserted) != count {
		t.Fatalf("inserted %d jobs, want %d", len(inserted), count)
	}

	var locked []PluginJobLifecycleRow
	err = harness.store.WithPluginJobLifecycleTx(harness.ctx, func(tx PluginJobLifecycleTx) error {
		var lockErr error
		locked, lockErr = tx.LockPluginJobs(harness.ctx, extensionID)
		return lockErr
	})
	if err != nil {
		t.Fatalf("lock bulk plugin jobs: %v", err)
	}
	if len(locked) != count {
		t.Fatalf("locked %d jobs, want all %d", len(locked), count)
	}
	for index, row := range locked {
		if row.JobID != inserted[index].Job.ID {
			t.Fatalf("bulk row %d id = %d, want %d", index, row.JobID, inserted[index].Job.ID)
		}
		if row.MaxAttempts != 3 || row.Queue != "plugin-bulk" || row.Priority != 2 {
			t.Fatalf("bulk row %d mapping = %#v", index, row)
		}
	}
}
