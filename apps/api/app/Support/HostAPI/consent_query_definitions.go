package hostapi

// 同意式（consent-gated）读取面的 Host 定义。
//
// 设计边界（见 knowledge/decisions/2026-10-18-consented-email-surface.md）：
//   - public.users 的 email 永远不进入 sforum_core_v1 视图，也不进入任何公开
//     稳定查询；它只出现在这一条 ConsentGated 定义里。
//   - 定义本身没有 filters/sorts：actor 由 host 签发的单次 query actor 委派解出
//     后注入 context，插件请求里没有任何 userId 参数可注入。
//   - actorless 稳定入口（query_id = sforum.core.identity.self_email）一律拒绝；
//     唯一可执行路径是它在 Query Registry 的显式投影
//     （core.query.identity.self_email），那里必须携带 actor_delegation。
const (
	// QuerySelfEmailRegistryID 是 stable 定义在 Query Registry 的显式投影 id，
	// 也是插件实际调用的 query id。
	QuerySelfEmailRegistryID              = "core.query.identity.self_email"
	QuerySelfEmailRegistryContractVersion = QuerySelfEmailRegistryID + "@1"
	QuerySelfEmailRegistryPlanVersion     = QuerySelfEmailRegistryID + ".plan@1"
	// ProtocolV2ConsentQueryScope 与 shortcode 投影的 "content" 区分：这一面只在
	// 宿主发起的 route/admin 调用里签发。
	ProtocolV2ConsentQueryScope = "route"
	// ProtocolV2ConsentQueryMaxCost 是单次读取的成本预算：字段 3 × 1 + 基础 10 +
	// 单行 1 = 14，远低于 50，留出 Query Registry 成本口径变化的余量。
	ProtocolV2ConsentQueryMaxCost = 50
)

// protocolV2ActorSQLToken 是 actor 注入占位符的通用名；底层字符串与 shortcode
// 投影共享同一个 token，别名只为让新代码读起来与「shortcode 专属」解耦。
const protocolV2ActorSQLToken = shortcodeProjectionActorSQLToken

// selfEmailQueryDefinition 返回「读取本人 email」的 Host 定义。
func selfEmailQueryDefinition() protocolV2QueryDefinition {
	return protocolV2QueryDefinition{
		ID: QuerySelfEmail, PlanVersion: QueryStableCorePlanVersion,
		ResultSchemaID: QuerySelfEmailSchemaID, ResultSchemaVersion: QueryStableCoreResultSchemaV1,
		// email 是站点保存的展示形态；唯一性走 email_lower，后者永远不返回。
		// 非 active 或无 email 的账号返回空行集合，不报错。
		From: `(SELECT users.email AS email,
				(users.email_verified_at IS NOT NULL) AS email_verified,
				users.updated_at AS updated_at
			FROM public.users AS users
			WHERE users.id = ` + protocolV2ActorSQLToken + `
				AND users.status = 'active'
				AND users.email <> '') AS stable`,
		Fields: []protocolV2QueryField{
			{Name: "email", Expression: "stable.email"},
			{Name: "email_verified", Expression: "stable.email_verified"},
			{Name: "updated_at", Expression: "stable.updated_at"},
		},
		Single:       true,
		ActorScoped:  true,
		ConsentGated: true,
	}
}

// protocolV2ConsentGatedDefinitions 是全部 ConsentGated 定义，供目录自检与
// 注册表投影一致性断言使用（新增一条就必须同时出现在这里）。
func protocolV2ConsentGatedDefinitions() []protocolV2QueryDefinition {
	return []protocolV2QueryDefinition{selfEmailQueryDefinition()}
}

// IsProtocolV2ConsentGatedQuery 判断一个 **Query Registry query id** 是否指向
// consent-gated Host 定义。除这条显式映射外不接受任何命名推断。
func IsProtocolV2ConsentGatedQuery(registryQueryID string) bool {
	for _, definition := range protocolV2ConsentGatedDefinitions() {
		mapping, err := queryRegistryCoreMappingFor(definition)
		if err != nil {
			continue
		}
		if mapping.QueryID == registryQueryID {
			return true
		}
	}
	return false
}
