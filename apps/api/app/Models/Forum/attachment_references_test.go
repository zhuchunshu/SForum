package forum

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReplaceForumAttachmentReferencesMaintainsCounts(t *testing.T) {
	tx := &attachmentReferenceTx{validIDs: []int64{11, 12}}
	err := replaceForumAttachmentReferences(context.Background(), tx, "topic", 7, 5, []int64{11, 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.queries) != 1 || !strings.Contains(tx.queries[0], "owner_user_id = $2") || !strings.Contains(tx.queries[0], "visibility = 'public'") {
		t.Fatalf("missing ownership/public validation: %#v", tx.queries)
	}
	joined := strings.Join(tx.execs, "\n")
	for _, fragment := range []string{"DELETE FROM attachment_references", "INSERT INTO attachment_references", "reference_count = reference_count + 1"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("missing %q in SQL: %s", fragment, joined)
		}
	}
}

func TestReplaceForumAttachmentReferencesFailsBeforeMutation(t *testing.T) {
	tx := &attachmentReferenceTx{validIDs: []int64{11}}
	err := replaceForumAttachmentReferences(context.Background(), tx, "comment", 8, 5, []int64{11, 12})
	if !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("expected invalid content, got %v", err)
	}
	if len(tx.execs) != 0 {
		t.Fatalf("invalid references mutated storage: %#v", tx.execs)
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

type attachmentReferenceTx struct {
	pgx.Tx
	validIDs []int64
	queries  []string
	execs    []string
}

func (t *attachmentReferenceTx) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	t.queries = append(t.queries, sql)
	return &attachmentReferenceRows{ids: append([]int64(nil), t.validIDs...)}, nil
}

func (t *attachmentReferenceTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.execs = append(t.execs, sql)
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
