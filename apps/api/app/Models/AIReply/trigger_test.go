package aireply_test

import (
	"context"
	"errors"
	"testing"

	aireply "github.com/zhuchunshu/sforum/apps/api/app/Models/AIReply"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
)

type fakeAccounts struct {
	account aireply.BotAccount
	ok      bool
	err     error
}

func (f *fakeAccounts) ReplyBotAccount(context.Context) (aireply.BotAccount, bool, error) {
	return f.account, f.ok, f.err
}

type fakeReader struct {
	comments map[int64]aireply.CommentContext
	kinds    map[int64]identity.UserKind
	kindErr  error
}

func (f *fakeReader) CommentContext(_ context.Context, commentID int64) (aireply.CommentContext, error) {
	if snapshot, ok := f.comments[commentID]; ok {
		return snapshot, nil
	}
	return aireply.CommentContext{}, errors.New("comment not found")
}

func (f *fakeReader) UserKind(_ context.Context, userID int64) (identity.UserKind, error) {
	if f.kindErr != nil {
		return "", f.kindErr
	}
	if kind, ok := f.kinds[userID]; ok {
		return kind, nil
	}
	return identity.UserKindHuman, nil
}

type fakeEnqueuer struct {
	inputs []aireply.ReplyJobInput
	err    error
}

func (f *fakeEnqueuer) EnqueueReply(_ context.Context, input aireply.ReplyJobInput) error {
	f.inputs = append(f.inputs, input)
	return f.err
}

func commentCreatedEvent(commentID, topicID, authorUserID int64) appevents.Envelope {
	return appevents.Envelope{
		Name:        appevents.CommentCreated,
		Kind:        appevents.KindObserve,
		ActorUserID: authorUserID,
		Payload: map[string]any{
			"commentId":    commentID,
			"topicId":      topicID,
			"authorUserId": authorUserID,
		},
	}
}

func newTrigger(enqueuer *fakeEnqueuer, reader *fakeReader) *aireply.Trigger {
	return aireply.NewTrigger(
		&fakeAccounts{account: aireply.BotAccount{UserID: 77, Username: "sforum_ai"}, ok: true},
		reader,
		enqueuer,
	)
}

func withComment(snapshot aireply.CommentContext) *fakeReader {
	return &fakeReader{comments: map[int64]aireply.CommentContext{snapshot.CommentID: snapshot}}
}

func activeComment(raw string) aireply.CommentContext {
	return aireply.CommentContext{CommentID: 501, TopicID: 9, AuthorUserID: 11, Status: "active", RawContent: raw}
}

func TestOrdinaryCommentDoesNotTrigger(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	newTrigger(enqueuer, withComment(activeComment("普通讨论，没有提及任何人。"))).
		CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 0 {
		t.Fatalf("an ordinary comment must not wake the bot: %+v", enqueuer.inputs)
	}
}

// 防循环是这里最重要的性质：机器人自己的发言永远不能再唤起一次生成。
func TestBotCommentNeverTriggersAnotherGeneration(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	reader := withComment(activeComment("@sforum_ai 帮我看看"))
	reader.kinds = map[int64]identity.UserKind{77: identity.UserKindBot}
	newTrigger(enqueuer, reader).CommentCreated(context.Background(), commentCreatedEvent(501, 9, 77))
	if len(enqueuer.inputs) != 0 {
		t.Fatalf("a bot comment must never trigger generation: %+v", enqueuer.inputs)
	}
}

func TestMentioningTheBotTriggers(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	newTrigger(enqueuer, withComment(activeComment("请问 @sforum_ai 这个怎么配置？"))).
		CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 1 {
		t.Fatalf("expected one enqueue, got %+v", enqueuer.inputs)
	}
	job := enqueuer.inputs[0]
	if job.TopicID != 9 || job.ParentCommentID != 501 || job.TriggerUserID != 11 || job.BotUserID != 77 {
		t.Fatalf("job = %+v", job)
	}
}

// 提及解析复用论坛层：代码块里的 @ 不算提及，这里确认这条规则确实生效。
func TestMentionInsideCodeBlockDoesNotTrigger(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	fenced := "示例代码：" + "\n\n" + "```" + "\n" + "@sforum_ai" + "\n" + "```"
	newTrigger(enqueuer, withComment(activeComment(fenced))).
		CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 0 {
		t.Fatalf("a mention inside a code block is not a mention: %+v", enqueuer.inputs)
	}
}

