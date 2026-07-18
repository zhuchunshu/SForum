package queryregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

const redisQueryCacheLuaHelpers = `
local function key_type(key)
  local result = redis.call('TYPE', key)
  if type(result) == 'table' then
    return result['ok']
  end
  return result
end

local function valid_generation(value, maximum)
  if not value or string.len(value) > 16 or string.match(value, '^[1-9][0-9]*$') == nil then
    return false
  end
  local number = tonumber(value)
  return number and number <= maximum
end

local function epoch(key, maximum)
  local kind = key_type(key)
  if kind == 'none' then
    error('QUERYCACHE_INACTIVE', 0)
  end
  if kind ~= 'string' then
    error('QUERYCACHE_POISONED', 0)
  end
  local value = redis.call('GET', key)
  if not valid_generation(value, maximum) or redis.call('PTTL', key) ~= -1 then
    error('QUERYCACHE_POISONED', 0)
  end
  return value
end

local function tag_generation(key, fallback, maximum, maximum_ttl)
  local kind = key_type(key)
  if kind == 'none' then
    return fallback, false
  end
  if kind ~= 'string' then
    error('QUERYCACHE_POISONED', 0)
  end
  local value = redis.call('GET', key)
  local ttl = redis.call('PTTL', key)
  if not valid_generation(value, maximum) or ttl <= 0 or ttl > maximum_ttl then
    error('QUERYCACHE_POISONED', 0)
  end
  return value, true
end
`

const redisQueryCacheSnapshotLua = redisQueryCacheLuaHelpers + `
if #KEYS < 2 or #ARGV ~= 2 then
  error('QUERYCACHE_POISONED', 0)
end
local maximum = tonumber(ARGV[1])
local maximum_ttl = tonumber(ARGV[2])
if not maximum or not maximum_ttl or maximum_ttl <= 0 then
  error('QUERYCACHE_POISONED', 0)
end
local current_epoch = epoch(KEYS[1], maximum)
local result = {current_epoch}
for index = 2, #KEYS do
  local value = tag_generation(KEYS[index], current_epoch, maximum, maximum_ttl)
  result[index] = value
end
return result
`

// The envelope is staged with SET outside Lua. Finalize receives only bounded
// epoch material, pins every tag before publication, then atomically renames
// the already-encoded value into place.
const redisQueryCacheFinalizeLua = redisQueryCacheLuaHelpers + `
if #KEYS < 4 or #ARGV ~= #KEYS + 1 then
  error('QUERYCACHE_POISONED', 0)
end
local expected_epoch = ARGV[1]
local maximum = tonumber(ARGV[2])
local value_ttl = tonumber(ARGV[3])
local tag_ttl = tonumber(ARGV[4])
if not valid_generation(expected_epoch, maximum) or not value_ttl or value_ttl <= 0 or
   not tag_ttl or tag_ttl <= value_ttl then
  error('QUERYCACHE_POISONED', 0)
end
if key_type(KEYS[1]) ~= 'string' then
  error('QUERYCACHE_TEMPORARY_MISSING', 0)
end
local temporary_ttl = redis.call('PTTL', KEYS[1])
if temporary_ttl <= 0 or temporary_ttl > tag_ttl then
  error('QUERYCACHE_POISONED', 0)
end
local final_kind = key_type(KEYS[2])
if final_kind ~= 'none' and final_kind ~= 'string' then
  error('QUERYCACHE_POISONED', 0)
end
if final_kind == 'string' then
  local final_ttl = redis.call('PTTL', KEYS[2])
  if final_ttl <= 0 or final_ttl > value_ttl then
    error('QUERYCACHE_POISONED', 0)
  end
end
local current_epoch = epoch(KEYS[3], maximum)
if current_epoch ~= expected_epoch then
  redis.call('DEL', KEYS[1])
  return 0
end
local missing = {}
for index = 4, #KEYS do
  local value, exists = tag_generation(KEYS[index], current_epoch, maximum, tag_ttl)
  if value ~= ARGV[index + 1] then
    redis.call('DEL', KEYS[1])
    return 0
  end
  if not exists then
    missing[#missing + 1] = KEYS[index]
    missing[#missing + 1] = value
  end
end
if #missing > 0 then
  redis.call('MSET', unpack(missing))
end
for index = 4, #KEYS do
  redis.call('PEXPIRE', KEYS[index], tag_ttl)
end
redis.call('PEXPIRE', KEYS[1], value_ttl)
redis.call('RENAME', KEYS[1], KEYS[2])
return 1
`

