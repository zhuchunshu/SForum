package queryregistry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisQueryResultCacheUsesEpochNamespaceAndOpaqueFence(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", MaxRetries: -1, ContextTimeoutEnabled: true,
	})
	t.Cleanup(func() { _ = client.Close() })
	installationID := strings.Repeat("a", 64)
	cache, err := NewRedisQueryResultCache(client, installationID)
	if err != nil {
		t.Fatal(err)
	}
	again, err := NewRedisQueryResultCache(client, installationID)
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewRedisQueryResultCache(client, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	if cache.root != again.root || cache.root == other.root || strings.Contains(cache.root, installationID) {
		t.Fatalf("installation roots cache=%q again=%q other=%q", cache.root, again.root, other.root)
	}
	key := strings.Repeat("c", 64)
	tags := redisQueryResultTestTags('a', 'b', 'c')
	physical, err := cache.physicalGenerationKeys(tags)
	if err != nil {
		t.Fatal(err)
	}
	hashTag := cache.root[strings.Index(cache.root, "{") : strings.Index(cache.root, "}")+1]
	physicalKeys := []string{cache.valueKey(key), cache.markerKey, cache.epochKey, cache.temporaryPrefix + "probe"}
	physicalKeys = append(physicalKeys, physical...)
	for _, candidate := range physicalKeys {
		if !strings.Contains(candidate, hashTag) || strings.Contains(candidate, key) {
			t.Fatalf("physical key %q is not opaque and installation-scoped", candidate)
		}
	}
	fence := newQueryResultCacheFence(cache.root, key, "9", physical, []string{"9", "7"})
	if !validRedisQueryResultCacheFence(fence, cache.root, key, physical) ||
		validRedisQueryResultCacheFence(fence, other.root, key, physical) ||
		validRedisQueryResultCacheFence(fence, cache.root, strings.Repeat("d", 64), physical) {
		t.Fatalf("fence binding is not exact: %#v", fence)
	}
	tampered := fence
	tamperedMaterial, ok := redisQueryResultCacheFenceValue(tampered)
	if !ok {
		t.Fatal("Redis fence material was not retained")
	}
	tamperedMaterial.generations = []string{"0", "8"}
	tampered = tamperedMaterial
	if validRedisQueryResultCacheFence(tampered, cache.root, key, physical) {
		t.Fatal("tampered generation fence was accepted")
	}
	tamperedMaterial, _ = redisQueryResultCacheFenceValue(fence)
	tamperedMaterial.epoch = "10"
	tampered = tamperedMaterial
	if validRedisQueryResultCacheFence(tampered, cache.root, key, physical) {
		t.Fatal("tampered allocator epoch was accepted")
	}
	if RedisQueryResultCacheTTL <= 0 || RedisQueryResultCacheTTL > 5*time.Minute ||
		redisQueryCacheTagTTL < RedisQueryResultCacheTTL+maximumExecutionTimeout+redisQueryCacheWAITAOFTimeout ||
		redisQueryCacheMaximumTags != 64 {
		t.Fatalf("Host cache bounds ttl=%s tagTTL=%s tags=%d",
			RedisQueryResultCacheTTL, redisQueryCacheTagTTL, redisQueryCacheMaximumTags)
	}
	for _, invalid := range []string{"", "installation", strings.Repeat("A", 64), strings.Repeat("a", 63)} {
		candidate, candidateErr := NewRedisQueryResultCache(client, invalid)
		if candidate != nil || !errors.Is(candidateErr, ErrExecutionInvalid) {
			t.Fatalf("invalid installation %q accepted: cache=%#v err=%v", invalid, candidate, candidateErr)
		}
	}
}

