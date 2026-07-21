package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	extensions "github.com/zhuchunshu/sforum/apps/api/app/Models/Extensions"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	apilts "github.com/zhuchunshu/sforum/apps/api/app/Support/APILTS"
	avatar "github.com/zhuchunshu/sforum/apps/api/app/Support/Avatar"
	appevents "github.com/zhuchunshu/sforum/apps/api/app/Support/Events"
	extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"
	hostapi "github.com/zhuchunshu/sforum/apps/api/app/Support/HostAPI"
	humanverify "github.com/zhuchunshu/sforum/apps/api/app/Support/HumanVerify"
	pages "github.com/zhuchunshu/sforum/apps/api/app/Support/Pages"
	seoregistry "github.com/zhuchunshu/sforum/apps/api/app/Support/SEORegistry"
	"github.com/zhuchunshu/sforum/apps/api/config"
)

type API struct {
	App  *fiber.App
	Addr string
	SEO  *seoregistry.ExecutionRuntime

	closeOnce sync.Once
	close     func()
	failures  <-chan error
}

type extensionRuntime interface {
	extensions.RuntimeManager
	appevents.Publisher
	RouteTarget(extensionID string) (extensionsruntime.RouteTarget, bool)
	AcquireActiveRuntimeCall(context.Context, string, extensionsruntime.RuntimeCallClass) (extensionsruntime.RuntimeInstanceSnapshot, *extensionsruntime.RuntimeAdmissionLease, error)
	AcquireRuntimeCall(context.Context, extensionsruntime.RuntimeInstanceIdentity, extensionsruntime.RuntimeCallClass) (*extensionsruntime.RuntimeAdmissionLease, error)
	AdminSurfaceSnapshot(string) extensionsruntime.AdminSurfaceRegistrySnapshot
	ResolveAdminSurface(string) (extensionsruntime.AdminSurfaceContract, error)
	InvokeAdminSurface(context.Context, extensionsruntime.AdminSurfaceInvocation) (extensionsruntime.AdminSurfaceInvocationResult, error)
	Reconcile(ctx context.Context, items []extensions.Extension)
	Close(ctx context.Context)
	// SendMail 供 embed worker 的 mail.deliver 复用同一 runtime（P0 共享插件进程）。
	SendMail(ctx context.Context, extensionID string, request extensionsruntime.MailProviderRequest) (extensionsruntime.MailProviderResponse, error)
	// 附件存储槽 RPC（E6.2）；与 StorageRuntime 对齐。
	extensionsruntime.StorageRuntime
	protocolV2ProviderBrokerSource
}

var newExtensionRuntimeManager = func(
	store extensions.Store,
	hostAPI extensionsruntime.HostAPIRegistrar,
	settings extensionsruntime.PluginSettings,
	trust extensionsruntime.RuntimeTrustSource,
	databaseLeases extensionsruntime.RuntimeDatabaseLeaseRegistry,
) extensionRuntime {
	if settings == nil {
		settings = store
	}
	return extensionsruntime.NewManager(extensionsruntime.ManagerConfig{
		Starter: extensionsruntime.NewProtocolStarter(extensionsruntime.ProtocolStarterConfig{
			Settings:       settings,
			HostAPI:        hostAPI,
			Trust:          trust,
			DatabaseLeases: databaseLeases,
			// 生产 V1 net/rpc 流量写入 process-local APILTS，供 LTS 删除门禁证明。
			ShimTelemetry: apilts.Process(),
		}),
		DeliveryStore: store,
	})
}

func bindAPIExtensionRuntime(
	store extensions.Store,
	hostAPI extensionsruntime.HostAPIRegistrar,
	service *extensions.Service,
	trust extensionsruntime.RuntimeTrustSource,
	databaseLeases extensionsruntime.RuntimeDatabaseLeaseRegistry,
) extensionRuntime {
	runtime := newExtensionRuntimeManager(store, hostAPI, service, trust, databaseLeases)
	extensions.WithRuntimeManager(runtime)(service)
	return runtime
}

func shouldEmbedWorkerInAPI(cfg config.Config) bool {
	return cfg.EmbedWorkerInAPI
}

func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	// 分阶段装配：foundation → extension platform → domain/HTTP/worker。
	// 各阶段在失败时自行关闭已获得的资源；成功后由 API.close 统一收尾。
	return buildProductionAPI(ctx, cfg, logger)
}

func secureSessionID() string {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		panic(fmt.Errorf("generate session id: %w", err))
	}
	return base64.RawURLEncoding.EncodeToString(token[:])
}

// shouldUseSecureCookie 委托 config 包，与 CSRF CookieSecure 共用同一判定。
func shouldUseSecureCookie(cfg config.Config) bool {
	return config.ShouldUseSecureCookie(cfg)
}

func (api *API) Close() {
	if api == nil {
		return
	}
	api.closeOnce.Do(func() {
		if api.close != nil {
			api.close()
		}
	})
}

// Failures reports terminal host-owned runtime failures that require process
// termination. Safe Mode and test instances without a coordinator return nil.
func (api *API) Failures() <-chan error {
	if api == nil {
		return nil
	}
	return api.failures
}

// identityPermissionAdapter 将 identity store 接到 Host API 权限检查。
type identityPermissionAdapter struct {
	store *identity.PostgresStore
}