// Invalidation computes the next allocator value before its first write and
// commits the allocator plus all tags in one MSET. The following PEXPIRE calls
// cannot expose a partially invalidated tag set.
const redisQueryCacheInvalidateLua = redisQueryCacheLuaHelpers + `
if #KEYS < 2 or #ARGV ~= 2 then
  error('QUERYCACHE_POISONED', 0)
end
local maximum = tonumber(ARGV[1])
local tag_ttl = tonumber(ARGV[2])
if not maximum or not tag_ttl or tag_ttl <= 0 then
  error('QUERYCACHE_POISONED', 0)
end
local current_epoch = epoch(KEYS[1], maximum)
local current_number = tonumber(current_epoch)
if not current_number or current_number >= maximum then
  error('QUERYCACHE_POISONED', 0)
end
for index = 2, #KEYS do
  tag_generation(KEYS[index], current_epoch, maximum, tag_ttl)
end
local next_epoch = string.format('%.0f', current_number + 1)
local writes = {KEYS[1], next_epoch}
for index = 2, #KEYS do
  writes[#writes + 1] = KEYS[index]
  writes[#writes + 1] = next_epoch
end
redis.call('MSET', unpack(writes))
for index = 2, #KEYS do
  redis.call('PEXPIRE', KEYS[index], tag_ttl)
end
return #KEYS - 1
`

const redisQueryCacheActivateLua = redisQueryCacheLuaHelpers + `
if #KEYS ~= 2 or #ARGV ~= 3 then
  error('QUERYCACHE_POISONED', 0)
end
local expected_marker = ARGV[1]
local initial = ARGV[2]
local maximum = tonumber(ARGV[3])
if not expected_marker or string.len(expected_marker) == 0 or string.len(expected_marker) > 128 or
   not valid_generation(initial, maximum) then
  error('QUERYCACHE_POISONED', 0)
end
local marker_kind = key_type(KEYS[1])
local epoch_kind = key_type(KEYS[2])
local next_epoch = initial
if marker_kind == 'none' and epoch_kind == 'none' then
  -- The marker and allocator share one first-install write. A later one-key
  -- loss can therefore never be mistaken for a clean installation.
elseif marker_kind ~= 'string' or epoch_kind ~= 'string' then
  error('QUERYCACHE_POISONED', 0)
else
  if redis.call('GET', KEYS[1]) ~= expected_marker or redis.call('PTTL', KEYS[1]) ~= -1 then
    error('QUERYCACHE_POISONED', 0)
  end
  local current = redis.call('GET', KEYS[2])
  if not valid_generation(current, maximum) or redis.call('PTTL', KEYS[2]) ~= -1 then
    error('QUERYCACHE_POISONED', 0)
  end
  local current_number = tonumber(current)
  if not current_number or current_number >= maximum then
    error('QUERYCACHE_POISONED', 0)
  end
  next_epoch = string.format('%.0f', current_number + 1)
end
redis.call('MSET', KEYS[1], expected_marker, KEYS[2], next_epoch)
return next_epoch
`

const redisQueryCacheCapabilityLua = `
redis.call('SET', KEYS[1], 'probe', 'PX', 10000)
redis.call('MSET', KEYS[3], '1')
redis.call('PEXPIRE', KEYS[3], 10000)
redis.call('RENAME', KEYS[1], KEYS[2])
redis.call('PEXPIRE', KEYS[2], 10000)
redis.call('DEL', KEYS[2], KEYS[3])
return 1
`

var (
	redisQueryCacheSnapshotScript   = redis.NewScript(redisQueryCacheSnapshotLua)
	redisQueryCacheFinalizeScript   = redis.NewScript(redisQueryCacheFinalizeLua)
	redisQueryCacheInvalidateScript = redis.NewScript(redisQueryCacheInvalidateLua)
	redisQueryCacheActivateScript   = redis.NewScript(redisQueryCacheActivateLua)
	redisQueryCacheCapabilityScript = redis.NewScript(redisQueryCacheCapabilityLua)
)

// Activate fails closed unless Redis can preserve the permanent allocator and
// acknowledge every authoritative cache write through local AOF.
func (c *RedisQueryResultCache) Activate(ctx context.Context) error {
	if c == nil || c.client == nil || ctx == nil {
		return ErrExecutionInvalid
	}
	if c.state.Load() == redisQueryCacheStateFailed {
		return ErrCacheCapability
	}
	if err := c.CheckCapabilities(ctx); err != nil {
		return c.failAuthority(err)
	}
	conn := c.client.Conn()
	defer conn.Close()
	epoch, err := redisQueryCacheActivateScript.Run(
		ctx, conn, []string{c.markerKey, c.epochKey}, redisQueryCacheActivationMarker,
		redisQueryCacheInitialEpoch, redisQueryCacheMaximumGeneration,
	).Text()
	if redisQueryCacheMarkedPoisoned(err) {
		return c.failAuthority(ErrCachePoisoned)
	}
	if err != nil || !validRedisQueryCacheEpoch(epoch) {
		return c.failAuthority(errors.Join(ErrCacheCapability, err))
	}
	if err := waitForRedisQueryCacheAOF(ctx, conn, c.waitAOFTimeout); err != nil {
		return c.failAuthority(errors.Join(ErrCacheCapability, err))
	}
	return c.markActive()
}

