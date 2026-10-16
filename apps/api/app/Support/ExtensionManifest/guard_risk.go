package extensionmanifest

import "strings"

// 守卫风险词汇（M5 操作可读性）。
//
// 这里的风险是**运行时授权归属**的分类，不是权限分级：
//   - host_owned：宿主核心守卫自行评估，插件不参与该路由的授权决定；
//   - extension_code：kind=custom 的插件守卫用自己的代码决定该路由的授权；
//   - raw_request：该路由把原始请求（含 cookie/authorization 等凭据头）交给插件。
//
// 分类必须与 v3_validate.go 允许的 guard kind 完全一致；新增 kind 时必须同时更新
// GuardRuntimeRisk，并让测试覆盖，否则操作员会看到错误的授权归属。
const (
	GuardRiskHostOwned     = "host_owned"
	GuardRiskExtensionCode = "extension_code"
	GuardRiskRawRequest    = "raw_request"

	GuardKindCustom     = "custom"
	GuardKindRawRequest = "raw_request"
)

// GuardRuntimeRisk 返回某条路由步骤的授权归属风险。
// guardKind 为空表示该守卫由宿主拥有（核心守卫或未知引用，两者都在准入处 fail-closed）。
func GuardRuntimeRisk(guardID, guardKind string) string {
	if strings.TrimSpace(guardID) == GuardCoreRaw || strings.TrimSpace(guardKind) == GuardKindRawRequest {
		return GuardRiskRawRequest
	}
	if strings.TrimSpace(guardKind) == GuardKindCustom {
		return GuardRiskExtensionCode
	}
	return GuardRiskHostOwned
}

// GuardRequiresExactArtifactTrust 报告该守卫是否要执行插件代码，从而要求精确制品信任。
func GuardRequiresExactArtifactTrust(guardID, guardKind string) bool {
	risk := GuardRuntimeRisk(guardID, guardKind)
	return risk != GuardRiskHostOwned
}
