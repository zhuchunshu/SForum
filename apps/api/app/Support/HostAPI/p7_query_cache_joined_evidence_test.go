package hostapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const p7QueryCacheJoinedEvidenceTable = "p7_query_cache_joined_evidence"

type p7QueryCacheJoinedRedisEvidence struct {
	staleKey    string
	staleDigest string
	liveKey     string
	liveDigest  string
}

type p7QueryCacheJoinedRedisEnvelope struct {
	Result struct {
		CacheKey string           `json:"cacheKey"`
		Rows     []map[string]any `json:"rows"`
	} `json:"result"`
}

func (fixture *p7QueryCacheJoinedFixture) createOwnershipEvidence(t *testing.T, token string) {
	t.Helper()
	if _, err := fixture.pool.Exec(fixture.ctx, `
		CREATE TABLE `+p7QueryCacheJoinedEvidenceTable+` (
			seed TEXT PRIMARY KEY,
			ownership_token TEXT NOT NULL,
			redis_run_id TEXT,
			stale_redis_key TEXT,
			stale_redis_digest TEXT,
			live_redis_key TEXT,
			live_redis_digest TEXT
		)
	`); err != nil {
		t.Fatalf("create joined Query ownership evidence: %v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO `+p7QueryCacheJoinedEvidenceTable+` (seed, ownership_token)
		VALUES ($1, $2)
	`, fixture.seed, token); err != nil {
		t.Fatalf("record joined Query ownership evidence: %v", err)
	}
}

func (fixture *p7QueryCacheJoinedFixture) requireOwnershipEvidence(t *testing.T, token string) {
	t.Helper()
	matched, err := fixture.matchesOwnershipEvidence(token)
	if err != nil {
		t.Fatalf("load joined Query ownership evidence: %v", err)
	}
	if !matched {
		t.Fatal("joined Query database ownership evidence is missing; refusing destructive cleanup")
	}
}

func (fixture *p7QueryCacheJoinedFixture) matchesOwnershipEvidence(token string) (bool, error) {
	var tableName *string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT to_regclass('public.`+p7QueryCacheJoinedEvidenceTable+`')::text
	`).Scan(&tableName); err != nil {
		return false, err
	}
	if tableName == nil {
		return false, nil
	}
	var stored string
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT ownership_token
		FROM `+p7QueryCacheJoinedEvidenceTable+`
		WHERE seed = $1
	`, fixture.seed).Scan(&stored); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if stored != token {
		return false, fmt.Errorf("ownership token mismatch")
	}
	return true, nil
}

func (fixture *p7QueryCacheJoinedFixture) recordRedisRunID(t *testing.T, runID string) {
	t.Helper()
	result, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE `+p7QueryCacheJoinedEvidenceTable+`
		SET redis_run_id = $2
		WHERE seed = $1
	`, fixture.seed, runID)
	if err != nil {
		t.Fatalf("record joined Query Redis run id: %v", err)
	}
	if result.RowsAffected() != 1 {
		t.Fatalf("joined Query Redis run id rows=%d, want 1", result.RowsAffected())
	}
}

func (fixture *p7QueryCacheJoinedFixture) recordRedisEvidence(
	t *testing.T,
	evidence p7QueryCacheJoinedRedisEvidence,
) {
	t.Helper()
	result, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE `+p7QueryCacheJoinedEvidenceTable+`
		SET stale_redis_key = $2,
		    stale_redis_digest = $3,
		    live_redis_key = $4,
		    live_redis_digest = $5
		WHERE seed = $1
	`, fixture.seed, evidence.staleKey, evidence.staleDigest, evidence.liveKey, evidence.liveDigest)
	if err != nil {
		t.Fatalf("record joined Query Redis envelope evidence: %v", err)
	}
	if result.RowsAffected() != 1 {
		t.Fatalf("joined Query Redis envelope evidence rows=%d, want 1", result.RowsAffected())
	}
}

func (fixture *p7QueryCacheJoinedFixture) requireRedisRestartEvidence(
	t *testing.T,
	ctx context.Context,
	client *redis.Client,
	currentRunID string,
) {
	t.Helper()
	var seededRunID string
	var evidence p7QueryCacheJoinedRedisEvidence
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT redis_run_id, stale_redis_key, stale_redis_digest,
		       live_redis_key, live_redis_digest
		FROM `+p7QueryCacheJoinedEvidenceTable+`
		WHERE seed = $1
	`, fixture.seed).Scan(
		&seededRunID, &evidence.staleKey, &evidence.staleDigest,
		&evidence.liveKey, &evidence.liveDigest,
	); err != nil {
		t.Fatalf("load joined Query Redis restart evidence: %v", err)
	}
	if seededRunID == "" || currentRunID == "" || seededRunID == currentRunID {
		t.Fatalf("Redis process was not restarted: seed_run_id=%q verify_run_id=%q", seededRunID, currentRunID)
	}
	requireP7QueryCacheJoinedRedisValue(t, ctx, client, evidence.staleKey, evidence.staleDigest)
	requireP7QueryCacheJoinedRedisValue(t, ctx, client, evidence.liveKey, evidence.liveDigest)
}

func snapshotP7QueryCacheJoinedRedisEnvelope(
	t *testing.T,
	ctx context.Context,
	client *redis.Client,
	cacheKey string,
	wantName string,
) (string, string) {
	t.Helper()
	var cursor uint64
	matchedKey := ""
	matchedDigest := ""
	for {
		keys, next, err := client.Scan(ctx, cursor, "*", 256).Result()
		if err != nil {
			t.Fatalf("scan joined Query Redis evidence: %v", err)
		}
		for _, key := range keys {
			encoded, err := client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			var envelope p7QueryCacheJoinedRedisEnvelope
			if json.Unmarshal(encoded, &envelope) != nil || envelope.Result.CacheKey != cacheKey ||
				len(envelope.Result.Rows) != 1 || envelope.Result.Rows[0]["name"] != wantName {
				continue
			}
			if matchedKey != "" {
				t.Fatalf("multiple joined Query Redis envelopes matched cache key %q", cacheKey)
			}
			digest := sha256.Sum256(encoded)
			matchedKey = key
			matchedDigest = hex.EncodeToString(digest[:])
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if matchedKey == "" {
		t.Fatalf("joined Query Redis envelope for cache key %q was not found", cacheKey)
	}
	return matchedKey, matchedDigest
}

func requireP7QueryCacheJoinedRedisValue(
	t *testing.T,
	ctx context.Context,
	client *redis.Client,
	key string,
	wantDigest string,
) {
	t.Helper()
	if strings.TrimSpace(key) == "" || len(wantDigest) != sha256.Size*2 {
		t.Fatal("joined Query Redis evidence is incomplete")
	}
	encoded, err := client.Get(ctx, key).Bytes()
	if err != nil {
		t.Fatalf("load persisted joined Query Redis envelope %q: %v", key, err)
	}
	digest := sha256.Sum256(encoded)
	if got := hex.EncodeToString(digest[:]); got != wantDigest {
		t.Fatalf("persisted joined Query Redis envelope %q digest=%q, want %q", key, got, wantDigest)
	}
}
