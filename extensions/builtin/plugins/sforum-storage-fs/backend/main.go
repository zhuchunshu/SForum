//go:build !protocol_v1

package main

import pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"

func main() {
	pluginv2.Serve(newFSStoragePluginV2())
}
