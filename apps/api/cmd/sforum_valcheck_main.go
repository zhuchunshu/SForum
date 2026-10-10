package main

import (
	"fmt"

	extensionmanifest "github.com/zhuchunshu/sforum/apps/api/app/Support/ExtensionManifest"
)

func main() {
	m, err := extensionmanifest.LoadPackage("../../extensions/builtin/plugins/sforum-gamification")
	if err != nil {
		fmt.Println("LOAD:", err)
		return
	}
	fmt.Println("VALIDATE:", extensionmanifest.Validate(m))
}
