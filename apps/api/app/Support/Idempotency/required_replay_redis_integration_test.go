package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRequiredReplayRedisBackendIntegration(t *testing.T) {
	client, ctx := requiredReplayRedisClient(t)
	store := NewStore(NewRedisBackend(client), DefaultTTL).
		WithRequiredReplayCipher(requiredReplayRedisCipher(t))
	unique := strconv.FormatInt(time.Now().UnixNano(), 10)
	scope := requiredReplayTestScope("actor:806:bearer")
	scope.ExtensionID = "redis.replay." + unique
	scope.RouteID = scope.ExtensionID + ".create"
	scope.ContractVersion = scope.RouteID + "@1"

	t.Run("begin complete replay TTL and ciphertext", func(t *testing.T) {
		key := "redis-roundtrip-" + unique
		fullKey := requiredReplayRedisCleanKey(t, client, store, scope, key)
		fingerprint := strings.Repeat("a", 64)
		binding := requiredReplayV3TestBinding(fingerprint)
		binding.PlanDigest = strings.Repeat("d", 64)
		secret := "redis-mutation-secret-" + unique
		bodySecret := "redis-response-body-secret-" + unique
		headerSecret := "redis-response-header-secret-" + unique
		canonicalSecret := "/redis/canonical-secret-" + unique
		lease, replay, err := store.BeginRequiredReplayBound(ctx, scope, key, binding)
		if err != nil || replay != nil || lease.storageKey == "" {
			t.Fatalf("begin: lease=%#v replay=%#v error=%v", lease, replay, err)
		}
		response := RequiredReplayResponse{
			Status: http.StatusCreated,
			Headers: http.Header{
				"Content-Type": {"application/json"},
				"X-Result":     {headerSecret},
			},
			Body:          []byte(bodySecret),
			CanonicalPath: canonicalSecret,
			Authorization: requiredReplayRedisAuthorization(secret),
		}
		if err := store.CompleteRequiredReplay(ctx, lease, response); err != nil {
			t.Fatal(err)
		}

		raw, err := client.Get(ctx, fullKey).Bytes()
		if err != nil {
			t.Fatal(err)
		}
		for _, plaintext := range []string{
			secret, bodySecret, headerSecret, canonicalSecret, "requestMutations", "beforeDigest", "afterDigest",
			"sforum.route-replay-authorization@1", `"response"`, `"authorizationCiphertext"`,
		} {
			if strings.Contains(string(raw), plaintext) {
				t.Fatalf("Redis record contains mutation plaintext %q: %s", plaintext, raw)
			}
		}
		if !strings.Contains(string(raw), `"payloadCiphertext"`) {
			t.Fatalf("Redis record has no encrypted payload: %s", raw)
		}
		ttl, err := client.PTTL(ctx, fullKey).Result()
		if err != nil || ttl > DefaultTTL || ttl < DefaultTTL-2*time.Minute {
			t.Fatalf("Redis replay PTTL=%s error=%v", ttl, err)
		}

		_, first, err := store.BeginRequiredReplayBound(ctx, scope, key, binding)
		if err != nil || !requiredReplayRedisResponseMatches(first, secret, bodySecret) {
			t.Fatalf("first replay=%#v error=%v", first, err)
		}
		first.Body[0] = '!'
		first.Headers.Set("X-Result", "mutated")
		first.Authorization.RequestMutations[0].Operations[0].Value[1] = 'X'
		_, detached, err := store.BeginRequiredReplayBound(ctx, scope, key, binding)
		if err != nil || !requiredReplayRedisResponseMatches(detached, secret, bodySecret) ||
			detached.Headers.Get("X-Result") != headerSecret || detached.CanonicalPath != canonicalSecret {
			t.Fatalf("detached replay=%#v error=%v", detached, err)
		}
	})

	t.Run("abort and CAS lease fence", func(t *testing.T) {
		key := "redis-cas-" + unique
		requiredReplayRedisCleanKey(t, client, store, scope, key)
		fingerprint := strings.Repeat("b", 64)
		stale, _, err := store.BeginRequiredReplayBound(ctx, scope, key, requiredReplayV3TestBinding(fingerprint))
		if err != nil {
			t.Fatal(err)
		}
		if err := store.AbortRequiredReplay(ctx, stale); err != nil {
			t.Fatal(err)
		}
		owner, _, err := store.BeginRequiredReplayBound(ctx, scope, key, requiredReplayV3TestBinding(fingerprint))
		if err != nil {
			t.Fatal(err)
		}
		// A stale abort is deliberately idempotent but must not delete the new owner.
		if err := store.AbortRequiredReplay(ctx, stale); err != nil {
			t.Fatal(err)
		}
		if _, _, err := store.BeginRequiredReplayBound(
			ctx, scope, key, requiredReplayV3TestBinding(fingerprint),
		); !errors.Is(err, ErrRequiredReplayInProgress) {
			t.Fatalf("stale abort removed replacement owner: %v", err)
		}
		if err := store.CompleteRequiredReplay(ctx, stale, RequiredReplayResponse{
			Status: http.StatusOK,
		}); !errors.Is(err, ErrRequiredReplayLeaseLost) {
			t.Fatalf("stale completion error=%v", err)
		}
		if err := store.CompleteRequiredReplay(ctx, owner, RequiredReplayResponse{
			Status: http.StatusOK, Body: []byte("owner"),
		}); err != nil {
			t.Fatal(err)
		}
		_, replay, err := store.BeginRequiredReplayBound(ctx, scope, key, requiredReplayV3TestBinding(fingerprint))
		if err != nil || replay == nil || string(replay.Body) != "owner" {
			t.Fatalf("owner replay=%#v error=%v", replay, err)
		}
	})

	t.Run("64 callers acquire exactly once then replay detached", func(t *testing.T) {
		const callers = 64
		key := "redis-concurrent-" + unique
		requiredReplayRedisCleanKey(t, client, store, scope, key)
		fingerprint := strings.Repeat("c", 64)
		binding := requiredReplayV3TestBinding(fingerprint)
		binding.PlanDigest = strings.Repeat("d", 64)
		secret := "redis-concurrent-secret-" + unique
		results := requiredReplayRedisConcurrentBegin(ctx, store, scope, key, binding, callers)
		winners := make([]RequiredReplayLease, 0, 1)
		inProgress := 0
		for _, result := range results {
			switch {
			case result.err == nil && result.replay == nil && result.lease.storageKey != "":
				winners = append(winners, result.lease)
			case errors.Is(result.err, ErrRequiredReplayInProgress) && result.replay == nil && result.lease.storageKey == "":
				inProgress++
			default:
				t.Fatalf("unexpected concurrent begin=%#v", result)
			}
		}
		if len(winners) != 1 || inProgress != callers-1 {
			t.Fatalf("winners=%d in_progress=%d", len(winners), inProgress)
		}
		if err := store.CompleteRequiredReplay(ctx, winners[0], RequiredReplayResponse{
			Status:        http.StatusCreated,
			Headers:       http.Header{"X-Result": {"winner"}},
			Body:          []byte(`{"winner":true}`),
			Authorization: requiredReplayRedisAuthorization(secret),
		}); err != nil {
			t.Fatal(err)
		}

		replays := requiredReplayRedisConcurrentBegin(ctx, store, scope, key, binding, callers)
		for index, result := range replays {
			if result.err != nil || result.lease.storageKey != "" ||
				!requiredReplayRedisResponseMatches(result.replay, secret, `{"winner":true}`) ||
				result.replay.Headers.Get("X-Result") != "winner" {
				t.Fatalf("concurrent replay %d=%#v", index, result)
			}
			// Every caller receives a detached body, header map, and transcript.
			result.replay.Body[0] = byte('a' + index%26)
			result.replay.Headers.Set("X-Result", strconv.Itoa(index))
			result.replay.Authorization.RequestMutations[0].Operations[0].Value[1] = byte('A' + index%26)
		}
		_, finalReplay, err := store.BeginRequiredReplayBound(ctx, scope, key, binding)
		if err != nil || !requiredReplayRedisResponseMatches(finalReplay, secret, `{"winner":true}`) ||
			finalReplay.Headers.Get("X-Result") != "winner" {
			t.Fatalf("final detached replay=%#v error=%v", finalReplay, err)
		}
	})
}

