//go:build !windows

package main

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// M8 任务 10：构建子进程必须可被整体回收。真机 e2e 里 `bun run build` 派生的
// `sleep 45` 在 SIGINT 之后存活过——只杀直接子进程不够，必须杀整个进程组。
func TestDevChildProcessKillsWholeProcessGroupOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(ctx, "sh", "-c", "sleep 30")
	devConfigureChildProcess(command)
	if command.SysProcAttr == nil || !command.SysProcAttr.Setpgid {
		t.Fatalf("child must run in its own process group: %#v", command.SysProcAttr)
	}
	if command.Cancel == nil {
		t.Fatalf("child must install a group-killing cancel")
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	pid := command.Process.Pid

	// 进程组在取消前必须存在。
	if err := syscall.Kill(-pid, 0); err != nil {
		t.Fatalf("process group missing before cancel: %v", err)
	}
	cancel()
	_ = command.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for {
		err := syscall.Kill(-pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("process group %d survived cancellation (err=%v)", pid, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// 已退出的子进程再取消不得报错（Cancel 里的 ESRCH 必须被视为成功）。
func TestDevChildProcessCancelIsIdempotentAfterExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := exec.CommandContext(ctx, "true")
	devConfigureChildProcess(command)
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	if err := command.Cancel(); err != nil {
		t.Fatalf("cancel after exit must be a no-op: %v", err)
	}
}
