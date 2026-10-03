package aireply

import (
	"context"

	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// GatewayCompleter 把生成器的请求转成一次网关调用。
//
// 它固定使用经济档位：回复是高频、低难度的任务，不该占用高成本模型。运营者
// 想换模型时改的是成本等级绑定，而不是这里的代码。
type GatewayCompleter struct {
	gateway *supportai.Gateway
}

func NewGatewayCompleter(gateway *supportai.Gateway) *GatewayCompleter {
	return &GatewayCompleter{gateway: gateway}
}

func (c *GatewayCompleter) Complete(ctx context.Context, input CompletionInput) (CompletionOutput, error) {
	if c == nil || c.gateway == nil {
		return CompletionOutput{}, ErrGeneratorUnavailable
	}
	result, err := c.gateway.Execute(ctx, supportai.CompletionRequest{
		Purpose:       input.Purpose,
		CostClass:     supportai.CostClassEconomy,
		System:        input.System,
		Messages:      []supportai.Message{{Role: supportai.RoleUser, Parts: []supportai.Part{{Type: supportai.PartText, Text: input.User}}}},
		MaxTokens:     input.MaxTokens,
		PromptVersion: input.PromptVersion,
		// SubjectUserID 让触发者的每日配额能被正确计量：机器人花的是提问者的额度。
		SubjectUserID: input.SubjectUserID,
	})
	if err != nil {
		return CompletionOutput{}, err
	}
	return CompletionOutput{Text: result.Text}, nil
}
