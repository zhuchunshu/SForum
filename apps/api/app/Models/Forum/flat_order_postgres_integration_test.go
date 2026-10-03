package forum

import (
	"testing"
	"time"
)

// 评论区时间流契约（真实 Postgres，fixture 自带最小 schema）：
//  1. flat 视图按 created_at ASC, id ASC 返回；新回复恒在最底楼，不会插到被回复评论后面。
//  2. CountCommentsBefore 与 flat 列表位置逐条对齐（跨页深链反查页码依赖它）。
//  3. after keyset 按同一排序键续页，不重不漏。
//  4. tree 视图仍按 path_key 保留父子结构（同一份数据，两种视图语义分离）。
func TestPostgresFlatCommentStreamIsChronological(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	store := NewPostgresStore(fixture.pool)
	authorID := fixture.insertUser(t, "flat_stream_author")
	topic := fixture.insertBareTopic(t, authorID, "flat-stream")

	base := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	// 时间线：根 c1、根 c2，然后有人回复 c1（c3）——树序会把 c3 拽到 c2 前面，时间流必须排在 c2 之后。
	c1 := fixture.insertTimedComment(t, store, topic.id, authorID, nil, base)
	c2 := fixture.insertTimedComment(t, store, topic.id, authorID, nil, base.Add(1*time.Minute))
	c3 := fixture.insertTimedComment(t, store, topic.id, authorID, &c1, base.Add(2*time.Minute))
	c4 := fixture.insertTimedComment(t, store, topic.id, authorID, nil, base.Add(3*time.Minute))
	c5 := fixture.insertTimedComment(t, store, topic.id, authorID, &c3, base.Add(4*time.Minute))
	// 同刻并列：只靠 id 决胜，保证游标不跳行。
	c6 := fixture.insertTimedComment(t, store, topic.id, authorID, &c1, base.Add(4*time.Minute))

	want := []int64{c1, c2, c3, c4, c5, c6}

	list, err := store.ListComments(fixture.ctx, CommentListInput{TopicID: topic.id, View: "flat", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("ListComments flat: %v", err)
	}
	if len(list.Items) != len(want) {
		t.Fatalf("flat items = %d, want %d", len(list.Items), len(want))
	}
	for i, id := range want {
		if list.Items[i].ID != id {
			t.Fatalf("flat order[%d] = %d, want %d (full=%v)", i, list.Items[i].ID, id, commentIDs(list.Items))
		}
	}

	// 回复行必须保留 replyTo 引用：时间流靠引用块表达"回复谁"，父评论不再相邻。
	replyRow := list.Items[2]
	if replyRow.ID != c3 || replyRow.ReplyTo == nil || replyRow.ReplyTo.ID != c1 {
		t.Fatalf("reply row = id:%d replyTo:%#v, want reply to %d", replyRow.ID, replyRow.ReplyTo, c1)
	}

	// 逐条对齐 CountCommentsBefore（= 深链反查页码的分子）。
	for index, item := range list.Items {
		before, countErr := store.CountCommentsBefore(fixture.ctx, topic.id, item.CreatedAt, item.ID, false, 0)
		if countErr != nil {
			t.Fatalf("CountCommentsBefore(%d): %v", item.ID, countErr)
		}
		if before != int64(index) {
			t.Fatalf("CountCommentsBefore(%d) = %d, want %d", item.ID, before, index)
		}
	}

	// after keyset 续页：2 条一页走完，顺序与整页一致。
	collected := make([]int64, 0, len(want))
	after := ""
	for page := 0; page < 4; page++ {
		pageList, pageErr := store.ListComments(fixture.ctx, CommentListInput{
			TopicID: topic.id, View: "flat", Page: 1, PerPage: 2, After: after,
		})
		if pageErr != nil {
			t.Fatalf("ListComments flat keyset page %d: %v", page, pageErr)
		}
		for _, item := range pageList.Items {
			collected = append(collected, item.ID)
		}
		if !pageList.HasMore {
			break
		}
		after = pageList.NextCursor
		if after == "" {
			t.Fatalf("keyset page %d reported hasMore without cursor", page)
		}
	}
	if len(collected) != len(want) {
		t.Fatalf("keyset collected = %v, want %v", collected, want)
	}
	for i, id := range want {
		if collected[i] != id {
			t.Fatalf("keyset order[%d] = %d, want %d (full=%v)", i, collected[i], id, collected)
		}
	}

	// tree 视图不受影响：c1 仍是根，c3/c6 仍挂在它下面。
	tree, err := store.ListComments(fixture.ctx, CommentListInput{TopicID: topic.id, View: "tree", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("ListComments tree: %v", err)
	}
	if len(tree.Items) != 3 {
		t.Fatalf("tree roots = %d, want 3 (%v)", len(tree.Items), commentIDs(tree.Items))
	}
	if tree.Items[0].ID != c1 || len(tree.Items[0].Children) != 2 {
		t.Fatalf("tree root = %d children=%d, want %d with 2 children",
			tree.Items[0].ID, len(tree.Items[0].Children), c1)
	}
	if tree.Items[0].Children[0].ID != c3 || len(tree.Items[0].Children[0].Children) != 1 {
		t.Fatalf("tree c1 first child = %d, want %d with 1 child",
			tree.Items[0].Children[0].ID, c3)
	}
}

func (f *revisionLedgerPGFixture) insertTimedComment(t *testing.T, store *PostgresStore, topicID, authorID int64, parentID *int64, createdAt time.Time) int64 {
	t.Helper()
	var parent *CommentSummary
	if parentID != nil {
		summary, err := store.GetCommentSummary(f.ctx, *parentID)
		if err != nil {
			t.Fatalf("load parent summary %d: %v", *parentID, err)
		}
		parent = &summary
	}
	comment, err := store.CreateComment(f.ctx, CreateCommentRecord{
		TopicID:      topicID,
		AuthorUserID: authorID,
		ParentID:     parentID,
		Parent:       parent,
		Content:      renderedFixtureContent(t, "时间流夹具评论"),
		Status:       CommentStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	// 写入路径用 now()；夹具显式回填发表时间以获得确定的时间线。
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE comments SET created_at = $2, updated_at = $2 WHERE id = $1
	`, comment.ID, createdAt); err != nil {
		t.Fatalf("stamp comment created_at: %v", err)
	}
	return comment.ID
}

func commentIDs(items []Comment) []int64 {
	out := make([]int64, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}
