package notifications

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRevisionWakeUnavailable = errors.New("notifications: revision wake source unavailable")
	ErrRevisionConnectionLimit = errors.New("notifications: revision connection limit reached")
)

const (
	maxRevisionConnections        = 512
	maxRevisionConnectionsPerUser = 4
)

// RevisionHub uses one reconnecting PostgreSQL LISTEN connection per API
// process. NOTIFY only wakes subscribers; every subscriber re-reads its durable
// revision before sending an SSE signal.
type RevisionHub struct {
	ctx    context.Context
	pool   *pgxpool.Pool
	mu     sync.Mutex
	nextID uint64
	total  int
	byUser map[int64]map[uint64]chan struct{}
}

func NewRevisionHub(ctx context.Context, pool *pgxpool.Pool) *RevisionHub {
	if ctx == nil {
		ctx = context.Background()
	}
	hub := &RevisionHub{ctx: ctx, pool: pool, byUser: make(map[int64]map[uint64]chan struct{})}
	go hub.listen()
	return hub
}

func (h *RevisionHub) Subscribe(userID int64) (<-chan struct{}, func(), error) {
	if h == nil || h.pool == nil || userID <= 0 {
		return nil, nil, ErrRevisionWakeUnavailable
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.total >= maxRevisionConnections || len(h.byUser[userID]) >= maxRevisionConnectionsPerUser {
		return nil, nil, ErrRevisionConnectionLimit
	}
	h.nextID++
	id := h.nextID
	ch := make(chan struct{}, 1)
	if h.byUser[userID] == nil {
		h.byUser[userID] = make(map[uint64]chan struct{})
	}
	h.byUser[userID][id] = ch
	h.total++
	var once sync.Once
	release := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if subscribers := h.byUser[userID]; subscribers != nil {
				if _, ok := subscribers[id]; ok {
					delete(subscribers, id)
					h.total--
				}
				if len(subscribers) == 0 {
					delete(h.byUser, userID)
				}
			}
		})
	}
	return ch, release, nil
}

func (h *RevisionHub) publish(userID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.byUser[userID] {
		select {
		case ch <- struct{}{}:
		default:
			// 合并重复 wake；durable revision 会保留全部状态变化。
		}
	}
}

func (h *RevisionHub) listen() {
	backoff := 100 * time.Millisecond
	for h.ctx.Err() == nil {
		conn, err := h.pool.Acquire(h.ctx)
		if err != nil {
			h.waitBackoff(backoff)
			backoff = min(backoff*2, 5*time.Second)
			continue
		}
		_, err = conn.Exec(h.ctx, `LISTEN sforum_notification_revision`)
		if err == nil {
			backoff = 100 * time.Millisecond
			for h.ctx.Err() == nil {
				notification, waitErr := conn.Conn().WaitForNotification(h.ctx)
				if waitErr != nil {
					err = waitErr
					break
				}
				userID, parseErr := strconv.ParseInt(notification.Payload, 10, 64)
				if parseErr == nil && userID > 0 {
					h.publish(userID)
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

func (h *RevisionHub) waitBackoff(delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-h.ctx.Done():
	case <-timer.C:
	}
}
