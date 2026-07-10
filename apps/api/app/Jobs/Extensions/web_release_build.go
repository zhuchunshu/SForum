package extensionjobs

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

type WebReleaseBuildArgs struct {
	ReleaseID int64 `json:"release_id" river:"unique"`
}

func (WebReleaseBuildArgs) Kind() string {
	return "extension.web_release_build"
}

func (WebReleaseBuildArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueTheme,
		MaxAttempts: 3,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type WebReleaseBuildDispatcher interface {
	EnqueueTx(context.Context, pgx.Tx, river.JobArgs, supportjobs.EnqueueOptions) (*rivertype.JobInsertResult, error)
}

type WebReleaseBuildDispatcherAdapter struct {
	Dispatcher WebReleaseBuildDispatcher
}

func (a WebReleaseBuildDispatcherAdapter) EnqueueWebReleaseBuildTx(ctx context.Context, tx pgx.Tx, releaseID int64) error {
	if releaseID <= 0 {
		return fmt.Errorf("web release build requires a positive release id")
	}
	if a.Dispatcher == nil {
		return fmt.Errorf("web release build dispatcher is unavailable")
	}
	args := WebReleaseBuildArgs{ReleaseID: releaseID}
	_, err := a.Dispatcher.EnqueueTx(ctx, tx, args, args.EnqueueOptions())
	return err
}
