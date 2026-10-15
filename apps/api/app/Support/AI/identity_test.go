package ai_test

import (
	"strings"
	"testing"

	ai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// 模型身份是运行期事实：同一个系统提示词服务三档成本等级绑定，三档可能是三个不同
// 的模型。因此「你是谁」只能由解析出 profile 的那一层写进 system。
func TestApplyIdentityNamesTheResolvedProfile(t *testing.T) {
	profile := deepseekProfile()
	system := ai.ApplyIdentity("你是社区助手。", profile, true)
	if !strings.Contains(system, "你是社区助手。") {
		t.Fatalf("调用方的 system 被丢弃：%q", system)
	}
	if !strings.Contains(system, profile.Label) || !strings.Contains(system, profile.Model) {
		t.Fatalf("身份声明缺少模型标识：%q", system)
	}
	if !strings.HasPrefix(system, "你是社区助手。") {
		t.Fatalf("身份声明必须追加在调用方指令之后：%q", system)
	}
}

// 关闭公开不是「让模型不知道」，而是给它一个可以说出口的确定答案：编造供应商与
// 模型名同样是错误答案。
func TestApplyIdentityConcealsModelWhenDisabled(t *testing.T) {
	profile := deepseekProfile()
	system := ai.ApplyIdentity("你是社区助手。", profile, false)
	if strings.Contains(system, profile.Model) || strings.Contains(system, profile.Label) {
		t.Fatalf("不公开时不得出现模型标识：%q", system)
	}
	if !strings.Contains(system, "AI 助手") {
		t.Fatalf("不公开时模型仍需知道自己是什么：%q", system)
	}
}

// system 为空时身份声明就是全部系统指令：模型身份与「有没有别的指令」无关。
func TestApplyIdentityFillsEmptySystem(t *testing.T) {
	system := ai.ApplyIdentity("   ", deepseekProfile(), true)
	if strings.TrimSpace(system) == "" {
		t.Fatal("空 system 也必须拿到身份声明")
	}
	if !strings.Contains(system, deepseekProfile().Model) {
		t.Fatalf("system = %q", system)
	}
}

// 公开模式要求 profile 有可读的标识与模型名；缺失时退回不公开声明，而不是写出一句
// 「本次使用的模型是 （）」。
func TestApplyIdentityFallsBackWhenProfileIsUnnamed(t *testing.T) {
	system := ai.ApplyIdentity("", ai.Profile{Protocol: ai.ProtocolOpenAIChat}, true)
	if strings.Contains(system, "本次使用的模型是") {
		t.Fatalf("无名 profile 不得渲染出半句声明：%q", system)
	}
	if !strings.Contains(system, "AI 助手") {
		t.Fatalf("system = %q", system)
	}
}

// 缺省（nil）必须是公开：存量配置文档里没有这一项，反序列化到零值 Settings 时
// 若默认不公开，升级后的站点会原样保留「模型不知道自己是哪个模型」这个缺陷。
func TestIdentitySettingsDefaultToDisclosing(t *testing.T) {
	var zero ai.IdentitySettings
	if !zero.ResolvedDiscloseModel() {
		t.Fatal("缺省必须是公开模型标识")
	}
	off := false
	if (ai.IdentitySettings{DiscloseModel: &off}).ResolvedDiscloseModel() {
		t.Fatal("显式关闭必须生效")
	}
	on := true
	if !(ai.IdentitySettings{DiscloseModel: &on}).ResolvedDiscloseModel() {
		t.Fatal("显式开启必须生效")
	}
}

func TestRecommendedSettingsDiscloseModel(t *testing.T) {
	if !ai.RecommendedSettings().Identity.ResolvedDiscloseModel() {
		t.Fatal("推荐默认必须公开模型标识")
	}
}
