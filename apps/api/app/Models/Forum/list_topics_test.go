package forum

import (
	"strings"
	"testing"
)

func TestClassifyListTopicsTotal_D1Modes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		category     string
		tag          string
		wantMode     listTopicsTotalMode
		wantApprox   bool
	}{
		{name: "home", category: "", tag: "", wantMode: listTopicsTotalHome, wantApprox: true},
		{name: "category only", category: "general", tag: "", wantMode: listTopicsTotalCategory, wantApprox: false},
		{name: "tag only", category: "", tag: "go", wantMode: listTopicsTotalTag, wantApprox: false},
		{name: "multi filter", category: "general", tag: "go", wantMode: listTopicsTotalMulti, wantApprox: true},
		{name: "trim spaces", category: "  general ", tag: "  ", wantMode: listTopicsTotalCategory, wantApprox: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mode, approx := classifyListTopicsTotal(tc.category, tc.tag)
			if mode != tc.wantMode {
				t.Fatalf("mode = %q, want %q", mode, tc.wantMode)
			}
			if approx != tc.wantApprox {
				t.Fatalf("approximate = %v, want %v", approx, tc.wantApprox)
			}
		})
	}
}

func TestListTopicsWhereSQL_NoILIKE(t *testing.T) {
	t.Parallel()
	sql := listTopicsWhereSQL()
	if strings.Contains(strings.ToUpper(sql), "ILIKE") {
		t.Fatalf("list WHERE must not contain ILIKE; got:\n%s", sql)
	}
	if strings.Contains(sql, "plain_text") {
		t.Fatalf("list WHERE must not reference plain_text; got:\n%s", sql)
	}
	if !strings.Contains(sql, "topic_tags") {
		t.Fatalf("expected tag EXISTS via topic_tags")
	}
}

func TestTopicListOrderBy_ActiveUsesActivityColumns(t *testing.T) {
	t.Parallel()
	// 默认 active 排序列序需与 topics_category_activity_idx 后缀对齐（is_pinned, last_activity_at, id）。
	order := topicListOrderBy("active")
	for _, frag := range []string{
		"topics.is_pinned DESC",
		"topics.last_activity_at DESC",
		"topics.id DESC",
	} {
		if !strings.Contains(order, frag) {
			t.Fatalf("active order missing %q in %q", frag, order)
		}
	}
	// 空 sort 与 active 相同（service 会填默认；store 侧 default 分支）。
	if topicListOrderBy("") != order {
		t.Fatalf("empty sort should match active")
	}
}

func TestTopicListOrderBy_HotUsesHotScoreColumn(t *testing.T) {
	t.Parallel()
	// M2：hot 走 hot_score 列，禁止再表达式排序 comment_count*5+view_count。
	order := topicListOrderBy("hot")
	for _, frag := range []string{
		"topics.is_pinned DESC",
		"topics.hot_score DESC",
		"topics.id DESC",
	} {
		if !strings.Contains(order, frag) {
			t.Fatalf("hot order missing %q in %q", frag, order)
		}
	}
	if strings.Contains(order, "comment_count") || strings.Contains(order, "view_count") {
		t.Fatalf("hot order must not use live expression: %q", order)
	}
}

func TestTopicSummarySQL_UsesPlainTextPrefixNotFullBody(t *testing.T) {
	t.Parallel()
	sql := topicSummarySQL()
	// 列表摘要 SQL 使用 left(plain_text, N)，不得 SELECT 全量 raw/html。
	if !strings.Contains(sql, "left(posts.plain_text") {
		t.Fatalf("expected left(posts.plain_text, …) prefix, got:\n%s", sql)
	}
	if strings.Contains(sql, "posts.raw_content") || strings.Contains(sql, "posts.html_content") {
		t.Fatalf("summary SQL must not select raw/html body columns")
	}
}

func TestListTopicsPageSQL_CategoryUsesCategoryIDEquality(t *testing.T) {
	t.Parallel()
	sql, args := listTopicsPageSQL("general", "", "active", 1, 20)
	if !strings.Contains(sql, "topics.category_id =") {
		t.Fatalf("category list should filter by category_id equality for index order, got:\n%s", sql)
	}
	if strings.Contains(strings.ToUpper(sql), "ILIKE") {
		t.Fatalf("page SQL must not contain ILIKE")
	}
	if len(args) != 4 {
		t.Fatalf("category page args len=%d want 4", len(args))
	}
	if args[0] != "general" || args[2] != 20 || args[3] != 0 {
		t.Fatalf("unexpected args %#v", args)
	}
}

func TestListTopicsPageSQL_HomeNoFullCount(t *testing.T) {
	t.Parallel()
	sql, args := listTopicsPageSQL("", "", "active", 1, 20)
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "COUNT(*)") {
		t.Fatalf("page SQL must not COUNT(*)")
	}
	if strings.Contains(upper, "ILIKE") {
		t.Fatalf("page SQL must not ILIKE")
	}
	if !strings.Contains(sql, "LIMIT $2 OFFSET $3") {
		t.Fatalf("home page expects LIMIT/OFFSET placeholders, got:\n%s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("home page args len=%d want 3", len(args))
	}
}
