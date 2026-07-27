// Package processmemory 提供主机进程表采样与扩展 backend 插件 RSS 归因。
// Admin overview 与扩展列表共用，避免 Models 包互相 import。
package processmemory

import (
	"os"
	"strings"
)

// Sample 描述一次可测试的进程采样结果。
type Sample struct {
	PID  int
	PPID int
	// RSSBytes 是该进程的常驻内存（字节）。
	RSSBytes uint64
	Command  string
}

// Sampler 读取本机进程列表。
type Sampler interface {
	List() ([]Sample, error)
}

// DefaultSampler 使用 OS 相关实现（见 sampler_*.go）。
var DefaultSampler Sampler = osSampler{}

// IsBackendPluginCommand 判断命令行是否为扩展 storage 下的 backend plugin 二进制。
// 仅匹配 SForum 扩展制品路径，避免误伤其它同名程序。
func IsBackendPluginCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	// 安装后运行路径：.../storage/extensions/<id>/<ver>/<digest>/backend/plugin
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

// AggregateFamily 从进程样本计算父进程 RSS 与「当前 API 拥有的」插件子进程全家内存。
// ownedPlugin 仅统计 PPID == selfPID 且命令匹配 backend plugin 的子进程；不包含 PPID=1 孤儿。
func AggregateFamily(selfPID int, samples []Sample) (selfRSS uint64, familyRSS uint64, pluginChildren int, selfFound bool) {
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
		if _, ok := liveAPI[s.PPID]; !ok && s.PPID != selfPID {
			continue
		}
		pluginChildren++
		familyRSS += s.RSSBytes
	}
	return selfRSS, familyRSS, pluginChildren, selfFound
}

// SampleSelfAndFamily 使用采样器读取本进程 RSS 与 owned 插件全家内存。
// 采样失败时 ok=false。
func SampleSelfAndFamily(sampler Sampler) (selfRSS uint64, familyRSS uint64, pluginChildren int, ok bool) {
	if sampler == nil {
		return 0, 0, 0, false
	}
	samples, err := sampler.List()
	if err != nil || len(samples) == 0 {
		return 0, 0, 0, false
	}
	selfPID := os.Getpid()
	selfRSS, familyRSS, pluginChildren, found := AggregateFamily(selfPID, samples)
	if !found || selfRSS == 0 {
		return selfRSS, familyRSS, pluginChildren, found && selfRSS > 0
	}
	return selfRSS, familyRSS, pluginChildren, true
}

// ExtensionIDFromPluginCommand 从 backend plugin 命令行解析扩展 ID。
// 优先匹配安装路径 storage/extensions/<id>/...；开发/容器 builtin 路径作次级回退。
func ExtensionIDFromPluginCommand(command string) (string, bool) {
	if !IsBackendPluginCommand(command) {
		return "", false
	}
	normalized := strings.ReplaceAll(strings.TrimSpace(command), "\\", "/")
	if fields := strings.Fields(normalized); len(fields) > 0 {
		normalized = fields[0]
	}
	for _, marker := range []string{"storage/extensions/", "/extensions/builtin/plugins/"} {
		idx := strings.Index(normalized, marker)
		if idx < 0 {
			continue
		}
		rest := normalized[idx+len(marker):]
		id, _, _ := strings.Cut(rest, "/")
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, " ") {
			continue
		}
		return id, true
	}
	return "", false
}

// OwnedPluginRSSByExtensionID 汇总当前 API 拥有的 backend plugin 子进程 RSS（按扩展 ID）。
// 同一扩展多个子进程时累加；无法解析 ID 的进程跳过。
func OwnedPluginRSSByExtensionID(selfPID int, samples []Sample) map[string]uint64 {
	out := make(map[string]uint64)
	if selfPID <= 0 || len(samples) == 0 {
		return out
	}
	for _, s := range samples {
		if s.PPID != selfPID || s.PID == selfPID {
			continue
		}
		if !IsBackendPluginCommand(s.Command) {
			continue
		}
		id, ok := ExtensionIDFromPluginCommand(s.Command)
		if !ok {
			continue
		}
		out[id] += s.RSSBytes
	}
	return out
}

// SampleOwnedPluginRSS 使用默认进程采样器读取「本进程拥有的」各插件 RSS。
// 采样失败返回空 map。
func SampleOwnedPluginRSS() map[string]uint64 {
	return SampleOwnedPluginRSSWith(DefaultSampler)
}

// SampleOwnedPluginRSSWith 允许测试注入固定样本。
func SampleOwnedPluginRSSWith(sampler Sampler) map[string]uint64 {
	if sampler == nil {
		return map[string]uint64{}
	}
	samples, err := sampler.List()
	if err != nil || len(samples) == 0 {
		return map[string]uint64{}
	}
	return OwnedPluginRSSByExtensionID(os.Getpid(), samples)
}
