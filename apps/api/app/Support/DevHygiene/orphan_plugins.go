package devhygiene

import "strings"

// ProcessRow 是孤儿插件筛选的输入行（与 ps 采样字段对齐，便于单测注入）。
type ProcessRow struct {
	PID     int
	PPID    int
	Command string
}

// IsExtensionBackendPluginCommand 仅匹配扩展制品 backend/plugin 路径。
func IsExtensionBackendPluginCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	if !strings.Contains(cmd, "backend/plugin") && !strings.Contains(cmd, "backend\\plugin") {
		return false
	}
	if strings.Contains(cmd, "storage/extensions/") || strings.Contains(cmd, "storage\\extensions\\") {
		return true
	}
	if strings.Contains(cmd, "/extensions/") || strings.Contains(cmd, "\\extensions\\") {
		return true
	}
	return false
}

// IsLiveSforumAPI 判断某 PID 是否为仍在运行的 sforum-api（由调用方根据进程表构建）。
func IsLiveSforumAPI(command string) bool {
	return strings.Contains(command, "sforum-api")
}

// SelectOrphanExtensionPluginPIDs 选出可安全清理的孤儿 backend plugin PID。
//
// 允许：
//   - 命令匹配扩展 backend/plugin；
//   - 且 PPID 为 1（已被 init 收养），或 PPID 不在 liveAPIParents 且父进程不是 live sforum-api。
//
// 拒绝：
//   - 非插件命令；
//   - PPID 属于仍存活的 sforum-api（仍被当前/其它 API 拥有）；
//   - PID 本身是 sforum-api。
func SelectOrphanExtensionPluginPIDs(rows []ProcessRow) []int {
	liveAPI := map[int]struct{}{}
	byPID := make(map[int]ProcessRow, len(rows))
	for _, row := range rows {
		byPID[row.PID] = row
		if IsLiveSforumAPI(row.Command) {
			liveAPI[row.PID] = struct{}{}
		}
	}

	selected := make([]int, 0)
	seen := map[int]struct{}{}
	for _, row := range rows {
		if row.PID <= 1 {
			continue
		}
		if _, dup := seen[row.PID]; dup {
			continue
		}
		if IsLiveSforumAPI(row.Command) {
			continue
		}
		if !IsExtensionBackendPluginCommand(row.Command) {
			continue
		}
		// 仍挂在 live sforum-api 下：禁止杀。
		if _, owned := liveAPI[row.PPID]; owned {
			continue
		}
		// 父进程仍是 sforum-api 命令（即使 live 表漏检）也拒绝。
		if parent, ok := byPID[row.PPID]; ok && IsLiveSforumAPI(parent.Command) {
			continue
		}
		// 仅清理「已失联」父进程：PPID=1，或父进程已不在表中，或父进程不是 sforum-api。
		if row.PPID != 1 {
			parent, parentOK := byPID[row.PPID]
			if parentOK && !IsLiveSforumAPI(parent.Command) {
				// 父进程存在但不是 API：可能是调试器/脚本拉起的插件，不杀。
				continue
			}
			if parentOK {
				continue
			}
			// 父进程行缺失：视为可清理孤儿（热重载后常见）。
		}
		seen[row.PID] = struct{}{}
		selected = append(selected, row.PID)
	}
	return selected
}
