// Contract Host API fixture plugin process (F4.1).
//
// Built as backend/plugin and loaded by host go-plugin tests.
// On Health it optionally pings Host API when SFORUM_HOST_API_* is set.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
)

type contractPlugin struct {
	pluginsdk.Noop
}

func (contractPlugin) Health() (pluginsdk.Health, error) {
	// 宿主注入 Host API 时探测 Ping，证明 SDK 客户端与网关联通。
	if os.Getenv("SFORUM_HOST_API_URL") != "" {
		host, err := pluginsdk.HostFromEnv()
		if err != nil {
			return pluginsdk.Health{OK: false}, fmt.Errorf("host from env: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		resp, err := pluginsdk.Ping(ctx, host)
		if err != nil {
			return pluginsdk.Health{OK: false}, fmt.Errorf("host ping transport: %w", err)
		}
		if !resp.OK {
			return pluginsdk.Health{OK: false}, fmt.Errorf("host ping denied: %s %s", resp.Reason, resp.Message)
		}
	}
	return pluginsdk.Health{OK: true}, nil
}

func (contractPlugin) InvokeHook(req pluginsdk.HookRequest) (pluginsdk.HookResponse, error) {
	// filter 允许通过；observe 确认收到即可。
	if req.Name == "topic.before_create" {
		return pluginsdk.HookResponse{OK: true}, nil
	}
	return pluginsdk.HookResponse{OK: true}, nil
}

func main() {
	pluginsdk.Serve(contractPlugin{})
}
