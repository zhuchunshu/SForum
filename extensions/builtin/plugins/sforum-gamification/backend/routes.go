package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 管理面与公开面路由声明 ID（与 manifest routes 一一对应）。
const (
	routeAdminBadges        = "sforum.gamification.route.admin.badges"
	routeAdminBadgeBySlug   = "sforum.gamification.route.admin.badge"
	routeAdminGrants        = "sforum.gamification.route.admin.grants"
	routeAdminGrantByUser   = "sforum.gamification.route.admin.grant"
	routeAdminGrantLog      = "sforum.gamification.route.admin.grant-log"
	routeAdminLevels        = "sforum.gamification.route.admin.levels"
	routeAdminSeedDefaults  = "sforum.gamification.route.admin.seed-defaults"
	routeAdminBackfill      = "sforum.gamification.route.admin.backfill"
	routeProfileAchievements = "sforum.gamification.route.profile.achievements"

	gamificationPermission = "sforum.gamification.manage"
)

// 成长值设置键（manifest settings schema 同步）。
const (
	settingPointsTopic       = "points_topic_created"
	settingPointsComment     = "points_comment_created"
	settingPointsRegister    = "points_registration"
	settingEnabledInProfile  = "enabled_in_profile"
	settingCelebrationEnabled = "celebration_enabled"
)

// gamificationRoutes 暴露插件路由。授权由 Host core.guard.permission 完成；
// 这里只做 handler 分发与二次兜底校验。
type gamificationRoutes struct {
	pluginv2.Server
	engine *Engine
}

// InvokeRoute 分发已声明路由。
func (p *gamificationPluginV2) InvokeRoute(ctx context.Context, request *pluginwire.RouteRequest) (*pluginwire.RouteResponse, error) {
	response := &pluginwire.RouteResponse{
		Context: routeResponseContext(request.GetContext()),
	}
	actor := request.GetContext().GetActor()
	var actorID int64
	if actor != nil {
		actorID = actor.GetUserId()
	}
	body := map[string]any{}
	if request.GetBody() != nil && request.GetBody().GetValue() != nil {
		body = request.GetBody().GetValue().AsMap()
	}
	status, payload, err := p.dispatchAdminRoute(ctx, request.GetRouteId(), request.GetMethod(),
		request.GetPathParameters(), body, actorID)
	// 响应 schema 必须逐条对应 manifest 声明（RouteRequest 不携带它）。
	responseSchema := routeResponseSchema(request.GetRouteId())
	if responseSchema == "" {
		responseSchema = "sforum.gamification.route.admin.action.response@1"
	}
	if err != nil {
		response.Error = &protocolwire.ErrorDetail{
			Code: protocolwire.ErrorCode_ERROR_CODE_INTERNAL, Reason: "gamification.route_failed", Message: err.Error(),
		}
		status = http.StatusInternalServerError
	}
	if payload == nil {
		payload = map[string]any{"ok": status < http.StatusBadRequest}
	}
	document, err := pluginv2.NewTypedDocument(responseSchema, payload)
	if err != nil {
		return nil, err
	}
	response.StatusCode = uint32(status)
	response.Body = document
	return response, nil
}

