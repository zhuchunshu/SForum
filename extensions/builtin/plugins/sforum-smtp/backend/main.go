package main

import extensionsruntime "github.com/zhuchunshu/sforum/apps/api/app/Support/Extensions"

func main() { extensionsruntime.ServeProtocolPlugin(smtpPlugin{}) }