func TestRedisQueryResultCacheEnvelopeIsStrictAndLossless(t *testing.T) {
	cache := newRedisQueryResultCacheUnit(t, strings.Repeat("a", 64))
	key := strings.Repeat("d", 64)
	tags := redisQueryResultTestTags('a', 'b', 'c')
	value := redisQueryResultTestValue(t, key, tags, false)
	physical, err := cache.physicalGenerationKeys(tags)
	if err != nil {
		t.Fatal(err)
	}
	fence := newQueryResultCacheFence(cache.root, key, "12", physical, []string{"12", "12"})
	fenceMaterial, ok := redisQueryResultCacheFenceValue(fence)
	if !ok {
		t.Fatal("Redis fence material was not retained")
	}
	envelope := redisQueryCacheEnvelope{
		Version: redisQueryCacheEnvelopeVersion, TagDigest: fenceMaterial.tagDigest,
		Result: redisCachedQueryResultFromValue(value),
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	_, decoded, err := cache.decodeEnvelope(encoded, key)
	if err != nil {
		t.Fatal(err)
	}
	number, ok := decoded.Rows[0]["sequence"].(json.Number)
	if !ok || number.String() != "9007199254740993123456789" {
		t.Fatalf("large integer was not lossless: %#v (%T)", decoded.Rows[0]["sequence"], decoded.Rows[0]["sequence"])
	}
	decoded.Rows[0]["title"] = "caller mutation"
	_, fresh, err := cache.decodeEnvelope(encoded, key)
	if err != nil || fresh.Rows[0]["title"] != "cached" {
		t.Fatalf("decoded ownership leaked: value=%#v err=%v", fresh.Rows, err)
	}

	core := redisQueryResultTestValue(t, key, tags, true)
	coreEnvelope := envelope
	coreEnvelope.Result = redisCachedQueryResultFromValue(core)
	coreEncoded, err := json.Marshal(coreEnvelope)
	if err != nil {
		t.Fatal(err)
	}
	_, decodedCore, err := cache.decodeEnvelope(coreEncoded, key)
	if err != nil || decodedCore.Artifact != core.Artifact || !validCoreArtifactSeal(decodedCore.Artifact) {
		t.Fatalf("core artifact seal was not restored: artifact=%#v err=%v", decodedCore.Artifact, err)
	}

	valid := string(encoded)
	rowNeedle := `"sequence":9007199254740993123456789`
	cases := map[string][]byte{
		"unknown envelope field": []byte(strings.Replace(valid, `"version":`, `"unknown":true,"version":`, 1)),
		"unknown result field":   []byte(strings.Replace(valid, `"schemaVersion":`, `"unknown":true,"schemaVersion":`, 1)),
		"duplicate field":        []byte(strings.Replace(valid, `"version":`, `"version":"forged","version":`, 1)),
		"duplicate row key":      []byte(strings.Replace(valid, rowNeedle, `"sequence":1,`+rowNeedle, 1)),
		"trailing document":      append(append([]byte(nil), encoded...), []byte(` {}`)...),
		"wrong envelope version": []byte(strings.Replace(valid, redisQueryCacheEnvelopeVersion, "sforum.query-result-redis@2", 1)),
		"invalid tag digest":     []byte(strings.Replace(valid, fenceMaterial.tagDigest, strings.Repeat("x", 64), 1)),
	}
	for name, poisoned := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, decodeErr := cache.decodeEnvelope(poisoned, key); !errors.Is(decodeErr, ErrCachePoisoned) {
				t.Fatalf("poisoned envelope error=%v", decodeErr)
			}
		})
	}

	deep := []byte(`{"value":` + strings.Repeat("[", redisQueryCacheMaximumJSONDepth+1) +
		`null` + strings.Repeat("]", redisQueryCacheMaximumJSONDepth+1) + `}`)
	if err := validateRedisQueryCacheJSON(deep); err == nil {
		t.Fatal("over-depth JSON was accepted")
	}
	oversized := make([]byte, redisQueryCacheMaximumEnvelope+1)
	if _, _, err := cache.decodeEnvelope(oversized, key); !errors.Is(err, ErrCachePoisoned) {
		t.Fatalf("oversized envelope error=%v", err)
	}
}

