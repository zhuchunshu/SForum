package extensions

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestExecutableTrustRevocationCommitErrorClassification(t *testing.T) {
	for name, commitErr := range map[string]error{
		"transaction rolled back": pgx.ErrTxCommitRollback,
		"request not sent":        executableTrustSafeRetryError{},
		"serialization rollback": &pgconn.PgError{
			Code: "40001", Message: "serialization failure",
		},
		"deadlock rollback": &pgconn.PgError{Code: "40P01", Message: "deadlock detected"},
	} {
		t.Run(name, func(t *testing.T) {
			if !executableTrustRevocationCommitDefinitelyFailed(commitErr) {
				t.Fatalf("definite commit failure classified as ambiguous: %v", commitErr)
			}
		})
	}

	for name, commitErr := range map[string]error{
		"connection lost after write": errors.New("connection lost after COMMIT write"),
		"statement completion unknown": &pgconn.PgError{
			Code: "40003", Message: "statement completion unknown",
		},
		"transaction resolution unknown": &pgconn.PgError{
			Code: "08007", Message: "transaction resolution unknown",
		},
		"server shutdown": &pgconn.PgError{Code: "57P01", Message: "admin shutdown"},
	} {
		t.Run(name, func(t *testing.T) {
			if executableTrustRevocationCommitDefinitelyFailed(commitErr) {
				t.Fatalf("ambiguous commit was classified as definite: %v", commitErr)
			}
		})
	}
}

func TestExecutableTrustRevocationDefiniteCommitFailureSkipsReadback(t *testing.T) {
	for name, commitErr := range map[string]error{
		"transaction rolled back": pgx.ErrTxCommitRollback,
		"request not sent":        executableTrustSafeRetryError{},
		"serialization rollback": &pgconn.PgError{
			Code: "40001", Message: "serialization failure",
		},
	} {
		t.Run(name, func(t *testing.T) {
			store := &PostgresExecutableTrustStore{}
			err := store.resolveExecutableTrustRevocationCommit(
				t.Context(),
				executableTrustRevocationExpectation{extensionID: "definite.failure"},
				commitErr,
			)
			if !errors.Is(err, commitErr) || errors.Is(err, ErrTrustRevocationCommitUnknown) {
				t.Fatalf("definite commit failure=%v", err)
			}
		})
	}
}

type executableTrustSafeRetryError struct{}

func (executableTrustSafeRetryError) Error() string     { return "request was not sent" }
func (executableTrustSafeRetryError) SafeToRetry() bool { return true }
