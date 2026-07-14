package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPostgresThemeRuntimeNodeLeaseAndExactAcknowledgement(t *testing.T) {
	fixture := newThemePublicationPGFixture(t, "node-ack")
	identity := ThemeRuntimeNodeIdentity{NodeID: "api-1", BootID: "boot-1"}
	node, err := fixture.store.RegisterThemeRuntimeNode(fixture.ctx, identity, time.Minute)
	if err != nil || node.NodeID != identity.NodeID || node.BootID != identity.BootID ||
		node.LastAppliedRevision != 0 || !node.LeaseExpiresAt.After(node.LastSeenAt) {
		t.Fatalf("registered node=%#v err=%v", node, err)
	}

	publication, err := fixture.store.LatestThemeRuntimePublication(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	ack, err := fixture.store.BeginThemeRuntimePublicationApply(fixture.ctx, identity, publication.Revision)
	if err != nil || ack.Status != ThemeRuntimeAckApplying || ack.AttemptCount != 1 || ack.Revision != 1 {
		t.Fatalf("begin ack=%#v err=%v", ack, err)
	}
	drifted := publication
	drifted.ThemeVersion = "9.9.9"
	if _, err := fixture.store.CompleteThemeRuntimePublicationApply(
		fixture.ctx, identity, drifted, ack.Revision,
	); !errors.Is(err, ErrThemeRuntimeAckConflict) {
		t.Fatalf("drifted completion error=%v", err)
	}
	applied, err := fixture.store.CompleteThemeRuntimePublicationApply(
		fixture.ctx, identity, publication, ack.Revision,
	)
	if err != nil || applied.Status != ThemeRuntimeAckApplied || applied.AppliedAt == nil ||
		applied.AppliedState != publication.DesiredState || applied.AppliedThemeID != publication.ThemeID ||
		applied.AppliedThemeVersion != publication.ThemeVersion || applied.AppliedPackageDigest != publication.PackageDigest ||
		applied.Revision != ack.Revision+1 {
		t.Fatalf("applied ack=%#v err=%v", applied, err)
	}
	node, err = fixture.store.GetThemeRuntimeNode(fixture.ctx, identity)
	if err != nil || node.LastAppliedRevision != publication.Revision {
		t.Fatalf("applied node=%#v err=%v", node, err)
	}
	if _, err := fixture.store.BeginThemeRuntimePublicationApply(
		fixture.ctx, identity, publication.Revision,
	); !errors.Is(err, ErrThemeRuntimeAckConflict) {
		t.Fatalf("applied replay error=%v", err)
	}

	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE theme_runtime_nodes
		SET first_seen_at = statement_timestamp() - interval '3 seconds',
		    last_seen_at = statement_timestamp() - interval '2 seconds',
		    lease_expires_at = statement_timestamp() - interval '1 second'
		WHERE node_id = $1 AND boot_id = $2
	`, identity.NodeID, identity.BootID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.HeartbeatThemeRuntimeNode(fixture.ctx, identity, time.Minute); !errors.Is(err, ErrThemeRuntimeNodeLeaseLost) {
		t.Fatalf("expired heartbeat error=%v", err)
	}
	if _, err := fixture.store.RegisterThemeRuntimeNode(fixture.ctx, identity, time.Minute); !errors.Is(err, ErrThemeRuntimeNodeLeaseLost) {
		t.Fatalf("expired boot resurrection error=%v", err)
	}
	newBoot := ThemeRuntimeNodeIdentity{NodeID: identity.NodeID, BootID: "boot-2"}
	if node, err = fixture.store.RegisterThemeRuntimeNode(fixture.ctx, newBoot, time.Minute); err != nil || node.LastAppliedRevision != 0 {
		t.Fatalf("new boot node=%#v err=%v", node, err)
	}
}

func TestPostgresThemeRuntimeFailedAckRetriesWithCAS(t *testing.T) {
	fixture := newThemePublicationPGFixture(t, "node-retry")
	identity := ThemeRuntimeNodeIdentity{NodeID: "api-2", BootID: "boot-retry"}
	if _, err := fixture.store.RegisterThemeRuntimeNode(fixture.ctx, identity, time.Minute); err != nil {
		t.Fatal(err)
	}
	target := fixture.saveTheme("target", "2.0.0", strings.Repeat("7", 64))
	current := fixture.activeTheme()
	activation, err := fixture.store.ActivateThemeExact(
		fixture.ctx, target.ID, exactThemeActivationInput(current, target, 88, false),
	)
	if err != nil {
		t.Fatal(err)
	}

	first, err := fixture.store.BeginThemeRuntimePublicationApply(
		fixture.ctx, identity, activation.Publication.Revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := fixture.store.FailThemeRuntimePublicationApply(
		fixture.ctx, identity, activation.Publication.Revision, first.Revision, "compile failed",
	)
	if err != nil || failed.Status != ThemeRuntimeAckFailed || failed.ErrorReason != "compile failed" ||
		failed.AttemptCount != 1 || failed.Revision != first.Revision+1 {
		t.Fatalf("failed ack=%#v err=%v", failed, err)
	}
	if _, err := fixture.store.FailThemeRuntimePublicationApply(
		fixture.ctx, identity, activation.Publication.Revision, first.Revision, "stale",
	); !errors.Is(err, ErrThemeRuntimeAckConflict) {
		t.Fatalf("stale failure error=%v", err)
	}
	retry, err := fixture.store.BeginThemeRuntimePublicationApply(
		fixture.ctx, identity, activation.Publication.Revision,
	)
	if err != nil || retry.Status != ThemeRuntimeAckApplying || retry.AttemptCount != 2 ||
		retry.Revision != failed.Revision+1 || retry.ErrorReason != "" {
		t.Fatalf("retry ack=%#v err=%v", retry, err)
	}
	applied, err := fixture.store.CompleteThemeRuntimePublicationApply(
		fixture.ctx, identity, activation.Publication, retry.Revision,
	)
	if err != nil || applied.Status != ThemeRuntimeAckApplied || applied.AttemptCount != 2 {
		t.Fatalf("retry applied=%#v err=%v", applied, err)
	}

	tooLong := strings.Repeat("x", 2049)
	if _, err := fixture.store.FailThemeRuntimePublicationApply(
		context.Background(), identity, activation.Publication.Revision, applied.Revision, tooLong,
	); !errors.Is(err, ErrThemeRuntimeAckConflict) {
		t.Fatalf("oversized reason error=%v", err)
	}
}
