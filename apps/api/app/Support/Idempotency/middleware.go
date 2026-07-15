package idempotency

import (
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
)

// ActorResolver 解析当前请求的幂等作用域主体（通常为登录用户 ID）。
// 返回 0 表示匿名；同一 key 在匿名与登录用户之间不共享。
type ActorResolver func(c fiber.Ctx) (int64, error)

// Middleware 仅在请求携带 Idempotency-Key 时生效；缺省不强制。
// 应挂在需要去重的写路由上（如 POST /topics、POST /comments）。
func Middleware(store *Store, resolveActor ActorResolver) fiber.Handler {
	return func(c fiber.Ctx) error {
		key := strings.TrimSpace(c.Get(HeaderName))
		if key == "" || store == nil {
			return c.Next()
		}
		if err := ValidateKey(key); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "idempotency.key_invalid")
		}

		var actorID int64
		if resolveActor != nil {
			id, err := resolveActor(c)
			if err != nil {
				return err
			}
			actorID = id
		}

		// Path() 不含 query，避免 ? 参数破坏作用域。
		storageKey := StorageKey(actorID, c.Method(), c.Path(), key)
		rec, started, conflict, err := store.Begin(c.Context(), storageKey)
		if err != nil {
			// 存储故障时 fail-open：不阻断业务写路径。
			return c.Next()
		}
		if conflict {
			return fiber.NewError(fiber.StatusConflict, "idempotency.in_progress")
		}
		if !started && rec.State == stateCompleted {
			c.Set(ReplayedHeader, "true")
			return c.Status(rec.Status).Send(rec.Body)
		}

		err = c.Next()
		status := c.Response().StatusCode()
		// 仅缓存 2xx 成功响应；客户端可用同一 key 重试 4xx/5xx。
		if err == nil && status >= 200 && status < 300 {
			body := append([]byte(nil), c.Response().Body()...)
			_ = store.Complete(c.Context(), storageKey, status, body)
			return nil
		}
		_ = store.Abort(c.Context(), storageKey)
		return err
	}
}

// ValidateKey 限制可见 ASCII 与长度，避免控制字符。
func ValidateKey(key string) error {
	if key == "" || utf8.RuneCountInString(key) > MaxKeyLength {
		return errInvalidKey
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f {
			return errInvalidKey
		}
	}
	return nil
}

type invalidKeyError struct{}

func (invalidKeyError) Error() string { return "idempotency key invalid" }

var errInvalidKey = invalidKeyError{}
