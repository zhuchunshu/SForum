package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestLockUserAuthorityPairTxOrdersByUserID(t *testing.T) {
	fixture := newIdentityPersistencePGFixture(t)
	installP7JoinedIdentitySchema(t, fixture)
	lowUserID, highUserID := fixture.actorUserID, fixture.targetUserID
	if lowUserID > highUserID {
		lowUserID, highUserID = highUserID, lowUserID
	}

	for _, input := range []struct {
		name         string
		actorUserID  int64
		targetUserID int64
	}{
		{name: "low_to_high", actorUserID: lowUserID, targetUserID: highUserID},
		{name: "high_to_low", actorUserID: highUserID, targetUserID: lowUserID},
	} {
		t.Run(input.name, func(t *testing.T) {
			blockerPool := fixture.newPool("p7-user-pair-blocker-" + input.name)
			blockerTx, err := blockerPool.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			blockerReleased := false
			defer func() {
				if !blockerReleased {
					_ = blockerTx.Rollback(context.Background())
				}
			}()
			if _, err := blockerTx.Exec(t.Context(), `
				SELECT id FROM users WHERE id = $1 FOR UPDATE
			`, highUserID); err != nil {
				t.Fatal(err)
			}

			pairPool := fixture.newPool("p7-user-pair-writer-" + input.name)
			pairConn, err := pairPool.Acquire(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer pairConn.Release()
			var pairPID int32
			if err := pairConn.QueryRow(t.Context(), `SELECT pg_backend_pid()`).Scan(&pairPID); err != nil {
				t.Fatal(err)
			}
			pairTx, err := pairConn.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			pairCommitted := false
			defer func() {
				if !pairCommitted {
					_ = pairTx.Rollback(context.Background())
				}
			}()
			pairResult := make(chan struct {
				lock UserAuthorityPairLock
				err  error
			}, 1)
			go func() {
				lock, lockErr := LockUserAuthorityPairTx(
					t.Context(), pairTx, input.actorUserID, input.targetUserID,
				)
				pairResult <- struct {
					lock UserAuthorityPairLock
					err  error
				}{lock: lock, err: lockErr}
			}()
			assertIdentityPersistenceBackendBlocked(t, fixture.admin, pairPID)

			probePool := fixture.newPool("p7-user-pair-probe-" + input.name)
			probeTx, err := probePool.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			_, probeErr := probeTx.Exec(t.Context(), `
				SELECT id FROM users WHERE id = $1 FOR UPDATE NOWAIT
			`, lowUserID)
			_ = probeTx.Rollback(context.Background())
			var pgErr *pgconn.PgError
			if !errors.As(probeErr, &pgErr) || pgErr.Code != "55P03" {
				t.Fatalf("low user row was not locked first: %v", probeErr)
			}

			if err := blockerTx.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}
			blockerReleased = true
			select {
			case result := <-pairResult:
				if result.err != nil || !result.lock.ActorExists || !result.lock.TargetExists {
					t.Fatalf("pair lock=%#v err=%v", result.lock, result.err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("pair lock did not finish after blocker release")
			}
			if err := pairTx.Commit(t.Context()); err != nil {
				t.Fatal(err)
			}
			pairCommitted = true
		})
	}
}