func (a identityPermissionAdapter) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	if a.store == nil {
		return false, fmt.Errorf("identity store unavailable")
	}
	actor, err := a.store.LoadActor(ctx, userID)
	if err != nil {
		return false, err
	}
	return actor.Can(permission), nil
}

// identityUserAdapter 提供安全用户字段给 Host API。
// 扩展用户字段走 Host-owned value store 的实时权限与 Schema 校验。
type identityUserAdapter struct {
	store  *identity.PostgresStore
	fields identity.SafeUserFieldReader
}

func (a identityUserAdapter) GetUserSafe(
	ctx context.Context,
	userID int64,
	actorUserID int64,
	declaredFields []string,
) (map[string]any, error) {
	if a.store == nil {
		return nil, fmt.Errorf("identity store unavailable")
	}
	user, err := a.store.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, hostapi.ErrNotFound
	}
	projected, err := identity.ProjectSafeUser(ctx, user, actorUserID, declaredFields, a.fields)
	if err != nil {
		return nil, err
	}
	return projected, nil
}

func apiAddress(cfg config.Config) string {
	return fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort)
}

func newHumanVerifyService(cfg config.Config, store humanverify.Store) (*humanverify.Service, error) {
	return humanverify.NewConfiguredService(humanVerifyConfigFromConfig(cfg), store)
}

type avatarOptionsAdapter struct {
	options *options.Service
}

func (a avatarOptionsAdapter) AvatarOptions(ctx context.Context) (avatar.Options, error) {
	if a.options == nil {
		return avatar.DefaultOptions(), nil
	}
	resolved, err := a.options.AvatarOptions(ctx)
	if err != nil {
		return avatar.Options{}, err
	}
	return avatar.Options{
		AllowUpload:           resolved.AllowUpload,
		DefaultProvider:       resolved.DefaultProvider,
		GravatarBaseURL:       resolved.GravatarBaseURL,
		GravatarHashAlgorithm: resolved.GravatarHashAlgorithm,
		DefaultStaticURL:      resolved.DefaultStaticURL,
		MaxSizeKB:             resolved.MaxSizeKB,
		MaxDimension:          resolved.MaxDimension,
		AllowGIF:              resolved.AllowGIF,
		CompressEnabled:       resolved.CompressEnabled,
		TargetDimension:       resolved.TargetDimension,
		CompressQuality:       resolved.CompressQuality,
	}, nil
}

func optionsDefaultsFromConfig(cfg config.Config) options.Defaults {
	return options.Defaults{
		SiteName:                  cfg.AppName,
		SiteURL:                   cfg.AppURL,
		DefaultLocale:             cfg.AppLocale,
		SupportedLocales:          cfg.SupportedLocales,
		HumanVerificationProvider: cfg.HumanVerificationProvider,
		AltchaSecret:              cfg.AltchaSecret,
		AltchaChallengeTTL:        cfg.AltchaChallengeTTL,
		AltchaCost:                cfg.AltchaCost,
	}
}

func humanVerifyConfigFromConfig(cfg config.Config) humanverify.RuntimeConfig {
	return humanverify.RuntimeConfig{
		Provider:        cfg.HumanVerificationProvider,
		AltchaSecret:    cfg.AltchaSecret,
		AltchaTTL:       cfg.AltchaChallengeTTL,
		AltchaCost:      cfg.AltchaCost,
		RateLimit:       60,
		RateLimitWindow: time.Minute,
	}
}

// pageRouteTargetAdapter 为 PageDataLoader 绑定 exact runtime admission lease。
type pageRouteTargetAdapter struct {
	runtime extensionRuntime
}

func (a pageRouteTargetAdapter) AcquireRouteTarget(ctx context.Context, artifact pages.RuntimeArtifact) (pages.LoaderRouteTarget, bool) {
	if a.runtime == nil {
		return pages.LoaderRouteTarget{}, false
	}
	target, admission, err := a.runtime.AcquireActiveRuntimeCall(ctx, artifact.ExtensionID, extensionsruntime.RuntimeCallPage)
	if err != nil {
		return pages.LoaderRouteTarget{}, false
	}
	if artifact.ExtensionVersion != "" && target.ExtensionVersion != artifact.ExtensionVersion ||
		artifact.PackageDigest != "" && target.ArtifactDigest != artifact.PackageDigest ||
		artifact.RuntimeInstanceID != "" && target.Identity.InstanceID != artifact.RuntimeInstanceID {
		admission.Release()
		return pages.LoaderRouteTarget{}, false
	}
	if strings.TrimSpace(target.Target.BaseURL) == "" {
		admission.Release()
		return pages.LoaderRouteTarget{}, false
	}
	return pages.LoaderRouteTarget{
		BaseURL: target.Target.BaseURL,
		Context: admission.Context,
		Release: admission.Release,
	}, true
}

// pagePackageRootAdapter 解析扩展包内容根（loader schema 文件）。
type pagePackageRootAdapter struct {
	store extensions.Store
}

func (a pagePackageRootAdapter) PackageRoot(extensionID string) (string, bool) {
	if a.store == nil {
		return "", false
	}
	item, err := a.store.Get(context.Background(), extensionID)
	if err != nil {
		return "", false
	}
	root := extensions.PackageContentRoot(item)
	if root == "" {
		return "", false
	}
	return root, true
}
