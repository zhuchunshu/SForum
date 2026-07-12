package jobs

import (
	"fmt"
	"strings"
)

// ScheduleEnabledOptionName 返回某条 schedule 在 web_options 中的启用开关键。
// 缺失时视为启用（与 CoreScheduleDefinitions 默认 Enabled=true 一致）。
func ScheduleEnabledOptionName(scheduleID string) string {
	return "jobs.schedule." + scheduleID + ".enabled"
}

// ParseScheduleEnabled 解析 option 值；空/缺失视为 true。
func ParseScheduleEnabled(value string, present bool) bool {
	if !present {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		// 无法识别时保守视为启用，避免运维误写导致整站维护任务静默停掉。
		return true
	}
}

// FormatScheduleEnabled 写入 web_options 的规范化值。
func FormatScheduleEnabled(enabled bool) string {
	if enabled {
		return "true"
	}
	return "false"
}

// ValidateScheduleID 校验 schedule id 形态（防路径注入进 option name）。
func ValidateScheduleID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("schedule id is required")
	}
	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("schedule id must not contain whitespace")
	}
	if len(id) > 128 {
		return fmt.Errorf("schedule id is too long")
	}
	return nil
}
