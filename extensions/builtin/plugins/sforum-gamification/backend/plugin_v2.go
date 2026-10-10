package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
	pluginwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/plugin/v2"
	protocolwire "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2/gen/sforum/protocol/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

// 三条 observe 订阅（Host Events catalog）：
//   - user.registered  payload: userId, username, email, locale
//   - topic.created    payload: topicId, authorUserId, ...
//   - comment.created  payload: commentId, topicId, authorUserId, parentId
const (
	hookUserRegistered = "sforum.gamification.event.user-registered"
	hookTopicCreated   = "sforum.gamification.event.topic-created"
	hookCommentCreated = "sforum.gamification.event.comment-created"

	hookContractVersion = "@1"
)

// gamificationPluginV2 是协议 v2 运行时：HookRegistry 分发 observe 事件到引擎。
type gamificationPluginV2 struct {
	*pluginv2.Server
	engine *Engine
}

func newGamificationPluginV2() (*gamificationPluginV2, error) {
	store, err := newGamificationStore()
	if err != nil {
		return nil, fmt.Errorf("gamification store init: %w", err)
	}
	if err := store.EnsureSchema(context.Background()); err != nil {
		_ = store.close()
		return nil, fmt.Errorf("gamification schema ensure: %w", err)
	}
	engine := NewEngine(store)
	if values, err := loadGrowthValuesFromEnv(); err == nil {
		engine.Values = values
	}

	hooks, err := pluginv2.NewHookRegistry(
		observeHook(hookUserRegistered, "user.registered", "sforum.gamification.event.user-registered.input@1", engine),
		observeHook(hookTopicCreated, "topic.created", "sforum.gamification.event.topic-created.input@1", engine),
		observeHook(hookCommentCreated, "comment.created", "sforum.gamification.event.comment-created.input@1", engine),
	)
	if err != nil {
		_ = store.close()
		return nil, err
	}
	jobs, err := pluginv2.NewJobRegistry(
		pluginv2.JobDefinition{
			ID:              "sforum.gamification.job.daily-scan",
			ContractVersion: "sforum.gamification.job.daily-scan@1",
			Name:            "sforum.gamification.daily-scan",
			Handler:         "gamification.job.daily-scan",
			PayloadSchema:   "sforum.gamification.job.daily-scan.payload@1",
			RetryPolicy:     "bounded", MaxAttempts: 3, RetryDelaySeconds: 30, ConcurrencyLimit: 1,
			Execute: makeDailyScanHandler(engine),
		},
	)
	if err != nil {
		_ = store.close()
		return nil, err
	}
	server := pluginv2.NewServer().WithHookRegistry(hooks).WithJobRegistry(jobs)
	plugin := &gamificationPluginV2{Server: server, engine: engine}
	// 庆祝通知：走 Host notifications.emit@1，需要握手后的 Host 客户端。
	engine.Celebrate = func(ctx context.Context, userID int64, badge BadgeDefinition, level string) {
		host, err := plugin.Host()
		if err != nil || host == nil {
			return
		}
		emitBadgeGranted(ctx, host, nil, userID, badge, level)
	}
	return plugin, nil
}

// makeDailyScanHandler 每日巡检：批量遍历注册快照，授予周年徽章。
func makeDailyScanHandler(engine *Engine) pluginv2.JobHandler {
	return func(ctx context.Context, call *pluginv2.JobCall) error {
		granted, err := engine.RunDailyScan(ctx, 100)
		if err != nil {
			return err
		}
		log.Printf("gamification daily scan granted=%d", granted)
		return nil
	}
}

func observeHook(id, name, inputSchema string, engine *Engine) pluginv2.HookDefinition {
	return pluginv2.HookDefinition{
		ID:              id,
		Name:            name,
		Kind:            "observe",
		ContractVersion: id + hookContractVersion,
		Handler:         "gamification.handle",
		InputSchema:     inputSchema,
		ResultSchema:    "sforum.gamification.event.result@1",
		Execute:         makeEventHandler(name, engine),
	}
}

