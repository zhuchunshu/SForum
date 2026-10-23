package ai

import (
	"errors"
	"fmt"
	"strings"
)

// ReasonEmptyCompletion 表示供应商正常返回、但没有给出任何正文。
//
// 它与 provider_failed 的区别是运营者要采取的动作：provider_failed 去查网络与
// 密钥，empty_completion 去查模型、输出预算与协议兼容性。把它并进 succeeded 是
// 本模块修过的一个真实故障——控制台一片「成功」，社区一条回复都没有：推理模型把
// 预算花在思考上、或兼容网关回了 200 空补全时，网关此前会照常记账成功。
const ReasonEmptyCompletion = "empty_completion"

// ErrEmptyCompletion 是空补全的哨兵错误。调用方用 errors.Is 判定，
// 与 ErrGatewayDisabled 等同类哨兵保持同一形态。
var ErrEmptyCompletion = errors.New("ai: provider returned an empty completion")

// validateCompletion 把「调用成功但没有产出」判成失败。
//
// 判定只针对「既没有正文、也没有工具调用」的补全：只要求工具的响应是工具循环的
// 正常一步（Run 会带着它继续），不是空回答。
//
// 它必须在网关里判，而不是留给各调用方：缓存、用量记账、执行记录、失败姿态都挂
// 在这条路径上，任何一处漏判都会让一次空回答变成一次「成功」——那正是这次故障的
// 形状。错误里带上 finish_reason、是否有思考、用量：这三项是同一句话的唯一可诊断
// 形态（length + 有思考 = 预算被思考吃光；都没有 = 端点没在产出回答）。
func validateCompletion(result CompletionResult) error {
	if len(result.ToolCalls) > 0 {
		return nil
	}
	if strings.TrimSpace(result.Text) != "" {
		return nil
	}
	return fmt.Errorf(
		"%w [%s] (finish_reason=%s reasoning=%t usage=%d/%d)",
		ErrEmptyCompletion,
		ReasonEmptyCompletion,
		emptyFallback(result.StopReason, "unknown"),
		strings.TrimSpace(result.Reasoning) != "",
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
	)
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
