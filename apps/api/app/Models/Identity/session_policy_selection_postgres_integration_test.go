package identity

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"testing"

	identityregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/IdentityRegistry"
)

func TestIdentitySessionPolicyPostgresSelectResetRuntimeRebindAndSafeMode(t *testing.T) {
	fixture := newIdentityPersistencePGFixture(t)
	ctx := fixture.ctx

	current, err := fixture.sessionPolicy.Current(ctx)
	if err != nil || current.PolicyID != IdentitySessionPolicyCoreDefault ||
		current.Revision != 0 || !current.Implicit {
		t.Fatalf("implicit Core selection = %#v, %v", current, err)
	}
	fixture.assertSessionPolicyCounts(0, 0, 0)
	effective, err := fixture.sessionPolicy.Resolve(ctx)
	if err != nil || effective.PolicyID != IdentitySessionPolicyCoreDefault ||
		effective.Source != IdentitySessionPolicySourceCore || effective.Selection == nil ||
		effective.Selection.Revision != 0 || effective.Provider != nil {
		t.Fatalf("implicit Core resolution = %#v, %v", effective, err)
	}

	candidate, err := fixture.sessionPolicy.Candidate(ctx, fixture.sessionProvider.ID)
	if err != nil {
		t.Fatalf("resolve exact session candidate: %v", err)
	}
	if candidate.PolicyID != fixture.sessionProvider.ID ||
		candidate.ProviderContractVersion != fixture.sessionProvider.ContractVersion ||
		candidate.OwnerExtensionID != fixture.extensionID ||
		candidate.OwnerExtensionVersionID != fixture.versionID ||
		candidate.DeclarationRevision <= 0 {
		t.Fatalf("candidate = %#v", candidate)
	}

	selected, err := fixture.sessionPolicy.Select(ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
	})
	if err != nil || !selected.Changed || selected.Event == nil ||
		selected.Selection.Revision != 1 || selected.Selection.Implicit ||
		selected.Selection.IdentitySessionPolicyEvidence != candidate ||
		selected.Event.PreviousSelection != nil || selected.Event.SelectedSelection == nil ||
		*selected.Event.SelectedSelection != candidate ||
		selected.Selection.SelectionAuditEventID != selected.Event.AuditEventID {
		t.Fatalf("select result = %#v, %v", selected, err)
	}
	fixture.assertSessionPolicyCounts(1, 1, 1)
	readback, found, err := fixture.sessionPolicy.readbackIdentitySessionPolicyMutation(ctx, selected)
	if err != nil || !found || !reflect.DeepEqual(readback.Selection, selected.Selection) ||
		readback.Event == nil || readback.Event.ID != selected.Event.ID {
		t.Fatalf("commit marker readback = %#v, found=%v, error=%v", readback, found, err)
	}
	effective, err = fixture.sessionPolicy.Resolve(ctx)
	if err != nil || effective.Source != IdentitySessionPolicySourcePlugin ||
		effective.Selection == nil || effective.Selection.Revision != 1 ||
		effective.Provider == nil || effective.Provider.ID != candidate.PolicyID ||
		effective.RegistryRevision == 0 || effective.RegistryDigest == "" {
		t.Fatalf("selected resolution = %#v, %v", effective, err)
	}
	fixture.assertSessionPolicyAuditMetadata(
		selected.Event.AuditEventID,
		identitySessionPolicyAuditMetadata{
			SelectedSelection: &candidate,
			SelectionRevision: 1,
		},
	)

	noChange, err := fixture.sessionPolicy.Select(ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 1, ActorUserID: fixture.adminUserID,
	})
	if err != nil || noChange.Changed || noChange.Event != nil || noChange.Selection.Revision != 1 {
		t.Fatalf("exact-current select = %#v, %v", noChange, err)
	}
	if _, err := fixture.sessionPolicy.Select(ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
	}); !errors.Is(err, ErrIdentitySessionPolicyRevisionConflict) {
		t.Fatalf("stale identical select error = %v", err)
	}
	fixture.assertSessionPolicyCounts(1, 1, 1)

	reset, err := fixture.sessionPolicy.Reset(ctx, ResetIdentitySessionPolicyInput{
		ExpectedRevision: 1, ActorUserID: fixture.adminUserID, ReasonCode: "operator_reset",
	})
	if err != nil || !reset.Changed || reset.Event == nil || reset.Selection.Revision != 2 ||
		reset.Selection.PolicyID != IdentitySessionPolicyCoreDefault || reset.Selection.Implicit ||
		reset.Event.SelectedSelection != nil || reset.Event.PreviousSelection == nil ||
		*reset.Event.PreviousSelection != candidate {
		t.Fatalf("reset result = %#v, %v", reset, err)
	}
	fixture.assertSessionPolicyCounts(1, 2, 2)
	fixture.assertSessionPolicyAuditMetadata(
		reset.Event.AuditEventID,
		identitySessionPolicyAuditMetadata{
			PreviousSelection: &candidate,
			SelectionRevision: 2,
			ReasonCode:        "operator_reset",
		},
	)

	noChange, err = fixture.sessionPolicy.Reset(ctx, ResetIdentitySessionPolicyInput{
		ExpectedRevision: 2, ActorUserID: fixture.adminUserID, ReasonCode: "operator_reset",
	})
	if err != nil || noChange.Changed || noChange.Selection.Revision != 2 {
		t.Fatalf("exact-current Core reset = %#v, %v", noChange, err)
	}
	if _, err := fixture.sessionPolicy.Reset(ctx, ResetIdentitySessionPolicyInput{
		ExpectedRevision: 1, ActorUserID: fixture.adminUserID, ReasonCode: "operator_reset",
	}); !errors.Is(err, ErrIdentitySessionPolicyRevisionConflict) {
		t.Fatalf("stale Core reset error = %v", err)
	}

	reselected, err := fixture.sessionPolicy.Select(ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 2, ActorUserID: fixture.adminUserID,
	})
	if err != nil || !reselected.Changed || reselected.Event == nil ||
		reselected.Selection.Revision != 3 || reselected.Event.PreviousSelection == nil ||
		reselected.Event.PreviousSelection.PolicyID != IdentitySessionPolicyCoreDefault {
		t.Fatalf("reselect result = %#v, %v", reselected, err)
	}
	fixture.assertSessionPolicyCounts(1, 3, 3)

	restartPublication, err := identityPersistenceProviderPublication(
		fixture.extensionID,
		fixture.versionID,
		candidate.OwnerPackageDigest,
		"identity-runtime-restart",
	)
	if err != nil {
		t.Fatal(err)
	}
	restartRegistry := identityregistry.New()
	if _, err := restartRegistry.Publish(restartPublication); err != nil {
		t.Fatal(err)
	}
	restartStore, err := NewPostgresIdentitySessionPolicyStore(fixture.pool, restartRegistry)
	if err != nil {
		t.Fatal(err)
	}
	restartCandidate, err := restartStore.Candidate(ctx, candidate.PolicyID)
	if err != nil || restartCandidate != candidate {
		t.Fatalf("restart candidate = %#v, %v", restartCandidate, err)
	}
	restarted, err := restartStore.Current(ctx)
	if err != nil || restarted.Revision != 3 || restarted.IdentitySessionPolicyEvidence != candidate {
		t.Fatalf("restart selection = %#v, %v", restarted, err)
	}

	beforeSafeMode := restarted
	if _, err := restartRegistry.ReplaceAll([]identityregistry.Publication{restartPublication}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := restartStore.Candidate(ctx, candidate.PolicyID); !errors.Is(err, ErrIdentitySessionPolicySafeMode) {
		t.Fatalf("Safe Mode candidate error = %v", err)
	}
	if _, err := restartStore.Select(ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 3, ActorUserID: fixture.adminUserID,
	}); !errors.Is(err, ErrIdentitySessionPolicySafeMode) {
		t.Fatalf("Safe Mode select error = %v", err)
	}
	afterSafeMode, err := restartStore.Current(ctx)
	if err != nil || !reflect.DeepEqual(afterSafeMode, beforeSafeMode) {
		t.Fatalf("Safe Mode mutated desired selection before=%#v after=%#v error=%v", beforeSafeMode, afterSafeMode, err)
	}
	effective, err = restartStore.Resolve(ctx)
	if err != nil || effective.PolicyID != IdentitySessionPolicyCoreDefault ||
		effective.Source != IdentitySessionPolicySourceSafeMode ||
		effective.Selection != nil || effective.Provider != nil {
		t.Fatalf("Safe Mode effective resolution = %#v, %v", effective, err)
	}
	fixture.assertSessionPolicyCounts(1, 3, 3)
	if _, err := restartRegistry.ReplaceAll([]identityregistry.Publication{restartPublication}, false); err != nil {
		t.Fatal(err)
	}
	effective, err = restartStore.Resolve(ctx)
	if err != nil || effective.Source != IdentitySessionPolicySourcePlugin ||
		effective.Selection == nil || effective.Selection.Revision != 3 || effective.Provider == nil {
		t.Fatalf("post-Safe-Mode resolution = %#v, %v", effective, err)
	}
	if _, err := restartRegistry.ReplaceAll([]identityregistry.Publication{restartPublication}, true); err != nil {
		t.Fatal(err)
	}
	// Explicit reset is a Host recovery action, not an automatic Safe Mode
	// side effect, and intentionally remains available while plugins are bypassed.
	safeModeReset, err := restartStore.Reset(ctx, ResetIdentitySessionPolicyInput{
		ExpectedRevision: 3, ActorUserID: fixture.adminUserID, ReasonCode: "operator_reset",
	})
	if err != nil || !safeModeReset.Changed || safeModeReset.Selection.Revision != 4 ||
		safeModeReset.Selection.PolicyID != IdentitySessionPolicyCoreDefault {
		t.Fatalf("Safe Mode explicit reset = %#v, %v", safeModeReset, err)
	}
	fixture.assertSessionPolicyCounts(1, 4, 4)
	if _, err := restartRegistry.ReplaceAll([]identityregistry.Publication{restartPublication}, false); err != nil {
		t.Fatal(err)
	}
	effective, err = restartStore.Resolve(ctx)
	if err != nil || effective.Source != IdentitySessionPolicySourceCore ||
		effective.Selection == nil || effective.Selection.Revision != 4 || effective.Provider != nil {
		t.Fatalf("explicit Core after Safe Mode = %#v, %v", effective, err)
	}

	events, err := restartStore.ListEvents(ctx, 100)
	if err != nil || len(events) != 4 ||
		events[0].SelectionRevision != 4 || events[1].SelectionRevision != 3 ||
		events[2].SelectionRevision != 2 || events[3].SelectionRevision != 1 {
		t.Fatalf("events = %#v, %v", events, err)
	}
}