func makeEventHandler(name string, engine *Engine) pluginv2.HookHandler {
	return func(ctx context.Context, call *pluginv2.HookCall) (*pluginv2.HookResult, error) {
		if call == nil || call.Payload == nil || call.Payload.Value == nil {
			return &pluginv2.HookResult{Accepted: false}, nil
		}
		payload := call.Payload.Value.AsMap()
		userID := payloadInt(payload, "userId")
		if userID == 0 {
			userID = payloadInt(payload, "authorUserId")
		}
		if userID <= 0 {
			// 非用户事件载荷：接受但忽略（Host 目录口径内的字段缺失）。
			return &pluginv2.HookResult{Accepted: true}, nil
		}
		event := EngineEvent{
			EventID: eventIDFromCall(call, userID),
			Name:    name,
			UserID:  userID,
		}
		if name == "user.registered" {
			// 注册快照取接收时刻（payload 无时间字段；周年语义容忍一天的偏移）。
			event.RegisteredAt = engine.Now().UTC()
		}
		outcomes, err := engine.HandleEvent(ctx, event)
		if err != nil {
			// 引擎/存储故障：返回非 accepted，让 River 按投递重试语义处理。
			log.Printf("gamification handle %s user=%d: %v", name, userID, err)
			return &pluginv2.HookResult{Accepted: false}, nil
		}
		result := map[string]any{"granted": len(outcomes)}
		for i, outcome := range outcomes {
			if !outcome.Newly {
				continue
			}
			result[fmt.Sprintf("badge_%d", i)] = outcome.Badge.Slug
		}
		value, err := structpb.NewStruct(result)
		if err != nil {
			return &pluginv2.HookResult{Accepted: true}, nil
		}
		return &pluginv2.HookResult{
			Accepted: true,
			Result: &protocolwire.TypedDocument{
				SchemaId: "sforum.gamification.event.result", SchemaVersion: "1", Value: value,
			},
		}, nil
	}
}

// eventIDFromCall 组合幂等锚点。delivery_id 在 Host 重投时可能变化，
// 因此用 (event name, user, 实体 id) 的稳定组合。
func eventIDFromCall(call *pluginv2.HookCall, userID int64) string {
	payload := map[string]any{}
	if call.Payload != nil && call.Payload.Value != nil {
		payload = call.Payload.Value.AsMap()
	}
	entity := payloadInt(payload, "topicId") + payloadInt(payload, "commentId")
	if entity == 0 {
		entity = userID // 注册事件无实体 id
	}
	return fmt.Sprintf("%s:%d:%d", strings.TrimPrefix(call.Name, ""), userID, entity)
}

func payloadInt(payload map[string]any, key string) int64 {
	switch v := payload[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		var parsed int64
		_, _ = fmt.Sscanf(v, "%d", &parsed)
		return parsed
	}
	return 0
}

// loadGrowthValuesFromEnv 从进程环境读成长值覆盖（M4 管理界面接管前的
// 开发通道；正式设置走 plugin settings）。
func loadGrowthValuesFromEnv() (GrowthValues, error) {
	values := DefaultGrowthValues()
	parseInt := func(key string, target *int64) error {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			return nil
		}
		var parsed int64
		if _, err := fmt.Sscanf(raw, "%d", &parsed); err != nil || parsed < 0 {
			return fmt.Errorf("invalid %s: %q", key, raw)
		}
		*target = parsed
		return nil
	}
	if err := parseInt("SFORUM_GAMIFICATION_TOPIC_POINTS", &values.TopicCreated); err != nil {
		return values, err
	}
	if err := parseInt("SFORUM_GAMIFICATION_COMMENT_POINTS", &values.CommentCreated); err != nil {
		return values, err
	}
	if err := parseInt("SFORUM_GAMIFICATION_REGISTRATION_POINTS", &values.Registration); err != nil {
		return values, err
	}
	return values, nil
}

var _ = pluginwire.HookRequest{} // 保持导入引用（HookCall 类型来自插件 wire）
