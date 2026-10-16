//go:build unix

// Package StableFile 提供跨平台的"稳定读取打开"原语：在 Unix 上以
// O_NONBLOCK 打开常规文件，避免目录诱导的阻塞 IO 卡住宿主线程。
// 它是通用 IO 基元，从 Extensions 领域包抽出（2026-10-17，为
// lifecycle orphan reclaim 腾出 flat-directory 名额，见该任务书）。
package stablefile

import (
	"os"
	"syscall"
)

// OpenReadFile 以非阻塞只读语义打开路径。
func OpenReadFile(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}

// OpenRootReadFile 在 os.Root 边界内以非阻塞只读语义打开相对路径。
func OpenRootReadFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
