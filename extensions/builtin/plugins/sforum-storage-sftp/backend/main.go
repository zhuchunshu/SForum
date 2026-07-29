package main

import (
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

func main() {
	pluginv2.Serve(pluginsdk.NewStorageProviderV2(newSFTPStorageProvider()))
}
