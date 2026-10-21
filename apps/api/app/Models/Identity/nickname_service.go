package identity

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// AuthorNameSyncer 让昵称改名后回写派生索引（站内搜索文档里的作者昵称）。
// 它是可选依赖：搜索同步失败不应该让改名失败。
type AuthorNameSyncer interface {
	SyncAuthorDisplayName(ctx context.Context, userID int64, displayName string) error
}

// NicknameService 编排昵称读取、改名与预检。它独立于 Service，避免把
// 账号大对象再撑大一圈；依赖都是窄接口，测试可以只造假存储。
type NicknameService struct {
	store     NicknameStore
	policy    NicknamePolicyResolver
	usernames UsernamePolicyResolver
	sync      AuthorNameSyncer
	now       func() time.Time
}

func NewNicknameService(store NicknameStore, policy NicknamePolicyResolver, usernames UsernamePolicyResolver) *NicknameService {
	return &NicknameService{store: store, policy: policy, usernames: usernames, now: time.Now}
}

func (s *NicknameService) WithAuthorNameSyncer(sync AuthorNameSyncer) *NicknameService {
	if s != nil {
		s.sync = sync
	}
	return s
}

// WithClock 只用于测试：冷却边界必须能确定性地断言。
func (s *NicknameService) WithClock(now func() time.Time) *NicknameService {
	if s != nil && now != nil {
		s.now = now
	}
	return s
}

func (s *NicknameService) clock() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

// State 返回当前用户的昵称状态（含冷却与重名提示）。
func (s *NicknameService) State(ctx context.Context, userID int64) (NicknameState, error) {
	if s == nil || s.store == nil {
		return NicknameState{}, ErrNicknameDisabled
	}
	policy, err := s.policy.NicknamePolicy(ctx)
	if err != nil {
		return NicknameState{}, err
	}
	snapshot, err := s.store.NicknameSnapshot(ctx, userID)
	if err != nil {
		return NicknameState{}, err
	}
	return s.decorate(ctx, snapshot, policy)
}

// Change 执行一次自助改名。校验顺序固定为：策略开关 → 格式/长度/保留词 →
// 冷却 → 用户名撞名 → 写入。冷却放在撞名之前，是因为用户更关心「我什么时候
// 能改」而不是「这个名字能不能用」。
func (s *NicknameService) Change(ctx context.Context, actor Actor, displayName string) (NicknameState, error) {
	if s == nil || s.store == nil {
		return NicknameState{}, ErrNicknameDisabled
	}
	if actor.ID <= 0 || !actor.IsActive() {
		return NicknameState{}, ErrPermissionDenied
	}
	policy, err := s.policy.NicknamePolicy(ctx)
	if err != nil {
		return NicknameState{}, err
	}
	if !policy.AllowUserChange {
		return NicknameState{}, ErrNicknameDisabled
	}
	usernamePolicy, err := s.usernames.UsernamePolicy(ctx)
	if err != nil {
		return NicknameState{}, err
	}
	snapshot, err := s.store.NicknameSnapshot(ctx, actor.ID)
	if err != nil {
		return NicknameState{}, err
	}

	value, fields := validateNicknameInput(displayName, snapshot.Username, policy, usernamePolicy.Reserved)
	if len(fields) > 0 {
		return NicknameState{}, &NicknameInvalidError{Fields: fields}
	}
	if value == snapshot.DisplayName {
		return s.decorate(ctx, snapshot, policy)
	}
	if retryAt := nextNicknameChangeAt(snapshot.ChangedAt, policy.CooldownMinutes); retryAt != nil && s.clock().Before(*retryAt) {
		return NicknameState{}, &NicknameCooldownError{RetryAt: *retryAt}
	}
	taken, err := s.store.UsernameOwnedByOther(ctx, actor.ID, value)
	if err != nil {
		return NicknameState{}, err
	}
	if taken {
		return NicknameState{}, &NicknameInvalidError{
			Fields: FieldMessages{FieldDisplayName: {MessageNicknameUsernameTaken}},
		}
	}

	updated, err := s.store.ChangeSelfNickname(ctx, ChangeNicknameInput{
		UserID:          actor.ID,
		ActorID:         actor.ID,
		DisplayName:     value,
		CooldownMinutes: policy.CooldownMinutes,
	})
	if err != nil {
		return NicknameState{}, err
	}

	if s.sync != nil && updated.DisplayName != snapshot.DisplayName {
		if err := s.sync.SyncAuthorDisplayName(ctx, actor.ID, updated.DisplayName); err != nil {
			// 派生索引可以重建，改名本身已经提交；这里只留线索。
			slog.Warn("identity: sync author display name failed",
				"userId", actor.ID, "displayName", updated.DisplayName, "error", err)
		}
	}
	return s.decorate(ctx, updated, policy)
}

