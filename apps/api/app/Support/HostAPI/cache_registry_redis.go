package hostapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	hostCacheRedisMaximumEnvelopeBytes = HostCacheMaximumValueBytes * 2
	hostCacheRedisMaximumTagMembers    = 10_000
)

var (
	hostCacheRedisSetScript = redis.NewScript(`
local expected = ARGV[3]
local current = redis.call('GET', KEYS[1])
if expected ~= '' then
  if not current then
    return redis.error_reply('HOSTCACHE_CONFLICT')
  end
  local ok, decoded = pcall(cjson.decode, current)
  if not ok or decoded['revision'] ~= expected then
    return redis.error_reply('HOSTCACHE_CONFLICT')
  end
end
for index = 2, #KEYS do
  if redis.call('SCARD', KEYS[index]) >= tonumber(ARGV[5]) and redis.call('SISMEMBER', KEYS[index], KEYS[1]) == 0 then
    return redis.error_reply('HOSTCACHE_TAG_LIMIT')
  end
end
if current then
  local ok, decoded = pcall(cjson.decode, current)
  if ok and type(decoded['tags']) == 'table' then
    for _, tag in ipairs(decoded['tags']) do
      if type(tag) == 'string' and string.sub(tag, 1, string.len(ARGV[4])) == ARGV[4] then
        redis.call('SREM', tag, KEYS[1])
      end
    end
  end
end
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
for index = 2, #KEYS do
  redis.call('SADD', KEYS[index], KEYS[1])
  local current_ttl = redis.call('PTTL', KEYS[index])
  if current_ttl < tonumber(ARGV[2]) then
    redis.call('PEXPIRE', KEYS[index], ARGV[2])
  end
end
return 1
`)
	hostCacheRedisDeleteScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if not current then
  return 0
end
local ok, decoded = pcall(cjson.decode, current)
if ok and type(decoded['tags']) == 'table' then
  for _, tag in ipairs(decoded['tags']) do
    if type(tag) == 'string' and string.sub(tag, 1, string.len(ARGV[1])) == ARGV[1] then
      redis.call('SREM', tag, KEYS[1])
    end
  end
end
redis.call('DEL', KEYS[1])
return 1
`)
	hostCacheRedisInvalidateScript = redis.NewScript(`
local limit = tonumber(ARGV[3])
local function is_value_member(member)
  if type(member) ~= 'string' or string.sub(member, 1, string.len(ARGV[2])) ~= ARGV[2] then
    return false
  end
  local suffix = string.sub(member, string.len(ARGV[2]) + 1)
  if string.len(suffix) ~= 135 or string.sub(suffix, 65, 71) ~= ':value:' then
    return false
  end
  local segment = string.sub(suffix, 1, 64)
  local digest = string.sub(suffix, 72)
  return string.match(segment, '^[0-9a-f]+$') ~= nil and string.match(digest, '^[0-9a-f]+$') ~= nil
end
for index = 1, #KEYS do
  if redis.call('SCARD', KEYS[index]) > limit then
    return redis.error_reply('HOSTCACHE_INVALIDATION_LIMIT')
  end
end
local members = {}
local total = 0
for index = 1, #KEYS do
  local values = redis.call('SMEMBERS', KEYS[index])
  for _, member in ipairs(values) do
    if not is_value_member(member) then
      return redis.error_reply('HOSTCACHE_POISONED')
    end
    if not members[member] then
      total = total + 1
      if total > limit then
        return redis.error_reply('HOSTCACHE_INVALIDATION_LIMIT')
      end
      members[member] = true
    end
  end
end
local count = 0
for member, _ in pairs(members) do
  local current = redis.call('GET', member)
  if current then
    local ok, decoded = pcall(cjson.decode, current)
    if ok and type(decoded['tags']) == 'table' then
      for _, tag in ipairs(decoded['tags']) do
        if type(tag) == 'string' and string.sub(tag, 1, string.len(ARGV[1])) == ARGV[1] then
          redis.call('SREM', tag, member)
        end
      end
    end
    redis.call('DEL', member)
    count = count + 1
  end
end
for index = 1, #KEYS do
  redis.call('DEL', KEYS[index])
end
return count
`)
	hostCacheRedisIncrementScript = redis.NewScript(`
local value = redis.call('INCRBY', KEYS[1], ARGV[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return value
`)
	hostCacheRedisReleaseLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)
	hostCacheRedisRenewLockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 0
`)
	hostCacheRedisSetAndReleaseLockScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[6] then
  return redis.error_reply('HOSTCACHE_LOCK_NOT_OWNED')
end
local expected = ARGV[3]
local current = redis.call('GET', KEYS[1])
if expected ~= '' then
  if not current then
    return redis.error_reply('HOSTCACHE_CONFLICT')
  end
  local ok, decoded = pcall(cjson.decode, current)
  if not ok or decoded['revision'] ~= expected then
    return redis.error_reply('HOSTCACHE_CONFLICT')
  end
end
for index = 3, #KEYS do
  if redis.call('SCARD', KEYS[index]) >= tonumber(ARGV[5]) and redis.call('SISMEMBER', KEYS[index], KEYS[1]) == 0 then
    return redis.error_reply('HOSTCACHE_TAG_LIMIT')
  end
end
if current then
  local ok, decoded = pcall(cjson.decode, current)
  if ok and type(decoded['tags']) == 'table' then
    for _, tag in ipairs(decoded['tags']) do
      if type(tag) == 'string' and string.sub(tag, 1, string.len(ARGV[4])) == ARGV[4] then
        redis.call('SREM', tag, KEYS[1])
      end
    end
  end
end
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
for index = 3, #KEYS do
  redis.call('SADD', KEYS[index], KEYS[1])
  local current_ttl = redis.call('PTTL', KEYS[index])
  if current_ttl < tonumber(ARGV[2]) then
    redis.call('PEXPIRE', KEYS[index], ARGV[2])
  end
end
redis.call('DEL', KEYS[2])
return 1
`)
)

// HostRedisCacheBackend uses the shared production go-redis client. The caller
// owns the client lifecycle, matching Support/Cache.RedisCache.
type HostRedisCacheBackend struct {
	client *redis.Client
}

func NewHostRedisCacheBackend(client *redis.Client) (*HostRedisCacheBackend, error) {
	if client == nil {
		return nil, ErrHostCacheInvalid
	}
	return &HostRedisCacheBackend{client: client}, nil
}

func (b *HostRedisCacheBackend) Get(ctx context.Context, key string) (HostCacheStoredValue, bool, error) {
	if b == nil || b.client == nil || ctx == nil {
		return HostCacheStoredValue{}, false, ErrHostCacheInvalid
	}
	encoded, err := b.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return HostCacheStoredValue{}, false, nil
	}
	if err != nil {
		return HostCacheStoredValue{}, false, err
	}
	if len(encoded) == 0 || len(encoded) > hostCacheRedisMaximumEnvelopeBytes {
		return HostCacheStoredValue{}, false, ErrHostCachePoisoned
	}
	var value HostCacheStoredValue
	if err := json.Unmarshal(encoded, &value); err != nil {
		return HostCacheStoredValue{}, false, ErrHostCachePoisoned
	}
	return cloneHostCacheStoredValue(value), true, nil
}

func (b *HostRedisCacheBackend) Set(
	ctx context.Context,
	key string,
	value HostCacheStoredValue,
	ttl time.Duration,
	expectedRevision string,
	tagPrefix string,
) error {
	if b == nil || b.client == nil || ctx == nil || validateHostCacheTTL(ttl) != nil ||
		!strings.HasSuffix(tagPrefix, ":tag:") || len(value.Tags) > HostCacheMaximumTags {
		return ErrHostCacheInvalid
	}
	for _, tag := range value.Tags {
		if !strings.HasPrefix(tag, tagPrefix) {
			return ErrHostCacheInvalid
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > hostCacheRedisMaximumEnvelopeBytes {
		return ErrHostCacheInvalid
	}
	keys := make([]string, 1, len(value.Tags)+1)
	keys[0] = key
	keys = append(keys, value.Tags...)
	err = hostCacheRedisSetScript.Run(ctx, b.client, keys,
		encoded, ttl.Milliseconds(), expectedRevision, tagPrefix, hostCacheRedisMaximumTagMembers,
	).Err()
	if hostCacheRedisError(err, "HOSTCACHE_CONFLICT") {
		return ErrHostCacheConflict
	}
	return err
}

func (b *HostRedisCacheBackend) Delete(ctx context.Context, key, tagPrefix string) (bool, error) {
	if b == nil || b.client == nil || ctx == nil || !strings.HasSuffix(tagPrefix, ":tag:") {
		return false, ErrHostCacheInvalid
	}
	value, err := hostCacheRedisDeleteScript.Run(ctx, b.client, []string{key}, tagPrefix).Int64()
	return value == 1, err
}

func (b *HostRedisCacheBackend) InvalidateTags(
	ctx context.Context,
	tags []string,
	tagPrefix string,
) (uint64, error) {
	if b == nil || b.client == nil || ctx == nil || len(tags) == 0 || len(tags) > HostCacheMaximumTags ||
		!strings.HasSuffix(tagPrefix, ":tag:") {
		return 0, ErrHostCacheInvalid
	}
	for _, tag := range tags {
		if !strings.HasPrefix(tag, tagPrefix) {
			return 0, ErrHostCacheInvalid
		}
	}
	entryPrefix := strings.TrimSuffix(tagPrefix, ":tag:") + ":segment:"
	value, err := hostCacheRedisInvalidateScript.Run(ctx, b.client, tags,
		tagPrefix, entryPrefix, hostCacheRedisMaximumTagMembers,
	).Uint64()
	if hostCacheRedisError(err, "HOSTCACHE_POISONED") {
		return 0, ErrHostCachePoisoned
	}
	return value, err
}

func (b *HostRedisCacheBackend) Increment(
	ctx context.Context,
	key string,
	delta int64,
	ttl time.Duration,
) (int64, error) {
	if b == nil || b.client == nil || ctx == nil || delta == 0 || validateHostCacheTTL(ttl) != nil {
		return 0, ErrHostCacheInvalid
	}
	return hostCacheRedisIncrementScript.Run(ctx, b.client, []string{key}, delta, ttl.Milliseconds()).Int64()
}

func (b *HostRedisCacheBackend) AcquireLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	if b == nil || b.client == nil || ctx == nil || validateHostCacheRevision(owner, false) != nil ||
		ttl < HostCacheMinimumLockTTL || ttl > HostCacheMaximumLockTTL {
		return false, ErrHostCacheInvalid
	}
	return b.client.SetNX(ctx, key, owner, ttl).Result()
}

func (b *HostRedisCacheBackend) ReleaseLock(ctx context.Context, key, owner string) (bool, error) {
	if b == nil || b.client == nil || ctx == nil || validateHostCacheRevision(owner, false) != nil {
		return false, ErrHostCacheInvalid
	}
	value, err := hostCacheRedisReleaseLockScript.Run(ctx, b.client, []string{key}, owner).Int64()
	return value == 1, err
}

func (b *HostRedisCacheBackend) RenewLock(
	ctx context.Context,
	key string,
	owner string,
	ttl time.Duration,
) (bool, error) {
	if b == nil || b.client == nil || ctx == nil || validateHostCacheRevision(owner, false) != nil ||
		ttl < HostCacheMinimumLockTTL || ttl > HostCacheMaximumLockTTL {
		return false, ErrHostCacheInvalid
	}
	value, err := hostCacheRedisRenewLockScript.Run(ctx, b.client, []string{key}, owner, ttl.Milliseconds()).Int64()
	return value == 1, err
}

func (b *HostRedisCacheBackend) SetAndReleaseLock(
	ctx context.Context,
	key string,
	value HostCacheStoredValue,
	ttl time.Duration,
	expectedRevision string,
	tagPrefix string,
	lockKey string,
	owner string,
) error {
	if b == nil || b.client == nil || ctx == nil || validateHostCacheTTL(ttl) != nil ||
		validateHostCacheRevision(owner, false) != nil || !strings.HasSuffix(tagPrefix, ":tag:") ||
		len(value.Tags) > HostCacheMaximumTags {
		return ErrHostCacheInvalid
	}
	for _, tag := range value.Tags {
		if !strings.HasPrefix(tag, tagPrefix) {
			return ErrHostCacheInvalid
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded) > hostCacheRedisMaximumEnvelopeBytes {
		return ErrHostCacheInvalid
	}
	keys := make([]string, 2, len(value.Tags)+2)
	keys[0], keys[1] = key, lockKey
	keys = append(keys, value.Tags...)
	err = hostCacheRedisSetAndReleaseLockScript.Run(ctx, b.client, keys,
		encoded, ttl.Milliseconds(), expectedRevision, tagPrefix, hostCacheRedisMaximumTagMembers, owner,
	).Err()
	switch {
	case hostCacheRedisError(err, "HOSTCACHE_LOCK_NOT_OWNED"):
		return ErrHostCacheLockNotOwned
	case hostCacheRedisError(err, "HOSTCACHE_CONFLICT"):
		return ErrHostCacheConflict
	default:
		return err
	}
}

func hostCacheRedisError(err error, code string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(err.Error()), strings.ToUpper(code))
}

var _ HostCacheBackend = (*HostRedisCacheBackend)(nil)
