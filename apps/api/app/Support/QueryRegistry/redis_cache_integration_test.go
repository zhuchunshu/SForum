package queryregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// This gate intentionally does not use SFORUM_TEST_REDIS_ADDR. Query cache
// durability tests need a dedicated Redis >=7.2 with appendonly=yes; sharing the
// session/queue Redis would let poison and CONFIG state leak across suites.
func TestRedisQueryResultCacheGenerationIntegration(t *testing.T) {
	client := queryResultRedisIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	seed := fmt.Sprintf("query-generation-%d", time.Now().UnixNano())
	first := queryResultRedisIntegrationCache(t, client, seed+"-first")
	second := queryResultRedisIntegrationCache(t, client, seed+"-second")
	cleanupQueryResultRedisPrefix(t, client, first.root)
	cleanupQueryResultRedisPrefix(t, client, second.root)
	if err := first.Activate(ctx); err != nil {
		t.Fatalf("activate first Query cache: %v", err)
	}
	if err := second.Activate(ctx); err != nil {
		t.Fatalf("activate second Query cache: %v", err)
	}
	if ttl, err := client.PTTL(ctx, first.epochKey).Result(); err != nil || ttl != -1 {
		t.Fatalf("permanent allocator TTL=%s err=%v", ttl, err)
	}
	if marker, err := client.Get(ctx, first.markerKey).Result(); err != nil || marker != redisQueryCacheActivationMarker {
		t.Fatalf("activation marker=%q err=%v", marker, err)
	}
	if ttl, err := client.PTTL(ctx, first.markerKey).Result(); err != nil || ttl != -1 {
		t.Fatalf("activation marker TTL=%s err=%v", ttl, err)
	}
	t.Cleanup(func() {
		cleanupQueryResultRedisPrefix(t, client, first.root)
		cleanupQueryResultRedisPrefix(t, client, second.root)
	})

	t.Run("miss store hit and bounded retention", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-hit")
		tags := queryResultRedisIntegrationTags(seed + "-hit")
		value := redisQueryResultTestValue(t, key, tags, false)
		loaded, fence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || found || loaded.CacheKey != "" ||
			!validRedisQueryResultCacheFence(fence, first.root, key, mustPhysicalGenerations(t, first, tags)) {
			t.Fatalf("initial miss found=%t value=%#v fence=%#v err=%v", found, loaded, fence, err)
		}
		if err := first.StoreQueryResult(ctx, key, value, tags, fence); err != nil {
			t.Fatalf("Store: %v", err)
		}
		loaded, replayFence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || !found || loaded.Rows[0]["title"] != "cached" ||
			!sameQueryResultCacheFence(fence, replayFence) {
			t.Fatalf("cache hit found=%t value=%#v fence=%#v err=%v", found, loaded, replayFence, err)
		}
		ttl, err := client.PTTL(ctx, first.valueKey(key)).Result()
		if err != nil || ttl <= 0 || ttl > RedisQueryResultCacheTTL {
			t.Fatalf("Host TTL=%s err=%v", ttl, err)
		}
	})

	t.Run("invalidation prevents stale provider revival", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-revival")
		tags := queryResultRedisIntegrationTags(seed + "-revival")
		value := redisQueryResultTestValue(t, key, tags, false)
		_, staleFence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || found {
			t.Fatalf("pre-provider miss found=%t err=%v", found, err)
		}
		if rotated, err := first.InvalidateTags(ctx, []string{tags[0]}); err != nil || rotated != 1 {
			t.Fatalf("durable invalidation rotated=%d err=%v", rotated, err)
		}
		if err := first.StoreQueryResult(ctx, key, value, tags, staleFence); !errors.Is(err, ErrCacheFenceConflict) {
			t.Fatalf("stale provider Store error=%v", err)
		}
		_, currentFence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || found || sameQueryResultCacheFence(staleFence, currentFence) {
			t.Fatalf("post-invalidation miss found=%t stale=%#v current=%#v err=%v", found, staleFence, currentFence, err)
		}
		if err := first.StoreQueryResult(ctx, key, value, tags, currentFence); err != nil {
			t.Fatalf("fresh generation Store: %v", err)
		}
		if _, _, found, err := first.LoadQueryResult(ctx, key, tags); err != nil || !found {
			t.Fatalf("fresh generation load found=%t err=%v", found, err)
		}
	})

	t.Run("invalidation wins while Store is staged", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-staged-race")
		tags := queryResultRedisIntegrationTags(seed + "-staged-race")
		value := redisQueryResultTestValue(t, key, tags, false)
		_, staleFence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || found {
			t.Fatalf("initial load found=%t err=%v", found, err)
		}

		storeClient := queryResultRedisIntegrationClient(t)
		storeCache := queryResultRedisIntegrationCache(t, storeClient, seed+"-first")
		if err := storeCache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		_, staleFence, found, err = first.LoadQueryResult(ctx, key, tags)
		if err != nil || found {
			t.Fatalf("post-activation load found=%t err=%v", found, err)
		}
		pause := newRedisQueryCacheFinalizePause()
		storeClient.AddHook(pause)
		storeDone := make(chan error, 1)
		go func() {
			storeDone <- storeCache.StoreQueryResult(ctx, key, value, tags, staleFence)
		}()
		select {
		case <-pause.entered:
		case <-time.After(10 * time.Second):
			t.Fatal("Store did not reach the staged finalize boundary")
		}
		if _, err := first.InvalidateTags(ctx, []string{tags[0]}); err != nil {
			t.Fatal(err)
		}
		close(pause.release)
		if err := <-storeDone; !errors.Is(err, ErrCacheFenceConflict) {
			t.Fatalf("staged stale Store error=%v", err)
		}
		if keys := scanQueryResultRedisKeys(t, client, first.temporaryPrefix+"*"); len(keys) != 0 {
			t.Fatalf("staged temporary envelopes leaked: %v", keys)
		}
		if _, _, found, err := first.LoadQueryResult(ctx, key, tags); err != nil || found {
			t.Fatalf("staged stale value revived found=%t err=%v", found, err)
		}
	})

	t.Run("activation rotates allocator without flushing valid tags", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-activation")
		tags := queryResultRedisIntegrationTags(seed + "-activation")
		value := redisQueryResultTestValue(t, key, tags, false)
		_, staleFence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || found {
			t.Fatalf("initial load found=%t err=%v", found, err)
		}
		if err := first.StoreQueryResult(ctx, key, value, tags, staleFence); err != nil {
			t.Fatal(err)
		}
		before, err := client.Get(ctx, first.epochKey).Uint64()
		if err != nil {
			t.Fatal(err)
		}
		if err := first.Activate(ctx); err != nil {
			t.Fatalf("reactivate: %v", err)
		}
		after, err := client.Get(ctx, first.epochKey).Uint64()
		if err != nil || after != before+1 {
			t.Fatalf("allocator before=%d after=%d err=%v", before, after, err)
		}
		loaded, currentFence, found, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil || !found || loaded.Rows[0]["title"] != "cached" ||
			sameQueryResultCacheFence(staleFence, currentFence) ||
			!sameQueryResultCacheTagSnapshot(staleFence, currentFence) {
			t.Fatalf("post-activation hit found=%t value=%#v stale=%#v current=%#v err=%v",
				found, loaded, staleFence, currentFence, err)
		}
		if err := first.StoreQueryResult(ctx, key, value, tags, staleFence); !errors.Is(err, ErrCacheFenceConflict) {
			t.Fatalf("pre-activation fence Store error=%v", err)
		}
	})

	t.Run("selective invalidation preserves unrelated hits", func(t *testing.T) {
		firstKey := queryResultRedisIntegrationKey(seed + "-selective-first")
		firstTags := queryResultRedisIntegrationTags(seed + "-selective-first")
		secondKey := queryResultRedisIntegrationKey(seed + "-selective-second")
		secondTags := queryResultRedisIntegrationTags(seed + "-selective-second")
		for _, item := range []struct {
			key  string
			tags []string
		}{{firstKey, firstTags}, {secondKey, secondTags}} {
			_, fence, found, err := first.LoadQueryResult(ctx, item.key, item.tags)
			if err != nil || found {
				t.Fatalf("initial load found=%t err=%v", found, err)
			}
			if err := first.StoreQueryResult(ctx, item.key, redisQueryResultTestValue(t, item.key, item.tags, false), item.tags, fence); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := first.InvalidateTags(ctx, []string{firstTags[0]}); err != nil {
			t.Fatal(err)
		}
		if _, _, found, err := first.LoadQueryResult(ctx, firstKey, firstTags); err != nil || found {
			t.Fatalf("targeted value found=%t err=%v", found, err)
		}
		if _, _, found, err := first.LoadQueryResult(ctx, secondKey, secondTags); err != nil || !found {
			t.Fatalf("unrelated value found=%t err=%v", found, err)
		}
	})

	t.Run("tag retention outlives value", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-tag-ttl")
		tags := queryResultRedisIntegrationTags(seed + "-tag-ttl")
		_, fence, _, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil {
			t.Fatal(err)
		}
		if err := first.StoreQueryResult(ctx, key, redisQueryResultTestValue(t, key, tags, false), tags, fence); err != nil {
			t.Fatal(err)
		}
		valueTTL, err := client.PTTL(ctx, first.valueKey(key)).Result()
		if err != nil {
			t.Fatal(err)
		}
		for _, physical := range mustPhysicalGenerations(t, first, tags) {
			tagTTL, tagErr := client.PTTL(ctx, physical).Result()
			if tagErr != nil || tagTTL <= valueTTL || tagTTL > redisQueryCacheTagTTL {
				t.Fatalf("value TTL=%s tag TTL=%s err=%v", valueTTL, tagTTL, tagErr)
			}
		}
	})

	t.Run("permanent value poison fails closed and latches", func(t *testing.T) {
		cache := queryResultRedisIntegrationCache(t, client, seed+"-permanent-value-cache")
		cleanupQueryResultRedisPrefix(t, client, cache.root)
		defer cleanupQueryResultRedisPrefix(t, client, cache.root)
		if err := cache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		permanentKey := queryResultRedisIntegrationKey(seed + "-permanent-value")
		permanentTags := queryResultRedisIntegrationTags(seed + "-permanent-value")
		if err := client.Set(ctx, cache.valueKey(permanentKey), "forged", 0).Err(); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := cache.LoadQueryResult(ctx, permanentKey, permanentTags); !errors.Is(err, ErrCachePoisoned) {
			t.Fatalf("permanent value load error=%v", err)
		}
		if err := client.Del(ctx, cache.valueKey(permanentKey)).Err(); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := cache.LoadQueryResult(ctx, permanentKey, permanentTags); !errors.Is(err, ErrCacheCapability) {
			t.Fatalf("poisoned cache resumed error=%v", err)
		}
	})

	t.Run("finalize poison cleans temporary envelope", func(t *testing.T) {
		cache := queryResultRedisIntegrationCache(t, client, seed+"-store-poison-cache")
		cleanupQueryResultRedisPrefix(t, client, cache.root)
		defer cleanupQueryResultRedisPrefix(t, client, cache.root)
		if err := cache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		storeKey := queryResultRedisIntegrationKey(seed + "-store-poison")
		storeTags := queryResultRedisIntegrationTags(seed + "-store-poison")
		_, fence, _, err := cache.LoadQueryResult(ctx, storeKey, storeTags)
		if err != nil {
			t.Fatal(err)
		}
		physical := mustPhysicalGenerations(t, cache, storeTags)
		if err := client.HSet(ctx, physical[0], "forged", "tag").Err(); err != nil {
			t.Fatal(err)
		}
		if err := cache.StoreQueryResult(ctx, storeKey, redisQueryResultTestValue(t, storeKey, storeTags, false), storeTags, fence); !errors.Is(err, ErrCachePoisoned) {
			t.Fatalf("poisoned Store error=%v", err)
		}
		if keys := scanQueryResultRedisKeys(t, client, cache.temporaryPrefix+"*"); len(keys) != 0 {
			t.Fatalf("temporary envelopes leaked: %v", keys)
		}
		if err := client.Del(ctx, physical[0]).Err(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rotation makes values unreadable without reverse deletion", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-lazy-expiry")
		tags := queryResultRedisIntegrationTags(seed + "-lazy-expiry")
		value := redisQueryResultTestValue(t, key, tags, false)
		_, fence, _, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil {
			t.Fatal(err)
		}
		if err := first.StoreQueryResult(ctx, key, value, tags, fence); err != nil {
			t.Fatal(err)
		}
		if rotated, err := first.InvalidateTags(ctx, tags); err != nil || rotated != uint64(len(tags)) {
			t.Fatalf("rotate all tags=%d err=%v", rotated, err)
		}
		if exists, err := client.Exists(ctx, first.valueKey(key)).Result(); err != nil || exists != 1 {
			t.Fatalf("generation invalidation reverse-deleted value exists=%d err=%v", exists, err)
		}
		if _, _, found, err := first.LoadQueryResult(ctx, key, tags); err != nil || found {
			t.Fatalf("rotated value remained readable found=%t err=%v", found, err)
		}
	})

	t.Run("installation namespace isolates values and generations", func(t *testing.T) {
		key := queryResultRedisIntegrationKey(seed + "-isolation")
		tags := queryResultRedisIntegrationTags(seed + "-isolation")
		firstValue := redisQueryResultTestValue(t, key, tags, false)
		secondValue := redisQueryResultTestValue(t, key, tags, false)
		secondValue.Rows[0]["title"] = "second installation"
		_, firstFence, _, err := first.LoadQueryResult(ctx, key, tags)
		if err != nil {
			t.Fatal(err)
		}
		_, secondFence, _, err := second.LoadQueryResult(ctx, key, tags)
		if err != nil {
			t.Fatal(err)
		}
		if err := first.StoreQueryResult(ctx, key, firstValue, tags, firstFence); err != nil {
			t.Fatal(err)
		}
		if err := second.StoreQueryResult(ctx, key, secondValue, tags, secondFence); err != nil {
			t.Fatal(err)
		}
		if _, err := first.InvalidateTags(ctx, tags); err != nil {
			t.Fatal(err)
		}
		loaded, _, found, err := second.LoadQueryResult(ctx, key, tags)
		if err != nil || !found || loaded.Rows[0]["title"] != "second installation" || first.root == second.root {
			t.Fatalf("cross-installation leak found=%t value=%#v err=%v", found, loaded, err)
		}
	})

	t.Run("sixty four tag bound is constant work", func(t *testing.T) {
		tags := make([]string, redisQueryCacheMaximumTags)
		for index := range tags {
			tags[index] = queryResultRedisSharedTag(seed + "-bound-" + strconv.Itoa(index))
		}
		if rotated, err := first.InvalidateTags(ctx, tags); err != nil || rotated != redisQueryCacheMaximumTags {
			t.Fatalf("64-tag rotation=%d err=%v", rotated, err)
		}
		tooMany := append(append([]string(nil), tags...), queryResultRedisSharedTag(seed+"-bound-overflow"))
		if rotated, err := first.InvalidateTags(ctx, tooMany); !errors.Is(err, ErrExecutionInvalid) || rotated != 0 {
			t.Fatalf("65-tag invalidation rotated=%d err=%v", rotated, err)
		}
	})

	t.Run("poisoned generation preflight makes zero writes", func(t *testing.T) {
		cacheSeed := seed + "-poison-generation-cache"
		cache := queryResultRedisIntegrationCache(t, client, cacheSeed)
		cleanupQueryResultRedisPrefix(t, client, cache.root)
		defer cleanupQueryResultRedisPrefix(t, client, cache.root)
		if err := cache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		tags := queryResultRedisIntegrationTags(seed + "-poison-generation")
		physical := mustPhysicalGenerations(t, cache, tags)
		if err := client.HSet(ctx, physical[1], "forged", "generation").Err(); err != nil {
			t.Fatal(err)
		}
		if rotated, err := cache.InvalidateTags(ctx, tags); !errors.Is(err, ErrCachePoisoned) || rotated != 0 {
			t.Fatalf("poisoned rotation=%d err=%v", rotated, err)
		}
		if exists, err := client.Exists(ctx, physical[0]).Result(); err != nil || exists != 0 {
			t.Fatalf("preflight partially rotated earlier generation exists=%d err=%v", exists, err)
		}
		reader := queryResultRedisIntegrationCache(t, client, cacheSeed)
		if err := reader.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := reader.LoadQueryResult(ctx, queryResultRedisIntegrationKey(seed+"-poison-generation"), tags); !errors.Is(err, ErrCachePoisoned) {
			t.Fatalf("poisoned generation load error=%v", err)
		}
		if err := client.Del(ctx, physical[1]).Err(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("poisoned envelope fails closed", func(t *testing.T) {
		cache := queryResultRedisIntegrationCache(t, client, seed+"-poison-envelope-cache")
		cleanupQueryResultRedisPrefix(t, client, cache.root)
		defer cleanupQueryResultRedisPrefix(t, client, cache.root)
		if err := cache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		key := queryResultRedisIntegrationKey(seed + "-poison-envelope")
		tags := queryResultRedisIntegrationTags(seed + "-poison-envelope")
		value := redisQueryResultTestValue(t, key, tags, false)
		_, fence, _, err := cache.LoadQueryResult(ctx, key, tags)
		if err != nil {
			t.Fatal(err)
		}
		if err := cache.StoreQueryResult(ctx, key, value, tags, fence); err != nil {
			t.Fatal(err)
		}
		if err := client.SetArgs(ctx, cache.valueKey(key), "{not-json", redis.SetArgs{KeepTTL: true}).Err(); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := cache.LoadQueryResult(ctx, key, tags); !errors.Is(err, ErrCachePoisoned) {
			t.Fatalf("poisoned envelope load error=%v", err)
		}
	})

	t.Run("allocator or marker loss is not treated as first install", func(t *testing.T) {
		for _, lost := range []string{"marker", "epoch"} {
			t.Run(lost, func(t *testing.T) {
				cache := queryResultRedisIntegrationCache(t, client, seed+"-loss-"+lost)
				cleanupQueryResultRedisPrefix(t, client, cache.root)
				t.Cleanup(func() { cleanupQueryResultRedisPrefix(t, client, cache.root) })
				if err := cache.Activate(ctx); err != nil {
					t.Fatal(err)
				}
				key := cache.markerKey
				if lost == "epoch" {
					key = cache.epochKey
				}
				if err := client.Del(ctx, key).Err(); err != nil {
					t.Fatal(err)
				}
				if err := cache.Activate(ctx); !errors.Is(err, ErrCachePoisoned) {
					t.Fatalf("lost %s reactivation error=%v", lost, err)
				}
			})
		}
	})
}

// TestRedisQueryResultCacheAOFRestartPhase is an explicit two-process gate.
// Run phase=seed, restart the dedicated Redis container, then run phase=verify
// with the same seed. Normal package tests skip it rather than managing Docker.
func TestRedisQueryResultCacheAOFRestartPhase(t *testing.T) {
	phase := strings.TrimSpace(os.Getenv("SFORUM_QUERY_CACHE_TEST_RESTART_PHASE"))
	seed := strings.TrimSpace(os.Getenv("SFORUM_QUERY_CACHE_TEST_RESTART_SEED"))
	if phase == "" && seed == "" {
		t.Skip("set SFORUM_QUERY_CACHE_TEST_RESTART_PHASE and SFORUM_QUERY_CACHE_TEST_RESTART_SEED")
	}
	if seed == "" {
		t.Fatal("SFORUM_QUERY_CACHE_TEST_RESTART_SEED is required")
	}
	client := queryResultRedisIntegrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cache := queryResultRedisIntegrationCache(t, client, "restart-"+seed)
	hitKey := queryResultRedisIntegrationKey(seed + "-restart-hit")
	hitTags := queryResultRedisIntegrationTags(seed + "-restart-hit")
	invalidatedKey := queryResultRedisIntegrationKey(seed + "-restart-invalidated")
	invalidatedTags := queryResultRedisIntegrationTags(seed + "-restart-invalidated")

	switch phase {
	case "seed":
		cleanupQueryResultRedisPrefix(t, client, cache.root)
		if err := cache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		for _, item := range []struct {
			key  string
			tags []string
		}{{hitKey, hitTags}, {invalidatedKey, invalidatedTags}} {
			_, fence, found, err := cache.LoadQueryResult(ctx, item.key, item.tags)
			if err != nil || found {
				t.Fatalf("seed miss found=%t err=%v", found, err)
			}
			if err := cache.StoreQueryResult(ctx, item.key, redisQueryResultTestValue(t, item.key, item.tags, false), item.tags, fence); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := cache.InvalidateTags(ctx, []string{invalidatedTags[0]}); err != nil {
			t.Fatal(err)
		}
		if _, _, found, err := cache.LoadQueryResult(ctx, hitKey, hitTags); err != nil || !found {
			t.Fatalf("seed hit found=%t err=%v", found, err)
		}
		if _, _, found, err := cache.LoadQueryResult(ctx, invalidatedKey, invalidatedTags); err != nil || found {
			t.Fatalf("seed invalidated value found=%t err=%v", found, err)
		}
	case "verify":
		defer cleanupQueryResultRedisPrefix(t, client, cache.root)
		before, err := client.Get(ctx, cache.epochKey).Uint64()
		if err != nil {
			t.Fatal(err)
		}
		if marker, err := client.Get(ctx, cache.markerKey).Result(); err != nil || marker != redisQueryCacheActivationMarker {
			t.Fatalf("persisted activation marker=%q err=%v", marker, err)
		}
		if err := cache.Activate(ctx); err != nil {
			t.Fatal(err)
		}
		after, err := client.Get(ctx, cache.epochKey).Uint64()
		if err != nil || after != before+1 {
			t.Fatalf("restart activation before=%d after=%d err=%v", before, after, err)
		}
		if _, _, found, err := cache.LoadQueryResult(ctx, hitKey, hitTags); err != nil || !found {
			t.Fatalf("activation flushed persisted hit found=%t err=%v", found, err)
		}
		if _, _, found, err := cache.LoadQueryResult(ctx, invalidatedKey, invalidatedTags); err != nil || found {
			t.Fatalf("invalidated value revived after restart found=%t err=%v", found, err)
		}
	default:
		t.Fatalf("unsupported restart phase %q", phase)
	}
}

func queryResultRedisIntegrationClient(t *testing.T) *redis.Client {
	t.Helper()
	address := strings.TrimSpace(os.Getenv("SFORUM_QUERY_CACHE_TEST_REDIS_ADDR"))
	if address == "" {
		t.Skip("set SFORUM_QUERY_CACHE_TEST_REDIS_ADDR to a dedicated Redis >=7.2 with appendonly=yes")
	}
	client := redis.NewClient(&redis.Options{
		Addr: address, Password: os.Getenv("SFORUM_QUERY_CACHE_TEST_REDIS_PASSWORD"), DB: 0,
		DialTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		MaxRetries: -1, ContextTimeoutEnabled: true,
	})
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("dedicated Query cache Redis unavailable: %v", err)
	}
	persistence, err := client.Info(ctx, "persistence").Result()
	if err != nil || !strings.Contains(persistence, "aof_enabled:1") {
		t.Fatalf("Query cache gate requires appendonly=yes: err=%v persistence=%q", err, persistence)
	}
	server, err := client.Info(ctx, "server").Result()
	if err != nil || !redisVersionAtLeast72(server) {
		t.Fatalf("Query cache gate requires Redis >=7.2 for WAITAOF: err=%v", err)
	}
	return client
}

func redisVersionAtLeast72(info string) bool {
	for _, line := range strings.Split(info, "\n") {
		if !strings.HasPrefix(line, "redis_version:") {
			continue
		}
		parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "redis_version:")), ".")
		if len(parts) < 2 {
			return false
		}
		major, majorErr := strconv.Atoi(parts[0])
		minor, minorErr := strconv.Atoi(parts[1])
		return majorErr == nil && minorErr == nil && (major > 7 || major == 7 && minor >= 2)
	}
	return false
}

