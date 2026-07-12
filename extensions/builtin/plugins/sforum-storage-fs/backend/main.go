package main

import pluginsdk "github.com/zhuchunshu/sforum/apps/api/sdk/plugin"

func main() {
	pluginsdk.Serve(newFSStoragePlugin())
}
