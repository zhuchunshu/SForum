package webreleaseruntime

import extensionpackage "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionPackage"

const (
	AdminSDKVersion      = 1
	BuildContractVersion = 1
	BunVersion           = "1.3.14"
)

func HostPeers() extensionpackage.HostPeers {
	return extensionpackage.HostPeers{
		"vue":               "3.5.39",
		"nuxt":              "4.4.8",
		"@nuxt/ui":          "4.9.0",
		"vue-router":        "5.1.0",
		"@sforum/admin-sdk": "1.0.0",
	}
}
