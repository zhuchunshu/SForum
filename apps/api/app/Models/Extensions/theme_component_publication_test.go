package extensions

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type themeComponentTransitionCall struct {
	kind     string
	targetID string
	sourceID string
	revision int64
}

type recordingThemeComponentRegistry struct {
	validateErr error
	publishErrs []error
	rollbackErr error
	calls       []themeComponentTransitionCall
}

func (r *recordingThemeComponentRegistry) ValidateThemeTransition(target, source *Extension) error {
	r.calls = append(r.calls, themeComponentTransitionCall{
		kind: "validate", targetID: extensionPointerID(target), sourceID: extensionPointerID(source),
	})
	return r.validateErr
}

func (r *recordingThemeComponentRegistry) PublishThemeTransition(
	target,
	source *Extension,
	revision int64,
) error {
	r.calls = append(r.calls, themeComponentTransitionCall{
		kind: "publish", targetID: extensionPointerID(target), sourceID: extensionPointerID(source), revision: revision,
	})
	if len(r.publishErrs) == 0 {
		return nil
	}
	err := r.publishErrs[0]
	r.publishErrs = r.publishErrs[1:]
	return err
}

func (r *recordingThemeComponentRegistry) RollbackThemeTransition(
	target,
	source *Extension,
	revision int64,
) error {
	r.calls = append(r.calls, themeComponentTransitionCall{
		kind: "rollback", targetID: extensionPointerID(target), sourceID: extensionPointerID(source), revision: revision,
	})
	return r.rollbackErr
}

func extensionPointerID(extension *Extension) string {
	if extension == nil {
		return ""
	}
	return extension.ID
}

func TestServiceThemeComponentPreflightRejectsBeforeDatabaseSwitch(t *testing.T) {
	current, target, store := themeComponentActivationFixture(t, "preflight")
	injected := errors.New("component graph conflict")
	components := &recordingThemeComponentRegistry{validateErr: injected}
	pages := &themeActivationApprovalRegistry{}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", LocalRuntimeManager{},
		WithPageRegistry(pages), WithComponentRegistry(components),
	)

	_, err := service.ActivateThemeFromPreview(
		context.Background(), extensionManager(), target.ID, exactThemeComponentActivationInput(current, target),
	)
	if !errors.Is(err, ErrBuildFailed) || !strings.Contains(err.Error(), injected.Error()) {
		t.Fatalf("preflight error=%v", err)
	}
	if store.activeThemeID != current.ID || store.activateThemeExactCalls != 0 || pages.ordinary != 0 {
		t.Fatalf("preflight changed state: active=%q dbCalls=%d pageCalls=%d", store.activeThemeID, store.activateThemeExactCalls, pages.ordinary)
	}
	if len(components.calls) != 1 || components.calls[0].kind != "validate" ||
		components.calls[0].targetID != target.ID || components.calls[0].sourceID != current.ID {
		t.Fatalf("component preflight calls=%#v", components.calls)
	}
}

func TestServiceThemeComponentPublishFailureCompensatesDatabaseBeforePage(t *testing.T) {
	current, target, store := themeComponentActivationFixture(t, "component-failure")
	injected := errors.New("component publication failed")
	components := &recordingThemeComponentRegistry{publishErrs: []error{injected, nil}}
	pages := &themeActivationApprovalRegistry{}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", LocalRuntimeManager{},
		WithPageRegistry(pages), WithComponentRegistry(components),
	)

	_, err := service.ActivateThemeFromPreview(
		context.Background(), extensionManager(), target.ID, exactThemeComponentActivationInput(current, target),
	)
	if !errors.Is(err, ErrBuildFailed) || !strings.Contains(err.Error(), injected.Error()) {
		t.Fatalf("publication error=%v", err)
	}
	if store.activeThemeID != current.ID || pages.ordinary != 0 ||
		store.latestThemePublication.Reason != ThemeRuntimePublicationCompensation {
		t.Fatalf("failed publication split state: active=%q pageCalls=%d publication=%#v", store.activeThemeID, pages.ordinary, store.latestThemePublication)
	}
	if len(components.calls) != 3 || components.calls[1].kind != "publish" ||
		components.calls[1].targetID != target.ID || components.calls[1].sourceID != current.ID ||
		components.calls[2].kind != "publish" || components.calls[2].targetID != current.ID ||
		components.calls[2].sourceID != target.ID ||
		components.calls[2].revision != store.latestThemePublication.Revision {
		t.Fatalf("component compensation calls=%#v publication=%#v", components.calls, store.latestThemePublication)
	}
}

