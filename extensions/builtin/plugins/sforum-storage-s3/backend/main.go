package main

import (
	pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"
	pluginv2 "github.com/zhuchunshu/sforum/apps/api/sdk/plugin/v2"
)

func main() {
	provider := pluginsdk.NewMultiStorageProvider("storage.s3", newS3Backend)
	pluginv2.Serve(pluginsdk.NewStorageProviderV2(provider))
}
