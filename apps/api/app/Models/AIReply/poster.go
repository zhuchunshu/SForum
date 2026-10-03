package aireply

import (
	"context"
	"errors"

	forum "github.com/zhuchunshu/sforum/apps/api/app/Models/Forum"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

var (
	// ErrNotABot 表示试图让一个非机器人账号走生成回复的路径。
	ErrNotABot = errors.New("aireply: only bot accounts may post generated replies")
)

// CommentCreator 是以指定身份创建评论的最小面，由 forum.Service 实现。
type CommentCreator interface {
	CreateComment(ctx context.Context, actor identity.Actor, input forum.CreateCommentInput) (forum.Comment, error)
}

// BotCommentPoster 以机器人身份发布回复。
//
// 它刻意走与普通用户完全相同的写路径：审核决策、提及解析、通知扇出、修订记录
// 一个都不跳过。绕过其中任何一步，机器人都成了发布后门——用户只要在 @ 的内容里
// 塞进广告，就能借机器人把审核拦下的东西发出去。
type BotCommentPoster struct {
	users identity.ActorStore
	forum CommentCreator
}

func NewBotCommentPoster(users identity.ActorStore, creator CommentCreator) *BotCommentPoster {
	return &BotCommentPoster{users: users, forum: creator}
}

func (p *BotCommentPoster) PostBotComment(ctx context.Context, botUserID int64, input BotCommentInput) (int64, error) {
	if p == nil || p.users == nil || p.forum == nil {
		return 0, ErrGeneratorUnavailable
	}
	if botUserID <= 0 || input.TopicID <= 0 {
		return 0, ErrGeneratorUnavailable
	}
	actor, err := p.users.LoadActor(ctx, botUserID)
	if err != nil {
		return 0, err
	}
	// 防御：只有机器人账号能走这条路径。人类账号的发言必须来自它自己的会话，
	// 否则这里会变成一条以他人名义发帖的通道。
	if !actor.Kind.IsBot() {
		return 0, ErrNotABot
	}
	var parentID *int64
	if input.ParentID > 0 {
		parent := input.ParentID
		parentID = &parent
	}
	comment, err := p.forum.CreateComment(ctx, actor, forum.CreateCommentInput{
		TopicID:  input.TopicID,
		ParentID: parentID,
		Content: forum.ContentInput{
			RawContent:   input.Content,
			SourceFormat: "markdown",
			EditorType:   "markdown",
		},
	})
	if err != nil {
		return 0, err
	}
	return comment.ID, nil
}
