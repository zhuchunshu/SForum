//go:build protocol_v1

package main

import extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"

// protocol_v1 保留可实际构建的回滚入口；LTS 窗口与零 shim 遥测满足前不得删除。
func main() {
	extensionsruntime.ServeProtocolPlugin(smtpPlugin{})
}
