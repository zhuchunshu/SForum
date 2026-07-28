package forum

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

var errNotificationFanoutFixture = errors.New("notification fanout failed")

type recordingNotificationWriter struct {
	topics   []TopicNotificationInput
	comments []CommentNotificationInput
	err      error
}

func (w *recordingNotificationWriter) NotifyTopicTx(_ context.Context, _ pgx.Tx, input TopicNotificationInput) error {
	w.topics = append(w.topics, input)
	return w.err
}

func (w *recordingNotificationWriter) NotifyCommentTx(_ context.Context, _ pgx.Tx, input CommentNotificationInput) error {
	w.comments = append(w.comments, input)
	return w.err
}

func TestCreateTopicNotificationFanoutSharesOwningTransactionPostgres(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	authorID := fixture.insertUser(t, "fanout_topic_author")

	t.Run("active uses parsed recipients", func(t *testing.T) {
		writer := &recordingNotificationWriter{}
		store := NewPostgresStore(fixture.pool).WithCommentNotifications(writer)
		topic, err := store.CreateTopic(fixture.ctx, CreateTopicRecord{
			CategorySlug:       "general",
			AuthorUserID:      authorID,
			Title:             "Active notification topic",
			Slug:              "active-notification-topic",
			Content:           renderedFixtureContent(t, "hello @recipient"),
			Status:            TopicStatusActive,
			MentionedUsernames: []string{"recipient"},
		})
		if err != nil {
			t.Fatalf("create active topic: %v", err)
		}
		if len(writer.topics) != 1 || writer.topics[0].TopicID != topic.ID || writer.topics[0].ActorUserID != authorID {
			t.Fatalf("unexpected topic notification input: %#v", writer.topics)
		}
	})

	t.Run("pending defers fanout", func(t *testing.T) {
		writer := &recordingNotificationWriter{}
		store := NewPostgresStore(fixture.pool).WithCommentNotifications(writer)
		if _, err := store.CreateTopic(fixture.ctx, CreateTopicRecord{
			CategorySlug:       "general",
			AuthorUserID:      authorID,
			Title:             "Pending notification topic",
			Slug:              "pending-notification-topic",
			Content:           renderedFixtureContent(t, "hello @recipient"),
			Status:            TopicStatusPending,
			MentionedUsernames: []string{"recipient"},
		}); err != nil {
			t.Fatalf("create pending topic: %v", err)
		}
		if len(writer.topics) != 0 {
			t.Fatalf("pending topic fanned out before approval: %#v", writer.topics)
		}
	})

	t.Run("notification error rolls back topic", func(t *testing.T) {
		writer := &recordingNotificationWriter{err: errNotificationFanoutFixture}
		store := NewPostgresStore(fixture.pool).WithCommentNotifications(writer)
		_, err := store.CreateTopic(fixture.ctx, CreateTopicRecord{
			CategorySlug:  "general",
			AuthorUserID: authorID,
			Title:        "Rolled back notification topic",
			Slug:         "rolled-back-notification-topic",
			Content:      renderedFixtureContent(t, "body"),
			Status:       TopicStatusActive,
		})
		if !errors.Is(err, errNotificationFanoutFixture) {
			t.Fatalf("expected notification failure, got %v", err)
		}
		var count int
		if err := fixture.pool.QueryRow(fixture.ctx, `SELECT count(*) FROM topics WHERE slug='rolled-back-notification-topic'`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("notification failure committed %d topic rows", count)
		}
	})
}

func TestCreateCommentNotificationContextAndRollbackPostgres(t *testing.T) {
	fixture := newRevisionLedgerPGFixture(t)
	topicAuthorID := fixture.insertUser(t, "fanout_comment_topic_author")
	parentAuthorID := fixture.insertUser(t, "fanout_comment_parent_author")
	actorID := fixture.insertUser(t, "fanout_comment_actor")
	topic := fixture.insertBareTopic(t, topicAuthorID, "fanout-comment-topic")
	baseStore := NewPostgresStore(fixture.pool)
	parent, err := baseStore.CreateComment(fixture.ctx, CreateCommentRecord{
		TopicID:           topic.id,
		AuthorUserID:      parentAuthorID,
		TopicAuthorUserID: topicAuthorID,
		Content:           renderedFixtureContent(t, "parent"),
		Status:            CommentStatusActive,
	})
	if err != nil {
		t.Fatalf("create parent comment: %v", err)
	}
	parentSummary, err := baseStore.GetCommentSummary(fixture.ctx, parent.ID)
	if err != nil {
		t.Fatalf("load parent summary: %v", err)
	}

	writer := &recordingNotificationWriter{}
	store := NewPostgresStore(fixture.pool).WithCommentNotifications(writer)
	created, err := store.CreateComment(fixture.ctx, CreateCommentRecord{
		TopicID:             topic.id,
		AuthorUserID:        actorID,
		TopicAuthorUserID:   topicAuthorID,
		ParentID:            &parent.ID,
		Parent:              &parentSummary,
		Content:             renderedFixtureContent(t, "nested @recipient"),
		Status:              CommentStatusActive,
		MentionedUsernames:  []string{"recipient"},
	})
	if err != nil {
		t.Fatalf("create nested comment: %v", err)
	}
	if len(writer.comments) != 1 {
		t.Fatalf("notification calls=%d want=1", len(writer.comments))
	}
	got := writer.comments[0]
	if got.CommentID != created.ID || got.TopicID != topic.id || got.ActorUserID != actorID || got.TopicAuthorUserID != topicAuthorID || got.ParentAuthorUserID != parentAuthorID {
		t.Fatalf("unexpected comment notification context: %#v", got)
	}

	failing := NewPostgresStore(fixture.pool).WithCommentNotifications(&recordingNotificationWriter{err: errNotificationFanoutFixture})
	_, err = failing.CreateComment(fixture.ctx, CreateCommentRecord{
		TopicID:           topic.id,
		AuthorUserID:      actorID,
		TopicAuthorUserID: topicAuthorID,
		Content:           renderedFixtureContent(t, "must roll back"),
		Status:            CommentStatusActive,
	})
	if !errors.Is(err, errNotificationFanoutFixture) {
		t.Fatalf("expected notification failure, got %v", err)
	}
	var bodyCount int
	if err := fixture.pool.QueryRow(fixture.ctx, `SELECT count(*) FROM posts WHERE raw_content='must roll back'`).Scan(&bodyCount); err != nil {
		t.Fatal(err)
	}
	if bodyCount != 0 {
		t.Fatalf("notification failure committed %d comment posts", bodyCount)
	}
}
