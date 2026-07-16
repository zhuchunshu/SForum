package navigationregistry

import (
	"fmt"
	"sync"
	"testing"
)

func TestComposerConcurrentSelectionResetReadersObserveWholeViews(t *testing.T) {
	registry := New()
	alpha := publication("race.alpha", false, 'a')
	alphaReplace := navigation("race.alpha.header.replace", NavigationKindHeader, ActionReplace, CoreHeaderNavigationID, 0)
	alphaReplace.Label = "Alpha"
	alpha.Navigation = []NavigationDeclaration{alphaReplace}
	beta := publication("race.beta", false, 'b')
	betaReplace := navigation("race.beta.header.replace", NavigationKindHeader, ActionReplace, CoreHeaderNavigationID, 0)
	betaReplace.Label = "Beta"
	beta.Navigation = []NavigationDeclaration{betaReplace}
	if _, err := registry.ReplaceAll([]Publication{CorePublication(), beta, alpha}); err != nil {
		t.Fatal(err)
	}
	runtime := newCompositionRuntime()
	composer := NewComposer(registry, runtime, runtime, NewTraceRing(64))

	start := make(chan struct{})
	problems := make(chan error, 1)
	report := func(err error) {
		select {
		case problems <- err:
		default:
		}
	}
	var group sync.WaitGroup
	group.Add(5)
	go func() {
		defer group.Done()
		<-start
		providers := []ProviderRef{
			{ContributionID: alphaReplace.ID, Artifact: alpha.Artifact},
			{ContributionID: betaReplace.ID, Artifact: beta.Artifact},
		}
		for index := 0; index < 200; index++ {
			if index%3 == 2 {
				_, _, err := registry.ResetProvider(ResetProviderRequest{
					ExpectedRevision: registry.Revision(), Family: ProviderFamilyNavigation, TargetID: CoreHeaderNavigationID,
				})
				if err != nil {
					report(fmt.Errorf("reset: %w", err))
					return
				}
				continue
			}
			_, err := registry.SelectProvider(SelectProviderRequest{
				ExpectedRevision: registry.Revision(), Family: ProviderFamilyNavigation, TargetID: CoreHeaderNavigationID,
				Provider: providers[index%2],
			})
			if err != nil {
				report(fmt.Errorf("select: %w", err))
				return
			}
		}
	}()
	for reader := 0; reader < 4; reader++ {
		go func(reader int) {
			defer group.Done()
			<-start
			for index := 0; index < 400; index++ {
				composition, err := composer.Compose(t.Context(), CompositionRequest{Locale: "en-US"})
				if err != nil {
					report(fmt.Errorf("reader %d compose: %w", reader, err))
					return
				}
				header := composedByID(composition.Navigation, CoreHeaderNavigationID)
				if header == nil || composition.Revision == 0 || composition.Digest == "" || composition.CacheKey == "" ||
					(header.Label != "Alpha" && header.Label != "Beta" && header.Label != "Header") {
					report(fmt.Errorf("reader %d partial composition: %#v", reader, composition))
					return
				}
			}
		}(reader)
	}
	close(start)
	group.Wait()
	select {
	case err := <-problems:
		t.Fatal(err)
	default:
	}
}
