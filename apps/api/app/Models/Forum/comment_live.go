package forum

import (
	"context"
	"errors"
	"time"
)

// 评论区实时信号错误码（对外 API 契约的一部分）。
const (
	// CodeCommentStreamLimited 连接预算耗尽：客户端应退回轮询而非重试轰炸。
	CodeCommentStreamLimited = "forum.comment_stream_limited"
	// CodeCommentStreamUnavailable 宿主未装配评论修订源。
	CodeCommentStreamUnavailable = "forum.comment_stream_unavailable"
)

// ErrCommentRevisionUnavailable 表示当前 store 组合没有提供修订号能力。
var ErrCommentRevisionUnavailable = errors.New("forum: comment revision unavailable")

// CommentRevisionState 是公开评论区的修订状态：SSE 只推它，正文永远走 ListComments。
//   - Revision 单调递增，任何公开可见变化 +1；仅作「变了」的信号。
//   - CommentCount 与公开列表 total 同源（topics.comment_count）。
//   - LastCommentID/LastCommentCreatedAt 是当前末尾评论，可直接作为 flat keyset 的续页锚点。
type CommentRevisionState struct {
	Revision             int64      `json:"revision"`
	CommentCount         int64      `json:"commentCount"`
	LastCommentID        int64      `json:"lastCommentId,omitempty"`
	LastCommentCreatedAt *time.Time `json:"lastCommentCreatedAt,omitempty"`
}

// CommentRevisionStore 是 store 的可选能力：读取主题级评论修订号。
// 与 forum.Store 分开声明，避免为此给所有测试替身补方法；
// 生产组合（PostgresStore / CachedStore）通过编译期断言保证实现。
type CommentRevisionStore interface {
	CommentRevisionState(ctx context.Context, topicID int64) (CommentRevisionState, error)
}

// CommentLiveService 是评论区实时信号的最小协作者：公开可见性规则复用 store.GetTopic，
// 修订号读取委托给 store 能力，唤醒来源是进程内 CommentRevisionHub（可为 nil）。
type CommentLiveService struct {
	store    Store
	revision CommentRevisionStore
	hub      *CommentRevisionHub
}

func NewCommentLiveService(store Store) *CommentLiveService {
	service := &CommentLiveService{store: store}
	if capability, ok := store.(CommentRevisionStore); ok {
		service.revision = capability
	}
	return service
}

// WithWakeHub 注入进程级 LISTEN 唤醒源；nil 时退化为仅靠对账 tick 的轮询式 SSE。
func (s *CommentLiveService) WithWakeHub(hub *CommentRevisionHub) *CommentLiveService {
	if s != nil {
		s.hub = hub
	}
	return s
}

// RevisionState 读取主题修订状态。不可见（隐藏/删除/非公开分类）统一 ErrTopicNotFound，
// 与 ListComments / ResolveCommentPage 同口径，避免成为隐藏主题的探测通道。
func (s *CommentLiveService) RevisionState(ctx context.Context, topicID int64) (CommentRevisionState, error) {
	if s == nil || s.store == nil || s.revision == nil || topicID <= 0 {
		return CommentRevisionState{}, ErrCommentRevisionUnavailable
	}
	if _, err := s.store.GetTopic(ctx, topicID); err != nil {
		return CommentRevisionState{}, err
	}
	return s.revision.CommentRevisionState(ctx, topicID)
}

// Subscribe 订阅主题评论修订唤醒。没有唤醒源时返回 nil channel（调用方仍靠对账 tick 收敛）。
func (s *CommentLiveService) Subscribe(topicID int64, clientKey string) (<-chan struct{}, func(), error) {
	if s == nil || s.hub == nil {
		return nil, nil, ErrCommentRevisionWakeUnavailable
	}
	return s.hub.Subscribe(topicID, clientKey)
}
