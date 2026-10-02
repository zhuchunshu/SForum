package aicontroller

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	apphttp "github.com/zhuchunshu/sforum/apps/api/app/Http"
	modelsai "github.com/zhuchunshu/sforum/apps/api/app/Models/AI"
	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
	authsession "github.com/zhuchunshu/sforum/apps/api/app/Support/AuthSession"
	secretstore "github.com/zhuchunshu/sforum/apps/api/app/Support/SecretStore"
)

type Controller struct {
	service  *modelsai.Service
	users    identity.ActorStore
	sessions *authsession.Manager
	secrets  *secretstore.Service
}

func NewController(service *modelsai.Service, users identity.ActorStore, sessions *authsession.Manager, secrets *secretstore.Service) *Controller {
	return &Controller{service: service, users: users, sessions: sessions, secrets: secrets}
}

// updateSettingsRequest 刻意不接受 revision / updatedBy / updatedAt：
// 它们由服务端权威决定，客户端提交只会被忽略或造成假象。
type updateSettingsRequest struct {
	Enabled               bool                         `json:"enabled"`
	Profiles              []supportai.Profile          `json:"profiles"`
	CostClassProfiles     map[string]string            `json:"costClassProfiles"`
	PurposeFailurePosture map[string]string            `json:"purposeFailurePosture"`
	AutoAction            supportai.AutoActionSettings `json:"autoAction"`
	Gates                 supportai.GateSettings       `json:"gates"`
	Redaction             supportai.RedactionSettings  `json:"redaction"`
}

type settingsResponse struct {
	Settings supportai.Settings `json:"settings"`
	// Warnings 是非阻断提示（例如未配置单价导致预算闸门不生效）。
	Warnings []string `json:"warnings"`
	// EncryptionEnabled 表示 Secret Store 是否具备加密能力；为 false 时不应保存真实密钥。
	EncryptionEnabled bool `json:"encryptionEnabled"`
	// CredentialStatus 按 profile id 报告密钥是否已配置。它不含密钥值。
	CredentialStatus map[string]bool `json:"credentialStatus"`
}

// profileCredentialRequest 是密钥录入的唯一载荷。密钥值只在这次请求里存在，
// 写入 Secret Store 后立即丢弃，之后只能通过状态位得知「已配置」。
type profileCredentialRequest struct {
	APIKey string `json:"apiKey"`
}

type profileCredentialResponse struct {
	ProfileID string `json:"profileId"`
	Version   int64  `json:"version"`
}

type usageResponse struct {
	Usage    supportai.UsageByScope `json:"usage"`
	Budget   int64                  `json:"budgetMicros"`
	Warnings []string               `json:"warnings"`
}

type executionsResponse struct {
	Items []supportai.ExecutionRecord `json:"items"`
}

func (h *Controller) getSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.GetSettings(c.Context(), actor)
	if err != nil {
		return mapAIError(err)
	}
	return h.settingsPayload(c, actor, settings)
}

// settingsPayload 统一组装配置响应。密钥状态每次都重新读取，因此刚保存过的
// 密钥会立刻显示为「已配置」。
func (h *Controller) settingsPayload(c fiber.Ctx, actor identity.Actor, settings supportai.Settings) error {
	status, err := h.service.CredentialStatus(c.Context(), actor)
	if err != nil {
		return mapAIError(err)
	}
	return apphttp.OK(c, settingsResponse{
		Settings:          settings,
		Warnings:          settings.Warnings(),
		EncryptionEnabled: h.encryptionEnabled(),
		CredentialStatus:  status,
	})
}

func (h *Controller) putProfileCredential(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	profileID := c.Params("profileId")
	if profileID == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "ai.settings_invalid")
	}
	var request profileCredentialRequest
	if err := c.Bind().JSON(&request); err != nil || request.APIKey == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "ai.credential_required")
	}
	version, err := h.service.SaveProfileCredential(c.Context(), actor, profileID, request.APIKey)
	if err != nil {
		return mapAIError(err)
	}
	return apphttp.OK(c, profileCredentialResponse{ProfileID: profileID, Version: version})
}

