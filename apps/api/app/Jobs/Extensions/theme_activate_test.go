package extensionjobs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/riverqueue/river"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	themeruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/ThemeRuntime"
)

func TestActivateThemeArgsKindAndOptions(t *testing.T) {
	args := ActivateThemeArgs{ReleaseID: 7, ExtensionID: "starter.theme"}
	if args.Kind() != "extension.theme_activate" {
		t.Fatalf("unexpected kind %q", args.Kind())
	}
	opts := args.EnqueueOptions()
	if opts.Queue != "theme" || opts.MaxAttempts != 1 || !opts.Unique.ByArgs {
		t.Fatalf("unexpected enqueue options: %#v", opts)
	}
}

func TestActivateThemeWorkerUsesThemeBuilderTimeoutInsteadOfRiverDefault(t *testing.T) {
	worker := ActivateThemeWorker{}

	if timeout := worker.Timeout(&river.Job[ActivateThemeArgs]{}); timeout != -1 {
		t.Fatalf("expected theme activation worker to disable River's default job timeout, got %s", timeout)
	}
}

func TestActivateThemeWorkerMarksReleaseActive(t *testing.T) {
	store := &fakeThemeStore{
		extension: extensions.Extension{ID: "starter.theme", Version: "1.0.0", Type: extensions.TypeTheme},
		release:   extensions.ThemeRelease{ID: 7, ExtensionID: "starter.theme", Status: extensions.ThemeReleaseQueued, LayerPath: "/tmp/layer"},
	}
	builder := &fakeThemeBuilder{
		result: themeruntime.BuildResult{ArtifactPath: "/tmp/out", ServerEntry: "/tmp/out/server/index.mjs", BuildLog: "ok"},
	}
	worker := ActivateThemeWorker{Store: store, Builder: builder}
	err := worker.Work(context.Background(), &river.Job[ActivateThemeArgs]{
		Args: ActivateThemeArgs{ReleaseID: 7, ExtensionID: "starter.theme"},
	})
	if err != nil {
		t.Fatalf("work: %v", err)
	}
	if store.activeThemeID != "starter.theme" {
		t.Fatalf("expected active theme starter.theme, got %q", store.activeThemeID)
	}
	if store.release.Status != extensions.ThemeReleaseActive {
		t.Fatalf("expected active release, got %#v", store.release)
	}
	if builder.current.ExtensionID != "starter.theme" {
		t.Fatalf("expected current release write, got %#v", builder.current)
	}
}

func TestActivateThemeWorkerMarksReleaseFailedWithCanceledJobContext(t *testing.T) {
	store := &fakeThemeStore{
		extension:            extensions.Extension{ID: "starter.theme", Version: "1.0.0", Type: extensions.TypeTheme},
		release:              extensions.ThemeRelease{ID: 7, ExtensionID: "starter.theme", Status: extensions.ThemeReleaseQueued, LayerPath: "/tmp/layer"},
		respectContextCancel: true,
	}
	builder := &fakeThemeBuilder{
		result: themeruntime.BuildResult{BuildLog: "partial build output"},
		err:    errors.New("theme build failed: signal: killed"),
	}
	worker := ActivateThemeWorker{Store: store, Builder: builder}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := worker.Work(ctx, &river.Job[ActivateThemeArgs]{
		Args: ActivateThemeArgs{ReleaseID: 7, ExtensionID: "starter.theme"},
	})

	if err == nil || !strings.Contains(err.Error(), "signal: killed") {
		t.Fatalf("expected build failure to be returned, got %v", err)
	}
	if store.release.Status != extensions.ThemeReleaseFailed {
		t.Fatalf("expected failed release after canceled job context, got %#v", store.release)
	}
	if store.release.BuildLog != "partial build output" {
		t.Fatalf("expected partial build log to be preserved, got %q", store.release.BuildLog)
	}
}

type fakeThemeStore struct {
	extension            extensions.Extension
	release              extensions.ThemeRelease
	activeThemeID        string
	respectContextCancel bool
}

func (s *fakeThemeStore) Get(_ context.Context, id string) (extensions.Extension, error) {
	if id != s.extension.ID {
		return extensions.Extension{}, extensions.ErrExtensionNotFound
	}
	return s.extension, nil
}

func (s *fakeThemeStore) ActivateTheme(_ context.Context, id string) (extensions.Extension, error) {
	if id != s.extension.ID {
		return extensions.Extension{}, extensions.ErrExtensionNotFound
	}
	s.activeThemeID = id
	s.extension.Status = extensions.StatusEnabled
	return s.extension, nil
}

func (s *fakeThemeStore) UpdateThemeRelease(ctx context.Context, input extensions.ThemeReleaseUpdate) (extensions.ThemeRelease, error) {
	if s.respectContextCancel && ctx.Err() != nil {
		return extensions.ThemeRelease{}, ctx.Err()
	}
	if input.ID != s.release.ID {
		return extensions.ThemeRelease{}, extensions.ErrExtensionNotFound
	}
	s.release.Status = input.Status
	if input.ArtifactPath != "" {
		s.release.ArtifactPath = input.ArtifactPath
	}
	if input.ServerEntry != "" {
		s.release.ServerEntry = input.ServerEntry
	}
	s.release.Message = input.Message
	s.release.BuildLog = input.BuildLog
	return s.release, nil
}

func (s *fakeThemeStore) LatestThemeRelease(_ context.Context, extensionID string) (extensions.ThemeRelease, error) {
	if extensionID != s.release.ExtensionID {
		return extensions.ThemeRelease{}, extensions.ErrExtensionNotFound
	}
	return s.release, nil
}

type fakeThemeBuilder struct {
	result  themeruntime.BuildResult
	current themeruntime.CurrentRelease
	err     error
}

func (b *fakeThemeBuilder) Build(context.Context, themeruntime.BuildInput) (themeruntime.BuildResult, error) {
	return b.result, b.err
}

func (b *fakeThemeBuilder) WriteCurrent(_ context.Context, current themeruntime.CurrentRelease) error {
	b.current = current
	return nil
}
