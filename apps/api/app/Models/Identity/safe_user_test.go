package identity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestProjectSafeUserCoreOnly(t *testing.T) {
	user := CurrentUser{
		ID: 7, Username: "alice", DisplayName: "Alice", Status: UserStatusActive,
	}
	full, err := ProjectSafeUser(t.Context(), user, 0, nil, nil)
	if err != nil || full["id"] != int64(7) || full["username"] != "alice" ||
		full["displayName"] != "Alice" || full["status"] != string(UserStatusActive) {
		t.Fatalf("full core=%#v err=%v", full, err)
	}

	subset, err := ProjectSafeUser(t.Context(), user, 0, []string{"id", "username"}, nil)
	if err != nil || len(subset) != 2 || subset["id"] != int64(7) || subset["username"] != "alice" {
		t.Fatalf("subset=%#v err=%v", subset, err)
	}
}

func TestProjectSafeUserExtensionFieldsRequireActorAndPermission(t *testing.T) {
	user := CurrentUser{ID: 9, Username: "bob", DisplayName: "Bob", Status: UserStatusActive}
	reader := &fakeSafeUserFieldReader{
		values: map[string]json.RawMessage{
			"plugin.membership.tier": json.RawMessage(`"gold"`),
		},
	}

	if _, err := ProjectSafeUser(t.Context(), user, 0, []string{"plugin.membership.tier"}, reader); !errors.Is(err, ErrIdentityUserFieldPermissionDenied) {
		t.Fatalf("actorless err=%v", err)
	}
	if reader.calls != 0 {
		t.Fatalf("actorless must not call store, calls=%d", reader.calls)
	}

	if _, err := ProjectSafeUser(t.Context(), user, 3, []string{"plugin.membership.tier"}, nil); !errors.Is(err, ErrIdentityUserFieldValueStoreUnavailable) {
		t.Fatalf("missing store err=%v", err)
	}

	deniedReader := &fakeSafeUserFieldReader{err: ErrIdentityUserFieldPermissionDenied}
	if _, err := ProjectSafeUser(t.Context(), user, 3, []string{"plugin.membership.tier"}, deniedReader); !errors.Is(err, ErrIdentityUserFieldPermissionDenied) {
		t.Fatalf("denied err=%v", err)
	}

	allowedReader := &fakeSafeUserFieldReader{
		values: map[string]json.RawMessage{
			"plugin.membership.tier": json.RawMessage(`"gold"`),
		},
	}
	result, err := ProjectSafeUser(
		t.Context(), user, 3,
		[]string{"id", "plugin.membership.tier", "plugin.membership.tier", "displayName"},
		allowedReader,
	)
	if err != nil || result["id"] != int64(9) || result["displayName"] != "Bob" ||
		result["plugin.membership.tier"] != "gold" || allowedReader.calls != 1 ||
		allowedReader.lastActor != 3 || allowedReader.lastUser != 9 {
		t.Fatalf("allowed result=%#v reader=%#v err=%v", result, allowedReader, err)
	}
}

func TestProjectSafeUserOmitsMissingExtensionValues(t *testing.T) {
	user := CurrentUser{ID: 1, Username: "c", DisplayName: "C", Status: UserStatusActive}
	reader := &fakeSafeUserFieldReader{err: ErrIdentityUserFieldValueNotFound}
	result, err := ProjectSafeUser(t.Context(), user, 2, []string{"id", "plugin.membership.tier"}, reader)
	if err != nil || len(result) != 1 || result["id"] != int64(1) {
		t.Fatalf("missing field result=%#v err=%v", result, err)
	}
}

type fakeSafeUserFieldReader struct {
	values    map[string]json.RawMessage
	err       error
	calls     int
	lastActor int64
	lastUser  int64
}

func (f *fakeSafeUserFieldReader) Get(
	_ context.Context,
	input ReadIdentityUserFieldValueInput,
) (IdentityUserFieldValueRead, error) {
	f.calls++
	f.lastActor = input.ActorUserID
	f.lastUser = input.UserID
	if f.err != nil {
		return IdentityUserFieldValueRead{}, f.err
	}
	value, ok := f.values[input.FieldID]
	if !ok {
		return IdentityUserFieldValueRead{}, ErrIdentityUserFieldValueNotFound
	}
	return IdentityUserFieldValueRead{Value: append([]byte(nil), value...)}, nil
}