func queryResultRedisIntegrationCache(t *testing.T, client *redis.Client, seed string) *RedisQueryResultCache {
	t.Helper()
	digest := sha256.Sum256([]byte(seed))
	cache, err := NewRedisQueryResultCache(client, hex.EncodeToString(digest[:]))
	if err != nil {
		t.Fatal(err)
	}
	return cache
}

func queryResultRedisIntegrationKey(seed string) string {
	digest := sha256.Sum256([]byte("key\x00" + seed))
	return hex.EncodeToString(digest[:])
}

func queryResultRedisIntegrationTags(seed string) []string {
	shared := sha256.Sum256([]byte("shared\x00" + seed))
	isolated := sha256.Sum256([]byte("isolated\x00" + seed))
	semantic := sha256.Sum256([]byte("semantic\x00" + seed))
	return []string{
		"query:shared:" + hex.EncodeToString(shared[:16]),
		"query:" + hex.EncodeToString(isolated[:16]) + ":" + hex.EncodeToString(semantic[:16]),
	}
}

func queryResultRedisSharedTag(seed string) string {
	digest := sha256.Sum256([]byte("shared\x00" + seed))
	return "query:shared:" + hex.EncodeToString(digest[:16])
}

func mustPhysicalGenerations(t *testing.T, cache *RedisQueryResultCache, tags []string) []string {
	t.Helper()
	physical, err := cache.physicalGenerationKeys(tags)
	if err != nil {
		t.Fatal(err)
	}
	return physical
}

