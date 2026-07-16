package installationidentity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	IdentityBytes     = 32
	IdentityHexLength = IdentityBytes * 2
)

var (
	ErrUnavailable = errors.New("host installation identity is unavailable")
	ErrInvalid     = errors.New("host installation identity is invalid")
)

type postgresIdentityStore interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Repository initializes and reads the database-owned installation identity.
// It never derives authority from process, hostname, DSN, credentials, or Redis.
type Repository struct {
	store postgresIdentityStore
}

func NewPostgresRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{store: pool}
}

// Ensure races safely across API and worker processes. INSERT completion is a
// visibility fence: a conflicting initializer commits before the following
// SELECT takes its new READ COMMITTED snapshot.
func (r *Repository) Ensure(ctx context.Context) (string, error) {
	if r == nil || r.store == nil || ctx == nil {
		return "", ErrUnavailable
	}
	candidate, err := newIdentity()
	if err != nil {
		return "", errors.Join(ErrUnavailable, err)
	}
	if _, err := r.store.Exec(ctx, `
		INSERT INTO host_installation_identity (singleton, installation_id)
		VALUES (TRUE, $1)
		ON CONFLICT (singleton) DO NOTHING
	`, candidate); err != nil {
		return "", fmt.Errorf("%w: initialize: %v", ErrUnavailable, err)
	}
	var identity string
	if err := r.store.QueryRow(ctx, `
		SELECT installation_id
		FROM host_installation_identity
		WHERE singleton = TRUE
	`).Scan(&identity); err != nil {
		return "", fmt.Errorf("%w: read: %v", ErrUnavailable, err)
	}
	if !Valid(identity) {
		return "", ErrInvalid
	}
	return identity, nil
}

func Valid(identity string) bool {
	if len(identity) != IdentityHexLength || identity != strings.ToLower(identity) {
		return false
	}
	decoded, err := hex.DecodeString(identity)
	return err == nil && len(decoded) == IdentityBytes
}

func newIdentity() (string, error) {
	random := make([]byte, IdentityBytes)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return hex.EncodeToString(random), nil
}
