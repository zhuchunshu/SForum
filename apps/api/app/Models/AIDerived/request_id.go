package aiderived

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

// NewRequestID 生成一次性触发标识，参与 River 唯一键。
//
// 为什么不用「主题 + 用途」做唯一键：那会让「重新生成一份」永远被去重挡回，
// 而模型输出不理想时重新生成正是这类功能最常用的动作。唯一性只覆盖「同一次
// 触发被重复投递」（CLI 重试、按钮双击），因此标识必须每次触发都不同。
//
// 随机源失败时退化为时间戳：唯一键的强度只需要「同一毫秒内两次触发不撞」，
// 而随机源不可用不该让整个触发失败。
func NewRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "ts-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buffer)
}