type requiredReplayRedisBeginResult struct {
	lease  RequiredReplayLease
	replay *RequiredReplayResponse
	err    error
}

func requiredReplayRedisConcurrentBegin(
	ctx context.Context,
	store *Store,
	scope RequiredReplayScope,
	key string,
	binding RequiredReplayBinding,
	callers int,
) []requiredReplayRedisBeginResult {
	start := make(chan struct{})
	results := make(chan requiredReplayRedisBeginResult, callers)
	var wait sync.WaitGroup
	wait.Add(callers)
	for range callers {
		go func() {
			defer wait.Done()
			<-start
			lease, replay, err := store.BeginRequiredReplayBound(
				ctx, scope, key, binding,
			)
			results <- requiredReplayRedisBeginResult{lease: lease, replay: replay, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	collected := make([]requiredReplayRedisBeginResult, 0, callers)
	for result := range results {
		collected = append(collected, result)
	}
	return collected
}

func requiredReplayRedisClient(t *testing.T) (*redis.Client, context.Context) {
	t.Helper()
	address := strings.TrimSpace(os.Getenv("SFORUM_TEST_REDIS_ADDR"))
	if address == "" {
		t.Skip("SFORUM_TEST_REDIS_ADDR is required for required replay Redis integration tests")
	}
	database := 15
	if raw := strings.TrimSpace(os.Getenv("SFORUM_TEST_REDIS_DB")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			t.Fatalf("invalid SFORUM_TEST_REDIS_DB %q", raw)
		}
		database = value
	}
	client := redis.NewClient(&redis.Options{
		Addr: address, Password: os.Getenv("SFORUM_TEST_REDIS_PASSWORD"), DB: database,
		DialTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		PoolSize: 96,
	})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping test Redis: %v", err)
	}
	return client, ctx
}

func requiredReplayRedisCleanKey(
	t *testing.T,
	client *redis.Client,
	store *Store,
	scope RequiredReplayScope,
	key string,
) string {
	t.Helper()
	storageKey, err := requiredReplayStorageKey(scope, key)
	if err != nil {
		t.Fatal(err)
	}
	fullKey := store.fullKey(storageKey)
	if err := client.Del(t.Context(), fullKey).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Del(cleanupCtx, fullKey).Err()
	})
	return fullKey
}

