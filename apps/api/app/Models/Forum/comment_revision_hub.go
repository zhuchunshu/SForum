package forum

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CommentRevisionHub 与 Notifications.RevisionHub 同构：每个 API 进程一条可重连的
// PostgreSQL LISTEN 连接；NOTIFY 只做唤醒，订阅者每次唤醒后重新读取 topics.comment_revision
// 这一持久事实，因此连接抖动、通知合并或丢失都不会让信号永久错误。
//
// 与通知 hub 的差别只在分桶维度：评论流是公开读取，按 topicID 分桶（而非按收件人），
// 并额外限制单客户端（IP）连接数，避免匿名读者把连接预算吃光。
type CommentRevisionHub struct {
	ctx    context.Context
	cancel context.CancelFunc
	pool   *pgxpool.Pool

	mu       sync.Mutex
	nextID   uint64
	total    int
	byTopic  map[int64]map[uint64]chan struct{}
	byClient map[string]int

	done      chan struct{}
	closeOnce sync.Once
}

const (
	commentRevisionChannel = "sforum_forum_comment_revision"

	maxCommentStreamConnections          = 1024
	maxCommentStreamConnectionsPerTopic  = 256
	maxCommentStreamConnectionsPerClient = 8
)

var (
	ErrCommentRevisionWakeUnavailable = errors.New("forum: comment revision wake source unavailable")
	ErrCommentRevisionConnectionLimit = errors.New("forum: comment revision connection limit reached")
)

// NewCommentRevisionHub 启动 LISTEN 循环。ctx 结束时循环退出并释放专用连接，
// 因此调用方只需在进程关闭前取消 ctx（无需额外 Close 顺序约束）。
func NewCommentRevisionHub(ctx context.Context, pool *pgxpool.Pool) *CommentRevisionHub {
	if ctx == nil {
		ctx = context.Background()
	}
	listenCtx, cancel := context.WithCancel(ctx)
	hub := &CommentRevisionHub{
		ctx: listenCtx, cancel: cancel, pool: pool,
		byTopic: make(map[int64]map[uint64]chan struct{}), byClient: make(map[string]int),
		done: make(chan struct{}),
	}
	go hub.listen()
	return hub
}

// Close 取消 LISTEN 并等待循环退出；可在测试或显式关停路径调用。
func (h *CommentRevisionHub) Close() {
	if h == nil {
		return
	}
	h.closeOnce.Do(func() {
		if h.cancel != nil {
			h.cancel()
		}
		if h.done != nil {
			<-h.done
		}
	})
}

// Subscribe 订阅某个主题的评论修订唤醒。返回的 release 必须恰好调用一次。
func (h *CommentRevisionHub) Subscribe(topicID int64, clientKey string) (<-chan struct{}, func(), error) {
	if h == nil || h.pool == nil || topicID <= 0 {
		return nil, nil, ErrCommentRevisionWakeUnavailable
	}
	if clientKey == "" {
		clientKey = "unknown"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.total >= maxCommentStreamConnections ||
		len(h.byTopic[topicID]) >= maxCommentStreamConnectionsPerTopic ||
		h.byClient[clientKey] >= maxCommentStreamConnectionsPerClient {
		return nil, nil, ErrCommentRevisionConnectionLimit
	}
	h.nextID++
	id := h.nextID
	ch := make(chan struct{}, 1)
	if h.byTopic[topicID] == nil {
		h.byTopic[topicID] = make(map[uint64]chan struct{})
	}
	h.byTopic[topicID][id] = ch
	h.byClient[clientKey]++
	h.total++
	var once sync.Once
	release := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if subscribers := h.byTopic[topicID]; subscribers != nil {
				if _, ok := subscribers[id]; ok {
					delete(subscribers, id)
					h.total--
				}
				if len(subscribers) == 0 {
					delete(h.byTopic, topicID)
				}
			}
			if h.byClient[clientKey] > 0 {
				h.byClient[clientKey]--
			}
			if h.byClient[clientKey] == 0 {
				delete(h.byClient, clientKey)
			}
		})
	}
	return ch, release, nil
}

func (h *CommentRevisionHub) publish(topicID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.byTopic[topicID] {
		select {
		case ch <- struct{}{}:
		default:
			// 合并重复唤醒：持久修订号会保留全部状态变化。
		}
	}
}

func (h *CommentRevisionHub) listen() {
	defer close(h.done)
	backoff := 100 * time.Millisecond
	for h.ctx.Err() == nil {
		conn, err := h.pool.Acquire(h.ctx)
		if err != nil {
			h.waitBackoff(backoff)
			backoff = min(backoff*2, 5*time.Second)
			continue
		}
		_, err = conn.Exec(h.ctx, `LISTEN `+commentRevisionChannel)
		if err == nil {
			backoff = 100 * time.Millisecond
			for h.ctx.Err() == nil {
				notification, waitErr := conn.Conn().WaitForNotification(h.ctx)
				if waitErr != nil {
					err = waitErr
					break
				}
				topicID, parseErr := strconv.ParseInt(notification.Payload, 10, 64)
				if parseErr == nil && topicID > 0 {
					h.publish(topicID)
				}
			}
		}
		conn.Release()
		if h.ctx.Err() == nil {
			h.waitBackoff(backoff)
			backoff = min(backoff*2, 5*time.Second)
		}
	}
}

func (h *CommentRevisionHub) waitBackoff(delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-h.ctx.Done():
	case <-timer.C:
	}
}
