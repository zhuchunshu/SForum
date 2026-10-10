// sforum.gamification 徽章/成就插件（受保护内置）——进程入口。
package main

import (
	"log"

	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

func main() {
	server, err := newGamificationPluginV2()
	if err != nil {
		log.Fatalf("configure gamification protocol v2: %v", err)
	}
	pluginv2.Serve(server)
}
