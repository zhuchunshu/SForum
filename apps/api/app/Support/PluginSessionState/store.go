package pluginsessionstate

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Store 是有界、按扩展与会话隔离的插件状态存储。
//
// 键结构（S1，全部变量经哈希后拼接，无原始标识符、无分隔符歧义）：
//
//	sforum:plugin-session:v1:installation:<inst>:extension:<extHash>:session:<sidHash>:key:<keyHash>
//	sforum:plugin-session:v1:installation:<inst>:session:<sidHash>:keys        SET 成员 "<extHash>:<keyHash>"
//	sforum:plugin-session:v1:installation:<inst>:session:<sidHash>:extensions  SET 成员 "<extHash>"
//	sforum:plugin-session:v1:installation:<inst>:extension:<extHash>:sessions  SET 成员 "<sidHash>"
//
// 为什么需要三个索引：Redis 没有按前缀删除，而「被撤销的会话其键仍然存在」（Core 的
// 撤销是每请求 PG 检查，不是 Redis 删除），所以失效必须能枚举出「某会话写过什么」与
// 「某扩展被哪些会话写过」。没有第三张索引，就没有办法在会话失效后把第二张索引里的
// 成员摘掉，第二张索引会无限长。
//
// 计数语义（诚实说明）：上限**不是**用集合基数（SCARD）判定，而是**扫描成员并逐个确认
// 值键是否还在**（`countLiveSessionKeys` / `countLiveSessions`），顺手把已经不存在的成员
// 摘掉。为什么不用 SCARD：索引成员可能比值键活得久（值键有自己的、更短的 TTL），
// SCARD 会把陈旧成员算进去并错误地拒绝写入；扫描 + 修剪能同时得到准确计数与自我修复。
// 代价是最坏 O(会话数 × 每会话成员数) 的有界遍历，上界由 S6 的限额保证。
type Store struct {
	backend Backend
	now     func() time.Time
}

// NewStore 组装存储。backend 为 nil 时直接失败：一个「没有后端」的会话状态存储
// 只会在运行期变成静默丢数据，宁可装配期就报错。
func NewStore(backend Backend) (*Store, error) {
	if backend == nil {
		return nil, ErrInvalid
	}
	return &Store{backend: backend, now: time.Now}, nil
}

// WithClock 注入时钟用于测试，仅影响 TTL 计算与过期判定。
func (s *Store) WithClock(now func() time.Time) *Store {
	if now != nil {
		s.now = now
	}
	return s
}

func valueKey(installationID, extensionHash, sessionFingerprint, namespaceKey string) string {
	return NamespacePrefix + "installation:" + installationID +
		":extension:" + extensionHash +
		":session:" + sessionFingerprint +
		":key:" + Fingerprint(namespaceKey)
}

func sessionKeysIndex(installationID, sessionFingerprint string) string {
	return NamespacePrefix + "installation:" + installationID + ":session:" + sessionFingerprint + ":keys"
}

func sessionExtensionsIndex(installationID, sessionFingerprint string) string {
	return NamespacePrefix + "installation:" + installationID + ":session:" + sessionFingerprint + ":extensions"
}

func extensionSessionsIndex(installationID, extensionHash string) string {
	return NamespacePrefix + "installation:" + installationID + ":extension:" + extensionHash + ":sessions"
}

// indexBound 是索引键的 TTL：取该会话当前允许的最大寿命，写时刷新。
// 索引宁可比值键多活一会儿（失效路径容忍缺失键），也不能比值键先消失——
// 索引先消失会让失效无法枚举，残留值只能等自己的 TTL 到期。
func (s *Store) indexBound(scope Scope, now time.Time) time.Duration {
	bound := MaxTTL
	if !scope.SessionIDleExpiresAt.IsZero() {
		if remaining := scope.SessionIDleExpiresAt.Sub(now); remaining > 0 && remaining < bound {
			bound = remaining
		}
	}
	if !scope.SessionAbsoluteExpiresAt.IsZero() {
		if remaining := scope.SessionAbsoluteExpiresAt.Sub(now); remaining > 0 && remaining < bound {
			bound = remaining
		}
	}
	if bound < MinTTL {
		bound = MinTTL
	}
	return bound
}