func TestRedisQueryResultCacheScriptsUseBoundedAtomicEpochProtocol(t *testing.T) {
	combined := redisQueryCacheSnapshotLua + redisQueryCacheFinalizeLua + redisQueryCacheInvalidateLua
	for _, forbidden := range []string{"ZRANGE", "ZMSCORE", "HGETALL", "HSET", "ZADD", "SCAN", "redis.call('KEYS'", "redis.pcall('KEYS'"} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("reverse-index or scan primitive remains: %s", forbidden)
		}
	}
	for _, required := range []string{"redis.call('MSET'", "error('QUERYCACHE_POISONED'", "return #KEYS - 1"} {
		if !strings.Contains(redisQueryCacheInvalidateLua, required) {
			t.Fatalf("epoch invalidation is missing %q", required)
		}
	}
	if strings.Contains(redisQueryCacheInvalidateLua, "INCR") || strings.Contains(redisQueryCacheInvalidateLua, "cjson") {
		t.Fatal("invalidation uses partial writes or value material")
	}
	for _, required := range []string{"redis.call('MSET'", "redis.call('PEXPIRE'", "redis.call('RENAME'"} {
		if !strings.Contains(redisQueryCacheFinalizeLua, required) {
			t.Fatalf("Store finalize is missing %q", required)
		}
	}
	if strings.Index(redisQueryCacheFinalizeLua, "redis.call('MSET'") > strings.Index(redisQueryCacheFinalizeLua, "redis.call('RENAME'") ||
		strings.Contains(redisQueryCacheFinalizeLua, "cjson") || strings.Contains(redisQueryCacheFinalizeLua, "encoded") {
		t.Fatal("Store publishes before tag materialization or accepts an envelope in Lua")
	}
	if !strings.Contains(redisQueryCacheSnapshotLua, "return result") ||
		strings.Contains(redisQueryCacheSnapshotLua, "redis.call('SET'") || strings.Contains(redisQueryCacheSnapshotLua, "redis.call('MSET'") {
		t.Fatal("generation snapshot is not read-only")
	}
	for _, required := range []string{"marker_kind", "epoch_kind", "redis.call('MSET'"} {
		if !strings.Contains(redisQueryCacheActivateLua, required) {
			t.Fatalf("activation marker protocol is missing %q", required)
		}
	}
}

func TestRedisQueryResultCacheAuthorityFailureCannotReactivate(t *testing.T) {
	hook := newRedisQueryCacheCapabilityHook()
	hook.scriptReplies = []interface{}{int64(1), "1"}
	hook.waitReplies = []interface{}{[]interface{}{int64(1), int64(0)}, []interface{}{int64(1), int64(0)}}
	cache := newRedisQueryResultCacheWithHook(t, hook)
	if err := cache.Activate(context.Background()); err != nil {
		t.Fatal(err)
	}

	tags := redisQueryResultTestTags('a', 'b', 'c')
	hook.scriptReplies = []interface{}{int64(len(tags))}
	hook.waitReplies = []interface{}{[]interface{}{int64(0), int64(0)}}
	if _, err := cache.InvalidateTags(context.Background(), tags); !errors.Is(err, ErrCacheDurability) {
		t.Fatalf("uncertain invalidation error=%v", err)
	}
	commands := hook.processCalls
	if err := cache.Activate(context.Background()); !errors.Is(err, ErrCacheCapability) {
		t.Fatalf("failed cache reactivated: %v", err)
	}
	if hook.processCalls != commands {
		t.Fatalf("failed cache reached Redis again: before=%d after=%d", commands, hook.processCalls)
	}
}

func TestRedisQueryResultCacheRequiresActivation(t *testing.T) {
	hook := newRedisQueryCacheCapabilityHook()
	cache := newRedisQueryResultCacheWithHook(t, hook)
	key := strings.Repeat("d", 64)
	tags := redisQueryResultTestTags('a', 'b', 'c')
	if _, _, _, err := cache.LoadQueryResult(context.Background(), key, tags); !errors.Is(err, ErrCacheCapability) {
		t.Fatalf("inactive Load error=%v", err)
	}
	if _, err := cache.InvalidateTags(context.Background(), tags); !errors.Is(err, ErrCacheCapability) {
		t.Fatalf("inactive invalidation error=%v", err)
	}
	if hook.processCalls != 0 {
		t.Fatalf("inactive cache reached Redis %d times", hook.processCalls)
	}
}

