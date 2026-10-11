package aicontroller

import (
	"encoding/json"
	"testing"

	supportai "github.com/zhuchunshu/sforum/apps/api/app/Support/AI"
)

// 控制台的真实使用路径是「GET 默认配置 → 只改一个开关 → PUT 整份文档」。
// 这个往返必须始终可保存：一旦它被拒绝，用户看到的就是一个没有任何解释的 422。
func TestUpdateSettingsRequestRoundTripStaysSavable(t *testing.T) {
	settings := supportai.RecommendedSettings().Normalized()
	settings.Enabled = true

	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	var request updateSettingsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	assembled := settingsFromRequest(request)
	if !assembled.Enabled {
		t.Fatal("enabled flag did not survive the round trip")
	}
	if len(assembled.Profiles) != len(settings.Profiles) {
		t.Fatalf("profiles lost in round trip: %d vs %d", len(assembled.Profiles), len(settings.Profiles))
	}
	if err := assembled.Validate(); err != nil {
		t.Fatalf("enabling the gateway alone must stay savable: %v", err)
	}
	warnings := assembled.Warnings()
	if len(warnings) == 0 {
		t.Fatal("saving an enabled gateway without providers must warn")
	}
}

// 控制台是单选语义：选中的提供商必须真正被解析到，否则界面在撒谎。
func TestSelectedProviderIsTheOneResolved(t *testing.T) {
	for _, preset := range supportai.BuiltinProfiles() {
		settings := supportai.RecommendedSettings()
		// 模拟控制台提交：只启用选中的那个，并把三个成本等级都指向它。
		for index := range settings.Profiles {
			settings.Profiles[index].Enabled = settings.Profiles[index].ID == preset.ID
		}
		for _, class := range []string{supportai.CostClassEconomy, supportai.CostClassStandard, supportai.CostClassPremium} {
			settings.CostClassProfiles[class] = preset.ID
		}
		request := updateSettingsRequest{
			Enabled:               true,
			Profiles:              settings.Profiles,
			CostClassProfiles:     settings.CostClassProfiles,
			PurposeFailurePosture: settings.PurposeFailurePosture,
			AutoAction:            settings.AutoAction,
			Gates:                 settings.Gates,
			Redaction:             settings.Redaction,
		}
		assembled := settingsFromRequest(request)
		if err := assembled.Validate(); err != nil {
			t.Fatalf("preset %s: %v", preset.ID, err)
		}
		for _, class := range []string{supportai.CostClassEconomy, supportai.CostClassStandard, supportai.CostClassPremium} {
			resolved, err := assembled.ResolveProfile(class, "")
			if err != nil {
				t.Fatalf("preset %s class %s: %v", preset.ID, class, err)
			}
			if resolved.ID != preset.ID {
				t.Fatalf("selected %s but class %s resolves to %s", preset.ID, class, resolved.ID)
			}
		}
	}
}

// 每个内置预设单独启用后都必须可用：它们是运营者最先会打开的选项。
func TestBuiltinProfilesRemainIndividuallyEnableable(t *testing.T) {
	for _, preset := range supportai.BuiltinProfiles() {
		settings := supportai.RecommendedSettings().Normalized()
		settings.Enabled = true
		for index := range settings.Profiles {
			settings.Profiles[index].Enabled = settings.Profiles[index].ID == preset.ID
		}
		if err := settings.Validate(); err != nil {
			t.Fatalf("preset %s must be enableable on its own: %v", preset.ID, err)
		}
		warnings := settings.Warnings()
		for _, warning := range warnings {
			if warning == "gateway_enabled_without_enabled_profile:no_call_will_succeed" {
				t.Fatalf("preset %s enabled but the missing-provider warning still fired", preset.ID)
			}
		}
	}
}

// 前端一直在 PUT 文档里带 `reply`（提示词与工具白名单），而 DTO 曾漏掉这个字段：
// 保存设置会静默清空运营者写过的提示词，并让白名单配置根本无法落库。这条断言把
// 「往返不丢回复相关字段」钉在控制台边界上。
func TestUpdateSettingsRequestKeepsReplyOverrides(t *testing.T) {
	settings := supportai.RecommendedSettings().Normalized()
	settings.Reply.SystemPrompt = "只回答与本站相关的问题。"
	settings.Reply.Tools = &supportai.ReplyToolSettings{Allowed: []string{"forum-search", "vendor-plugin-tool"}}

	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var request updateSettingsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	assembled := settingsFromRequest(request)
	if assembled.Reply.SystemPrompt != settings.Reply.SystemPrompt {
		t.Fatalf("reply prompt lost in round trip: %q", assembled.Reply.SystemPrompt)
	}
	names, configured := assembled.Reply.ToolAllowlist()
	if !configured || len(names) != 2 || names[1] != "vendor-plugin-tool" {
		t.Fatalf("reply tool allowlist lost in round trip: %#v configured=%v", names, configured)
	}
	if err := assembled.Validate(); err != nil {
		t.Fatalf("well-formed reply overrides must stay savable: %v", err)
	}

	// 白名单形状校验必须走同一条保存路径：非法名字在保存时就该被拒。
	settings.Reply.Tools = &supportai.ReplyToolSettings{Allowed: []string{"Vendor Tool"}}
	raw, err = json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var invalid updateSettingsRequest
	if err := json.Unmarshal(raw, &invalid); err != nil {
		t.Fatal(err)
	}
	if err := settingsFromRequest(invalid).Validate(); err == nil {
		t.Fatal("an invalid tool name must be rejected before saving")
	}
}
