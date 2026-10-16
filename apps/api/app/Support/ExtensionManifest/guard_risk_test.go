package extensionmanifest

import "testing"

// M5：授权归属分类必须与 v3 允许的 guard kind 一一对应，否则操作员会在影响文档和
// 路由检视器里看到错误的「谁拥有这条路由的授权」。
func TestGuardRuntimeRiskClassification(t *testing.T) {
	for _, test := range []struct {
		name      string
		guardID   string
		kind      string
		wantRisk  string
		wantTrust bool
	}{
		{name: "core guard needs no trust", guardID: "core.guard.login", wantRisk: GuardRiskHostOwned},
		{name: "core inherit", guardID: GuardCoreInherit, wantRisk: GuardRiskHostOwned},
		{name: "unknown reference stays host owned", guardID: "core.guard.missing", wantRisk: GuardRiskHostOwned},
		{
			name: "custom guard owns authorization", guardID: "acme.guard.owner", kind: GuardKindCustom,
			wantRisk: GuardRiskExtensionCode, wantTrust: true,
		},
		{
			name: "declared raw request guard", guardID: "acme.guard.raw", kind: GuardKindRawRequest,
			wantRisk: GuardRiskRawRequest, wantTrust: true,
		},
		{
			// 空绑定的 core.guard.raw_request：零守卫代码即铸造原始请求权威（R3）。
			// 分类必须如实标注 raw_request，即使没有插件声明。
			name: "core raw request without a declaration", guardID: GuardCoreRaw,
			wantRisk: GuardRiskRawRequest, wantTrust: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := GuardRuntimeRisk(test.guardID, test.kind); got != test.wantRisk {
				t.Fatalf("GuardRuntimeRisk(%q, %q) = %q; want %q", test.guardID, test.kind, got, test.wantRisk)
			}
			if got := GuardRequiresExactArtifactTrust(test.guardID, test.kind); got != test.wantTrust {
				t.Fatalf("GuardRequiresExactArtifactTrust(%q, %q) = %v; want %v", test.guardID, test.kind, got, test.wantTrust)
			}
		})
	}
}

// 声明 kind 白名单与风险词汇必须保持一一对应：新增 kind 而不更新分类会让
// v3_validate 接受的声明被静默当成宿主守卫。这里用校验器真正接受的清单验证。
func TestGuardKindVocabularyMatchesValidationVocabulary(t *testing.T) {
	manifest := completeV3Manifest()
	if err := Validate(manifest); err != nil {
		t.Fatalf("baseline fixture must stay valid: %v", err)
	}
	if len(manifest.Guards) != 1 || manifest.Guards[0].Kind != GuardKindCustom {
		t.Fatalf("fixture guard kind = %#v", manifest.Guards)
	}
	if risk := GuardRuntimeRisk(manifest.Guards[0].ID, manifest.Guards[0].Kind); risk != GuardRiskExtensionCode {
		t.Fatalf("admitted custom guard classified as %q", risk)
	}
	for _, kind := range []string{GuardKindCustom, GuardKindRawRequest} {
		if risk := GuardRuntimeRisk("acme.guard.probe", kind); risk == GuardRiskHostOwned {
			t.Fatalf("admitted guard kind %q has no runtime risk mapping", kind)
		}
	}
	// raw_request 也必须真的被校验器接受，否则风险词汇会领先于准入白名单。
	raw := completeV3Manifest()
	raw.Guards[0].Kind = GuardKindRawRequest
	if err := Validate(raw); err != nil {
		t.Fatalf("raw_request guard kind must be admitted: %v", err)
	}
	if risk := GuardRuntimeRisk(raw.Guards[0].ID, raw.Guards[0].Kind); risk != GuardRiskRawRequest {
		t.Fatalf("admitted raw_request guard classified as %q", risk)
	}
}