// Availability 是改名前的预检：重名只计数（昵称非唯一），撞用户名才算冲突。
func (s *NicknameService) Availability(ctx context.Context, actor Actor, raw string) (NicknameAvailability, error) {
	if s == nil || s.store == nil {
		return NicknameAvailability{}, ErrNicknameDisabled
	}
	if actor.ID <= 0 || !actor.IsActive() {
		return NicknameAvailability{}, ErrPermissionDenied
	}
	policy, err := s.policy.NicknamePolicy(ctx)
	if err != nil {
		return NicknameAvailability{}, err
	}
	usernamePolicy, err := s.usernames.UsernamePolicy(ctx)
	if err != nil {
		return NicknameAvailability{}, err
	}
	snapshot, err := s.store.NicknameSnapshot(ctx, actor.ID)
	if err != nil {
		return NicknameAvailability{}, err
	}
	value, fields := validateNicknameInput(raw, snapshot.Username, policy, usernamePolicy.Reserved)
	if len(fields) > 0 {
		return NicknameAvailability{Available: false, Conflict: firstFieldMessage(fields)}, nil
	}
	duplicates, err := s.store.NicknameDuplicates(ctx, actor.ID, value)
	if err != nil {
		return NicknameAvailability{}, err
	}
	if value != snapshot.DisplayName {
		taken, err := s.store.UsernameOwnedByOther(ctx, actor.ID, value)
		if err != nil {
			return NicknameAvailability{}, err
		}
		if taken {
			return NicknameAvailability{
				Available:  false,
				Duplicates: duplicates,
				Conflict:   MessageNicknameUsernameTaken,
			}, nil
		}
	}
	return NicknameAvailability{Available: true, Duplicates: duplicates, Normalized: value}, nil
}

func (s *NicknameService) decorate(ctx context.Context, snapshot NicknameSnapshot, policy NicknamePolicy) (NicknameState, error) {
	duplicates, err := s.store.NicknameDuplicates(ctx, snapshot.UserID, snapshot.DisplayName)
	if err != nil {
		return NicknameState{}, err
	}
	state := NicknameState{
		DisplayName:  snapshot.DisplayName,
		ChangedAt:    snapshot.ChangedAt,
		Duplicates:   duplicates,
		NextChangeAt: nextNicknameChangeAt(snapshot.ChangedAt, policy.CooldownMinutes),
	}
	state.CanChange = policy.AllowUserChange &&
		(state.NextChangeAt == nil || !s.clock().Before(*state.NextChangeAt))
	return state, nil
}

func firstFieldMessage(fields FieldMessages) string {
	for _, messages := range fields {
		if len(messages) > 0 {
			return messages[0]
		}
	}
	return ""
}

// PublicPreviousNames 返回公开资料页允许展示的曾用名：只有运营打开
// profile.nickname.history_public 才有内容，且最多一条。改名轨迹本身是
// 隐私负担，默认只给后台看。
func (s *NicknameService) PublicPreviousNames(ctx context.Context, userID int64) ([]string, error) {
	if s == nil || s.store == nil || userID <= 0 {
		return nil, nil
	}
	policy, err := s.policy.NicknamePolicy(ctx)
	if err != nil {
		return nil, err
	}
	if !policy.HistoryPublic {
		return nil, nil
	}
	history, err := s.store.ListNicknameHistory(ctx, userID, 1)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(history))
	for _, item := range history {
		if name := strings.TrimSpace(item.OldName); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}