func TestReplyingToTheBotTriggers(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	snapshot := activeComment("那我再问一句。")
	snapshot.ParentID = 480
	snapshot.ParentAuthorUserID = 77
	newTrigger(enqueuer, withComment(snapshot)).CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 1 {
		t.Fatalf("replying to the bot must trigger: %+v", enqueuer.inputs)
	}
}

func TestReplyingToAnotherBotTriggers(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	snapshot := activeComment("另一个机器人的回答有问题。")
	snapshot.ParentID = 480
	snapshot.ParentAuthorUserID = 88
	reader := withComment(snapshot)
	reader.kinds = map[int64]identity.UserKind{88: identity.UserKindBot}
	newTrigger(enqueuer, reader).CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 1 {
		t.Fatalf("replying to any bot account should trigger: %+v", enqueuer.inputs)
	}
}

func TestReplyingToAHumanDoesNotTrigger(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	snapshot := activeComment("同意你的看法。")
	snapshot.ParentID = 480
	snapshot.ParentAuthorUserID = 12
	newTrigger(enqueuer, withComment(snapshot)).CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 0 {
		t.Fatalf("replying to a human must not trigger: %+v", enqueuer.inputs)
	}
}

// 待审评论还没公开，不该让机器人先看到它。
func TestPendingCommentDoesNotTrigger(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	snapshot := activeComment("@sforum_ai 帮我看看")
	snapshot.Status = "pending"
	newTrigger(enqueuer, withComment(snapshot)).CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 0 {
		t.Fatalf("pending content must not trigger: %+v", enqueuer.inputs)
	}
}

func TestNoBotAccountDisablesTheFeature(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	trigger := aireply.NewTrigger(&fakeAccounts{ok: false}, withComment(activeComment("@sforum_ai")), enqueuer)
	trigger.CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 0 {
		t.Fatalf("without a bot account nothing should happen: %+v", enqueuer.inputs)
	}
}

func TestEnqueueFailureIsSwallowed(t *testing.T) {
	enqueuer := &fakeEnqueuer{err: errors.New("queue unavailable")}
	newTrigger(enqueuer, withComment(activeComment("@sforum_ai 在吗"))).
		CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 1 {
		t.Fatal("the attempt should still be recorded by the fake")
	}
}

func TestReaderFailureIsSwallowed(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	reader := &fakeReader{kindErr: errors.New("db down")}
	newTrigger(enqueuer, reader).CommentCreated(context.Background(), commentCreatedEvent(501, 9, 11))
	if len(enqueuer.inputs) != 0 {
		t.Fatal("a reader failure must not enqueue anything")
	}
}

// Subscriber 是发布路径上的一层包装：它必须原样返回原发布结果，且只对
// comment.created 这一个事件做额外处理。
func TestSubscriberForwardsAndOnlyHandlesCommentCreated(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	inner := &recordingPublisher{result: appevents.Result{OK: true}}
	subscriber := aireply.NewSubscriber(inner, newTrigger(enqueuer, withComment(activeComment("@sforum_ai 在吗"))))

	if result := subscriber.Emit(context.Background(), commentCreatedEvent(501, 9, 11)); !result.OK {
		t.Fatalf("the inner result must be forwarded: %+v", result)
	}
	if len(inner.names) != 1 || inner.names[0] != appevents.CommentCreated {
		t.Fatalf("the original event must still be published: %+v", inner.names)
	}
	if len(enqueuer.inputs) != 1 {
		t.Fatalf("comment.created should reach the trigger: %+v", enqueuer.inputs)
	}

	subscriber.Emit(context.Background(), appevents.Envelope{Name: appevents.TopicCreated, Kind: appevents.KindObserve})
	if len(enqueuer.inputs) != 1 {
		t.Fatalf("unrelated events must not reach the trigger: %+v", enqueuer.inputs)
	}
}

type recordingPublisher struct {
	result appevents.Result
	names  []string
}

func (p *recordingPublisher) Emit(_ context.Context, envelope appevents.Envelope) appevents.Result {
	p.names = append(p.names, envelope.Name)
	return p.result
}
