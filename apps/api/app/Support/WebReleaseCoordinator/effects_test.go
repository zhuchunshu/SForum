package webreleasecoordinator

import (
	"context"
	"os"
	"strings"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

func TestFinalizeRevocationsDoesNotUseReservedGrantAlias(t *testing.T) {
	body, err := os.ReadFile("effects.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "AS grant\n") {
		t.Fatal("finalize revocations uses PostgreSQL reserved word grant as an alias")
	}
}

func TestPostgresStoreNextActivationKeepsLatestReadyRelease(t *testing.T) {
	releases := &fakeReleaseStore{items: []extensions.WebRelease{
		{ID: 9, Status: extensions.WebReleaseReady},
		{ID: 8, Status: extensions.WebReleaseReady},
		{ID: 7, Status: extensions.WebReleaseReady},
	}}
	store := NewPostgresStore(nil, releases, nil)
	detail, err := store.NextActivation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != 9 {
		t.Fatalf("expected latest release 9, got %d", detail.ID)
	}
	if len(releases.transitions) != 2 || releases.transitions[0].ID != 8 || releases.transitions[1].ID != 7 {
		t.Fatalf("older releases were not superseded: %#v", releases.transitions)
	}
	for _, transition := range releases.transitions {
		if transition.NextStatus != extensions.WebReleaseSuperseded {
			t.Fatalf("unexpected stale release transition: %#v", transition)
		}
	}
}

type fakeReleaseStore struct {
	items       []extensions.WebRelease
	transitions []extensions.WebReleaseTransitionInput
}

func (s *fakeReleaseStore) ListWebReleases(_ context.Context, input extensions.WebReleaseListInput) (extensions.WebReleasePage, error) {
	if input.Status == extensions.WebReleaseActivating {
		return extensions.WebReleasePage{}, nil
	}
	return extensions.WebReleasePage{Items: append([]extensions.WebRelease(nil), s.items...)}, nil
}

func (s *fakeReleaseStore) WebRelease(_ context.Context, id int64) (extensions.WebReleaseDetail, error) {
	return extensions.WebReleaseDetail{WebRelease: extensions.WebRelease{ID: id, Status: extensions.WebReleaseReady}}, nil
}

func (s *fakeReleaseStore) TransitionWebRelease(_ context.Context, input extensions.WebReleaseTransitionInput) (extensions.WebRelease, error) {
	s.transitions = append(s.transitions, input)
	return extensions.WebRelease{ID: input.ID, Status: input.NextStatus}, nil
}

// fakeLifecycle records ApplyApprovedLifecycleEffect calls for page registry sync tests.
type fakeLifecycle struct {
	calls []struct {
		id     string
		status string
	}
	err error
}

func (f *fakeLifecycle) ApplyApprovedLifecycleEffect(_ context.Context, extensionID string, targetStatus string) error {
	f.calls = append(f.calls, struct {
		id     string
		status string
	}{extensionID, targetStatus})
	return f.err
}

type fakeExtStore struct {
	items map[string]extensions.Extension
}

func (s *fakeExtStore) Get(_ context.Context, id string) (extensions.Extension, error) {
	if e, ok := s.items[id]; ok {
		return e, nil
	}
	return extensions.Extension{}, extensions.ErrExtensionNotFound
}

func TestApplyEffectsUsesLifecycleNotDirectEnable(t *testing.T) {
	life := &fakeLifecycle{}
	exts := &fakeExtStore{items: map[string]extensions.Extension{
		"sforum.page-registry-demo": {
			ID: "sforum.page-registry-demo", Type: extensions.TypePlugin, Status: extensions.StatusDisabled,
		},
	}}
	store := NewPostgresStore(nil, &fakeReleaseStore{}, exts).WithLifecycle(life)
	detail := extensions.WebReleaseDetail{
		Effects: []extensions.WebReleaseExtensionEffect{
			{
				ExtensionID:    "sforum.page-registry-demo",
				TargetStatus:   extensions.StatusEnabled,
				PreviousStatus: extensions.StatusDisabled,
			},
		},
	}
	if err := store.ApplyEffects(context.Background(), detail, true); err != nil {
		t.Fatal(err)
	}
	if len(life.calls) != 1 || life.calls[0].status != extensions.StatusEnabled {
		t.Fatalf("expected lifecycle enable call, got %#v", life.calls)
	}
	// reverse / rollback
	if err := store.ApplyEffects(context.Background(), detail, false); err != nil {
		t.Fatal(err)
	}
	if len(life.calls) != 2 || life.calls[1].status != extensions.StatusDisabled {
		t.Fatalf("expected lifecycle disable on reverse, got %#v", life.calls)
	}
}

func TestApplyEffectsRequiresLifecycle(t *testing.T) {
	store := NewPostgresStore(nil, &fakeReleaseStore{}, &fakeExtStore{items: map[string]extensions.Extension{
		"p": {ID: "p"},
	}})
	err := store.ApplyEffects(context.Background(), extensions.WebReleaseDetail{
		Effects: []extensions.WebReleaseExtensionEffect{{ExtensionID: "p", TargetStatus: extensions.StatusEnabled}},
	}, true)
	if err == nil || !strings.Contains(err.Error(), "lifecycle") {
		t.Fatalf("expected lifecycle required error, got %v", err)
	}
}