func (p *gamificationPluginV2) dispatchAdminRoute(ctx context.Context, routeID, method string,
	pathParams map[string]string, body map[string]any, actorID int64,
) (int, map[string]any, error) {
	engine := p.engine
	switch routeID {
	case routeAdminBadges:
		switch method {
		case http.MethodGet:
			badges, err := engine.Store.ListBadges(ctx, true)
			if err != nil {
				return 0, nil, err
			}
			encoded, err := json.Marshal(badgesForWire(badges))
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"badges": string(encoded)}, nil
		case http.MethodPost:
			def, err := badgeFromWire(body)
			if err != nil {
				return http.StatusUnprocessableEntity, map[string]any{"reason": "gamification.badge_invalid", "message": err.Error()}, nil
			}
			if err := engine.Store.UpsertBadge(ctx, def); err != nil {
				return 0, nil, err
			}
			return http.StatusCreated, map[string]any{"slug": def.Slug}, nil
		}
	case routeAdminBadgeBySlug:
		slug := strings.TrimSpace(pathParams["slug"])
		if slug == "" {
			return http.StatusBadRequest, map[string]any{"reason": "gamification.slug_required"}, nil
		}
		switch method {
		case http.MethodDelete:
			// 停用不删除：历史授予保留。
			if err := engine.Store.DisableBadge(ctx, slug); err != nil {
				return http.StatusNotFound, map[string]any{"reason": "gamification.badge_not_found"}, nil
			}
			return http.StatusOK, map[string]any{"slug": slug, "disabled": true}, nil
		case http.MethodPatch:
			def, err := engine.Store.GetBadge(ctx, slug)
			if err != nil {
				return http.StatusNotFound, map[string]any{"reason": "gamification.badge_not_found"}, nil
			}
			updated, err := badgeFromWire(mergeBadgePatch(def, body))
			if err != nil {
				return http.StatusUnprocessableEntity, map[string]any{"reason": "gamification.badge_invalid", "message": err.Error()}, nil
			}
			updated.Slug = slug
			if err := engine.Store.UpsertBadge(ctx, updated); err != nil {
				return 0, nil, err
			}
			return http.StatusOK, map[string]any{"slug": slug}, nil
		}
	case routeAdminGrants:
		if method != http.MethodPost {
			break
		}
		// username 由 Host 侧解析为 user id 后注入；这里接受 userId。
		userID := int64FromAny(body["userId"])
		slug := strings.TrimSpace(stringValue(body["badgeSlug"]))
		if userID <= 0 || slug == "" {
			return http.StatusUnprocessableEntity, map[string]any{"reason": "gamification.grant_payload_invalid"}, nil
		}
		result, err := engine.ManualGrant(ctx, slug, userID, actorID)
		if err != nil {
			return http.StatusUnprocessableEntity, map[string]any{"reason": "gamification.grant_failed", "message": err.Error()}, nil
		}
		return http.StatusCreated, map[string]any{
			"granted": result.Granted, "already": result.Already, "badgeSlug": slug, "userId": userID,
		}, nil
	case routeAdminGrantByUser:
		if method != http.MethodDelete {
			break
		}
		userID, _ := strconv.ParseInt(strings.TrimSpace(pathParams["userId"]), 10, 64)
		slug := strings.TrimSpace(pathParams["badgeSlug"])
		if userID <= 0 || slug == "" {
			return http.StatusBadRequest, map[string]any{"reason": "gamification.grant_payload_invalid"}, nil
		}
		if err := engine.Store.RevokeBadge(ctx, userID, slug); err != nil {
			return http.StatusNotFound, map[string]any{"reason": "gamification.grant_not_found"}, nil
		}
		log.Printf("gamification manual revoke actor=%d user=%d badge=%s", actorID, userID, slug)
		return http.StatusOK, map[string]any{"revoked": true}, nil
	case routeAdminGrantLog:
		if method != http.MethodGet {
			break
		}
		userID, _ := strconv.ParseInt(strings.TrimSpace(pathParams["userId"]), 10, 64)
		grants, err := engine.Store.ListUserBadges(ctx, userID)
		if err != nil {
			return 0, nil, err
		}
		encoded, err := json.Marshal(grantsForWire(grants))
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"grants": string(encoded)}, nil
	case routeAdminLevels:
		switch method {
		case http.MethodGet:
			levels, err := engine.Store.ListLevels(ctx)
			if err != nil {
				return 0, nil, err
			}
			encoded, err := json.Marshal(levels)
			if err != nil {
				return 0, nil, err
			}
			return http.StatusOK, map[string]any{"levels": string(encoded)}, nil
		case http.MethodPost:
			rule, err := levelFromWire(body)
			if err != nil {
				return http.StatusUnprocessableEntity, map[string]any{"reason": "gamification.level_invalid", "message": err.Error()}, nil
			}
			if err := engine.Store.UpsertLevel(ctx, rule); err != nil {
				return 0, nil, err
			}
			return http.StatusCreated, map[string]any{"minPoints": rule.MinPoints}, nil
		}
	case routeAdminSeedDefaults:
		if method != http.MethodPost {
			break
		}
		// 一键恢复推荐默认：徽章 + 等级。已存在的同名徽章被覆盖为默认规则，
		// 历史授予不动（幂等）。
		seeded, err := engine.SeedDefaults(ctx)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{
			"badges": seeded.Badges, "levels": seeded.Levels, "restored": true,
		}, nil
	case routeAdminBackfill:
		if method != http.MethodPost {
			break
		}
		// 管理员手动触发的历史回填（不做自动全表扫描）。
		users, points, err := engine.Backfill(ctx, body)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, map[string]any{"users": users, "points": points}, nil
	case routeProfileAchievements:
		if method != http.MethodGet {
			break
		}
		userID, _ := strconv.ParseInt(strings.TrimSpace(pathParams["userId"]), 10, 64)
		wall, err := engine.ProfileWall(ctx, userID)
		if err != nil {
			return 0, nil, err
		}
		return http.StatusOK, wall, nil
	}
	return http.StatusNotFound, map[string]any{"reason": "gamification.route_not_found"}, nil
}

