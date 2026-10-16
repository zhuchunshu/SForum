package pluginsessionstate

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Backend 的方法名常量，用于按方法注入故障与统计调用次数。
const (
	fakeMethodGet        = "Get"
	fakeMethodSet        = "Set"
	fakeMethodDelete     = "Delete"
	fakeMethodSetAdd     = "SetAdd"
	fakeMethodSetMembers = "SetMembers"
	fakeMethodSetRemove  = "SetRemove"
	fakeMethodSetCard    = "SetCard"
)

// fakeBackendMethods 是 Backend 的全部方法，测试据此遍历「每个方法注入故障」。
var fakeBackendMethods = []string{
	fakeMethodGet,
	fakeMethodSet,
	fakeMethodDelete,
	fakeMethodSetAdd,
	fakeMethodSetMembers,
	fakeMethodSetRemove,
	fakeMethodSetCard,
}

// fakeBackendClock 是 Store 与 fake 后端共用的可注入时钟。
//
// 两者必须落在同一条时间线上：Store 用它算生效 TTL，fake 用它判断键是否过期。只有这样
// 「推进时间 → 值键过期 → 索引里的成员变成陈旧成员」这一条 store.go 关键路径才能被断言。
type fakeBackendClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeBackendClock(now time.Time) *fakeBackendClock {
	return &fakeBackendClock{now: now}
}

func (c *fakeBackendClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeBackendClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type fakeBackendValue struct {
	data      []byte
	expiresAt time.Time // 零值表示不过期
}

type fakeBackendSet struct {
	members   map[string]struct{}
	expiresAt time.Time
}

// fakeBackend 是 Backend 的内存替身。它刻意对齐 Redis 的两条语义，因为 store.go 的分支
// 正是围绕它们写的：
//
//  1. SET 键只有整体 TTL，成员不会单独过期。SetMembers/SetCard 必须原样返回陈旧成员，
//     否则「有界修剪 + 重新计数」这条路径就被替身替掉了。
//  2. 变参 SREM 至少需要 1 个成员（Redis arity），RedisBackend 因此对空列表短路；
//     替身同样短路，使「幂等」在两种实现下含义一致。
type fakeBackend struct {
	mu    sync.Mutex
	clock func() time.Time

	values map[string]fakeBackendValue
	sets   map[string]fakeBackendSet

	failureAll    error
	failureMethod map[string]error
	calls         map[string]int

	// syntheticMembers 让 SetMembers 在指定索引键上多报成员；syntheticKeyPrefixes 让 Get
	// 认为这些前缀下的值键存在。用于把扩展级计数推过 MaxKeysPerExtension 而不真写 4096
	// 个键（见 store_test.go 的 TestExtensionKeyLimitSyntheticPopulation）。
	syntheticMembers     map[string][]string
	syntheticKeyPrefixes []string

	// 记录调用方传入的 TTL，用于断言「生效 TTL 被真正传给后端」。
	valueTTL  map[string]time.Duration
	setAddTTL map[string]time.Duration
}

func newFakeBackend(clock func() time.Time) *fakeBackend {
	if clock == nil {
		clock = time.Now
	}
	return &fakeBackend{
		clock:            clock,
		values:           map[string]fakeBackendValue{},
		sets:             map[string]fakeBackendSet{},
		failureMethod:    map[string]error{},
		calls:            map[string]int{},
		syntheticMembers: map[string][]string{},
		valueTTL:         map[string]time.Duration{},
		setAddTTL:        map[string]time.Duration{},
	}
}

func (b *fakeBackend) failAll(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failureAll = err
}

func (b *fakeBackend) failMethod(method string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failureMethod[method] = err
}

func (b *fakeBackend) clearFailures() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failureAll = nil
	b.failureMethod = map[string]error{}
}

func (b *fakeBackend) callCount(method string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls[method]
}