func (s *Store) backendError(err error) error {
	if err == nil {
		return nil
	}
	// 后端错误一律折叠为「不可用」，调用方必须 fail closed；
	// 不把底层错误文本透给插件，避免把部署细节当能力探测面。
	return ErrBackendUnavailable
}

// Get 读取一个命名空间键。会话已过期时按未命中处理，并且顺手清掉该键：
// 读路径必须自己复核寿命，不能只依赖 Redis TTL。
func (s *Store) Get(ctx context.Context, scope Scope, namespaceKey string) ([]byte, bool, error) {
	if s == nil || ctx == nil || scope.Validate() != nil {
		return nil, false, ErrInvalid
	}
	// 键错误必须具体化：Set 会返回 ErrKeyTooLong，读/删路径把它折叠成 ErrInvalid
	// 会让「同一把键、不同操作得到不同错误码」成为调用方无法处理的意外。
	if err := ValidateNamespaceKey(namespaceKey); err != nil {
		return nil, false, err
	}
	now := s.now()
	key := valueKey(scope.InstallationID, Fingerprint(scope.ExtensionID), scope.SessionFingerprint, namespaceKey)
	value, found, err := s.backend.Get(ctx, key)
	if err != nil {
		return nil, false, s.backendError(err)
	}
	if !found {
		return nil, false, nil
	}
	if scope.Expired(now) {
		if _, delErr := s.backend.Delete(ctx, key); delErr != nil {
			return nil, false, s.backendError(delErr)
		}
		return nil, false, nil
	}
	return value, true, nil
}

// Set 写入一个命名空间键并返回生效 TTL。新键才受上限约束：更新已有键永远允许，
// 否则一个被削过上限的插件会连自己的旧数据都改不了。
func (s *Store) Set(ctx context.Context, scope Scope, namespaceKey string, value []byte, requestedTTL time.Duration) (time.Duration, error) {
	if s == nil || ctx == nil || scope.Validate() != nil {
		return 0, ErrInvalid
	}
	if err := ValidateNamespaceKey(namespaceKey); err != nil {
		return 0, err
	}
	if err := ValidateValue(value); err != nil {
		return 0, err
	}
	now := s.now()
	effective, err := scope.EffectiveTTL(now, requestedTTL)
	if err != nil {
		return 0, err
	}
	extensionHash := Fingerprint(scope.ExtensionID)
	key := valueKey(scope.InstallationID, extensionHash, scope.SessionFingerprint, namespaceKey)
	if _, exists, err := s.backend.Get(ctx, key); err != nil {
		return 0, s.backendError(err)
	} else if !exists {
		if err := s.enforceCaps(ctx, scope, extensionHash, namespaceKey); err != nil {
			return 0, err
		}
	}
	if err := s.backend.Set(ctx, key, value, effective); err != nil {
		return 0, s.backendError(err)
	}
	indexTTL := s.indexBound(scope, now)
	member := extensionHash + ":" + Fingerprint(namespaceKey)
	if _, err := s.backend.SetAdd(ctx, sessionKeysIndex(scope.InstallationID, scope.SessionFingerprint), member, indexTTL); err != nil {
		return 0, s.backendError(err)
	}
	if _, err := s.backend.SetAdd(ctx, sessionExtensionsIndex(scope.InstallationID, scope.SessionFingerprint), extensionHash, indexTTL); err != nil {
		return 0, s.backendError(err)
	}
	if _, err := s.backend.SetAdd(ctx, extensionSessionsIndex(scope.InstallationID, extensionHash), scope.SessionFingerprint, indexTTL); err != nil {
		return 0, s.backendError(err)
	}
	return effective, nil
}

// enforceCaps 在写入新键之前检查两个上限，并先有界修剪陈旧成员。
func (s *Store) enforceCaps(ctx context.Context, scope Scope, extensionHash, namespaceKey string) error {
	sessionKeys := sessionKeysIndex(scope.InstallationID, scope.SessionFingerprint)
	extensionSessions := extensionSessionsIndex(scope.InstallationID, extensionHash)

	sessionCard, err := s.countLiveSessionKeys(ctx, scope, sessionKeys)
	if err != nil {
		return err
	}
	if sessionCard >= MaxKeysPerSession {
		return ErrKeyLimit
	}

	liveSessions, err := s.countLiveSessions(ctx, scope, extensionHash, extensionSessions)
	if err != nil {
		return err
	}
	if liveSessions >= MaxKeysPerExtension {
		return ErrExtensionLimit
	}
	return nil
}

