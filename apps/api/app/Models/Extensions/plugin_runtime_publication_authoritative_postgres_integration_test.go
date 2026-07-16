package extensions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRetryablePluginRuntimePublicationError(t *testing.T) {
	for _, code := range []string{"40001", "40P01"} {
		if !retryablePluginRuntimePublicationError(fmt.Errorf("transaction: %w", &pgconn.PgError{Code: code})) {
			t.Fatalf("PostgreSQL %s must be retried", code)
		}
	}
	if retryablePluginRuntimePublicationError(&pgconn.PgError{Code: "23505"}) ||
		retryablePluginRuntimePublicationError(errors.New("not a PostgreSQL transaction error")) {
		t.Fatal("non-serialization errors must not be retried")
	}
}

func TestPublishAuthoritativePluginRuntimeSetSeedsAndReusesEmptyRevision(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_empty")

	seed, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if seed.Revision != 1 || seed.MemberCount != 0 || len(seed.Members) != 0 ||
		seed.Reason != PluginRuntimePublicationStartupReconcile || seed.ActorUserID != 0 {
		t.Fatalf("unexpected empty seed: %+v", seed)
	}

	replayed, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationRecovery, 71,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !samePluginRuntimePublication(replayed, seed) {
		t.Fatalf("same desired set appended or rewrote evidence: seed=%+v replay=%+v", seed, replayed)
	}
	assertPluginRuntimePublicationCount(t, fixture, 1)
}

