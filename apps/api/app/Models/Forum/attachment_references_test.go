package forum

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReplaceForumAttachmentReferencesWritesOneBulkInsert(t *testing.T) {
	tx := &attachmentReferenceTx{validIDs: []int64{11, 12}}
	err := replaceForumAttachmentReferences(context.Background(), tx, "topic", 7, 5, []int64{11, 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.queries) != 2 {
		t.Fatalf("expected validate + clear queries, got %#v", tx.queries)
	}
	validate := tx.queries[0]
	for _, fragment := range []string{
		"a.owner_user_id = $2",
		"FROM attachment_references ar",
		"ar.resource_type = $3",
		"a.visibility = 'public'",
		"ORDER BY a.id",
		"FOR UPDATE",
	} {
		if !strings.Contains(validate, fragment) {
			t.Fatalf("validate SQL missing %q: %s", fragment, validate)
		}
	}
	if !strings.Contains(tx.queries[1], "DELETE FROM attachment_references") {
		t.Fatalf("clear SQL=%s", tx.queries[1])
	}

	// 两个附件只允许一次 INSERT：逐行插入既慢，也会在唯一性收紧/放宽时产生
	// N 次往返与部分失败。
	if len(tx.execs) != 2 {
		t.Fatalf("expected bulk insert + recount, got %#v", tx.execs)
	}
	insert := tx.execs[0]
	if !strings.Contains(insert, "INSERT INTO attachment_references") ||
		!strings.Contains(insert, "SELECT unnest($1::bigint[])") ||
		!strings.Contains(insert, "ON CONFLICT DO NOTHING") {
		t.Fatalf("insert SQL=%s", insert)
	}
	if !strings.Contains(tx.execs[1], "reference_count = counts.amount") {
		t.Fatalf("recount SQL=%s", tx.execs[1])
	}
}

func TestReplaceForumAttachmentReferencesReportsUnavailableIDs(t *testing.T) {
	tx := &attachmentReferenceTx{validIDs: []int64{11}}
	err := replaceForumAttachmentReferences(context.Background(), tx, "comment", 8, 5, []int64{11, 12})
	if !errors.Is(err, ErrAttachmentUnavailable) {
		t.Fatalf("expected attachment unavailable, got %v", err)
	}
	var unavailable *AttachmentUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("expected typed error, got %T", err)
	}
	if len(unavailable.AttachmentIDs) != 1 || unavailable.AttachmentIDs[0] != 12 {
		t.Fatalf("unavailable ids=%v", unavailable.AttachmentIDs)
	}
	if unavailable.ResourceType != "comment" || unavailable.ResourceID != 8 {
		t.Fatalf("resource=%s/%d", unavailable.ResourceType, unavailable.ResourceID)
	}
	if len(tx.execs) != 0 {
		t.Fatalf("invalid references mutated storage: %#v", tx.execs)
	}
}

func TestReplaceForumAttachmentReferencesMapsUniqueViolation(t *testing.T) {
	tx := &attachmentReferenceTx{
		validIDs: []int64{11, 12},
		execErr: &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "attachment_references_slot_context_unique",
		},
	}
	err := replaceForumAttachmentReferences(context.Background(), tx, "topic", 7, 5, []int64{11, 12})
	if !errors.Is(err, ErrAttachmentConflict) {
		t.Fatalf("expected attachment conflict, got %v", err)
	}
	var conflict *AttachmentConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected typed conflict, got %T", err)
	}
	if conflict.Constraint != "attachment_references_slot_context_unique" {
		t.Fatalf("constraint=%s", conflict.Constraint)
	}
}

func TestReplaceForumAttachmentReferencesClearsWithoutRecountWhenEmpty(t *testing.T) {
	tx := &attachmentReferenceTx{}
	if err := replaceForumAttachmentReferences(context.Background(), tx, "topic", 7, 5, nil); err != nil {
		t.Fatal(err)
	}
	// 没有引用也没有附件：只需要一次 DELETE，不应再去 INSERT 或重算。
	if len(tx.execs) != 0 {
		t.Fatalf("unexpected writes: %#v", tx.execs)
	}
	if len(tx.queries) != 1 || !strings.Contains(tx.queries[0], "DELETE FROM attachment_references") {
		t.Fatalf("queries=%#v", tx.queries)
	}
}

func TestReplaceForumAttachmentReferencesRejectsUnknownResourceType(t *testing.T) {
	tx := &attachmentReferenceTx{}
	if err := replaceForumAttachmentReferences(context.Background(), tx, "site", 0, 5, []int64{11}); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("expected invalid content, got %v", err)
	}
	if len(tx.queries) != 0 || len(tx.execs) != 0 {
		t.Fatalf("unexpected storage access: %#v %#v", tx.queries, tx.execs)
	}
}

func TestClearTopicAttachmentReferencesIncludesComments(t *testing.T) {
	tx := &attachmentReferenceTx{}
	if err := clearTopicAttachmentReferences(context.Background(), tx, 7); err != nil {
		t.Fatal(err)
	}
	if len(tx.execs) != 1 || !strings.Contains(tx.execs[0], "SELECT id FROM comments WHERE topic_id = $1") || !strings.Contains(tx.execs[0], "reference_count") {
		t.Fatalf("topic cleanup SQL=%#v", tx.execs)
	}
}

func TestUnionAttachmentIDsKeepsOrderAndDeduplicates(t *testing.T) {
	got := unionAttachmentIDs([]int64{9, 3, 9}, []int64{3, 12, 0})
	want := []int64{9, 3, 12}
	if len(got) != len(want) {
		t.Fatalf("merged=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("merged=%v", got)
		}
	}
}

type attachmentReferenceTx struct {
	pgx.Tx
	validIDs []int64
	execErr  error
	queries  []string
	execs    []string
}

func (t *attachmentReferenceTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	t.queries = append(t.queries, sql)
	return &attachmentReferenceRows{ids: append([]int64(nil), t.validIDs...)}, nil
}

func (t *attachmentReferenceTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.execs = append(t.execs, sql)
	if t.execErr != nil {
		err := t.execErr
		// 只让第一次写入失败，模拟「INSERT 撞约束、后续重算不会发生」。
		t.execErr = nil
		return pgconn.CommandTag{}, err
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

type attachmentReferenceRows struct {
	pgx.Rows
	ids    []int64
	index  int
	closed bool
}

func (r *attachmentReferenceRows) Next() bool {
	if r.closed || r.index >= len(r.ids) {
		return false
	}
	r.index++
	return true
}

func (r *attachmentReferenceRows) Scan(dest ...any) error {
	*(dest[0].(*int64)) = r.ids[r.index-1]
	return nil
}

func (r *attachmentReferenceRows) Err() error { return nil }
func (r *attachmentReferenceRows) Close()     { r.closed = true }