func badgeFromWire(body map[string]any) (BadgeDefinition, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return BadgeDefinition{}, err
	}
	var def BadgeDefinition
	if err := json.Unmarshal(raw, &def); err != nil {
		return BadgeDefinition{}, err
	}
	return def, validateBadgeDefinition(def)
}

// mergeBadgePatch 让 PATCH 只覆盖提供的字段（slug 固定）。
func mergeBadgePatch(current BadgeDefinition, patch map[string]any) map[string]any {
	merged := map[string]any{
		"slug": current.Slug, "name": current.Name, "description": current.Description,
		"rarity": current.Rarity, "rule": current.Rule, "icon": current.Icon,
		"grantsPoints": current.Points, "sort": current.Sort,
	}
	for key, value := range patch {
		merged[key] = value
	}
	return merged
}

func levelFromWire(body map[string]any) (LevelRule, error) {
	rule := LevelRule{
		MinPoints: int64FromAny(body["minPoints"]),
		Title:     stringValue(body["title"]),
		Icon:      stringValue(body["icon"]),
	}
	if rule.Title == "" || rule.MinPoints < 0 {
		return LevelRule{}, fmt.Errorf("level needs a title and min_points >= 0")
	}
	return rule, nil
}

func badgesForWire(badges []BadgeDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(badges))
	for _, badge := range badges {
		result = append(result, map[string]any{
			"slug": badge.Slug, "name": badge.Name, "description": badge.Description,
			"rarity": badge.Rarity, "rule": badge.Rule, "icon": badge.Icon,
			"grantsPoints": badge.Points, "sort": badge.Sort,
			"disabled": badge.DisabledAt != nil,
		})
	}
	return result
}

func grantsForWire(grants []UserBadge) []map[string]any {
	result := make([]map[string]any, 0, len(grants))
	for _, grant := range grants {
		result = append(result, map[string]any{
			"userId": grant.UserID, "badgeSlug": grant.BadgeSlug, "source": grant.Source,
			"actorId": grant.ActorID, "grantedAt": grant.GrantedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return result
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case string:
		var parsed int64
		if _, err := fmt.Sscanf(typed, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

// routeResponseContext 必须原样回传 request id / trace / extension 身份，
// 否则 Host 判定响应与「精确运行时请求」不匹配。
// routeResponseSchema 与 manifest routes[].responseSchema 一一对应。
func routeResponseSchema(routeID string) string {
	switch routeID {
	case routeAdminBadges:
		return "sforum.gamification.route.admin.badges.response@1"
	case routeAdminBadgeBySlug:
		return "sforum.gamification.route.admin.badge.response@1"
	case routeAdminGrants:
		return "sforum.gamification.route.admin.grants.response@1"
	case routeAdminGrantLog:
		return "sforum.gamification.route.admin.grant-log.response@1"
	case routeAdminLevels:
		return "sforum.gamification.route.admin.levels.response@1"
	case routeProfileAchievements:
		return "sforum.gamification.route.profile.achievements.response@1"
	}
	return "sforum.gamification.route.admin.action.response@1"
}

func routeResponseContext(request *protocolwire.RequestContext) *protocolwire.ResponseContext {
	if request == nil {
		return &protocolwire.ResponseContext{ServerTime: timestamppb.New(time.Now().UTC())}
	}
	return &protocolwire.ResponseContext{
		RequestId:  request.GetRequestId(),
		Trace:      proto.Clone(request.GetTrace()).(*protocolwire.TraceContext),
		Extension:  proto.Clone(request.GetExtension()).(*protocolwire.ExtensionIdentity),
		ServerTime: timestamppb.New(time.Now().UTC()),
	}
}
