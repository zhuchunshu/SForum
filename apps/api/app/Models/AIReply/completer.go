package aireply

import (
	"context"

	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// GatewayCompleter 把生成器的请求转成一次网关调用。
//
// 它固定使用经济档位：回复是高频、低难度的任务，不该占用高成本模型。运营者
// 想换模型时改的是成本等级绑定，而不是这里的代码。
//
// 工具循环由 Host 的编排器执行（步数预算、逐步闸门、逐步执行记录都在那里）。
// orchestrator 为 nil 时退化为单次补全，与工具能力引入前完全一致——一个还没有
// 注册表的部署不会因此失去回复功能。
type GatewayCompleter struct {
	gateway      *supportai.Gateway
	orchestrator *supportai.Orchestrator
}

func NewGatewayCompleter(gateway *supportai.Gateway, orchestrator *supportai.Orchestrator) *GatewayCompleter {
	return &GatewayCompleter{gateway: gateway, orchestrator: orchestrator}
}

func (c *GatewayCompleter) Complete(ctx context.Context, input CompletionInput) (CompletionOutput, error) {
	if c == nil || c.gateway == nil {
		return CompletionOutput{}, ErrGeneratorUnavailable
	}
	if c.orchestrator != nil && len(input.Tools) > 0 {
		result, err := c.orchestrator.Run(ctx, supportai.RunInput{
			Purpose:       input.Purpose,
			CostClass:     supportai.CostClassEconomy,
			System:        input.System,
			User:          input.User,
			MaxTokens:     input.MaxTokens,
			PromptVersion: input.PromptVersion,
			SubjectUserID: input.SubjectUserID,
			ToolNames:     input.Tools,
			ToolContext:   input.ToolContext,
		})
		if err != nil {
			return CompletionOutput{}, err
		}
		return CompletionOutput{Text: result.Text}, nil
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
