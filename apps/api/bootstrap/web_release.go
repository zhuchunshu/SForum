package bootstrap

import (
	"context"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

type initialWebReleaseReader interface {
	HasLiveWebRelease(context.Context) (bool, error)
}

type initialWebReleaseQueuer interface {
	PlanAndQueue(context.Context, extensions.QueueWebReleaseInput) (extensions.WebReleaseQueueResult, error)
}

func ensureInitialWebRelease(ctx context.Context, reader initialWebReleaseReader, queuer initialWebReleaseQueuer) error {
	hasLive, err := reader.HasLiveWebRelease(ctx)
	if err != nil || hasLive {
		return err
	}
	_, err = queuer.PlanAndQueue(ctx, extensions.QueueWebReleaseInput{Plan: extensions.PlanWebReleaseInput{
		TriggerKind: extensions.WebReleaseTriggerRebuild,
		ReloadMode:  extensions.WebReleaseReloadPrompt,
	}})
	return err
}
