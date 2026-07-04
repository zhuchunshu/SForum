package postgres

import "testing"

func TestBuildPoolConfigAppliesMaxConns(t *testing.T) {
	cfg, err := BuildPoolConfig("postgres://sforum:sforum@localhost:5432/sforum?sslmode=disable", 12)
	if err != nil {
		t.Fatalf("build pool config: %v", err)
	}

	if cfg.MaxConns != 12 {
		t.Fatalf("expected max conns 12, got %d", cfg.MaxConns)
	}
	if cfg.ConnConfig.Database != "sforum" {
		t.Fatalf("expected database sforum, got %q", cfg.ConnConfig.Database)
	}
}

func TestBuildPoolConfigIgnoresNonPositiveMaxConns(t *testing.T) {
	cfg, err := BuildPoolConfig("postgres://sforum:sforum@localhost:5432/sforum?sslmode=disable", 0)
	if err != nil {
		t.Fatalf("build pool config: %v", err)
	}

	if cfg.MaxConns <= 0 {
		t.Fatalf("expected pgx default max conns to remain positive, got %d", cfg.MaxConns)
	}
}

func TestBuildPoolConfigRejectsInvalidURL(t *testing.T) {
	if _, err := BuildPoolConfig("://not-a-postgres-url", 10); err == nil {
		t.Fatal("expected invalid database URL to fail")
	}
}