func TestServicePageFailureCompensatesExactComponentPublication(t *testing.T) {
	current, target, store := themeComponentActivationFixture(t, "page-failure")
	components := &recordingThemeComponentRegistry{}
	pages := &themeActivationApprovalRegistry{err: errors.New("page publication failed")}
	service := NewServiceWithOptions(
		store, t.TempDir(), "", LocalRuntimeManager{},
		WithPageRegistry(pages), WithComponentRegistry(components),
	)

	_, err := service.ActivateThemeFromPreview(
		context.Background(), extensionManager(), target.ID, exactThemeComponentActivationInput(current, target),
	)
	if !errors.Is(err, ErrBuildFailed) || store.activeThemeID != current.ID || pages.ordinary != 1 {
		t.Fatalf("page failure error=%v active=%q calls=%d", err, store.activeThemeID, pages.ordinary)
	}
	if len(components.calls) != 3 || components.calls[1].targetID != target.ID ||
		components.calls[2].targetID != current.ID || components.calls[2].sourceID != target.ID ||
		components.calls[2].revision <= components.calls[1].revision {
		t.Fatalf("exact component rollback calls=%#v", components.calls)
	}
}

func TestApplyThemeRuntimePublicationSynchronizesAndRollsBackComponentRegistry(t *testing.T) {
	current, target, store := themeComponentActivationFixture(t, "watcher")
	current.Status = StatusDisabled
	target.ID = DefaultThemeID
	target.Status = StatusEnabled
	store.items = map[string]Extension{current.ID: current, target.ID: target}
	store.activeThemeID = target.ID
	publication := ThemeRuntimePublication{
		Revision: 71, DesiredState: ThemeRuntimePublicationActive,
		ThemeID: target.ID, ThemeVersion: target.Version, PackageDigest: target.PackageDigest,
		SourceThemeID: current.ID, SourceThemeVersion: current.Version, SourcePackageDigest: current.PackageDigest,
		Reason: ThemeRuntimePublicationActivation,
	}

	t.Run("publish", func(t *testing.T) {
		components := &recordingThemeComponentRegistry{}
		pages := &themeActivationApprovalRegistry{}
		service := NewServiceWithOptions(
			store, t.TempDir(), "", LocalRuntimeManager{},
			WithPageRegistry(pages), WithComponentRegistry(components),
		)
		if err := service.ApplyThemeRuntimePublication(context.Background(), publication); err != nil {
			t.Fatal(err)
		}
		if pages.ordinary != 1 || len(components.calls) != 2 ||
			components.calls[1].kind != "publish" || components.calls[1].revision != publication.Revision ||
			components.calls[1].targetID != target.ID || components.calls[1].sourceID != current.ID {
			t.Fatalf("watcher publication pageCalls=%d componentCalls=%#v", pages.ordinary, components.calls)
		}
	})

	t.Run("page failure rollback", func(t *testing.T) {
		components := &recordingThemeComponentRegistry{}
		pages := &themeActivationApprovalRegistry{err: errors.New("watcher page failure")}
		service := NewServiceWithOptions(
			store, t.TempDir(), "", LocalRuntimeManager{},
			WithPageRegistry(pages), WithComponentRegistry(components),
		)
		err := service.ApplyThemeRuntimePublication(context.Background(), publication)
		if !errors.Is(err, ErrThemeRuntimeApplyFailed) || len(components.calls) != 3 ||
			components.calls[2].kind != "rollback" || components.calls[2].revision != publication.Revision ||
			components.calls[2].targetID != current.ID || components.calls[2].sourceID != target.ID {
			t.Fatalf("watcher rollback error=%v calls=%#v", err, components.calls)
		}
	})

	t.Run("uninstalled source exact fence", func(t *testing.T) {
		missingSourceStore := newFakeExtensionStore(map[string]Extension{target.ID: target})
		missingSourceStore.activeThemeID = target.ID
		components := &recordingThemeComponentRegistry{}
		pages := &themeActivationApprovalRegistry{}
		service := NewServiceWithOptions(
			missingSourceStore, t.TempDir(), "", LocalRuntimeManager{},
			WithPageRegistry(pages), WithComponentRegistry(components),
		)
		if err := service.ApplyThemeRuntimePublication(context.Background(), publication); err != nil {
			t.Fatal(err)
		}
		if len(components.calls) != 2 || components.calls[1].sourceID != current.ID ||
			components.calls[1].targetID != target.ID {
			t.Fatalf("uninstalled source fence calls=%#v", components.calls)
		}
	})
}

func themeComponentActivationFixture(
	t *testing.T,
	prefix string,
) (Extension, Extension, *fakeExtensionStore) {
	t.Helper()
	current := exactThemeRuntimeExtensionFixture(t, prefix+".current", "/current")
	current.Status = StatusEnabled
	target := exactThemeRuntimeExtensionFixture(t, prefix+".target", "/target")
	target.Status = StatusInstalled
	store := newFakeExtensionStore(map[string]Extension{current.ID: current, target.ID: target})
	store.activeThemeID = current.ID
	return current, target, store
}

func exactThemeComponentActivationInput(current, target Extension) ThemeActivationInput {
	return ThemeActivationInput{
		Version: target.Version, PackageDigest: target.PackageDigest,
		CurrentThemeID: current.ID, CurrentThemeVersion: current.Version,
		CurrentThemeDigest: current.PackageDigest,
	}
}
