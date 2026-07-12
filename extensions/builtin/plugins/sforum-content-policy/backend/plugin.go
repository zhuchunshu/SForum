package main

import (
	"context"
	"os"
	"time"

	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

// contentPolicyPlugin 是 E5 工作流参考插件：关键词门禁 + 公共 UI 贡献。
// 不实现 mail.provider；SendMail 走 Noop 默认拒绝。
type contentPolicyPlugin struct {
	pluginsdk.Noop
}

func (contentPolicyPlugin) Health() (pluginsdk.Health, error) {
	// 可选：Host API 注入时做连通性探测（与 hostapi fixture 一致）。
	if os.Getenv("SFORUM_HOST_API_URL") != "" {
		host, err := pluginsdk.HostFromEnv()
		if err != nil {
			return pluginsdk.Health{OK: false}, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		resp, err := pluginsdk.Ping(ctx, host)
		if err != nil {
			return pluginsdk.Health{OK: false}, err
		}
		if !resp.OK {
			return pluginsdk.Health{OK: false}, nil
		}
	}
	return pluginsdk.Health{OK: true}, nil
}

func (contentPolicyPlugin) RouteTarget() (pluginsdk.RouteTarget, error) {
	// 纯 filter + 贡献描述符，无 HTTP 代理路由。
	return pluginsdk.RouteTarget{}, nil
}

func (p contentPolicyPlugin) InvokeHook(req pluginsdk.HookRequest) (pluginsdk.HookResponse, error) {
	// observe 类事件若被误投递则放行；本插件只声明 filter。
	switch req.Name {
	case "topic.before_create", "topic.before_update", "comment.before_create":
		return p.handleFilter(req)
	default:
		return pluginsdk.HookResponse{OK: true}, nil
	}
}

func (contentPolicyPlugin) handleFilter(req pluginsdk.HookRequest) (pluginsdk.HookResponse, error) {
	// filter 必须廉价：只读 env 配置 + 子串匹配，不发 Host API、不入队。
	cfg := loadPolicyConfigFromEnv()
	title := payloadString(req.Payload, "title")
	content := payloadString(req.Payload, "content")
	decision := evaluateContent(cfg, req.Name, title, content)
	if !decision.OK {
		return pluginsdk.HookResponse{
			OK:      false,
			Reason:  decision.Reason,
			Message: decision.Message,
		}, nil
	}
	if decision.PatchTag == "" {
		return pluginsdk.HookResponse{OK: true}, nil
	}
	// mode=tag：仅 patch tagSlugs（宿主 allowlist 字段）。
	merged := mergeTagSlugs(req.Payload["tagSlugs"], decision.PatchTag)
	return pluginsdk.HookResponse{
		OK:      true,
		Reason:  decision.Reason,
		Message: decision.Message,
		Patch: map[string]any{
			"tagSlugs": merged,
		},
	}, nil
}
