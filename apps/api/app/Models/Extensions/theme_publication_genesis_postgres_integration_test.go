package extensions

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEnsureInitialThemeRuntimePublicationImportsExactActiveThemeOnce(t *testing.T) {
	fixture := newThemePublicationPGFixtureWithoutPublication(t, "genesis_exact")
	active := fixture.activeTheme()
	actorID := fixture.addUser("genesis-approval")
	pageID := fixture.prefix + ".approved"
	fixture.pageIDs = append(fixture.pageIDs, pageID)
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO page_provider_bindings (
			page_id, extension_id, contribution_id, version, package_digest, approved_by
		) VALUES ($1, $2, 'genesis-approved', $3, $4, $5)
	`, pageID, active.ID, active.Version, active.PackageDigest, actorID); err != nil {
		t.Fatal(err)
	}

	publication, err := fixture.store.EnsureInitialThemeRuntimePublication(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if publication.Revision <= 0 || publication.CreatedAt.IsZero() ||
		publication.DesiredState != ThemeRuntimePublicationActive ||
		publication.ThemeID != active.ID || publication.ThemeVersion != active.Version ||
		publication.PackageDigest != active.PackageDigest ||
		publication.Reason != ThemeRuntimePublicationStartupRepair ||
		!publication.CoreReplacementsApproved || publication.ActorUserID != actorID ||
		publication.SourceThemeID != "" || publication.SourceThemeVersion != "" ||
		publication.SourcePackageDigest != "" || publication.SourceCoreReplacementsApproved ||
		publication.SourceActorUserID != 0 {
		t.Fatalf("initial publication=%+v active=%+v", publication, active)
	}
	assertThemeRuntimePublicationCount(t, fixture, 1)

	// Existing immutable authority must short-circuit before consulting mutable
	// extension state. Holding the mutable table proves replay is read-only.
	mutableLock, err := fixture.pool.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mutableLock.Exec(fixture.ctx, `LOCK TABLE extensions IN ACCESS EXCLUSIVE MODE`); err != nil {
		_ = mutableLock.Rollback(fixture.ctx)
		t.Fatal(err)
	}
	replayCtx, cancelReplay := context.WithTimeout(fixture.ctx, 2*time.Second)
	replayed, replayErr := fixture.store.EnsureInitialThemeRuntimePublication(replayCtx)
	cancelReplay()
	if rollbackErr := mutableLock.Rollback(fixture.ctx); rollbackErr != nil {
		t.Fatal(rollbackErr)
	}
	if replayErr != nil || !sameThemeRuntimePublication(replayed, publication) {
		t.Fatalf("replayed=%+v error=%v", replayed, replayErr)
	}
	assertThemeRuntimePublicationCount(t, fixture, 1)
}

func TestEnsureInitialThemeRuntimePublicationConcurrentProducersCreateOneGenesis(t *testing.T) {
	fixture := newThemePublicationPGFixtureWithoutPublication(t, "genesis_concurrent")
	secondPool, err := pgxpool.NewWithConfig(fixture.ctx, fixture.pool.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer secondPool.Close()
	stores := []*PostgresStore{fixture.store, NewPostgresStore(secondPool)}

	const producers = 32
	start := make(chan struct{})
	results := make(chan ThemeRuntimePublication, producers)
	errorsFound := make(chan error, producers)
	ctx, cancel := context.WithTimeout(fixture.ctx, 10*time.Second)
	defer cancel()
	var wait sync.WaitGroup
	for index := range producers {
		wait.Add(1)
		go func(store *PostgresStore) {
			defer wait.Done()
			<-start
			publication, err := store.EnsureInitialThemeRuntimePublication(ctx)
			if err != nil {
				errorsFound <- err
				return
			}
			results <- publication
		}(stores[index%len(stores)])
	}
	close(start)
	wait.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Errorf("concurrent genesis failed: %v", err)
	}
	var expected ThemeRuntimePublication
	for publication := range results {
		if expected.Revision == 0 {
			expected = publication
			continue
		}
		if !sameThemeRuntimePublication(publication, expected) {
			t.Errorf("concurrent publication=%+v want=%+v", publication, expected)
		}
	}
	if expected.Revision <= 0 {
		t.Fatal("concurrent producers returned no publication")
	}
	assertThemeRuntimePublicationCount(t, fixture, 1)
}

func TestEnsureInitialThemeRuntimePublicationDoesNotOverwriteActivation(t *testing.T) {
	fixture := newThemePublicationPGFixtureWithoutPublication(t, "genesis_activation_race")
	current := fixture.activeTheme()
	target := fixture.saveTheme("target", "2.0.0", strings.Repeat("a", 64))
	secondPool, err := pgxpool.NewWithConfig(fixture.ctx, fixture.pool.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer secondPool.Close()

	type genesisResult struct {
		publication ThemeRuntimePublication
		err         error
	}
	type activationResult struct {
		result ThemeActivationResult
		err    error
	}
	start := make(chan struct{})
	genesis := make(chan genesisResult, 1)
	activation := make(chan activationResult, 1)
	go func() {
		<-start
		publication, ensureErr := fixture.store.EnsureInitialThemeRuntimePublication(fixture.ctx)
		genesis <- genesisResult{publication: publication, err: ensureErr}
	}()
	go func() {
		<-start
		result, activateErr := NewPostgresStore(secondPool).ActivateThemeExact(
			fixture.ctx, target.ID, exactThemeActivationInput(current, target, 99, false),
		)
		activation <- activationResult{result: result, err: activateErr}
	}()
	close(start)
	ensured := <-genesis
	activated := <-activation
	if ensured.err != nil || activated.err != nil {
		t.Fatalf("genesis=%+v activation=%+v", ensured, activated)
	}
	latest, err := fixture.store.LatestThemeRuntimePublication(fixture.ctx)
	if err != nil || !sameThemeRuntimePublication(latest, activated.result.Publication) {
		t.Fatalf("latest=%+v activation=%+v error=%v", latest, activated.result.Publication, err)
	}
	if sameThemeRuntimePublication(ensured.publication, activated.result.Publication) {
		assertThemeRuntimePublicationCount(t, fixture, 1)
		return
	}
	if ensured.publication.ThemeID != current.ID ||
		ensured.publication.Revision >= activated.result.Publication.Revision {
		t.Fatalf("genesis=%+v activation=%+v", ensured.publication, activated.result.Publication)
	}
	assertThemeRuntimePublicationCount(t, fixture, 2)
}

func TestEnsureInitialThemeRuntimePublicationRejectsInvalidMutableState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *themePublicationPGFixture)
	}{
		{
			name: "no active theme",
			mutate: func(t *testing.T, fixture *themePublicationPGFixture) {
				if _, err := fixture.pool.Exec(fixture.ctx, `
					UPDATE extensions SET status = 'installed'
					WHERE type = 'theme' AND status = 'enabled'
				`); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "missing active version",
			mutate: func(t *testing.T, fixture *themePublicationPGFixture) {
				active := fixture.activeTheme()
				if _, err := fixture.pool.Exec(fixture.ctx, `
					UPDATE extensions SET active_version_id = NULL WHERE id = $1
				`, active.ID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "mismatched version owner",
			mutate: func(t *testing.T, fixture *themePublicationPGFixture) {
				active := fixture.activeTheme()
				other := fixture.saveTheme("other", "1.0.0", strings.Repeat("b", 64))
				if _, err := fixture.pool.Exec(fixture.ctx, `
					UPDATE extensions SET active_version_id = $2 WHERE id = $1
				`, active.ID, other.ActiveVersionID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "mismatched manifest",
			mutate: func(t *testing.T, fixture *themePublicationPGFixture) {
				active := fixture.activeTheme()
				if _, err := fixture.pool.Exec(fixture.ctx, `
					UPDATE extension_versions SET manifest = '{}'::jsonb WHERE id = $1
				`, active.ActiveVersionID); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newThemePublicationPGFixtureWithoutPublication(t, "genesis_invalid")
			test.mutate(t, fixture)
			if _, err := fixture.store.EnsureInitialThemeRuntimePublication(fixture.ctx); !errors.Is(err, ErrThemePublicationConflict) {
				t.Fatalf("invalid state error=%v", err)
			}
			assertThemeRuntimePublicationCount(t, fixture, 0)
		})
	}
}

func assertThemeRuntimePublicationCount(t *testing.T, fixture *themePublicationPGFixture, want int) {
	t.Helper()
	var count int
	if err := fixture.pool.QueryRow(
		fixture.ctx, `SELECT count(*) FROM theme_runtime_publications`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("theme runtime publication count=%d want=%d", count, want)
	}
}
