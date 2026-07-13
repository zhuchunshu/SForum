//go:build protocol_v1

package main

import pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"

// protocol_v1 保留可实际构建的回滚入口；P13 退出门禁通过前不得删除。
func main() {
	pluginsdk.Serve(contentPolicyPlugin{})
}