// countLiveSessionKeys 返回该 (安装, 会话) 下仍然有值的键数，顺带摘掉已过期的成员。
func (s *Store) countLiveSessionKeys(ctx context.Context, scope Scope, sessionKeys string) (int64, error) {
	members, err := s.backend.SetMembers(ctx, sessionKeys)
	if err != nil {
		return 0, s.backendError(err)
	}
	live := int64(0)
	for _, member := range members {
		extensionHash, keyHash, ok := strings.Cut(member, ":")
		if !ok {
			// 结构不符的成员只可能来自更早的版本；摘掉而不是让它永久占额度。
			if err := s.backend.SetRemove(ctx, sessionKeys, member); err != nil {
				return 0, s.backendError(err)
			}
			continue
		}
		key := NamespacePrefix + "installation:" + scope.InstallationID +
			":extension:" + extensionHash + ":session:" + scope.SessionFingerprint + ":key:" + keyHash
		if _, found, err := s.backend.Get(ctx, key); err != nil {
			return 0, s.backendError(err)
		} else if found {
			live++
		} else if err := s.backend.SetRemove(ctx, sessionKeys, member); err != nil {
			return 0, s.backendError(err)
		}
	}
	return live, nil
}

// countLiveSessions 返回该扩展安装下仍然有值的会话键总数，顺带摘掉陈旧的会话成员。
// 这是最坏 O(会话数) 的有界遍历：每个会话的成员数本身不超过 MaxKeysPerSession。
func (s *Store) countLiveSessions(ctx context.Context, scope Scope, extensionHash, extensionSessions string) (int64, error) {
	fingerprints, err := s.backend.SetMembers(ctx, extensionSessions)
	if err != nil {
		return 0, s.backendError(err)
	}
	live := int64(0)
	for _, fingerprint := range fingerprints {
		if !validFingerprint(fingerprint) {
			if err := s.backend.SetRemove(ctx, extensionSessions, fingerprint); err != nil {
				return 0, s.backendError(err)
			}
			continue
		}
		sessionScope := scope
		sessionScope.SessionFingerprint = fingerprint
		count, err := s.countLiveSessionKeys(ctx, sessionScope, sessionKeysIndex(scope.InstallationID, fingerprint))
		if err != nil {
			return 0, err
		}
		if count == 0 {
			if err := s.backend.SetRemove(ctx, extensionSessions, fingerprint); err != nil {
				return 0, s.backendError(err)
			}
			continue
		}
		live += count
	}
	return live, nil
}

// Delete 删除一个命名空间键，并同步摘掉索引成员，使上限计数立刻反映真实状态。
func (s *Store) Delete(ctx context.Context, scope Scope, namespaceKey string) (bool, error) {
	if s == nil || ctx == nil || scope.Validate() != nil {
		return false, ErrInvalid
	}
	if err := ValidateNamespaceKey(namespaceKey); err != nil {
		return false, err
	}
	extensionHash := Fingerprint(scope.ExtensionID)
	keyHash := Fingerprint(namespaceKey)
	key := valueKey(scope.InstallationID, extensionHash, scope.SessionFingerprint, namespaceKey)
	deleted, err := s.backend.Delete(ctx, key)
	if err != nil {
		return false, s.backendError(err)
	}
	sessionKeys := sessionKeysIndex(scope.InstallationID, scope.SessionFingerprint)
	if err := s.backend.SetRemove(ctx, sessionKeys, extensionHash+":"+keyHash); err != nil {
		return false, s.backendError(err)
	}
	return deleted, nil
}