func TestIdentitySessionPolicyPostgresStaleDesiredFailsClosed(t *testing.T) {
	fixture := newIdentityPersistencePGFixture(t)
	candidate, err := fixture.sessionPolicy.Candidate(fixture.ctx, fixture.sessionProvider.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.sessionPolicy.Select(fixture.ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.retireProvider(); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.sessionPolicy.Resolve(fixture.ctx); !errors.Is(err, ErrIdentitySessionPolicyDeclarationStale) {
		t.Fatalf("stale desired resolution error = %v", err)
	}
	desired, err := fixture.sessionPolicy.Current(fixture.ctx)
	if err != nil || desired.IdentitySessionPolicyEvidence != candidate || desired.Revision != 1 {
		t.Fatalf("stale desired selection = %#v, %v", desired, err)
	}
	fixture.assertSessionPolicyCounts(1, 1, 1)
}

func TestIdentitySessionPolicyPostgresRejectsAuthorityAndRollsBack(t *testing.T) {
	fixture := newIdentityPersistencePGFixture(t)
	candidate, err := fixture.sessionPolicy.Candidate(fixture.ctx, fixture.sessionProvider.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fixture.sessionPolicy.Select(fixture.ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.actorUserID,
	}); !errors.Is(err, ErrIdentitySessionPolicyPermissionDenied) {
		t.Fatalf("non-super-admin select error = %v", err)
	}
	forged := candidate
	forged.DeclarationRevision++
	if _, err := fixture.sessionPolicy.Select(fixture.ctx, SelectIdentitySessionPolicyInput{
		Candidate: forged, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
	}); !errors.Is(err, ErrIdentitySessionPolicyDeclarationStale) {
		t.Fatalf("forged exact claim error = %v", err)
	}
	fixture.assertSessionPolicyCounts(0, 0, 0)

	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE FUNCTION reject_session_policy_event() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'forced session-policy event failure';
		END;
		$$;
		CREATE TRIGGER reject_session_policy_event
		BEFORE INSERT ON identity_session_policy_selection_events
		FOR EACH ROW EXECUTE FUNCTION reject_session_policy_event();
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.sessionPolicy.Select(fixture.ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
	}); !errors.Is(err, ErrIdentitySessionPolicyStoreUnavailable) {
		t.Fatalf("forced event rollback error = %v", err)
	}
	fixture.assertSessionPolicyCounts(0, 0, 0)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		DROP TRIGGER reject_session_policy_event ON identity_session_policy_selection_events;
		DROP FUNCTION reject_session_policy_event();
	`); err != nil {
		t.Fatal(err)
	}

	if _, err := fixture.pool.Exec(fixture.ctx, `UPDATE users SET status='disabled' WHERE id=$1`, fixture.adminUserID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.sessionPolicy.Select(fixture.ctx, SelectIdentitySessionPolicyInput{
		Candidate: candidate, ExpectedRevision: 0, ActorUserID: fixture.adminUserID,
	}); !errors.Is(err, ErrIdentitySessionPolicyPermissionDenied) {
		t.Fatalf("inactive super-admin select error = %v", err)
	}
	fixture.assertSessionPolicyCounts(0, 0, 0)
}

func (f *identityPersistencePGFixture) assertSessionPolicyCounts(selections, events, audits int) {
	f.t.Helper()
	var gotSelections, gotEvents, gotAudits int
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM identity_session_policy_selection`).Scan(&gotSelections); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM identity_session_policy_selection_events`).Scan(&gotEvents); err != nil {
		f.t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `
		SELECT count(*) FROM audit_events WHERE action LIKE 'identity.session_policy.%'
	`).Scan(&gotAudits); err != nil {
		f.t.Fatal(err)
	}
	if gotSelections != selections || gotEvents != events || gotAudits != audits {
		f.t.Fatalf(
			"session policy counts selections=%d/%d events=%d/%d audits=%d/%d",
			gotSelections,
			selections,
			gotEvents,
			events,
			gotAudits,
			audits,
		)
	}
}

func (f *identityPersistencePGFixture) assertSessionPolicyAuditMetadata(
	auditID int64,
	want identitySessionPolicyAuditMetadata,
) {
	f.t.Helper()
	var action string
	var raw []byte
	if err := f.pool.QueryRow(f.ctx, `
		SELECT action, metadata FROM audit_events WHERE id=$1
	`, auditID).Scan(&action, &raw); err != nil {
		f.t.Fatal(err)
	}
	if action != "identity.session_policy."+identitySessionPolicyActionForAudit(want) {
		f.t.Fatalf("audit action = %q", action)
	}
	var got identitySessionPolicyAuditMetadata
	if err := json.Unmarshal(raw, &got); err != nil {
		f.t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		f.t.Fatalf("audit metadata = %#v, want %#v", got, want)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		f.t.Fatal(err)
	}
	gotKeys := make([]string, 0, len(keys))
	for key := range keys {
		gotKeys = append(gotKeys, key)
	}
	sort.Strings(gotKeys)
	wantKeys := []string{"previousSelection", "selectedSelection", "selectionRevision"}
	if want.ReasonCode != "" {
		wantKeys = append(wantKeys, "reasonCode")
		sort.Strings(wantKeys)
	}
	if !reflect.DeepEqual(gotKeys, wantKeys) {
		f.t.Fatalf("audit metadata keys = %v, want %v", gotKeys, wantKeys)
	}
}

func identitySessionPolicyActionForAudit(metadata identitySessionPolicyAuditMetadata) string {
	if metadata.SelectedSelection == nil {
		return IdentitySessionPolicyActionReset
	}
	return IdentitySessionPolicyActionSelect
}