func TestRedisQueryResultCacheSeparatesStoreCASFromTagHitDigest(t *testing.T) {
	cache := newRedisQueryResultCacheUnit(t, strings.Repeat("a", 64))
	key := strings.Repeat("d", 64)
	tags := redisQueryResultTestTags('a', 'b', 'c')
	physical, err := cache.physicalGenerationKeys(tags)
	if err != nil {
		t.Fatal(err)
	}
	first := newQueryResultCacheFence(cache.root, key, "7", physical, []string{"5", "7"})
	second := newQueryResultCacheFence(cache.root, key, "8", physical, []string{"5", "7"})
	if sameQueryResultCacheFence(first, second) || !sameQueryResultCacheTagSnapshot(first, second) {
		t.Fatalf("unrelated allocator advance changed hit tags: first=%#v second=%#v", first, second)
	}
	changed := newQueryResultCacheFence(cache.root, key, "8", physical, []string{"8", "7"})
	if sameQueryResultCacheTagSnapshot(first, changed) {
		t.Fatal("semantic tag advance did not invalidate the hit digest")
	}
}

func TestRedisQueryResultCacheEpochAndWAITAOFValidation(t *testing.T) {
	for _, valid := range []string{"1", "9", "9007199254740991"} {
		if !validRedisQueryCacheEpoch(valid) {
			t.Fatalf("valid generation %q rejected", valid)
		}
	}
	for _, invalid := range []string{"", "0", "00", "01", "-1", "1.0", "9007199254740992", strings.Repeat("9", 17)} {
		if validRedisQueryCacheEpoch(invalid) {
			t.Fatalf("invalid generation %q accepted", invalid)
		}
	}
	if err := validateRedisQueryCacheAOFReply([]int64{1, 0}); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]int64{nil, {0, 0}, {1}, {1, -1}, {1, 0, 0}} {
		if validateRedisQueryCacheAOFReply(invalid) == nil {
			t.Fatalf("invalid WAITAOF reply accepted: %v", invalid)
		}
	}
}

func TestNewRedisQueryResultCacheRequiresNonRetryingClient(t *testing.T) {
	installationID := strings.Repeat("a", 64)
	for name, options := range map[string]*redis.Options{
		"default retries": {Addr: "127.0.0.1:1"},
		"short read": {Addr: "127.0.0.1:1", MaxRetries: -1, ReadTimeout: 2 * time.Second,
			ContextTimeoutEnabled: true},
		"unbounded read": {Addr: "127.0.0.1:1", MaxRetries: -1, ReadTimeout: -1,
			ContextTimeoutEnabled: true},
		"unbounded write": {Addr: "127.0.0.1:1", MaxRetries: -1, ReadTimeout: 5 * time.Second,
			WriteTimeout: -1, ContextTimeoutEnabled: true},
		"context deadlines disabled": {Addr: "127.0.0.1:1", MaxRetries: -1,
			ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			client := redis.NewClient(options)
			t.Cleanup(func() { _ = client.Close() })
			cache, err := NewRedisQueryResultCache(client, installationID)
			if cache != nil || !errors.Is(err, ErrCacheCapability) {
				t.Fatalf("cache=%#v err=%v", cache, err)
			}
		})
	}
	for name, options := range map[string]*redis.Options{
		"bounded deadlines": {Addr: "127.0.0.1:1", MaxRetries: -1, ReadTimeout: 5 * time.Second,
			WriteTimeout: 5 * time.Second, ContextTimeoutEnabled: true},
	} {
		t.Run(name, func(t *testing.T) {
			client := redis.NewClient(options)
			t.Cleanup(func() { _ = client.Close() })
			cache, err := NewRedisQueryResultCache(client, installationID)
			if err != nil || cache == nil {
				t.Fatalf("cache=%#v err=%v", cache, err)
			}
		})
	}
}