// InvalidateSession 清空某个会话在**所有**扩展下的状态（登出、撤销、设备数驱逐、
// 登录替换掉旧 sid、绝对过期都走这里），并维护扩展侧索引。
// 幂等：没有状态时返回 0 且不报错。
func (s *Store) InvalidateSession(ctx context.Context, installationID, sessionFingerprint string) (int, error) {
	if s == nil || ctx == nil || !validIdentityPart(installationID) || !validFingerprint(sessionFingerprint) {
		return 0, ErrInvalid
	}
	sessionKeys := sessionKeysIndex(installationID, sessionFingerprint)
	members, err := s.backend.SetMembers(ctx, sessionKeys)
	if err != nil {
		return 0, s.backendError(err)
	}
	removed := 0
	for _, member := range members {
		extensionHash, keyHash, ok := strings.Cut(member, ":")
		if !ok {
			continue
		}
		key := NamespacePrefix + "installation:" + installationID +
			":extension:" + extensionHash + ":session:" + sessionFingerprint + ":key:" + keyHash
		if deleted, err := s.backend.Delete(ctx, key); err != nil {
			return removed, s.backendError(err)
		} else if deleted {
			removed++
		}
		if err := s.backend.SetRemove(ctx, extensionSessionsIndex(installationID, extensionHash), sessionFingerprint); err != nil {
			return removed, s.backendError(err)
		}
	}
	// 这里刻意没有 SetRemove(sessionKeys, members...)：紧接着就整体删除该索引键，
	// 逐成员删除是多余的一次往返；而当 members 为空时，Redis 的 SREM 还会因为
	// arity（key + ≥1 成员）报错，把一次**幂等**调用变成 ErrBackendUnavailable。
	// 幂等契约必须由本层保证，不能依赖适配层替我们短路。
	sessionExtensions := sessionExtensionsIndex(installationID, sessionFingerprint)
	if _, err := s.backend.Delete(ctx, sessionKeys); err != nil {
		return removed, s.backendError(err)
	}
	if _, err := s.backend.Delete(ctx, sessionExtensions); err != nil {
		return removed, s.backendError(err)
	}
	return removed, nil
}

// InvalidateExtension 清空某个扩展在**所有**会话下的状态（禁用、卸载走这里）。
// 幂等：没有状态时返回 0 且不报错。
func (s *Store) InvalidateExtension(ctx context.Context, installationID, extensionID string) (int, error) {
	if s == nil || ctx == nil || !validIdentityPart(installationID) || !validIdentityPart(extensionID) {
		return 0, ErrInvalid
	}
	extensionHash := Fingerprint(extensionID)
	extensionSessions := extensionSessionsIndex(installationID, extensionHash)
	fingerprints, err := s.backend.SetMembers(ctx, extensionSessions)
	if err != nil {
		return 0, s.backendError(err)
	}
	removed := 0
	for _, fingerprint := range fingerprints {
		sessionKeys := sessionKeysIndex(installationID, fingerprint)
		members, err := s.backend.SetMembers(ctx, sessionKeys)
		if err != nil {
			return removed, s.backendError(err)
		}
		for _, member := range members {
			memberExtension, keyHash, ok := strings.Cut(member, ":")
			if !ok || memberExtension != extensionHash {
				continue
			}
			key := NamespacePrefix + "installation:" + installationID +
				":extension:" + extensionHash + ":session:" + fingerprint + ":key:" + keyHash
			if deleted, err := s.backend.Delete(ctx, key); err != nil {
				return removed, s.backendError(err)
			} else if deleted {
				removed++
			}
			if err := s.backend.SetRemove(ctx, sessionKeys, member); err != nil {
				return removed, s.backendError(err)
			}
		}
		if err := s.backend.SetRemove(ctx, sessionExtensionsIndex(installationID, fingerprint), extensionHash); err != nil {
			return removed, s.backendError(err)
		}
	}
	if _, err := s.backend.Delete(ctx, extensionSessions); err != nil {
		return removed, s.backendError(err)
	}
	return removed, nil
}

// IsBackendUnavailable 供调用方区分「存储不可用」（必须 fail closed，不得当作未命中）
// 与其它错误。把后端故障当成「没有状态」会让插件在建言上被误导。
func IsBackendUnavailable(err error) bool { return errors.Is(err, ErrBackendUnavailable) }