func (c *RedisQueryResultCache) CheckCapabilities(ctx context.Context) error {
	if c == nil || c.client == nil || ctx == nil {
		return ErrExecutionInvalid
	}
	options := c.client.Options()
	if !validRedisQueryCacheClientOptions(options, c.waitAOFTimeout) {
		return ErrCacheCapability
	}
	server, err := c.client.Info(ctx, "server").Result()
	if err != nil || !redisQueryCacheVersionAtLeast72(server) {
		return errors.Join(ErrCacheCapability, err)
	}
	persistence, err := c.client.Info(ctx, "persistence").Result()
	if err != nil || redisQueryCacheInfoValue(persistence, "aof_enabled") != "1" ||
		redisQueryCacheInfoValue(persistence, "aof_last_write_status") != "ok" ||
		redisQueryCacheInfoValue(persistence, "aof_last_bgrewrite_status") != "ok" {
		return errors.Join(ErrCacheCapability, err)
	}
	appendonly, err := c.client.ConfigGet(ctx, "appendonly").Result()
	if err != nil || strings.ToLower(appendonly["appendonly"]) != "yes" {
		return errors.Join(ErrCacheCapability, err)
	}
	appendfsync, err := c.client.ConfigGet(ctx, "appendfsync").Result()
	fsyncPolicy := strings.ToLower(appendfsync["appendfsync"])
	if err != nil || (fsyncPolicy != "always" && fsyncPolicy != "everysec") {
		return errors.Join(ErrCacheCapability, err)
	}
	policy, err := c.client.ConfigGet(ctx, "maxmemory-policy").Result()
	if err != nil || strings.ToLower(policy["maxmemory-policy"]) != "noeviction" {
		return errors.Join(ErrCacheCapability, err)
	}

	conn := c.client.Conn()
	defer conn.Close()
	probe := c.root + "capability:"
	probeKeys := []string{probe + "source", probe + "target", probe + "tag"}
	defer c.cleanupRedisQueryCacheKeys(conn, probeKeys...)
	value, err := redisQueryCacheCapabilityScript.Run(
		ctx, conn, probeKeys,
	).Int64()
	if err != nil || value != 1 {
		return errors.Join(ErrCacheCapability, err)
	}
	if err := waitForRedisQueryCacheAOF(ctx, conn, c.waitAOFTimeout); err != nil {
		return errors.Join(ErrCacheCapability, err)
	}
	return nil
}

func newQueryResultCacheFence(
	root, key, epoch string,
	physical, generations []string,
) QueryResultCacheFence {
	material := redisQueryResultCacheFence{
		root: root, cacheKey: key, epoch: epoch,
		physical: slices.Clone(physical), generations: slices.Clone(generations),
	}
	material.tagDigest = queryResultCacheTagDigest(root, key, physical, generations)
	material.digest = queryResultCacheFenceDigest(root, key, epoch, material.tagDigest)
	return material
}

func (f redisQueryResultCacheFence) validFor(root, key string, physical []string) bool {
	if f.root != root || f.cacheKey != key || !validRedisQueryCacheEpoch(f.epoch) ||
		!validRedisQueryCacheDigest(f.tagDigest) || !validRedisQueryCacheDigest(f.digest) ||
		!slices.Equal(f.physical, physical) || len(f.generations) != len(physical) {
		return false
	}
	for _, generation := range f.generations {
		if !validRedisQueryCacheEpoch(generation) {
			return false
		}
	}
	return f.tagDigest == queryResultCacheTagDigest(f.root, f.cacheKey, f.physical, f.generations) &&
		f.digest == queryResultCacheFenceDigest(f.root, f.cacheKey, f.epoch, f.tagDigest)
}

func validRedisQueryResultCacheFence(f QueryResultCacheFence, root, key string, physical []string) bool {
	material, ok := redisQueryResultCacheFenceValue(f)
	return ok && material.validFor(root, key, physical)
}

