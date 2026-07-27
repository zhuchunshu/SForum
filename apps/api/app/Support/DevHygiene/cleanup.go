package devhygiene

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ProcessLister 列出本机进程；测试可注入固定表。
type ProcessLister func() ([]ProcessRow, error)

// ProcessSignaler 向 PID 发送信号；测试可注入记录器。
type ProcessSignaler func(pid int, signal syscall.Signal) error

// CleanupOptions 控制一次孤儿插件清理。
type CleanupOptions struct {
	DryRun bool
	List   ProcessLister
	Signal ProcessSignaler
	// Grace 是 TERM 后等待再 KILL 的间隔；0 表示默认 150ms。
	Grace time.Duration
}

// CleanupResult 是一次清理的结果摘要。
type CleanupResult struct {
	Selected []int
	Signaled []int
	DryRun   bool
}

// ListProcessesViaPS 用 ps 采集 PID/PPID/command（与 admin overview 采样一致，无 cgo）。
func ListProcessesViaPS() ([]ProcessRow, error) {
	cmd := exec.Command("ps", "-axo", "pid=,ppid=,command=")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps process list: %w", err)
	}
	return ParseProcessList(out)
}

// ParseProcessList 解析 `ps -axo pid=,ppid=,command=` 输出。
func ParseProcessList(out []byte) ([]ProcessRow, error) {
	lines := bytes.Split(out, []byte{'\n'})
	rows := make([]ProcessRow, 0, len(lines))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(string(line))
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil || ppid < 0 {
			continue
		}
		rows = append(rows, ProcessRow{
			PID:     pid,
			PPID:    ppid,
			Command: strings.Join(fields[2:], " "),
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("ps process list: empty")
	}
	return rows, nil
}

func defaultSignal(pid int, signal syscall.Signal) error {
	return syscall.Kill(pid, signal)
}

// CleanupOrphanExtensionPlugins 使用 SelectOrphanExtensionPluginPIDs 选出并终止孤儿插件。
// 这是 shell / CLI / 开发 reaper 的唯一执行入口，避免规则漂移。
func CleanupOrphanExtensionPlugins(opts CleanupOptions) (CleanupResult, error) {
	list := opts.List
	if list == nil {
		list = ListProcessesViaPS
	}
	signal := opts.Signal
	if signal == nil {
		signal = defaultSignal
	}
	grace := opts.Grace
	if grace <= 0 {
		grace = 150 * time.Millisecond
	}

	rows, err := list()
	if err != nil {
		return CleanupResult{}, err
	}
	selected := SelectOrphanExtensionPluginPIDs(rows)
	result := CleanupResult{Selected: append([]int(nil), selected...), DryRun: opts.DryRun}
	if opts.DryRun || len(selected) == 0 {
		return result, nil
	}

	self := os.Getpid()
	for _, pid := range selected {
		if pid == self {
			continue
		}
		// 发送前再确认进程仍匹配插件命令，降低 TOCTOU 误杀面。
		if cmd, ok := commandForPID(rows, pid); ok && !IsExtensionBackendPluginCommand(cmd) {
			continue
		}
		if err := signal(pid, syscall.SIGTERM); err != nil {
			// 进程可能已退出；继续处理其余目标。
			continue
		}
		result.Signaled = append(result.Signaled, pid)
	}

	time.Sleep(grace)
	for _, pid := range result.Signaled {
		// 仍存活则强杀。
		if err := signal(pid, 0); err != nil {
			continue
		}
		_ = signal(pid, syscall.SIGKILL)
	}
	return result, nil
}

func commandForPID(rows []ProcessRow, pid int) (string, bool) {
	for _, row := range rows {
		if row.PID == pid {
			return row.Command, true
		}
	}
	return "", false
}