func TestRedisQueryResultCacheCapabilityGate(t *testing.T) {
	for name, mutate := range map[string]func(*redisQueryCacheCapabilityHook){
		"redis 7.1": func(h *redisQueryCacheCapabilityHook) {
			h.server = "redis_version:7.1.9\r\n"
		},
		"AOF disabled": func(h *redisQueryCacheCapabilityHook) {
			h.persistence = strings.Replace(h.persistence, "aof_enabled:1", "aof_enabled:0", 1)
		},
		"AOF write unhealthy": func(h *redisQueryCacheCapabilityHook) {
			h.persistence = strings.Replace(h.persistence, "aof_last_write_status:ok", "aof_last_write_status:err", 1)
		},
		"AOF rewrite unhealthy": func(h *redisQueryCacheCapabilityHook) {
			h.persistence = strings.Replace(h.persistence, "aof_last_bgrewrite_status:ok", "aof_last_bgrewrite_status:err", 1)
		},
		"appendonly disabled": func(h *redisQueryCacheCapabilityHook) {
			h.config["appendonly"] = "no"
		},
		"appendfsync disabled": func(h *redisQueryCacheCapabilityHook) {
			h.config["appendfsync"] = "no"
		},
		"appendfsync unknown": func(h *redisQueryCacheCapabilityHook) {
			h.config["appendfsync"] = "forged"
		},
		"eviction enabled": func(h *redisQueryCacheCapabilityHook) {
			h.config["maxmemory-policy"] = "allkeys-lru"
		},
		"probe rejected": func(h *redisQueryCacheCapabilityHook) {
			h.scriptReply = int64(0)
		},
		"WAITAOF rejected": func(h *redisQueryCacheCapabilityHook) {
			h.waitReply = []interface{}{int64(0), int64(0)}
		},
		"WAITAOF malformed": func(h *redisQueryCacheCapabilityHook) {
			h.waitReply = []interface{}{int64(1)}
		},
	} {
		t.Run(name, func(t *testing.T) {
			hook := newRedisQueryCacheCapabilityHook()
			mutate(hook)
			cache := newRedisQueryResultCacheWithHook(t, hook)
			if err := cache.CheckCapabilities(context.Background()); !errors.Is(err, ErrCacheCapability) {
				t.Fatalf("capability error=%v", err)
			}
		})
	}

	for _, fsyncPolicy := range []string{"always", "everysec"} {
		t.Run("accept "+fsyncPolicy, func(t *testing.T) {
			hook := newRedisQueryCacheCapabilityHook()
			hook.config["appendfsync"] = fsyncPolicy
			cache := newRedisQueryResultCacheWithHook(t, hook)
			if err := cache.CheckCapabilities(context.Background()); err != nil {
				t.Fatal(err)
			}
			want := []interface{}{"WAITAOF", 1, 0, int64(redisQueryCacheWAITAOFTimeout.Milliseconds())}
			if len(hook.waitArgs) != len(want) {
				t.Fatalf("WAITAOF args=%v", hook.waitArgs)
			}
			for index := range want {
				if fmt.Sprint(hook.waitArgs[index]) != fmt.Sprint(want[index]) {
					t.Fatalf("WAITAOF args=%v want=%v", hook.waitArgs, want)
				}
			}
		})
	}
}

func TestRedisCachedQueryResultRejectsDuplicateOrMalformedTags(t *testing.T) {
	key := strings.Repeat("d", 64)
	tags := redisQueryResultTestTags('a', 'b', 'c')
	value := redisQueryResultTestValue(t, key, tags, false)
	if _, err := validateRedisCachedQueryResult(value, key); err != nil {
		t.Fatal(err)
	}
	value.CacheTags = []string{tags[0], tags[0]}
	if _, err := validateRedisCachedQueryResult(value, key); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("duplicate tags error=%v", err)
	}
	value.CacheTags = []string{"plugin-controlled"}
	if _, err := validateRedisCachedQueryResult(value, key); !errors.Is(err, ErrExecutionInvalid) {
		t.Fatalf("malformed tags error=%v", err)
	}
}

func newRedisQueryResultCacheUnit(t *testing.T, installationID string) *RedisQueryResultCache {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", MaxRetries: -1, ContextTimeoutEnabled: true,
	})
	t.Cleanup(func() { _ = client.Close() })
	cache, err := NewRedisQueryResultCache(client, installationID)
	if err != nil {
		t.Fatal(err)
	}
	return cache
}

type redisQueryCacheCapabilityHook struct {
	server        string
	persistence   string
	config        map[string]string
	scriptReply   interface{}
	waitReply     interface{}
	waitArgs      []interface{}
	scriptReplies []interface{}
	waitReplies   []interface{}
	processCalls  int
}

