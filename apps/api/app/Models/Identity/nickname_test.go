package identity

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeNicknamePolicy struct {
	policy NicknamePolicy
	err    error
}

func (f fakeNicknamePolicy) NicknamePolicy(context.Context) (NicknamePolicy, error) {
	return f.policy, f.err
}

type fakeUsernamePolicy struct {
	policy UsernamePolicy
}

func (f fakeUsernamePolicy) UsernamePolicy(context.Context) (UsernamePolicy, error) {
	return f.policy, nil
}

type fakeNicknameStore struct {
	snapshot   NicknameSnapshot
	duplicates int
	taken      bool
	history    []NicknameChange
	changes    []ChangeNicknameInput
	changeErr  error
	onChange   func(input ChangeNicknameInput) (NicknameSnapshot, error)
}

func (f *fakeNicknameStore) NicknameSnapshot(context.Context, int64) (NicknameSnapshot, error) {
	return f.snapshot, nil
}

func (f *fakeNicknameStore) ChangeSelfNickname(_ context.Context, input ChangeNicknameInput) (NicknameSnapshot, error) {
	f.changes = append(f.changes, input)
	if f.changeErr != nil {
		return NicknameSnapshot{}, f.changeErr
	}
	if f.onChange != nil {
		return f.onChange(input)
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	f.snapshot.DisplayName = input.DisplayName
	f.snapshot.ChangedAt = &now
	return f.snapshot, nil
}

func (f *fakeNicknameStore) NicknameDuplicates(context.Context, int64, string) (int, error) {
	return f.duplicates, nil
}

func (f *fakeNicknameStore) UsernameOwnedByOther(context.Context, int64, string) (bool, error) {
	return f.taken, nil
}

func (f *fakeNicknameStore) ListNicknameHistory(context.Context, int64, int) ([]NicknameChange, error) {
	return f.history, nil
}

type recordingAuthorNameSyncer struct {
	calls []string
	err   error
}

func (r *recordingAuthorNameSyncer) SyncAuthorDisplayName(_ context.Context, _ int64, displayName string) error {
	r.calls = append(r.calls, displayName)
	return r.err
}

func defaultNicknamePolicy() NicknamePolicy {
	return NicknamePolicy{
		AllowUserChange: true,
		CooldownMinutes: 10080,
		MinLength:       2,
		MaxLength:       24,
	}
}

func activeNicknameActor() Actor {
	return Actor{ID: 7, Status: UserStatusActive}
}

func TestNormalizeNicknameValue(t *testing.T) {
	cases := []struct {
		raw    string
		want   string
		wantOK bool
	}{
		{"  Alice  ", "Alice", true},
		{"Alice   Smith", "Alice Smith", true},
		{"Alice\u00a0Smith", "Alice Smith", true},
		{"Alice\u3000Smith", "Alice Smith", true},
		{"\u3000\u3000", "", true},
		{"Alice\nSmith", "", false},
		{"Alice\tSmith", "", false},
		{"Ali\u200bce", "", false}, // 零宽空格是 Cf
		{"🐱", "🐱", true},
	}
	for _, tc := range cases {
		got, ok := normalizeNicknameValue(tc.raw)
		if ok != tc.wantOK || got != tc.want {
			t.Fatalf("normalize(%q) = %q ok=%v, want %q ok=%v", tc.raw, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestValidateNicknameInput(t *testing.T) {
	policy := defaultNicknamePolicy()
	reserved := []string{"admin", "official"}

	if value, fields := validateNicknameInput("   ", "alice", policy, reserved); value != "alice" || len(fields) != 0 {
		t.Fatalf("blank nickname must fall back to username, got %q fields=%v", value, fields)
	}
	if _, fields := validateNicknameInput("a", "alice", policy, reserved); !fieldMessagesContain(fields, FieldDisplayName, MessageNicknameTooShort) {
		t.Fatalf("one-character nickname must be rejected: %v", fields)
	}
	if _, fields := validateNicknameInput(strings.Repeat("名", 25), "alice", policy, reserved); !fieldMessagesContain(fields, FieldDisplayName, MessageNicknameTooLong) {
		t.Fatalf("overlong nickname must be rejected: %v", fields)
	}
	if _, fields := validateNicknameInput("ADMIN", "alice", policy, reserved); !fieldMessagesContain(fields, FieldDisplayName, MessageNicknameReserved) {
		t.Fatalf("reserved nickname must be rejected case-insensitively: %v", fields)
	}
	if _, fields := validateNicknameInput("@alice", "alice", policy, reserved); !fieldMessagesContain(fields, FieldDisplayName, MessageNicknameAtPrefix) {
		t.Fatalf("leading @ must be rejected: %v", fields)
	}
	if _, fields := validateNicknameInput("Ali\nce", "alice", policy, reserved); !fieldMessagesContain(fields, FieldDisplayName, MessageNicknameInvalidChars) {
		t.Fatalf("control characters must be rejected: %v", fields)
	}
	if value, fields := validateNicknameInput("  爱丽丝  ", "alice", policy, reserved); value != "爱丽丝" || len(fields) != 0 {
		t.Fatalf("valid nickname = %q fields=%v", value, fields)
	}
}

func TestNextNicknameChangeAt(t *testing.T) {
	changedAt := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	if next := nextNicknameChangeAt(nil, 10080); next != nil {
		t.Fatalf("never-changed user must have no cooldown, got %v", next)
	}
	if next := nextNicknameChangeAt(&changedAt, 0); next != nil {
		t.Fatalf("zero cooldown must mean unlimited, got %v", next)
	}
	next := nextNicknameChangeAt(&changedAt, 90)
	if next == nil || !next.Equal(changedAt.Add(90*time.Minute)) {
		t.Fatalf("90-minute cooldown = %v", next)
	}
}

func TestNicknameServiceChangeRejectsWhenDisabled(t *testing.T) {
	store := &fakeNicknameStore{snapshot: NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice"}}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: NicknamePolicy{AllowUserChange: false}}, fakeUsernamePolicy{})
	if _, err := service.Change(context.Background(), activeNicknameActor(), "Alicia"); !errors.Is(err, ErrNicknameDisabled) {
		t.Fatalf("disabled nickname change error = %v", err)
	}
	if len(store.changes) != 0 {
		t.Fatal("disabled path must not write")
	}
}

func TestNicknameServiceChangeEnforcesCooldown(t *testing.T) {
	changedAt := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	store := &fakeNicknameStore{snapshot: NicknameSnapshot{
		UserID: 7, Username: "alice", DisplayName: "Alice", ChangedAt: &changedAt,
	}}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{}).
		WithClock(func() time.Time { return changedAt.Add(6 * 24 * time.Hour) })

	_, err := service.Change(context.Background(), activeNicknameActor(), "Alicia")
	var cooldown *NicknameCooldownError
	if !errors.As(err, &cooldown) {
		t.Fatalf("cooldown error = %v", err)
	}
	if want := changedAt.Add(7 * 24 * time.Hour); !cooldown.RetryAt.Equal(want) {
		t.Fatalf("retryAt = %v, want %v", cooldown.RetryAt, want)
	}
	if len(store.changes) != 0 {
		t.Fatal("cooldown path must not write")
	}
}

func TestNicknameServiceChangeRejectsUsernameCollision(t *testing.T) {
	store := &fakeNicknameStore{
		snapshot: NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice"},
		taken:    true,
	}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{})
	_, err := service.Change(context.Background(), activeNicknameActor(), "Bob")
	var invalid *NicknameInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("collision error = %v", err)
	}
	if !fieldMessagesContain(invalid.Fields, FieldDisplayName, MessageNicknameUsernameTaken) {
		t.Fatalf("collision fields = %v", invalid.Fields)
	}
}

func TestNicknameServiceChangeWritesAndSyncs(t *testing.T) {
	store := &fakeNicknameStore{snapshot: NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice"}}
	syncer := &recordingAuthorNameSyncer{}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{}).
		WithAuthorNameSyncer(syncer)

	state, err := service.Change(context.Background(), activeNicknameActor(), "  爱丽丝  ")
	if err != nil {
		t.Fatalf("change returned error: %v", err)
	}
	if state.DisplayName != "爱丽丝" {
		t.Fatalf("state display name = %q", state.DisplayName)
	}
	if len(store.changes) != 1 || store.changes[0].DisplayName != "爱丽丝" || store.changes[0].CooldownMinutes != 10080 {
		t.Fatalf("store change input = %+v", store.changes)
	}
	if len(syncer.calls) != 1 || syncer.calls[0] != "爱丽丝" {
		t.Fatalf("sync calls = %v", syncer.calls)
	}
}

// 派生索引同步失败不能把已经提交的改名变成一次失败请求。
func TestNicknameServiceChangeToleratesSyncFailure(t *testing.T) {
	store := &fakeNicknameStore{snapshot: NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice"}}
	syncer := &recordingAuthorNameSyncer{err: errors.New("index down")}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{}).
		WithAuthorNameSyncer(syncer)

	if _, err := service.Change(context.Background(), activeNicknameActor(), "Alicia"); err != nil {
		t.Fatalf("sync failure must not fail the change: %v", err)
	}
}

func TestNicknameServiceChangeNoOpDoesNotWrite(t *testing.T) {
	store := &fakeNicknameStore{snapshot: NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice"}}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{})
	if _, err := service.Change(context.Background(), activeNicknameActor(), "Alice"); err != nil {
		t.Fatalf("no-op change returned error: %v", err)
	}
	if len(store.changes) != 0 {
		t.Fatalf("no-op change must not write, got %+v", store.changes)
	}
}

func TestNicknameServiceAvailability(t *testing.T) {
	store := &fakeNicknameStore{
		snapshot:   NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice"},
		duplicates: 2,
	}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{})

	result, err := service.Availability(context.Background(), activeNicknameActor(), "Alice Smith")
	if err != nil {
		t.Fatalf("availability returned error: %v", err)
	}
	if !result.Available || result.Duplicates != 2 || result.Normalized != "Alice Smith" {
		t.Fatalf("availability = %+v", result)
	}

	invalid, err := service.Availability(context.Background(), activeNicknameActor(), "a")
	if err != nil {
		t.Fatalf("availability returned error: %v", err)
	}
	if invalid.Available || invalid.Conflict != MessageNicknameTooShort {
		t.Fatalf("short nickname availability = %+v", invalid)
	}

	store.taken = true
	collision, err := service.Availability(context.Background(), activeNicknameActor(), "Bob")
	if err != nil {
		t.Fatalf("availability returned error: %v", err)
	}
	if collision.Available || collision.Conflict != MessageNicknameUsernameTaken {
		t.Fatalf("collision availability = %+v", collision)
	}
}

func TestNicknameServiceStateComputesCooldown(t *testing.T) {
	changedAt := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	store := &fakeNicknameStore{
		snapshot:   NicknameSnapshot{UserID: 7, Username: "alice", DisplayName: "Alice", ChangedAt: &changedAt},
		duplicates: 1,
	}
	service := NewNicknameService(store, fakeNicknamePolicy{policy: defaultNicknamePolicy()}, fakeUsernamePolicy{}).
		WithClock(func() time.Time { return changedAt.Add(6 * 24 * time.Hour) })

	state, err := service.State(context.Background(), 7)
	if err != nil {
		t.Fatalf("state returned error: %v", err)
	}
	if state.CanChange {
		t.Fatal("user inside cooldown must not be able to change")
	}
	if state.NextChangeAt == nil || !state.NextChangeAt.Equal(changedAt.Add(7*24*time.Hour)) {
		t.Fatalf("next change at = %v", state.NextChangeAt)
	}
	if state.Duplicates != 1 {
		t.Fatalf("duplicates = %d", state.Duplicates)
	}
}
