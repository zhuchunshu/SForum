package adminoverview

import (
	"os"
	"strings"
)

// ProcessSample 描述一次可测试的进程采样结果。
type ProcessSample struct {
	PID     int
	PPID    int
	// RSSBytes 是该进程的常驻内存（字节）。
	RSSBytes uint64
	Command  string
}

// ProcessSampler 读取本机进程列表；失败时 Snapshot 仍返回 Go MemStats。
type ProcessSampler interface {
	// List 返回可见进程；测试可注入固定样本。
	List() ([]ProcessSample, error)
}

// defaultProcessSampler 使用 OS 相关实现（见 process_sampler_*.go）。
var defaultProcessSampler ProcessSampler = osProcessSampler{}

// IsBackendPluginCommand 判断命令行是否为扩展 storage 下的 backend plugin 二进制。
// 仅匹配 SForum 扩展制品路径，避免误伤其它同名程序。
func IsBackendPluginCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	// 安装后运行路径：.../storage/extensions/<id>/<ver>/<digest>/backend/plugin
	// 以及部分相对路径形式。
	if !strings.Contains(cmd, "backend/plugin") && !strings.Contains(cmd, "backend\\plugin") {
		return false
	}
	if strings.Contains(cmd, "storage/extensions/") || strings.Contains(cmd, "storage\\extensions\\") {
		return true
	}
	// 兼容仍指向 extensions/ 树的开发路径（builtin-dev 等）。
	if strings.Contains(cmd, "/extensions/") || strings.Contains(cmd, "\\extensions\\") {
		return true
	}
	return false
}

// IsSforumAPICommand 判断命令是否为本仓库 API 进程。
func IsSforumAPICommand(command string) bool {
	return strings.Contains(command, "sforum-api")
}

// AggregateProcessMemory 从进程样本计算父进程 RSS 与「当前 API 拥有的」插件子进程全家内存。
// ownedPlugin 仅统计 PPID == selfPID 且命令匹配 backend plugin 的子进程；不包含 PPID=1 孤儿。
func AggregateProcessMemory(selfPID int, samples []ProcessSample) (selfRSS uint64, familyRSS uint64, pluginChildren int, selfFound bool) {
	liveAPI := map[int]struct{}{}
	for _, s := range samples {
		if IsSforumAPICommand(s.Command) {
			liveAPI[s.PID] = struct{}{}
		}
	}

	for _, s := range samples {
		if s.PID != selfPID {
			continue
		}
		selfRSS = s.RSSBytes
		selfFound = true
		break
	}

	familyRSS = selfRSS
	for _, s := range samples {
		if s.PID == selfPID {
			continue
		}
		if s.PPID != selfPID {
			continue
		}
		if !IsBackendPluginCommand(s.Command) {
			continue
		}
		// 双重保险：父进程必须是当前 API（或样本中标记为 sforum-api）。
		if _, ok := liveAPI[s.PPID]; !ok && s.PPID != selfPID {
			continue
		}
		pluginChildren++
		familyRSS += s.RSSBytes
	}
	return selfRSS, familyRSS, pluginChildren, selfFound
}

// sampleSelfAndFamily 使用采样器读取本进程 RSS 与 owned 插件全家内存。
// 采样失败时 ok=false，调用方应省略 family 字段并保留 Sys 诊断值。
func sampleSelfAndFamily(sampler ProcessSampler) (selfRSS uint64, familyRSS uint64, pluginChildren int, ok bool) {
	if sampler == nil {
		return 0, 0, 0, false
	}
	samples, err := sampler.List()
	if err != nil || len(samples) == 0 {
		return 0, 0, 0, false
	}
	selfPID := os.Getpid()
	selfRSS, familyRSS, pluginChildren, found := AggregateProcessMemory(selfPID, samples)
	if !found || selfRSS == 0 {
		// 部分平台短时读不到自身时，仍尝试用 owned 子进程 + 尽力 RSS。
		return selfRSS, familyRSS, pluginChildren, found && selfRSS > 0
	}
	return selfRSS, familyRSS, pluginChildren, true
}
