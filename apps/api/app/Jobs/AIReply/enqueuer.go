package aireplyjobs

import (
	"context"

	aireply "github.com/zhuchunshu/sforum/apps/api/app/Models/AIReply"
	supportjobs "github.com/zhuchunshu/sforum/apps/api/app/Support/Jobs"
)

// Enqueuer 实现 aireply.Enqueuer：把触发判定产生的任务放进持久队列。
// 它放在任务包里而不是领域包里，是为了让依赖保持单向——领域包不认识队列。
type Enqueuer struct {
	dispatcher *supportjobs.Dispatcher
}

func NewEnqueuer(dispatcher *supportjobs.Dispatcher) *Enqueuer {
	return &Enqueuer{dispatcher: dispatcher}
}

func (e *Enqueuer) EnqueueReply(ctx context.Context, input aireply.ReplyJobInput) error {
	if e == nil || e.dispatcher == nil {
		return aireply.ErrGeneratorUnavailable
	}
	args := GenerateReplyArgs{
		TopicID:          input.TopicID,
		ParentCommentID:  input.ParentCommentID,
		TriggerCommentID: input.TriggerCommentID,
		TriggerUserID:    input.TriggerUserID,
		BotUserID:        input.BotUserID,
	}
	_, err := e.dispatcher.Enqueue(ctx, args, args.EnqueueOptions())
	return err
}
