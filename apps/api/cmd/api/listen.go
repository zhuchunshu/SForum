package main

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// air 热重载会先起新进程再杀旧进程，旧实例尚未释放端口时会出现短暂 EADDRINUSE。
// 开发环境下对「地址占用」做有限次重试，避免无意义的启动失败。
func listenWithAddrInUseRetry(app *fiber.App, addr string, logger *slog.Logger) error {
	const (
		maxAttempts = 20
		retryWait   = 150 * time.Millisecond
	)

	var last error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := app.Listen(addr)
		if err == nil {
			return nil
		}
		last = err
		if !isAddrInUse(err) || attempt == maxAttempts {
			return err
		}
		logger.Warn(
			"api listen address busy, retrying",
			"addr", addr,
			"attempt", attempt,
			"maxAttempts", maxAttempts,
			"error", err,
		)
		time.Sleep(retryWait)
	}
	return last
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	// 覆盖 net 包与部分包装错误文案（Fiber 会再包一层 failed to listen）。
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "address already in use") {
		return true
	}
	if strings.Contains(msg, "only one usage of each socket address") {
		// Windows
		return true
	}
	var errno interface{ Temporary() bool }
	if errors.As(err, &errno) {
		// 保底：部分平台用 Temporary 标记 bind 冲突并不稳定，上面字符串已覆盖主路径。
		_ = errno
	}
	return false
}