// addSyntheticLiveSession 注册一个「合成"存活"会话」：它出现在扩展会话索引里，其会话
// 键索引里多报 keyCount 个成员，且对应值键前缀下的任意键在 Get 时都被视为有值。
//
// 这是测试用的替身旁路，不改动 store.go 的任何判断：enforceCaps → countLiveSessions →
// liveSessions >= MaxKeysPerExtension 这条分支按原样执行，只是人口是伪造的。
func (b *fakeBackend) addSyntheticLiveSession(
	extensionSessionsKey string,
	sessionFingerprint string,
	sessionKeysKey string,
	memberPrefix string,
	keyCount int,
	valueKeyPrefix string,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.syntheticMembers[extensionSessionsKey] = append(b.syntheticMembers[extensionSessionsKey], sessionFingerprint)
	for i := 0; i < keyCount; i++ {
		b.syntheticMembers[sessionKeysKey] = append(
			b.syntheticMembers[sessionKeysKey],
			fmt.Sprintf("%s%064x", memberPrefix, i),
		)
	}
	b.syntheticKeyPrefixes = append(b.syntheticKeyPrefixes, valueKeyPrefix)
}

func (b *fakeBackend) before(ctx context.Context, method string) error {
	if ctx == nil {
		return ErrInvalid
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls[method]++
	if err := b.failureMethod[method]; err != nil {
		return err
	}
	return b.failureAll
}

// expiredLocked 判断一个到期时间是否已经过去（零值 = 不过期）。调用方必须持锁，
// 以便与 fake 时钟读值处于同一个临界区。
func (b *fakeBackend) expiredLocked(expiresAt time.Time) bool {
	return !expiresAt.IsZero() && !b.nowLocked().Before(expiresAt)
}

func (b *fakeBackend) nowLocked() time.Time {
	return b.clock()
}

func (b *fakeBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := b.before(ctx, fakeMethodGet); err != nil {
		return nil, false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if entry, found := b.values[key]; found {
		if b.expiredLocked(entry.expiresAt) {
			// 惰性过期：真实 Redis 里到期键已经不存在，所以这里连计数都不算。
			delete(b.values, key)
			delete(b.valueTTL, key)
			return nil, false, nil
		}
		return append([]byte(nil), entry.data...), true, nil
	}
	for _, prefix := range b.syntheticKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return []byte("synthetic-value"), true, nil
		}
	}
	return nil, false, nil
}

func (b *fakeBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := b.before(ctx, fakeMethodSet); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.values[key] = fakeBackendValue{
		data:      append([]byte(nil), value...),
		expiresAt: b.nowLocked().Add(ttl),
	}
	b.valueTTL[key] = ttl
	return nil
}

