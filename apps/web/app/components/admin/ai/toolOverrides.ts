import type { AdminAIPurposeOverride } from '~/composables/admin/useAdminAI'
import type { AIPurposeOverride } from '~/utils/admin/adminAI'
import { normalizePurposeToolNames } from '~/utils/admin/adminAI'

// 工具纯函数（utils/admin/adminAITools）产出的是「可能没有 allowed」的覆盖形状：它们
// 要能吃下旧文档里缺字段的配置。设置文档里的用途覆盖则要求 `tools.allowed` 一定存在。
//
// 界面写入草稿时在这里补齐，而不是在调用点做类型断言：断言会把「tools 存在但没有
// allowed」这种真问题静默带进配置文档，而服务端会把它当成一个空白名单。
export function toSettingsOverride(override: AIPurposeOverride | undefined): AdminAIPurposeOverride | undefined {
  if (!override) return undefined
  const next: AdminAIPurposeOverride = {}
  if (override.tools) next.tools = { allowed: normalizePurposeToolNames(override.tools.allowed) }
  const posture = typeof override.failurePosture === 'string' ? override.failurePosture.trim() : ''
  if (posture) next.failurePosture = posture
  return next
}

export function toSettingsOverrides(
  overrides: Record<string, AIPurposeOverride> | undefined
): Record<string, AdminAIPurposeOverride> {
  const result: Record<string, AdminAIPurposeOverride> = {}
  for (const [id, override] of Object.entries(overrides ?? {})) {
    const converted = toSettingsOverride(override)
    if (converted) result[id] = converted
  }
  return result
}
