//go:build !unix

// Package StableFile 提供跨平台的"稳定读取打开"原语。
// 非 Unix 平台没有 O_NONBLOCK 常规文件语义，退化为普通打开。
package stablefile

import "os"

// OpenReadFile 以只读语义打开路径。
func OpenReadFile(name string) (*os.File, error) {
	return os.Open(name)
}

// OpenRootReadFile 在 os.Root 边界内以只读语义打开相对路径。
func OpenRootReadFile(root *os.Root, name string) (*os.File, error) {
	return root.Open(name)
}