func newRedisQueryCacheCapabilityHook() *redisQueryCacheCapabilityHook {
	return &redisQueryCacheCapabilityHook{
		server: "redis_version:7.4.2\r\n",
		persistence: "aof_enabled:1\r\naof_last_write_status:ok\r\n" +
			"aof_last_bgrewrite_status:ok\r\n",
		config: map[string]string{
			"appendonly":       "yes",
			"appendfsync":      "everysec",
			"maxmemory-policy": "noeviction",
		},
		scriptReply: int64(1),
		waitReply:   []interface{}{int64(1), int64(0)},
	}
}

func (h *redisQueryCacheCapabilityHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (h *redisQueryCacheCapabilityHook) ProcessHook(_ redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		h.processCalls++
		switch cmd.Name() {
		case "info":
			arguments := cmd.Args()
			if len(arguments) != 2 {
				return fmt.Errorf("unexpected INFO arguments %v", arguments)
			}
			value := h.server
			if fmt.Sprint(arguments[1]) == "persistence" {
				value = h.persistence
			}
			cmd.(*redis.StringCmd).SetVal(value)
			return nil
		case "config":
			arguments := cmd.Args()
			if len(arguments) != 3 || fmt.Sprint(arguments[1]) != "get" {
				return fmt.Errorf("unexpected CONFIG arguments %v", arguments)
			}
			key := fmt.Sprint(arguments[2])
			cmd.(*redis.MapStringStringCmd).SetVal(map[string]string{key: h.config[key]})
			return nil
		case "evalsha", "eval":
			value := h.scriptReply
			if len(h.scriptReplies) > 0 {
				value = h.scriptReplies[0]
				h.scriptReplies = h.scriptReplies[1:]
			}
			cmd.(*redis.Cmd).SetVal(value)
			return nil
		case "waitaof":
			h.waitArgs = append([]interface{}(nil), cmd.Args()...)
			value := h.waitReply
			if len(h.waitReplies) > 0 {
				value = h.waitReplies[0]
				h.waitReplies = h.waitReplies[1:]
			}
			cmd.(*redis.Cmd).SetVal(value)
			return nil
		case "del":
			cmd.(*redis.IntCmd).SetVal(0)
			return nil
		default:
			return fmt.Errorf("unexpected Redis command %q", cmd.Name())
		}
	}
}

func (h *redisQueryCacheCapabilityHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		return next(ctx, commands)
	}
}

func newRedisQueryResultCacheWithHook(t *testing.T, hook redis.Hook) *RedisQueryResultCache {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", MaxRetries: -1, ReadTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second, ContextTimeoutEnabled: true,
	})
	client.AddHook(hook)
	t.Cleanup(func() { _ = client.Close() })
	cache, err := NewRedisQueryResultCache(client, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return cache
}

func redisQueryResultTestTags(shared, isolated, semantic byte) []string {
	return []string{
		"query:shared:" + strings.Repeat(string(shared), 32),
		"query:" + strings.Repeat(string(isolated), 32) + ":" + strings.Repeat(string(semantic), 32),
	}
}

func redisQueryResultTestValue(t *testing.T, key string, tags []string, core bool) CachedQueryResult {
	t.Helper()
	artifact := Artifact{
		ExtensionID: "plugin.redis-cache", ExtensionVersion: "1.0.0", PackageDigest: strings.Repeat("b", 64),
		VersionID: 17, RuntimeInstanceID: "runtime-redis-cache",
	}
	queryID := "plugin.redis-cache.items"
	if core {
		var err error
		artifact, err = NewCoreArtifact("core.forum", "1.0.0", strings.Repeat("a", 64))
		if err != nil {
			t.Fatal(err)
		}
		queryID = "core.forum.items"
	}
	return CachedQueryResult{
		SchemaVersion: resultCacheSchemaVersion, CacheKey: key,
		RegistryRevision: 9, RegistryDigest: strings.Repeat("1", 64), ShapeDigest: strings.Repeat("2", 64),
		FilterPlan: strings.Repeat("3", 64), QueryID: queryID, ContractVersion: queryID + "@1",
		PlanVersion: queryID + ".plan@1", ResultSchema: queryID + ".result@1",
		Artifact: artifact, ProviderDigest: strings.Repeat("4", 64),
		Page:      QueryResultPage{Mode: PaginationOffset, Limit: 10},
		Rows:      []QueryRow{{"id": "1", "title": "cached", "sequence": json.Number("9007199254740993123456789")}},
		CacheTags: slices.Clone(tags),
	}
}
