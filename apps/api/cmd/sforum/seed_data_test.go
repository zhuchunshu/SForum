package main

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

var (
	seedUsernamePattern = regexp.MustCompile(`^seed_[0-9a-f]{12}$`)
	seedEmailPattern    = regexp.MustCompile(`^seed_[0-9a-f]{12}@seed\.local$`)
)

func TestGenerateSeedUserFormat(t *testing.T) {
	u, err := generateSeedUser(3)
	if err != nil {
		t.Fatalf("generateSeedUser returned error: %v", err)
	}
	if !seedUsernamePattern.MatchString(u.Username) {
		t.Fatalf("unexpected username format: %q", u.Username)
	}
	if !seedEmailPattern.MatchString(u.Email) {
		t.Fatalf("unexpected email format: %q", u.Email)
	}
	// 显示名带序号（从 1 开始），便于人工区分。
	if u.DisplayName != "种子用户4" {
		t.Fatalf("unexpected display name: %q", u.DisplayName)
	}
	if !strings.HasPrefix(u.Password, "Seed-") {
		t.Fatalf("unexpected password format: %q", u.Password)
	}
}

func TestGenerateSeedUserSuffixIsRandom(t *testing.T) {
	// 连续生成多个用户，后缀应几乎不可能重复（6 字节 = 48 位随机）。
	seen := make(map[string]bool, 8)
	for i := 0; i < 8; i++ {
		u, err := generateSeedUser(i)
		if err != nil {
			t.Fatalf("generateSeedUser(%d) returned error: %v", i, err)
		}
		if seen[u.Username] {
			t.Fatalf("duplicate username across runs: %q", u.Username)
		}
		seen[u.Username] = true
	}
}

func TestGenerateSeedDatasetShape(t *testing.T) {
	opts := seedOptions{Count: 10, Users: 3, CommentsMax: 4, CategorySlug: "general"}
	rng := rand.New(rand.NewPCG(1, 2))
	dataset, err := generateSeedDataset(opts, rng)
	if err != nil {
		t.Fatalf("generateSeedDataset returned error: %v", err)
	}

	if len(dataset.Users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(dataset.Users))
	}
	if len(dataset.Topics) != 10 {
		t.Fatalf("expected 10 topics, got %d", len(dataset.Topics))
	}

	// 每个主题字段必填，作者下标在合法范围。
	for i, plan := range dataset.Topics {
		if strings.TrimSpace(plan.Topic.Title) == "" {
			t.Fatalf("topic %d has empty title", i)
		}
		if strings.TrimSpace(plan.Topic.Body) == "" {
			t.Fatalf("topic %d has empty body", i)
		}
		if plan.Topic.AuthorIndex < 0 || plan.Topic.AuthorIndex >= opts.Users {
			t.Fatalf("topic %d author index out of range: %d", i, plan.Topic.AuthorIndex)
		}
		// CommentsMax=4 → 评论数应在 [0,4]。
		if len(plan.Comments) > opts.CommentsMax {
			t.Fatalf("topic %d has %d comments, max %d", i, len(plan.Comments), opts.CommentsMax)
		}
		// 评论作者下标与 ParentOffset 合法性。
		for j, c := range plan.Comments {
			if c.AuthorIndex < 0 || c.AuthorIndex >= opts.Users {
				t.Fatalf("topic %d comment %d author index out of range: %d", i, j, c.AuthorIndex)
			}
			if strings.TrimSpace(c.Body) == "" {
				t.Fatalf("topic %d comment %d has empty body", i, j)
			}
			// ParentOffset=-1 表示顶层；否则必须指向更早的评论（避免前向引用导致 offset 越界）。
			if c.ParentOffset != -1 && (c.ParentOffset < 0 || c.ParentOffset >= j) {
				t.Fatalf("topic %d comment %d parent offset invalid: %d (j=%d)", i, j, c.ParentOffset, j)
			}
		}
	}
}

func TestGenerateSeedDatasetNoCommentsWhenMaxZero(t *testing.T) {
	opts := seedOptions{Count: 5, Users: 2, CommentsMax: 0}
	rng := rand.New(rand.NewPCG(7, 9))
	dataset, err := generateSeedDataset(opts, rng)
	if err != nil {
		t.Fatalf("generateSeedDataset returned error: %v", err)
	}
	for i, plan := range dataset.Topics {
		if len(plan.Comments) != 0 {
			t.Fatalf("topic %d should have no comments when CommentsMax=0, got %d", i, len(plan.Comments))
		}
	}
}

func TestValidateSeedOptions(t *testing.T) {
	cases := []struct {
		name    string
		opts    seedOptions
		wantErr bool
	}{
		{"valid", seedOptions{Profile: seedProfileSmall, Count: 100, Users: 10, CommentsMax: 5, Batch: 20}, false},
		{"zero count", seedOptions{Profile: seedProfileSmall, Count: 0, Users: 10, CommentsMax: 5}, true},
		{"zero users", seedOptions{Profile: seedProfileSmall, Count: 100, Users: 0, CommentsMax: 5}, true},
		{"negative comments", seedOptions{Profile: seedProfileSmall, Count: 100, Users: 10, CommentsMax: -1}, true},
		{"zero comments allowed", seedOptions{Profile: seedProfileSmall, Count: 100, Users: 10, CommentsMax: 0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSeedOptions(&tc.opts)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
