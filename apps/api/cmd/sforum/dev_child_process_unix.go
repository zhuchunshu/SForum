//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"syscall"
)

// devConfigureChildProcess 让构建子进程运行在**自己的进程组**里，并在 context 取消时
// 杀掉整个进程组。没有这一步时，`bun run build` 派生的孙进程（例如构建脚本里的
// 长命令）会在 SIGINT 之后存活——真机 e2e 里 `sleep 45` 就是这样残留的。
func devConfigureChildProcess(command *exec.Cmd) {
	if command == nil {
		return
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		// 负 pid 表示整个进程组；进程已退出时 ESRCH 视为成功。
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); err != nil &&
			!errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}
}
