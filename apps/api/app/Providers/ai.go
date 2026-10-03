package providers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	aicontroller "github.com/zhuchunshu/sforum/apps/api/app/Http/Controllers/AI"
	modelsai "github.com/zhuchunshu/sforum/apps/api/app/Models/AI"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	secretstore "github.com/zhuchunshu/sforum/apps/api/app/Support/SecretStore"
)

// identityStore 是 AI 控制台需要的身份能力：读取当前操作者，以及解析充当 AI
// 助手的机器人账号。两者都由 identity.PostgresStore 提供。
type identityStore interface {
	identity.ActorStore
	modelsai.BotAccountReader
}

type AIProvider struct {
	controller *aicontroller.Controller
	service    *modelsai.Service
}

// NewAIProvider 装配 AI 网关：配置与用量存储、Secret Store 凭证解析、受控出站
// 执行，以及权限感知的服务层。网关默认关闭：未显式启用前不产生任何出站调用。
func NewAIProvider(pool *pgxpool.Pool, secrets *secretstore.Service, users identityStore, sessions *authsession.Manager) *AIProvider {
	store := supportai.NewPostgresStore(pool)
	gateway := supportai.NewGateway(supportai.GatewayConfig{
		Settings: store,
		Usage:    store,
		Traces:   store,
		Creds:    supportai.NewSecretStoreResolver(secrets),
		Invoker:  supportai.NewHTTPInvoker(supportai.HTTPInvokerOptions{}),
	})
	service := modelsai.NewService(modelsai.Config{
		Settings:    store,
		Gateway:     gateway,
		Usage:       store,
		Traces:      store,
		Credentials: supportai.NewCredentialWriter(secrets),
		BotAccounts: users,
	})
	return &AIProvider{controller: aicontroller.NewController(service, users, sessions, secrets), service: service}
}

// Service 暴露服务层，供其他 Provider 把自己的 AI 用途接到同一网关上。
func (p *AIProvider) Service() *modelsai.Service { return p.service }

// Gateway 暴露底层网关，供回复生成等用途复用同一个装配好的实例。
func (p *AIProvider) Gateway() *supportai.Gateway {
	if p == nil || p.service == nil {
		return nil
	}
	return p.service.Gateway()
}

func (p *AIProvider) RegisterRoutes(api fiber.Router) { p.controller.RegisterRoutes(api) }
