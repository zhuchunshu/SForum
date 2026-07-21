package forum

import (
	"context"
	"testing"
)

func TestNormalizeTreeDescendantsPerRoot(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, RecommendedTreeDescendantsPerRoot},
		{-1, RecommendedTreeDescendantsPerRoot},
		{1, 1},
		{50, 50},
		{100, 100},
		{101, 100},
	}
	for _, tc := range cases {
		if got := normalizeTreeDescendantsPerRoot(tc.in); got != tc.want {
			t.Fatalf("normalizeTreeDescendantsPerRoot(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestMarkTreeHasMoreChildren 模拟 store 在 cap 后标记 hasMoreChildren 的语义：
// 子孙总数 > N 时 root.hasMoreChildren=true，且树中只应挂最多 N 个子孙节点。
func TestMarkTreeHasMoreChildren(t *testing.T) {
	const capN = 5
	rootID := int64(1)
	// 1 根 + 15 直系子孙（> cap）。
	items := []Comment{{ID: rootID, PathKey: "000000000001", Depth: 0}}
	for i := 0; i < 15; i++ {
		id := int64(100 + i)
		pid := rootID
		items = append(items, Comment{
			ID: id, ParentID: &pid, RootCommentID: rootID,
			PathKey: formatCommentPathSegment(rootID) + "." + formatCommentPathSegment(id),
			Depth:   1,
		})
	}
	// 模拟 store：只保留前 capN 个子孙，并标记 hasMore。
	kept := make([]Comment, 0, 1+capN)
	kept = append(kept, items[0])
	kept[0].HasMoreChildren = true
	kept = append(kept, items[1:1+capN]...)

	tree := buildCommentTree(kept)
	if len(tree) != 1 {
		t.Fatalf("roots = %d, want 1", len(tree))
	}
	if !tree[0].HasMoreChildren {
		t.Fatal("expected hasMoreChildren on truncated root")
	}
	if got := countDescendants(tree[0]); got != capN {
		t.Fatalf("descendants in tree = %d, want %d", got, capN)
	}
}

func countDescendants(c Comment) int {
	n := 0
	for _, child := range c.Children {
		n++
		n += countDescendants(child)
	}
	return n
}

func TestServiceListCommentsPassesTreeDescendantsCap(t *testing.T) {
	store := newServiceFakeStore()
	store.listCommentsResult = CommentList{Items: []Comment{{ID: 1}}, Total: 1, View: "tree"}
	settings := testForumSettings()
	settings.TreeDescendantsPerRoot = 7
	service := NewServiceWithSettingsAndEvents(store, fakeSettingsResolver{settings: settings}, nil)

	if _, err := service.ListComments(context.Background(), CommentListInput{TopicID: 10, View: "tree"}); err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if store.listCommentsInput.TreeDescendantsPerRoot != 7 {
		t.Fatalf("TreeDescendantsPerRoot = %d, want 7", store.listCommentsInput.TreeDescendantsPerRoot)
	}
}

func TestDefaultForumSettingsTreeDescendants(t *testing.T) {
	s := defaultForumSettings()
	if s.TreeDescendantsPerRoot != RecommendedTreeDescendantsPerRoot {
		t.Fatalf("default TreeDescendantsPerRoot = %d, want %d", s.TreeDescendantsPerRoot, RecommendedTreeDescendantsPerRoot)
	}
	if !isValidForumSettings(s) {
		t.Fatal("defaultForumSettings should be valid")
	}
}
