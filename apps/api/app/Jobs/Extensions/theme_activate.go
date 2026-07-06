package extensionjobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
)

type ThemeStore interface {
	Get(ctx context.Context, id string) (extensions.Extension, error)
	ActivateTheme(ctx context.Context, id string) (extensions.Extension, error)
	UpdateThemeRelease(ctx context.Context, input extensions.ThemeReleaseUpdate) (extensions.ThemeRelease, error)
	LatestThemeRelease(ctx context.Context, extensionID string) (extensions.ThemeRelease, error)
}

type ThemeBuilder interface {
	Build(ctx context.Context, input themeruntime.BuildInput) (themeruntime.BuildResult, error)
	WriteCurrent(ctx context.Context, current themeruntime.CurrentRelease) error
}

type ActivateThemeArgs struct {
	ReleaseID   int64  `json:"release_id" river:"unique"`
	ExtensionID string `json:"extension_id"`
}

func (ActivateThemeArgs) Kind() string {
	return "extension.theme_activate"
}

func (ActivateThemeArgs) EnqueueOptions() supportjobs.EnqueueOptions {
	return supportjobs.EnqueueOptions{
		Queue:       supportjobs.QueueTheme,
		MaxAttempts: 1,
		Unique:      river.UniqueOpts{ByArgs: true},
	}
}

type ActivateThemeWorker struct {
	river.WorkerDefaults[ActivateThemeArgs]
	Store   ThemeStore
	Builder ThemeBuilder
}

func (w *ActivateThemeWorker) Work(ctx context.Context, job *river.Job[ActivateThemeArgs]) error {
	if w.Store == nil {
		return fmt.Errorf("theme activation worker requires store")
	}
	if w.Builder == nil {
		return fmt.Errorf("theme activation worker requires builder")
	}
	extension, err := w.Store.Get(ctx, job.Args.ExtensionID)
	if err != nil {
		return err
	}
	if extension.Type != extensions.TypeTheme {
		return fmt.Errorf("extension %s is not a theme", extension.ID)
	}
	release, err := w.Store.LatestThemeRelease(ctx, extension.ID)
	if err != nil {
		return err
	}
	if release.ID != job.Args.ReleaseID {
		return fmt.Errorf("theme release mismatch: job=%d latest=%d", job.Args.ReleaseID, release.ID)
	}
	_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
		ID:      release.ID,
		Status:  extensions.ThemeReleaseBuilding,
		Message: "Building theme release.",
	})
	result, err := w.Builder.Build(ctx, themeruntime.BuildInput{
		ReleaseID:   release.ID,
		ExtensionID: extension.ID,
		LayerPath:   release.LayerPath,
	})
	if err != nil {
		_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
			ID:       release.ID,
			Status:   extensions.ThemeReleaseFailed,
			Message:  err.Error(),
			BuildLog: result.BuildLog,
		})
		return err
	}
	_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
		ID:           release.ID,
		Status:       extensions.ThemeReleaseActivating,
		ArtifactPath: result.ArtifactPath,
		ServerEntry:  result.ServerEntry,
		Message:      "Switching active web release.",
		BuildLog:     result.BuildLog,
	})
	if err := w.Builder.WriteCurrent(ctx, themeruntime.CurrentRelease{
		ReleaseID:   release.ID,
		ExtensionID: extension.ID,
		Server:      result.ServerEntry,
	}); err != nil {
		_, _ = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
			ID:       release.ID,
			Status:   extensions.ThemeReleaseFailed,
			Message:  err.Error(),
			BuildLog: result.BuildLog,
		})
		return err
	}
	if _, err := w.Store.ActivateTheme(ctx, extension.ID); err != nil {
		return err
	}
	_, err = w.Store.UpdateThemeRelease(ctx, extensions.ThemeReleaseUpdate{
		ID:           release.ID,
		Status:       extensions.ThemeReleaseActive,
		ArtifactPath: result.ArtifactPath,
		ServerEntry:  result.ServerEntry,
		Message:      "Theme release activated.",
		BuildLog:     result.BuildLog,
	})
	return err
}

func RegisterThemeActivationWorker(registry *supportjobs.Registry, store ThemeStore, builder ThemeBuilder) {
	registry.Add(func(workers *river.Workers) error {
		return river.AddWorkerSafely[ActivateThemeArgs](workers, &ActivateThemeWorker{Store: store, Builder: builder})
	})
}

type ActivationDispatcherAdapter struct {
	Dispatcher *supportjobs.Dispatcher
}

func (a ActivationDispatcherAdapter) EnqueueThemeActivation(ctx context.Context, release extensions.ThemeRelease) error {
	if a.Dispatcher == nil {
		return nil
	}
	args := ActivateThemeArgs{ReleaseID: release.ID, ExtensionID: release.ExtensionID}
	_, err := a.Dispatcher.Enqueue(ctx, args, args.EnqueueOptions())
	return err
}
