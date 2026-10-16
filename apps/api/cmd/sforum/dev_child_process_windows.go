//go:build windows

package main

import "os/exec"

// devConfigureChildProcess 在 Windows 上保持默认语义（取消时结束子进程）；
// 进程组语义由 Job Object 提供，超出当前 CLI 的范围。
func devConfigureChildProcess(_ *exec.Cmd) {}
