package migrator

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestUpCreatesRequiredRuntimeSchemas(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for migrator integration test")
	}

	ctx := context.Background()
	if err := Up(ctx, Config{DatabaseURL: databaseURL}); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	for _, table := range []string{
		"river_job",
		"river_migration",
		"extension_lifecycle_operations",
		"extension_lifecycle_steps",
	} {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected River table %s to exist after migrations", table)
		}
	}
}