func cleanupQueryResultRedisPrefix(t *testing.T, client *redis.Client, prefix string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, prefix+"*", 256).Result()
		if err != nil {
			t.Errorf("scan Query cache test prefix %q: %v", prefix, err)
			return
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				t.Errorf("delete Query cache test prefix %q: %v", prefix, err)
				return
			}
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func scanQueryResultRedisKeys(t *testing.T, client *redis.Client, pattern string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var (
		cursor uint64
		result []string
	)
	for {
		keys, next, err := client.Scan(ctx, cursor, pattern, 256).Result()
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, keys...)
		cursor = next
		if cursor == 0 {
			return result
		}
	}
}

type redisQueryCacheFinalizePause struct {
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func newRedisQueryCacheFinalizePause() *redisQueryCacheFinalizePause {
	return &redisQueryCacheFinalizePause{entered: make(chan struct{}), release: make(chan struct{})}
}

func (h *redisQueryCacheFinalizePause) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *redisQueryCacheFinalizePause) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, command redis.Cmder) error {
		arguments := command.Args()
		finalize := len(arguments) > 1 &&
			(command.Name() == "evalsha" && fmt.Sprint(arguments[1]) == redisQueryCacheFinalizeScript.Hash() ||
				command.Name() == "eval" && fmt.Sprint(arguments[1]) == redisQueryCacheFinalizeLua)
		if finalize {
			var wait bool
			h.once.Do(func() {
				wait = true
				close(h.entered)
			})
			if wait {
				select {
				case <-h.release:
				case <-ctx.Done():
					return context.Cause(ctx)
				}
			}
		}
		return next(ctx, command)
	}
}

func (h *redisQueryCacheFinalizePause) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		return next(ctx, commands)
	}
}