// Delete 同时覆盖值键与索引键，与 Redis DEL 一致（store 用同一个方法删 SET 索引）。
func (b *fakeBackend) Delete(ctx context.Context, key string) (bool, error) {
	if err := b.before(ctx, fakeMethodDelete); err != nil {
		return false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if entry, found := b.values[key]; found {
		delete(b.values, key)
		delete(b.valueTTL, key)
		return !b.expiredLocked(entry.expiresAt), nil
	}
	if set, found := b.sets[key]; found {
		delete(b.sets, key)
		delete(b.setAddTTL, key)
		return !b.expiredLocked(set.expiresAt), nil
	}
	return false, nil
}

func (b *fakeBackend) SetAdd(ctx context.Context, key string, member string, ttl time.Duration) (bool, error) {
	if err := b.before(ctx, fakeMethodSetAdd); err != nil {
		return false, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.nowLocked()
	set, found := b.sets[key]
	if !found || b.expiredLocked(set.expiresAt) {
		// 键整体过期等于键不存在：旧成员随之消失，不能延续到新 TTL。
		set = fakeBackendSet{members: map[string]struct{}{}}
	}
	if set.members == nil {
		set.members = map[string]struct{}{}
	}
	if _, existed := set.members[member]; existed {
		set.expiresAt = now.Add(ttl)
		b.sets[key] = set
		b.setAddTTL[key] = ttl
		return false, nil
	}
	set.members[member] = struct{}{}
	set.expiresAt = now.Add(ttl)
	b.sets[key] = set
	b.setAddTTL[key] = ttl
	return true, nil
}

func (b *fakeBackend) SetMembers(ctx context.Context, key string) ([]string, error) {
	if err := b.before(ctx, fakeMethodSetMembers); err != nil {
		return nil, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	var members []string
	if set, found := b.sets[key]; found && !b.expiredLocked(set.expiresAt) {
		members = make([]string, 0, len(set.members))
		for member := range set.members {
			members = append(members, member)
		}
	}
	members = append(members, b.syntheticMembers[key]...)
	sort.Strings(members)
	return members, nil
}

func (b *fakeBackend) SetRemove(ctx context.Context, key string, members ...string) error {
	if err := b.before(ctx, fakeMethodSetRemove); err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	set, found := b.sets[key]
	if !found || b.expiredLocked(set.expiresAt) {
		return nil
	}
	removing := map[string]struct{}{}
	for _, member := range members {
		delete(set.members, member)
		removing[member] = struct{}{}
	}
	if len(set.members) == 0 {
		delete(b.sets, key)
		delete(b.setAddTTL, key)
	} else {
		b.sets[key] = set
	}
	if len(b.syntheticMembers[key]) > 0 {
		kept := make([]string, 0, len(b.syntheticMembers[key]))
		for _, member := range b.syntheticMembers[key] {
			if _, drop := removing[member]; !drop {
				kept = append(kept, member)
			}
		}
		b.syntheticMembers[key] = kept
	}
	return nil
}

func (b *fakeBackend) SetCard(ctx context.Context, key string) (int64, error) {
	if err := b.before(ctx, fakeMethodSetCard); err != nil {
		return 0, err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	card := int64(len(b.syntheticMembers[key]))
	if set, found := b.sets[key]; found && !b.expiredLocked(set.expiresAt) {
		card += int64(len(set.members))
	}
	return card, nil
}

// liveValue 读取 fake 持有的值（不经过故障注入），过期即视为不存在。
func (b *fakeBackend) liveValue(key string) ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, found := b.values[key]
	if !found || b.expiredLocked(entry.expiresAt) {
		return nil, false
	}
	return append([]byte(nil), entry.data...), true
}

// liveValueKeys 返回仍然存活的值键集合（排序后，便于比较键结构）。
func (b *fakeBackend) liveValueKeys() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	keys := make([]string, 0, len(b.values))
	for key, entry := range b.values {
		if !b.expiredLocked(entry.expiresAt) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// realSetMembers 返回索引里的真实成员（不含合成人口，不做过期过滤——与 Redis 一致）。
func (b *fakeBackend) realSetMembers(key string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	set, found := b.sets[key]
	if !found || b.expiredLocked(set.expiresAt) {
		return nil
	}
	members := make([]string, 0, len(set.members))
	for member := range set.members {
		members = append(members, member)
	}
	sort.Strings(members)
	return members
}

func (b *fakeBackend) hasSetKey(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	set, found := b.sets[key]
	return found && !b.expiredLocked(set.expiresAt)
}

func (b *fakeBackend) lastValueTTL(key string) (time.Duration, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ttl, found := b.valueTTL[key]
	return ttl, found
}

func (b *fakeBackend) lastSetAddTTL(key string) (time.Duration, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ttl, found := b.setAddTTL[key]
	return ttl, found
}

// failErrFakeBackend 是注入故障用的哨兵：文本里带「部署细节」式内容，用于断言 store
// 不会把它透给调用方。
var errFakeBackend = errors.New("fake backend boom: redis 10.0.0.7:6379 noauth detail")

var _ Backend = (*fakeBackend)(nil)
