package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

const externalAuthT1DSessionCredentialsMigration = "202607270057_external_auth_t1d_session_credentials.sql"

// T8C：migration 057 不得冗余改动 password_hash；Down 必须保留 055/0001 的 NOT NULL 不变量。
func TestT8C_Migration057DoesNotTouchPasswordHash(t *testing.T) {
	body, err := fs.ReadFile(Files(), externalAuthT1DSessionCredentialsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration 057 has no Down section")
	}
	// 去掉 SQL 注释后再检查，避免说明性注释触发假阳性。
	upSQL := stripSQLComments(parts[0])
	downSQL := stripSQLComments(parts[1])
	up := strings.ToLower(upSQL)
	down := strings.ToLower(downSQL)

	for _, forbidden := range []string{
		"password_hash",
		"user_credentials",
	} {
		if strings.Contains(up, forbidden) {
			t.Fatalf("migration 057 Up SQL must not reference %q", forbidden)
		}
		if strings.Contains(down, forbidden) {
			t.Fatalf("migration 057 Down SQL must not reference %q (preserve NOT NULL invariant)", forbidden)
		}
	}
	// Down 不得 DROP NOT NULL（即使未来误写列名变体）。
	for _, forbidden := range []string{
		"drop not null",
		"drop notnull",
	} {
		if strings.Contains(down, forbidden) {
			t.Fatalf("migration 057 Down must not %q", forbidden)
		}
	}
}

// stripSQLComments 移除 -- line comments，便于断言可执行 SQL。
func stripSQLComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestT8C_Migration057RebuildsSessionBoundRecentAuth(t *testing.T) {
	body, err := fs.ReadFile(Files(), externalAuthT1DSessionCredentialsMigration)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(body), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration 057 has no Down section")
	}
	up := strings.Join(strings.Fields(parts[0]), " ")
	for _, clause := range []string{
		"DROP TABLE IF EXISTS user_recent_auth",
		"CREATE TABLE user_recent_auth",
		"session_fingerprint TEXT NOT NULL",
		"PRIMARY KEY (user_id, session_fingerprint)",
		"CREATE INDEX user_recent_auth_expires_idx",
	} {
		if !strings.Contains(up, clause) {
			t.Fatalf("migration 057 Up missing %q", clause)
		}
	}
	down := strings.Join(strings.Fields(parts[1]), " ")
	// Down 回退到 user-scoped 形态，但仍不得触碰 credentials。
	for _, clause := range []string{
		"DROP TABLE IF EXISTS user_recent_auth",
		"CREATE TABLE user_recent_auth",
		"user_id BIGINT PRIMARY KEY",
	} {
		if !strings.Contains(down, clause) {
			t.Fatalf("migration 057 Down missing %q", clause)
		}
	}
}
