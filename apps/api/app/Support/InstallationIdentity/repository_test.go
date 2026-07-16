package installationidentity

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRepositoryEnsureCreatesAndReusesOpaqueIdentity(t *testing.T) {
	store := &fakeIdentityStore{}
	repository := &Repository{store: store}
	first, err := repository.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !Valid(first) || store.execCalls != 2 || store.queryCalls != 2 {
		t.Fatalf("identities = %q, %q store=%#v", first, second, store)
	}
}

func TestRepositoryEnsureFailsClosedOnStorageAndTamperedValues(t *testing.T) {
	for name, repository := range map[string]*Repository{
		"nil repository": nil,
		"nil store":      {},
		"insert failure": {store: &fakeIdentityStore{execErr: errors.New("insert unavailable")}},
		"read failure":   {store: &fakeIdentityStore{rowErr: errors.New("read unavailable")}},
		"uppercase":      {store: &fakeIdentityStore{identity: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}},
		"short":          {store: &fakeIdentityStore{identity: "aa"}},
		"non-hex":        {store: &fakeIdentityStore{identity: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := repository.Ensure(context.Background())
			if err == nil || (!errors.Is(err, ErrUnavailable) && !errors.Is(err, ErrInvalid)) {
				t.Fatalf("Ensure error = %v", err)
			}
		})
	}
	if _, err := (&Repository{store: &fakeIdentityStore{}}).Ensure(nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil context error = %v", err)
	}
}

func TestValidInstallationIdentity(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if !Valid(valid) {
		t.Fatal("valid identity rejected")
	}
	for _, invalid := range []string{"", valid[:63], valid + "0", "A" + valid[1:], "g" + valid[1:]} {
		if Valid(invalid) {
			t.Fatalf("invalid identity accepted: %q", invalid)
		}
	}
}

type fakeIdentityStore struct {
	identity   string
	execErr    error
	rowErr     error
	execCalls  int
	queryCalls int
}

func (s *fakeIdentityStore) Exec(
	_ context.Context,
	_ string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	s.execCalls++
	if s.execErr != nil {
		return pgconn.CommandTag{}, s.execErr
	}
	if s.identity == "" && len(arguments) == 1 {
		s.identity, _ = arguments[0].(string)
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (s *fakeIdentityStore) QueryRow(context.Context, string, ...any) pgx.Row {
	s.queryCalls++
	return fakeIdentityRow{identity: s.identity, err: s.rowErr}
}

type fakeIdentityRow struct {
	identity string
	err      error
}

func (r fakeIdentityRow) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(destinations) != 1 {
		return errors.New("unexpected scan destination")
	}
	value, ok := destinations[0].(*string)
	if !ok {
		return errors.New("unexpected scan type")
	}
	*value = r.identity
	return nil
}
