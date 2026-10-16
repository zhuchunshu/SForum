package pluginsessionstate

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisSetAddScript 把「加入索引成员 + 刷新索引 TTL + 报告是否新成员」压成一次往返。
//
// TTL 直接采用调用方传入值，不做 host-cache 标签那种「只在现有 TTL 更短时才延长」的试探：
// 那里的标签集合允许比条目活得更久，而这里的索引 TTL 是 store 按会话剩余寿命算出来的
// 权威值（见 store.indexBound），再叠一层试探只会让索引寿命依赖写入顺序。
var redisSetAddScript = redis.NewScript(`
local added = redis.call('SADD', KEYS[1], ARGV[1])
redis.call('PEXPIRE', KEYS[1], ARGV[2])
return added
`)

// RedisBackend 是本包 Backend 的 go-redis 实现，沿用 Support/HostAPI 的既有形状：
// 客户端生命周期由调用方持有（共享连接池），这里只做键空间操作与边界守卫。
type RedisBackend struct {
	client *redis.Client
}

// NewRedisBackend 组装后端。nil client 是装配错误，必须在装配期暴露：一个没有连接的
// 后端只会在运行期表现为「每次读都未命中」，而插件会把未命中当成「没有状态」。
func NewRedisBackend(client *redis.Client) (*RedisBackend, error) {
	if client == nil {
		return nil, ErrInvalid
	}
	return &RedisBackend{client: client}, nil
}

// Get 读一个键。缺失用 (nil, false, nil) 表达，不用 redis.Nil 这种底层哨兵：
// 调用方（Store）需要区分「没有值」与「存储坏了」，而 store.go 只把 error 折叠成
// 不可用。这里返回的 []byte 由 go-redis 新分配，不会被后续命令复用。
func (b *RedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if !b.usable(ctx, key) {
		return nil, false, ErrInvalid
	}
	value, err := b.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

// Set 覆写一个键并应用 TTL。TTL 必须落在 [MinTTL, MaxTTL]：PEXPIRE 收到 0 或负值会
// 直接删除键，静默变成「写入即消失」，所以这里宁可返回 ErrTTLOutOfRange。
func (b *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if !b.usable(ctx, key) {
		return ErrInvalid
	}
	if !validBackendTTL(ttl) {
		return ErrTTLOutOfRange
	}
	if err := ValidateValue(value); err != nil {
		return err
	}
	return b.client.Set(ctx, key, value, ttl).Err()
}

// Delete 删除一个键（值键或索引键都可能走到这里）并报告它此前是否存在。
func (b *RedisBackend) Delete(ctx context.Context, key string) (bool, error) {
	if !b.usable(ctx, key) {
		return false, ErrInvalid
	}
	deleted, err := b.client.Del(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return deleted > 0, nil
}

// SetAdd 加入索引成员，把索引 TTL 设为传入值，并报告该成员是否为本次新增。
// 返回值被 store 忽略，但保持与 Redis 语义一致，避免后来者误以为它是恒真。
func (b *RedisBackend) SetAdd(ctx context.Context, key string, member string, ttl time.Duration) (bool, error) {
	if !b.usable(ctx, key) || member == "" {
		return false, ErrInvalid
	}
	if !validBackendTTL(ttl) {
		return false, ErrTTLOutOfRange
	}
	added, err := redisSetAddScript.Run(ctx, b.client, []string{key}, member, ttl.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}
	return added == 1, nil
}

// SetMembers 返回索引成员。注意这里不能顺手过滤「指向的值键已经过期」的成员：
// 成员级过期不是 Redis 语义（只有整个 SET 键才有 TTL），store 正是靠看到这些陈旧成员
// 才做有界修剪与计数重建（store.countLiveSessionKeys）。
func (b *RedisBackend) SetMembers(ctx context.Context, key string) ([]string, error) {
	if !b.usable(ctx, key) {
		return nil, ErrInvalid
	}
	return b.client.SMembers(ctx, key).Result()
}

// SetRemove 摘掉成员。空成员列表直接短路：Redis 7 的 SREM 至少需要 key + 1 个成员，
// 否则返回 arity 错误；而 store.InvalidateSession 在没有状态时的调用序列里会出现空列表
// （成员枚举结果为空），短路让「幂等返回 0」在真实 Redis 上成立，而不是变成后端不可用。
func (b *RedisBackend) SetRemove(ctx context.Context, key string, members ...string) error {
	if !b.usable(ctx, key) {
		return ErrInvalid
	}
	if len(members) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(members))
	for _, member := range members {
		args = append(args, member)
	}
	return b.client.SRem(ctx, key, args...).Err()
}

// SetCard 返回索引基数（含陈旧成员，与 SCARD 一致）。store 目前不用它做判据——
// 见 store_test.go 的 TestStoreCountsWithoutSetCard。
func (b *RedisBackend) SetCard(ctx context.Context, key string) (int64, error) {
	if !b.usable(ctx, key) {
		return 0, ErrInvalid
	}
	return b.client.SCard(ctx, key).Result()
}

func (b *RedisBackend) usable(ctx context.Context, key string) bool {
	return b != nil && b.client != nil && ctx != nil && key != ""
}

// validBackendTTL 是写入前的最后一道 TTL 守卫：越界值在这里被拒绝，而不是被 Redis
// 解释成「立即删除」或「永不过期」。
func validBackendTTL(ttl time.Duration) bool {
	return ttl >= MinTTL && ttl <= MaxTTL
}

var _ Backend = (*RedisBackend)(nil)
