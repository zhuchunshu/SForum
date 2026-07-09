package http

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type healthResponse struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	Environment      string    `json:"environment"`
	Locale           string    `json:"locale"`
	SupportedLocales []string  `json:"supportedLocales"`
	Time             time.Time `json:"time"`
}

type RouteProvider interface {
	RegisterRoutes(api fiber.Router)
}

type Dependencies struct {
	RouteProviders []RouteProvider
	Options        *options.Service
	// Storage 用于分布式限流。为 nil 时 limiter 退化为进程内存限流。
	Storage fiber.Storage
}

func NewApp(cfg config.Config, logger *slog.Logger, deps Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler(logger),
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
		BodyLimit:    cfg.HTTPBodyLimit,
	})
	app.Hooks().OnPreStartupMessage(func(sm *fiber.PreStartupMessageData) error {
		sm.BannerHeader = sforumStartupBanner
		return nil
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	// 响应压缩：根据 Accept-Encoding 自动 brotli/gzip，降低带宽占用。
	app.Use(compress.New(compress.Config{Level: compress.Level(cfg.CompressLevel)}))
	// 写接口限流：跳过 GET/HEAD/OPTIONS，按 IP 限制单位时间内的写操作次数。
	// Storage 注入时为分布式限流（多实例共享），否则退化为进程内存限流。
	if cfg.LimiterWriteMax > 0 && cfg.LimiterWindow > 0 {
		app.Use(limiter.New(limiter.Config{
			Storage:    deps.Storage,
			Max:        cfg.LimiterWriteMax,
			Expiration: cfg.LimiterWindow,
			Next: func(c fiber.Ctx) bool {
				// 只限写方法，读请求直接放行（读路径已有缓存挡）。
				switch c.Method() {
				case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
					return true
				}
				return false
			},
		}))
	}
	app.Use(localeMiddleware(cfg, deps.Options))

	registerRoutes(app, cfg, deps)

	return app
}

func registerRoutes(app *fiber.App, cfg config.Config, deps Dependencies) {
	api := app.Group("/api/v1")

	// CSRF 防护：double-submit cookie + Origin/Referer 校验，保护所有 unsafe 方法。
	// 注册在 /api/v1 group 上，使 GET（如 /auth/session）也能种下可读的 csrf_ cookie，
	// 供 SPA 读取后随 unsafe 请求回传 X-Csrf-Token header。
	// TrustedOrigins 必须包含公开站点 origin：API 在反向代理后看到的 Host 是内部地址，
	// 而 Origin 是公开站点，二者不匹配会被默认拒绝。默认从 APP_URL 派生。
	// CSRF 防护仅在配置启用时生效；测试场景通过 CSRF_ENABLED=false 显式关闭，
	// 避免每个测试请求都要携带 token。生产默认启用。
	if cfg.CSRFEnabled {
		api.Use(csrf.New(csrf.Config{
			Storage:         deps.Storage,
			CookieSameSite:  fiber.CookieSameSiteLaxMode,
			CookieSecure:    strings.EqualFold(cfg.AppEnv, "production"),
			CookieHTTPOnly:  false, // SPA 必须能读取 csrf_ cookie 以回传 token
			CookiePath:      "/",
			TrustedOrigins:  cfg.CSRFTrustedOrigins,
			ErrorHandler:    csrfErrorHandler,
		}))
	}

	for _, provider := range deps.RouteProviders {
		if provider != nil {
			provider.RegisterRoutes(api)
		}
	}

	api.Get("/health", func(c fiber.Ctx) error {
		settings := runtimeSettings(c.Context(), cfg, deps.Options)
		return OK(c, healthResponse{
			Name:             settings.SiteName,
			Status:           "ok",
			Environment:      cfg.AppEnv,
			Locale:           settings.DefaultLocale,
			SupportedLocales: settings.SupportedLocales,
			Time:             time.Now().UTC(),
		})
	})
}

func runtimeSettings(ctx context.Context, cfg config.Config, service *options.Service) options.RuntimeSettings {
	if service != nil {
		if settings, err := service.RuntimeSettings(ctx); err == nil {
			return settings
		}
	}
	return fallbackRuntimeSettings(cfg)
}

func fallbackRuntimeSettings(cfg config.Config) options.RuntimeSettings {
	return options.RuntimeSettings{
		SiteName:                  cfg.AppName,
		SiteURL:                   "",
		DefaultLocale:             cfg.AppLocale,
		SupportedLocales:          cfg.SupportedLocales,
		HumanVerificationProvider: cfg.HumanVerificationProvider,
	}
}

// csrfErrorHandler 把 CSRF 中间件的错误映射为统一 envelope reason。
// token 缺失/不匹配 → csrf.invalid；Origin/Referer 不匹配 → csrf.origin_invalid。
// 返回 *APIError 由全局 errorHandler 渲染成 {code,message,data:{reason}} 结构。
func csrfErrorHandler(_ fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, csrf.ErrOriginNoMatch),
		errors.Is(err, csrf.ErrOriginInvalid),
		errors.Is(err, csrf.ErrRefererNoMatch),
		errors.Is(err, csrf.ErrRefererInvalid),
		errors.Is(err, csrf.ErrRefererNotFound):
		return NewError(fiber.StatusForbidden, "csrf.origin_invalid")
	default:
		return NewError(fiber.StatusForbidden, "csrf.invalid")
	}
}

const sforumStartupBanner = `
   _____ ______
  / ___// ____/___  _______  ______ ___
  \__ \/ /_  / __ \/ ___/ / / / _  _  \
 ___/ / __/ / /_/ / /  / /_/ / / / / / /
/____/_/    \____/_/   \__,_/_/ /_/ /_/    SForum API
--------------------------------------------------`