func requiredReplayRedisCipher(t *testing.T) *RequiredReplayCipher {
	t.Helper()
	cipher, err := NewRequiredReplayCipher(strings.Repeat("0a", 32))
	if err != nil || !cipher.Enabled() {
		t.Fatalf("required replay cipher=%#v error=%v", cipher, err)
	}
	return cipher
}

func requiredReplayRedisAuthorization(secret string) *RequiredReplayAuthorization {
	return &RequiredReplayAuthorization{
		Schema:     "sforum.route-replay-authorization@1",
		PlanDigest: strings.Repeat("d", 64),
		BaseDigest: strings.Repeat("e", 64),
		RequestMutations: []RequiredReplayRequestMutation{{
			StepIndex:    0,
			BeforeDigest: strings.Repeat("a", 64),
			AfterDigest:  strings.Repeat("b", 64),
			Operations: []RequiredReplayPatchOperation{{
				Kind: "replace", Path: "/body/private", Value: json.RawMessage(fmt.Sprintf("%q", secret)),
			}},
		}},
	}
}

func requiredReplayRedisResponseMatches(response *RequiredReplayResponse, secret, body string) bool {
	return response != nil && response.Authorization != nil && string(response.Body) == body &&
		len(response.Authorization.RequestMutations) == 1 &&
		len(response.Authorization.RequestMutations[0].Operations) == 1 &&
		string(response.Authorization.RequestMutations[0].Operations[0].Value) == fmt.Sprintf("%q", secret)
}
