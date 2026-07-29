package main

import pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"

func main() {
	pluginv2.Serve(newSMTPPluginV2())
}
