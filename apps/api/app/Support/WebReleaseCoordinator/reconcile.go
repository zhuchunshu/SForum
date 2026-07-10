package webreleasecoordinator

import (
	"context"
	"errors"
	"fmt"
	"time"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	webreleaseruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/WebReleaseRuntime"
)

func (c *Coordinator) reconcileLocked(ctx context.Context) error {
	detail, err := c.store.NextActivation(ctx)
	if errors.Is(err, ErrNoActivation) {
		return nil
	}
	if err != nil {
		return err
	}
	if detail.Status == extensions.WebReleaseReady {
		release, err := c.store.Transition(ctx, extensions.WebReleaseTransitionInput{
			ID: detail.ID, ExpectedStatus: extensions.WebReleaseReady,
			NextStatus: extensions.WebReleaseActivating, ActivationCheckpoint: CheckpointPending,
			Reason: "web_release.activating",
		})
		if err != nil {
			return err
		}
		detail.WebRelease = release
	}
	if detail.Status != extensions.WebReleaseActivating {
		return fmt.Errorf("web release %d is not activatable from %s", detail.ID, detail.Status)
	}

	checkpoint := detail.ActivationCheckpoint
	if checkpoint == "" {
		checkpoint = CheckpointPending
	}
	if checkpoint == CheckpointPending {
		if err := c.runtime.Prepare(ctx, detail); err != nil {
			return c.fail(ctx, detail, "web_release.runtime_prepare_failed", err, false)
		}
		if err := c.store.SetCheckpoint(ctx, detail.ID, checkpoint, CheckpointRuntimePrepared); err != nil {
			return err
		}
		checkpoint = CheckpointRuntimePrepared
	}
	if checkpoint == CheckpointRuntimePrepared {
		if err := c.store.ApplyEffects(ctx, detail, true); err != nil {
			return c.fail(ctx, detail, "web_release.effect_commit_failed", err, true)
		}
		if err := c.store.SetCheckpoint(ctx, detail.ID, checkpoint, CheckpointEffectsCommitted); err != nil {
			return err
		}
		checkpoint = CheckpointEffectsCommitted
	}
	if checkpoint == CheckpointEffectsCommitted {
		if err := c.pointers.WriteCurrent(ctx, currentRelease(detail)); err != nil {
			return c.fail(ctx, detail, "web_release.pointer_write_failed", err, true)
		}
		if err := c.store.SetCheckpoint(ctx, detail.ID, checkpoint, CheckpointPointerWritten); err != nil {
			return err
		}
		checkpoint = CheckpointPointerWritten
	}

	if failure, err := c.pointers.ReadFailure(ctx, detail.ID); err == nil {
		return c.fail(ctx, detail, failure.Reason, errors.New(failure.Message), true)
	}
	active, err := c.pointers.ReadActive(ctx)
	if err != nil {
		return nil
	}
	if active.ReleaseID != detail.ID || active.CompositionHash != detail.CompositionHash || active.ArtifactDigest != detail.ArtifactDigest {
		return nil
	}
	if checkpoint == CheckpointPointerWritten {
		if err := c.store.SetCheckpoint(ctx, detail.ID, checkpoint, CheckpointSupervisorActive); err != nil {
			return err
		}
		checkpoint = CheckpointSupervisorActive
	}
	if checkpoint != CheckpointSupervisorActive {
		return fmt.Errorf("unknown web release checkpoint %q", checkpoint)
	}
	if err := c.runtime.Finalize(ctx, detail); err != nil {
		return c.fail(ctx, detail, "web_release.runtime_finalize_failed", err, true)
	}
	if err := c.store.FinalizeRevocations(ctx, detail.ID); err != nil {
		return err
	}
	_, err = c.store.Transition(ctx, extensions.WebReleaseTransitionInput{
		ID: detail.ID, ExpectedStatus: extensions.WebReleaseActivating,
		NextStatus: extensions.WebReleaseActive, ActivationCheckpoint: CheckpointSupervisorActive,
		ArtifactPath: detail.ArtifactPath, ArtifactDigest: detail.ArtifactDigest, ServerEntry: detail.ServerEntry,
		Reason: "web_release.active",
	})
	return err
}

func (c *Coordinator) fail(ctx context.Context, detail extensions.WebReleaseDetail, reason string, cause error, compensate bool) error {
	if compensate {
		_ = c.store.ApplyEffects(ctx, detail, false)
		_ = c.runtime.Compensate(ctx, detail)
		_ = c.pointers.RestorePrevious(ctx, detail)
	}
	_, transitionErr := c.store.Transition(ctx, extensions.WebReleaseTransitionInput{
		ID: detail.ID, ExpectedStatus: extensions.WebReleaseActivating,
		NextStatus: extensions.WebReleaseFailed, PublicReason: reason, PublicMessage: cause.Error(),
		Reason: reason, Message: cause.Error(),
	})
	if transitionErr != nil {
		return errors.Join(cause, transitionErr)
	}
	return cause
}

func currentRelease(detail extensions.WebReleaseDetail) webreleaseruntime.CurrentRelease {
	return webreleaseruntime.CurrentRelease{
		SchemaVersion: webreleaseruntime.ReleaseManifestSchemaVersion,
		ReleaseID:     detail.ID, CompositionHash: detail.CompositionHash,
		ArtifactPath: detail.ArtifactPath, ArtifactDigest: detail.ArtifactDigest,
		ServerEntry: detail.ServerEntry, ThemeID: detail.ActiveThemeID,
		ThemeVersion: detail.ThemeVersion, ReloadMode: detail.ReloadMode, RequestedAt: time.Now().UTC(),
	}
}