func (h *Controller) updateSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var request updateSettingsRequest
	if err := c.Bind().JSON(&request); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "ai.settings_invalid")
	}
	settings := settingsFromRequest(request)
	saved, err := h.service.UpdateSettings(c.Context(), actor, settings)
	if err != nil {
		return mapAIError(err)
	}
	return h.settingsPayload(c, actor, saved)
}

func (h *Controller) resetSettings(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	settings, err := h.service.ResetSettings(c.Context(), actor)
	if err != nil {
		return mapAIError(err)
	}
	return h.settingsPayload(c, actor, settings)
}

func (h *Controller) getUsage(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	usage, err := h.service.Usage(c.Context(), actor, time.Now())
	if err != nil {
		return mapAIError(err)
	}
	response := usageResponse{Usage: usage}
	if settings, settingsErr := h.service.GetSettings(c.Context(), actor); settingsErr == nil {
		response.Budget = settings.Gates.MonthlyBudgetMicros
		response.Warnings = settings.Warnings()
	}
	return apphttp.OK(c, response)
}

// diagnoseRequest 允许空 body：不填就用服务端默认的探测消息。
type diagnoseRequest struct {
	Message string `json:"message"`
}

// diagnose 发一次真实的最小调用，让运营者在配置完成后立刻知道能不能用。
func (h *Controller) diagnose(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	var request diagnoseRequest
	// 空 body 或非 JSON body 一律按「使用默认探测消息」处理。
	_ = c.Bind().JSON(&request)
	result, err := h.service.Diagnose(c.Context(), actor, request.Message)
	if err != nil {
		return mapAIError(err)
	}
	return apphttp.OK(c, result)
}

func (h *Controller) listExecutions(c fiber.Ctx) error {
	actor, err := h.actor(c)
	if err != nil {
		return err
	}
	items, err := h.service.Executions(c.Context(), actor, queryInt(c, "limit"))
	if err != nil {
		return mapAIError(err)
	}
	return apphttp.OK(c, executionsResponse{Items: items})
}

func (h *Controller) actor(c fiber.Ctx) (identity.Actor, error) {
	return apphttp.LoadActor(c, h.sessions, h.users)
}

// encryptionEnabled 让管理界面能提示「当前部署没有加密密钥」。没有加密能力时
// 保存 provider 密钥会失败，这必须在界面上说明，而不是等用户保存后才报错。
func (h *Controller) encryptionEnabled() bool {
	if h.secrets == nil {
		return false
	}
	return h.secrets.EncryptionEnabled()
}

// settingsFromRequest 把控制台提交的载荷组装成完整配置。Normalized 负责补齐结构
// 默认值，因此「用户只改了开关」这类最小提交也能通过校验。
func settingsFromRequest(request updateSettingsRequest) supportai.Settings {
	return supportai.Settings{
		Enabled:               request.Enabled,
		Profiles:              request.Profiles,
		CostClassProfiles:     request.CostClassProfiles,
		PurposeFailurePosture: request.PurposeFailurePosture,
		AutoAction:            request.AutoAction,
		Gates:                 request.Gates,
		Redaction:             request.Redaction,
	}.Normalized()
}

func mapAIError(err error) error {
	switch {
	case errors.Is(err, identity.ErrPermissionDenied):
		return fiber.NewError(fiber.StatusForbidden, "permission.denied")
	case errors.Is(err, supportai.ErrSettingsInvalid), errors.Is(err, supportai.ErrProfileInvalid):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "ai.settings_invalid")
	case errors.Is(err, supportai.ErrProfileUnavailable):
		return fiber.NewError(fiber.StatusNotFound, "ai.profile_not_found")
	case errors.Is(err, supportai.ErrCredentialMissing):
		return fiber.NewError(fiber.StatusUnprocessableEntity, "ai.credential_required")
	default:
		return err
	}
}

func queryInt(c fiber.Ctx, key string) int {
	value := c.Query(key)
	if value == "" {
		return 0
	}
	parsed := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		parsed = parsed*10 + int(r-'0')
		if parsed > 1000 {
			return 1000
		}
	}
	return parsed
}