func sameQueryResultCacheFence(left, right QueryResultCacheFence) bool {
	leftMaterial, leftOK := redisQueryResultCacheFenceValue(left)
	rightMaterial, rightOK := redisQueryResultCacheFenceValue(right)
	return leftOK && rightOK && leftMaterial.digest == rightMaterial.digest &&
		leftMaterial.root == rightMaterial.root && leftMaterial.cacheKey == rightMaterial.cacheKey &&
		leftMaterial.epoch == rightMaterial.epoch && leftMaterial.tagDigest == rightMaterial.tagDigest &&
		slices.Equal(leftMaterial.physical, rightMaterial.physical) &&
		slices.Equal(leftMaterial.generations, rightMaterial.generations)
}

func sameQueryResultCacheTagSnapshot(left, right QueryResultCacheFence) bool {
	leftMaterial, leftOK := redisQueryResultCacheFenceValue(left)
	rightMaterial, rightOK := redisQueryResultCacheFenceValue(right)
	return leftOK && rightOK && leftMaterial.root == rightMaterial.root &&
		leftMaterial.cacheKey == rightMaterial.cacheKey && leftMaterial.tagDigest == rightMaterial.tagDigest &&
		slices.Equal(leftMaterial.physical, rightMaterial.physical) &&
		slices.Equal(leftMaterial.generations, rightMaterial.generations)
}

func redisQueryResultCacheFenceValue(fence QueryResultCacheFence) (redisQueryResultCacheFence, bool) {
	material, ok := fence.(redisQueryResultCacheFence)
	return material, ok
}

func queryResultCacheTagDigest(root, key string, physical, generations []string) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, redisQueryCacheEnvelopeVersion+"\x00tags\x00"+root+"\x00"+key)
	for index := range physical {
		_, _ = io.WriteString(hash, "\x00"+physical[index]+"\x00"+generations[index])
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func queryResultCacheFenceDigest(root, key, epoch, tagDigest string) string {
	digest := sha256.Sum256([]byte(
		redisQueryCacheEnvelopeVersion + "\x00fence\x00" + root + "\x00" + key + "\x00" + epoch + "\x00" + tagDigest,
	))
	return hex.EncodeToString(digest[:])
}

func (c *RedisQueryResultCache) valueKey(key string) string {
	return c.valuePrefix + redisQueryCacheHash("value", key)
}

func (c *RedisQueryResultCache) physicalGenerationKeys(tags []string) ([]string, error) {
	if c == nil || len(tags) == 0 || len(tags) > redisQueryCacheMaximumTags {
		return nil, ErrExecutionInvalid
	}
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if !validLogicalQueryCacheTag(tag) {
			return nil, ErrExecutionInvalid
		}
		physical := c.tagPrefix + redisQueryCacheHash("tag", tag)
		if _, duplicate := seen[physical]; duplicate {
			return nil, ErrExecutionInvalid
		}
		seen[physical] = struct{}{}
		result = append(result, physical)
	}
	return result, nil
}

func redisQueryCacheHash(domain, value string) string {
	digest := sha256.Sum256([]byte(redisQueryCacheEnvelopeVersion + "\x00" + domain + "\x00" + value))
	return hex.EncodeToString(digest[:])
}

func validLogicalQueryCacheTag(value string) bool {
	const sharedPrefix = "query:shared:"
	if strings.HasPrefix(value, sharedPrefix) {
		return validRedisQueryCacheHex(value[len(sharedPrefix):], 32)
	}
	const isolatedPrefix = "query:"
	if !strings.HasPrefix(value, isolatedPrefix) {
		return false
	}
	suffix := value[len(isolatedPrefix):]
	return len(suffix) == 65 && suffix[32] == ':' &&
		validRedisQueryCacheHex(suffix[:32], 32) && validRedisQueryCacheHex(suffix[33:], 32)
}

func validRedisQueryCacheEpoch(value string) bool {
	if value == "" || len(value) > 16 || value[0] == '0' {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	generation, err := strconv.ParseUint(value, 10, 64)
	return err == nil && generation > 0 && generation <= redisQueryCacheMaximumGeneration
}

func validRedisQueryCacheDigest(value string) bool {
	return validRedisQueryCacheHex(value, 64)
}

func validRedisQueryCacheHex(value string, length int) bool {
	if len(value) != length || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == length/2
}

func redisQueryCacheMarkedPoisoned(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "QUERYCACHE_POISONED") ||
		strings.Contains(err.Error(), "WRONGTYPE"))
}

func redisQueryCacheMarkedInactive(err error) bool {
	return err != nil && strings.Contains(err.Error(), "QUERYCACHE_INACTIVE")
}

func redisQueryCacheVersionAtLeast72(info string) bool {
	version := redisQueryCacheInfoValue(info, "redis_version")
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	return majorErr == nil && minorErr == nil && (major > 7 || major == 7 && minor >= 2)
}

func redisQueryCacheInfoValue(info, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}
