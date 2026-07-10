package webreleasecoordinator

import (
	"context"
	"testing"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
)

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