func TestPublishAuthoritativePluginRuntimeSetTracksCompleteExecutableSet(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_changes")
	seed, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
	)
	if err != nil {
		t.Fatal(err)
	}

	setPluginRuntimeFixtureVersion(t, fixture, "fixture.plugin", 101, StatusEnabled,
		runtimeManifestBody(t, "fixture.plugin", "1.0.0", TypePlugin, "backend/plugin"))
	enabled, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationEnable, 41,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertPluginRuntimePublicationMembers(t, enabled, fixture.firstMember())
	if enabled.Revision <= seed.Revision || enabled.Reason != PluginRuntimePublicationEnable || enabled.ActorUserID != 41 {
		t.Fatalf("enable evidence mismatch: %+v", enabled)
	}

	replayed, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
	)
	if err != nil || !samePluginRuntimePublication(replayed, enabled) {
		t.Fatalf("startup replay replaced lifecycle evidence: publication=%+v err=%v", replayed, err)
	}

	upgradeManifest := runtimeManifestBody(t, "fixture.plugin", "1.1.0", TypePlugin, "backend/plugin")
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO extension_versions (id, extension_id, version, package_digest, manifest)
		VALUES (104, 'fixture.plugin', '1.1.0', repeat('e', 64), $1::jsonb)
	`, string(upgradeManifest)); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extensions SET active_version_id = 104 WHERE id = 'fixture.plugin'
	`); err != nil {
		t.Fatal(err)
	}
	upgraded, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationUpgrade, 42,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertPluginRuntimePublicationMembers(t, upgraded, PluginRuntimeMember{
		ExtensionID: "fixture.plugin", ExtensionVersionID: 104,
		ExtensionVersion: "1.1.0", PackageDigest: strings.Repeat("e", 64),
	})

	setPluginRuntimeFixtureVersion(t, fixture, "second.plugin", 102, StatusEnabled,
		runtimeManifestBody(t, "second.plugin", "2.0.0", TypePlugin, "backend/plugin"))
	multiple, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationEnable, 43,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertPluginRuntimePublicationMembers(t, multiple,
		PluginRuntimeMember{
			ExtensionID: "fixture.plugin", ExtensionVersionID: 104,
			ExtensionVersion: "1.1.0", PackageDigest: strings.Repeat("e", 64),
		},
		fixture.secondMember(),
	)

	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extensions SET status = 'disabled' WHERE id = 'fixture.plugin'
	`); err != nil {
		t.Fatal(err)
	}
	disabled, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationDisable, 44,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertPluginRuntimePublicationMembers(t, disabled, fixture.secondMember())

	latest, err := fixture.store.LatestPluginRuntimePublication(fixture.ctx)
	if err != nil || !samePluginRuntimePublication(latest, disabled) {
		t.Fatalf("latest publication does not equal final database state: latest=%+v err=%v", latest, err)
	}
}

func TestPublishAuthoritativePluginRuntimeSetExcludesStaticPluginsAndThemes(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_exclusions")
	setPluginRuntimeFixtureVersion(t, fixture, "fixture.plugin", 101, StatusEnabled,
		runtimeManifestBody(t, "fixture.plugin", "1.0.0", TypePlugin, "backend/plugin"))
	setPluginRuntimeFixtureVersion(t, fixture, "second.plugin", 102, StatusEnabled,
		runtimeManifestBody(t, "second.plugin", "2.0.0", TypePlugin, ""))
	setPluginRuntimeFixtureVersion(t, fixture, "fixture.theme", 103, StatusEnabled,
		runtimeManifestBody(t, "fixture.theme", "1.0.0", TypeTheme, ""))

	publication, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertPluginRuntimePublicationMembers(t, publication, fixture.firstMember())
}

func TestPublishAuthoritativePluginRuntimeSetRejectsInvalidActiveProjection(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *pluginRuntimePublicationPGFixture)
	}{
		{
			name: "missing active version",
			setup: func(t *testing.T, fixture *pluginRuntimePublicationPGFixture) {
				_, err := fixture.pool.Exec(fixture.ctx, `
					UPDATE extensions SET status = 'enabled', active_version_id = NULL
					WHERE id = 'fixture.plugin'
				`)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "corrupt manifest shape",
			setup: func(t *testing.T, fixture *pluginRuntimePublicationPGFixture) {
				setPluginRuntimeFixtureVersion(t, fixture, "fixture.plugin", 101, StatusEnabled, []byte(`{"id":42}`))
			},
		},
		{
			name: "manifest identity mismatch",
			setup: func(t *testing.T, fixture *pluginRuntimePublicationPGFixture) {
				setPluginRuntimeFixtureVersion(t, fixture, "fixture.plugin", 101, StatusEnabled,
					runtimeManifestBody(t, "other.plugin", "1.0.0", TypePlugin, "backend/plugin"))
			},
		},
		{
			name: "active version belongs to another extension",
			setup: func(t *testing.T, fixture *pluginRuntimePublicationPGFixture) {
				setPluginRuntimeFixtureVersion(t, fixture, "second.plugin", 102, StatusInstalled,
					runtimeManifestBody(t, "second.plugin", "2.0.0", TypePlugin, "backend/plugin"))
				_, err := fixture.pool.Exec(fixture.ctx, `
					UPDATE extensions SET status = 'enabled', active_version_id = 102
					WHERE id = 'fixture.plugin'
				`)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPluginRuntimePublicationPGFixture(t, fmt.Sprintf("authoritative_invalid_%d", index))
			test.setup(t, fixture)
			_, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
				fixture.ctx, PluginRuntimePublicationStartupReconcile, 0,
			)
			if !errors.Is(err, ErrPluginRuntimePublicationConflict) {
				t.Fatalf("expected fail-closed conflict, got %v", err)
			}
			assertPluginRuntimePublicationCount(t, fixture, 0)
		})
	}
}

func TestPublishAuthoritativePluginRuntimeSetConcurrentProducersConverge(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_concurrent")
	setPluginRuntimeFixtureVersion(t, fixture, "fixture.plugin", 101, StatusEnabled,
		runtimeManifestBody(t, "fixture.plugin", "1.0.0", TypePlugin, "backend/plugin"))
	setPluginRuntimeFixtureVersion(t, fixture, "second.plugin", 102, StatusEnabled,
		runtimeManifestBody(t, "second.plugin", "2.0.0", TypePlugin, "backend/plugin"))

	const producers = 16
	results := make(chan PluginRuntimePublication, producers)
	errorsFound := make(chan error, producers)
	publishCtx, cancelPublish := context.WithTimeout(fixture.ctx, 10*time.Second)
	defer cancelPublish()
	var wait sync.WaitGroup
	for index := 0; index < producers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			publication, err := fixture.store.PublishAuthoritativePluginRuntimeSet(
				publishCtx, PluginRuntimePublicationStartupReconcile, 0,
			)
			if err != nil {
				errorsFound <- err
				return
			}
			results <- publication
		}()
	}
	done := make(chan struct{})
	go func() {
		wait.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		cancelPublish()
		t.Fatal("concurrent producers did not finish")
	}
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Errorf("concurrent producer failed: %v", err)
	}
	for publication := range results {
		if publication.Revision != 1 {
			t.Errorf("concurrent producer created revision %d, want 1", publication.Revision)
		}
		assertPluginRuntimePublicationMembers(t, publication, fixture.firstMember(), fixture.secondMember())
	}
	assertPluginRuntimePublicationCount(t, fixture, 1)
}

func TestLoadAuthoritativePluginRuntimeMembersLocksCompleteProjection(t *testing.T) {
	tests := []struct {
		name   string
		update string
	}{
		{
			name:   "extension state",
			update: `UPDATE extensions SET status = 'disabled' WHERE id = 'fixture.plugin'`,
		},
		{
			name: "version manifest",
			update: `UPDATE extension_versions
				SET manifest = manifest || '{"description":"changed"}'::jsonb
				WHERE id = 101`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_projection_lock")
			setPluginRuntimeFixtureVersion(t, fixture, "fixture.plugin", 101, StatusEnabled,
				runtimeManifestBody(t, "fixture.plugin", "1.0.0", TypePlugin, "backend/plugin"))

			tx, err := fixture.pool.BeginTx(fixture.ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			members, err := loadAuthoritativePluginRuntimeMembers(fixture.ctx, tx)
			if err != nil || len(members) != 1 || members[0] != fixture.firstMember() {
				_ = tx.Rollback(fixture.ctx)
				t.Fatalf("members=%+v error=%v", members, err)
			}

			updater, err := fixture.pool.Acquire(fixture.ctx)
			if err != nil {
				_ = tx.Rollback(fixture.ctx)
				t.Fatal(err)
			}
			defer updater.Release()
			var updaterPID int32
			if err := updater.QueryRow(fixture.ctx, `SELECT pg_backend_pid()`).Scan(&updaterPID); err != nil {
				_ = tx.Rollback(fixture.ctx)
				t.Fatal(err)
			}

			updateCtx, cancelUpdate := context.WithTimeout(fixture.ctx, 5*time.Second)
			defer cancelUpdate()
			updated := make(chan error, 1)
			go func() {
				_, updateErr := updater.Exec(updateCtx, test.update)
				updated <- updateErr
			}()

			blocked := false
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				select {
				case updateErr := <-updated:
					_ = tx.Rollback(fixture.ctx)
					t.Fatalf("projection mutation completed before transaction release: %v", updateErr)
				default:
				}
				var blockerCount int
				if err := fixture.pool.QueryRow(fixture.ctx, `
					SELECT cardinality(pg_blocking_pids($1))
				`, updaterPID).Scan(&blockerCount); err != nil {
					_ = tx.Rollback(fixture.ctx)
					t.Fatal(err)
				}
				if blockerCount > 0 {
					blocked = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !blocked {
				_ = tx.Rollback(fixture.ctx)
				<-updated
				t.Fatal("projection mutation never blocked on the producer transaction")
			}
			if err := tx.Commit(fixture.ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-updated; err != nil {
				t.Fatalf("projection mutation did not resume after commit: %v", err)
			}
		})
	}
}

func TestReleasePluginRuntimeProducerConnectionDropsUnownedSession(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_unlock_false")
	connection, err := fixture.pool.Acquire(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	// No advisory lock was acquired on this session, so PostgreSQL returns
	// false. The release path must close it instead of poisoning the pool.
	destroyed, err := releasePluginRuntimeProducerConnection(fixture.ctx, connection, true)
	if err != nil || !destroyed {
		t.Fatalf("unlock false destroyed=%t error=%v", destroyed, err)
	}
	if acquired := fixture.pool.Stat().AcquiredConns(); acquired != 0 {
		t.Fatalf("hijacked connection remains acquired: %d", acquired)
	}
	if err := fixture.pool.Ping(fixture.ctx); err != nil {
		t.Fatalf("pool did not replace closed producer connection: %v", err)
	}
}

func TestFinalizePluginRuntimeProducerResultRecoversWithSingleConnection(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_single_connection")
	expected, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name        string
		acquireLock bool
	}{
		{name: "released session", acquireLock: true},
		{name: "destroyed session", acquireLock: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := fixture.pool.Config().Copy()
			config.MinConns = 0
			config.MaxConns = 1
			pool, err := pgxpool.NewWithConfig(fixture.ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			store := NewPostgresStore(pool)
			connection, err := pool.Acquire(fixture.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if test.acquireLock {
				if _, err := connection.Exec(fixture.ctx, `
					SELECT pg_advisory_lock(hashtextextended($1, 0))
				`, pluginRuntimeDesiredSetLock); err != nil {
					connection.Release()
					t.Fatal(err)
				}
			}

			recovered, err := store.finalizePluginRuntimeProducerResult(
				fixture.ctx,
				connection,
				true,
				expected,
				&pluginRuntimePublicationCommitUnknown{cause: errors.New("ambiguous commit transport")},
			)
			if err != nil || !samePluginRuntimePublication(recovered, expected) {
				t.Fatalf("recovered=%+v error=%v", recovered, err)
			}
			if acquired := pool.Stat().AcquiredConns(); acquired != 0 {
				t.Fatalf("producer connection remained acquired: %d", acquired)
			}
		})
	}
}

type pluginRuntimeCloseErrorConn struct {
	net.Conn
	err              error
	underlyingClosed *atomic.Bool
}

func (c *pluginRuntimeCloseErrorConn) Close() error {
	underlyingErr := c.Conn.Close()
	c.underlyingClosed.Store(true)
	return errors.Join(underlyingErr, c.err)
}

func TestFinalizePluginRuntimeProducerResultRecoversAfterDestroyedCloseError(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_close_error")
	stored, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		expected PluginRuntimePublication
		found    bool
	}{
		{name: "exact revision committed", expected: stored, found: true},
		{name: "exact revision absent", expected: func() PluginRuntimePublication {
			missing := stored
			missing.Revision++
			return missing
		}()},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := fixture.pool.Config().Copy()
			config.MinConns = 0
			config.MaxConns = 1
			originalDial := config.ConnConfig.DialFunc
			closeErr := errors.New("injected producer connection close error")
			var dialCount atomic.Int32
			var underlyingClosed atomic.Bool
			config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
				connection, dialErr := originalDial(ctx, network, address)
				if dialErr == nil && dialCount.Add(1) == 1 {
					return &pluginRuntimeCloseErrorConn{
						Conn: connection, err: closeErr, underlyingClosed: &underlyingClosed,
					}, nil
				}
				return connection, dialErr
			}
			pool, err := pgxpool.NewWithConfig(fixture.ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()

			producer, err := pool.Acquire(fixture.ctx)
			if err != nil {
				t.Fatal(err)
			}
			commitErr := errors.New("ambiguous commit transport")
			recovered, err := NewPostgresStore(pool).finalizePluginRuntimeProducerResult(
				fixture.ctx,
				producer,
				true,
				test.expected,
				&pluginRuntimePublicationCommitUnknown{cause: commitErr},
			)
			if test.found {
				if err != nil || !samePluginRuntimePublication(recovered, stored) {
					t.Fatalf("recovered=%+v error=%v", recovered, err)
				}
			} else if err == nil || !errors.Is(err, commitErr) ||
				!errors.Is(err, closeErr) || !errors.Is(err, ErrPluginRuntimePublicationNotFound) {
				t.Fatalf("missing recovery error=%v", err)
			}
			if acquired := pool.Stat().AcquiredConns(); acquired != 0 {
				t.Fatalf("producer connection remained acquired: %d", acquired)
			}
			if !underlyingClosed.Load() {
				t.Fatal("producer wrapper returned its close sentinel before closing the underlying connection")
			}
			if got := dialCount.Load(); got < 2 {
				t.Fatalf("exact recovery did not use a replacement physical connection: dials=%d", got)
			}
		})
	}
}

func TestPublishAuthoritativePluginRuntimeSetCancelWhileWaitingDestroysSession(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_cancel_lock")
	blocker, err := fixture.pool.Acquire(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	locked := false
	defer func() {
		if locked {
			_, _ = blocker.Exec(context.Background(), `
				SELECT pg_advisory_unlock(hashtextextended($1, 0))
			`, pluginRuntimeDesiredSetLock)
		}
		blocker.Release()
	}()
	var blockerPID int32
	if err := blocker.QueryRow(fixture.ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID); err != nil {
		t.Fatal(err)
	}
	if _, err := blocker.Exec(fixture.ctx, `
		SELECT pg_advisory_lock(hashtextextended($1, 0))
	`, pluginRuntimeDesiredSetLock); err != nil {
		t.Fatal(err)
	}
	locked = true

	publishCtx, cancelPublish := context.WithCancel(fixture.ctx)
	result := make(chan error, 1)
	go func() {
		_, publishErr := fixture.store.PublishAuthoritativePluginRuntimeSet(
			publishCtx, PluginRuntimePublicationStartupReconcile, 0,
		)
		result <- publishErr
	}()

	var waitingPID int32
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT waiter.pid
			FROM pg_locks AS held
			JOIN pg_locks AS waiter
			  ON waiter.locktype = held.locktype
			 AND waiter.database IS NOT DISTINCT FROM held.database
			 AND waiter.classid IS NOT DISTINCT FROM held.classid
			 AND waiter.objid IS NOT DISTINCT FROM held.objid
			 AND waiter.objsubid IS NOT DISTINCT FROM held.objsubid
			JOIN pg_stat_activity AS activity ON activity.pid = waiter.pid
			WHERE held.pid = $1
			  AND held.locktype = 'advisory'
			  AND held.granted
			  AND NOT waiter.granted
			  AND activity.application_name = $2
			LIMIT 1
		`, blockerPID, fixture.schema).Scan(&waitingPID)
		if err == nil {
			break
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			cancelPublish()
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if waitingPID == 0 {
		cancelPublish()
		<-result
		t.Fatal("producer never blocked on the advisory lock")
	}

	cancelPublish()
	select {
	case publishErr := <-result:
		if !errors.Is(publishErr, context.Canceled) {
			t.Fatalf("cancelled producer error=%v", publishErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled producer did not return")
	}

	backendClosed := false
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var exists bool
		if err := fixture.pool.QueryRow(fixture.ctx, `
			SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = $1)
		`, waitingPID).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			backendClosed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !backendClosed {
		t.Fatalf("cancelled producer backend %d remained alive", waitingPID)
	}
	if acquired := fixture.pool.Stat().AcquiredConns(); acquired != 1 {
		t.Fatalf("pool retained cancelled producer connection: acquired=%d", acquired)
	}
}

func TestRecoverAuthoritativePluginRuntimePublicationUsesExactCommittedRevision(t *testing.T) {
	fixture := newPluginRuntimePublicationPGFixture(t, "authoritative_commit_recovery")
	expected, err := fixture.store.publishPluginRuntimePublication(
		fixture.ctx, PluginRuntimePublicationStartupReconcile, 0, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := fixture.store.recoverAuthoritativePluginRuntimePublication(
		fixture.ctx, expected, errors.New("ambiguous commit transport"),
	)
	if err != nil || !samePluginRuntimePublication(recovered, expected) {
		t.Fatalf("recovered=%+v error=%v", recovered, err)
	}

	missing := expected
	missing.Revision++
	if _, err := fixture.store.recoverAuthoritativePluginRuntimePublication(
		fixture.ctx, missing, errors.New("commit did not happen"),
	); err == nil || !strings.Contains(err.Error(), "commit did not happen") {
		t.Fatalf("missing commit recovery error=%v", err)
	}
}

func runtimeManifestBody(t *testing.T, id, version, extensionType, backendEntry string) []byte {
	t.Helper()
	manifest := Manifest{
		ID: id, Name: "Runtime Fixture", Description: "Runtime publication fixture.",
		URL: "https://example.com/runtime-fixture", Author: ManifestAuthor{Name: "SForum"},
		Version: version, Type: extensionType, SForumVersion: "^1.0.0",
	}
	if backendEntry != "" {
		manifest.Backend = ManifestBackend{Entry: backendEntry, RPC: "hashicorp-go-plugin"}
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func setPluginRuntimeFixtureVersion(
	t *testing.T,
	fixture *pluginRuntimePublicationPGFixture,
	extensionID string,
	versionID int64,
	status string,
	manifest []byte,
) {
	t.Helper()
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extension_versions SET manifest = $3::jsonb
		WHERE id = $1 AND extension_id = $2
	`, versionID, extensionID, string(manifest)); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE extensions SET status = $3, active_version_id = $1
		WHERE id = $2
	`, versionID, extensionID, status); err != nil {
		t.Fatal(err)
	}
}

func assertPluginRuntimePublicationMembers(
	t *testing.T,
	publication PluginRuntimePublication,
	members ...PluginRuntimeMember,
) {
	t.Helper()
	canonical, digest, err := canonicalPluginRuntimeMembers(members)
	if err != nil {
		t.Fatal(err)
	}
	if publication.MemberCount != len(canonical) || publication.MembersDigest != digest ||
		len(publication.Members) != len(canonical) {
		t.Fatalf("publication set mismatch: publication=%+v members=%+v", publication, canonical)
	}
	for index := range canonical {
		if publication.Members[index] != canonical[index] {
			t.Fatalf("publication member[%d]=%+v, want %+v", index, publication.Members[index], canonical[index])
		}
	}
}

func assertPluginRuntimePublicationCount(
	t *testing.T,
	fixture *pluginRuntimePublicationPGFixture,
	want int,
) {
	t.Helper()
	var count int
	if err := fixture.pool.QueryRow(fixture.ctx, `SELECT count(*) FROM plugin_runtime_publications`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("publication count=%d, want %d", count, want)
	}
}
