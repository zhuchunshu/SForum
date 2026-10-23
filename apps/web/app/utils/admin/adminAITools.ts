// 模型工具目录的展示层：三源英文说明、用途生效集合、自动收敛与诊断。
//
// 工具是「代码声明的全局事实」，用途白名单是「某个用途的生效配置」。控制台要把
// 两者摆在一起看：某个工具被哪些用途启用、它的英文说明来自运营者覆盖还是作者
// 声明、哪些名字被引用却根本没有注册。
//
// 本模块只从 ./adminAI 取用途覆盖的既有归一化函数（单向依赖），不从 composable
// 引入任何东西，避免 util -> composable -> util 的环。

import type { AIPurposeOverride } from './adminAI'
import {
  normalizePurposeToolNames,
  prunePurposeOverrides,
  purposeOverrideIsEmpty,
  selectedPurposeTools
} from './adminAI'

export type AIToolDisplayLike = {
  name: string
  description?: string
  descriptionI18n?: Record<string, string>
  origin?: { kind?: string, extensionId?: string }
}

export type AIToolDisplaySource = 'operator' | 'declared' | 'none'

/** AIToolPurposeLike 是用途目录条目的最小结构要求（声明 + 标识）。 */
export type AIToolPurposeLike = {
  id: string
  declaredTools?: readonly string[]
}

export type AIToolFilterOrigin = 'all' | 'core' | 'plugin'

export type AIToolSupportPurposeLike = { costClass: string }

export type AIToolSupportSettingsLike = {
  costClassProfiles?: Record<string, string>
  profiles?: readonly { id: string, supportsTools?: boolean }[]
}

function localeText(map: Record<string, string> | undefined, locale: string): string {
  if (!map || typeof map !== 'object') return ''
  const value = map[locale]
  return typeof value === 'string' ? value.trim() : ''
}

/**
 * toolDisplayEnglish 解析控制台里那条英文说明的**唯一来源**，并把来源一起返回。
 *
 * 优先级是运营者覆盖 > 作者/内置声明（en-US，再退 en）。返回 source 而不是只返回
 * 文本，是因为界面必须能说清「这句话是谁写的」：把声明默认显示成运营者覆盖，会让
 * 一次误删看起来像一次编辑。
 */
export function toolDisplayEnglish(
  tool: AIToolDisplayLike,
  overrides: Record<string, Record<string, string>> | undefined
): { text: string, source: AIToolDisplaySource } {
  const name = typeof tool?.name === 'string' ? tool.name.trim() : ''
  const operator = name && overrides ? overrides[name] : undefined
  const fromOperator = localeText(operator, 'en-US')
  if (fromOperator) {
    return { text: fromOperator, source: 'operator' }
  }
  const declared = localeText(tool?.descriptionI18n, 'en-US') || localeText(tool?.descriptionI18n, 'en')
  if (declared) {
    return { text: declared, source: 'declared' }
  }
  return { text: '', source: 'none' }
}

/**
 * toolOriginLabel 是工具来源徽标的文本：Core 内置，或插件用 `@扩展标识` 表示。
 *
 * 缺失 extensionId 时退回 `@plugin`——宁可显示一个不精确的来源，也不要显示空徽标
 * 让人以为这个工具没有归属。
 */
export function toolOriginLabel(origin: { kind?: string, extensionId?: string } | undefined): string {
  const kind = typeof origin?.kind === 'string' ? origin.kind.trim() : ''
  if (kind === '' || kind === 'core') {
    return 'Core'
  }
  const extensionId = typeof origin?.extensionId === 'string' ? origin.extensionId.trim() : ''
  return `@${extensionId || (kind === 'plugin' ? 'plugin' : kind)}`
}

/**
 * purposeToolExposure 物化每个用途的**生效**工具集合：覆盖里的 allowed 存在就以它
 * 为准（空列表也是有效答案），否则用用途自己声明的默认列表。
 */
export function purposeToolExposure(
  purposes: readonly AIToolPurposeLike[],
  overrides: Record<string, AIPurposeOverride> | undefined
): Map<string, string[]> {
  const exposure = new Map<string, string[]>()
  for (const purpose of purposes ?? []) {
    const id = typeof purpose?.id === 'string' ? purpose.id.trim() : ''
    if (!id) continue
    exposure.set(id, selectedPurposeTools(purpose.declaredTools, overrides?.[id]))
  }
  return exposure
}

function sameToolSet(left: readonly string[], right: readonly string[]): boolean {
  if (left.length !== right.length) return false
  return left.every((name, index) => name === right[index])
}

/**
 * setPurposeTool 是「勾选/取消一个工具」的唯一写入口，也是自动收敛的核心。
 *
 * 它先把生效集合物化（覆盖优先、否则声明），再增删这个名字，然后与声明集合比较：
 * 一旦结果与声明完全一致，就把 tools 字段整个删掉——运营者「勾上再取消」之后配置
 * 必须回到未覆盖状态，而不是留下一个与声明同值、却永远显示「已自定义」的覆盖。
 * failurePosture 不属于工具语义，原样保留。
 */
export function setPurposeTool(
  declared: readonly string[] | undefined,
  override: AIPurposeOverride | undefined,
  name: string,
  enabled: boolean
): AIPurposeOverride | undefined {
  const tool = typeof name === 'string' ? name.trim() : ''
  const declaredNames = normalizePurposeToolNames(declared)
  const effective = new Set(selectedPurposeTools(declared, override))
  if (tool) {
    if (enabled) {
      effective.add(tool)
    } else {
      effective.delete(tool)
    }
  }
  const next = [...effective].sort()
  const posture = typeof override?.failurePosture === 'string' ? override.failurePosture.trim() : ''
  const result: AIPurposeOverride = {}
  if (!sameToolSet(next, declaredNames)) {
    result.tools = { allowed: next }
  }
  if (posture) {
    result.failurePosture = posture
  }
  return purposeOverrideIsEmpty(result) ? undefined : result
}

/**
 * resetPurposeTools 把某个用途的工具覆盖还原成「跟随声明」。
 *
 * 只移除 tools，failurePosture 原样保留：工具可见性与失败姿态是两个独立决定，
 * 恢复其中一个不该顺手清掉另一个。结果为空时返回 undefined，让上层删除整个用途键
 * （空覆盖在服务端归一化时同样会被丢弃）。
 */
export function resetPurposeTools(override: AIPurposeOverride | undefined): AIPurposeOverride | undefined {
  const posture = typeof override?.failurePosture === 'string' ? override.failurePosture.trim() : ''
  return posture ? { failurePosture: posture } : undefined
}

/**
 * removeToolFromAllPurposes 把一个工具从每个用途的生效集合里摘掉，并对每个用途单独
 * 自动收敛：某个用途的覆盖因此变回「与声明一致」时，它的 tools 字段就该消失。
 *
 * 不在当前用途目录里的覆盖会被原样保留——本函数只处理看得见的用途，不静默删除
 * 看不见的配置。
 */
export function removeToolFromAllPurposes(
  purposes: readonly AIToolPurposeLike[],
  overrides: Record<string, AIPurposeOverride> | undefined,
  name: string
): Record<string, AIPurposeOverride> | undefined {
  const tool = typeof name === 'string' ? name.trim() : ''
  if (!tool) {
    return prunePurposeOverrides(overrides)
  }
  const result: Record<string, AIPurposeOverride> = {}
  const visited = new Set<string>()
  for (const purpose of purposes ?? []) {
    const id = typeof purpose?.id === 'string' ? purpose.id.trim() : ''
    if (!id || visited.has(id)) continue
    visited.add(id)
    const next = setPurposeTool(purpose.declaredTools, overrides?.[id], tool, false)
    if (next) {
      result[id] = next
    }
  }
  for (const [rawId, override] of Object.entries(overrides ?? {})) {
    const id = rawId.trim()
    if (!id || visited.has(id)) continue
    result[id] = override
  }
  return prunePurposeOverrides(result)
}

/**
 * referencedButUnregistered 反向列出「白名单里有、登记表里没有」的工具名。
 *
 * 服务端对这类名字是静默跳过的：模型看不到它，运营者却在界面上看到它被勾着。
 * 把「谁在引用」一起返回，界面才能给出可操作的移除入口，而不是只报一句有脏名字。
 */
export function referencedButUnregistered(
  purposes: readonly AIToolPurposeLike[],
  overrides: Record<string, AIPurposeOverride> | undefined,
  registeredNames: Iterable<string>
): { name: string, purposes: string[] }[] {
  const registered = new Set<string>()
  for (const entry of registeredNames) {
    const name = typeof entry === 'string' ? entry.trim() : ''
    if (name) registered.add(name)
  }
  const referenced = new Map<string, Set<string>>()
  for (const [purposeId, names] of purposeToolExposure(purposes, overrides)) {
    for (const name of names) {
      if (registered.has(name)) continue
      const owners = referenced.get(name) ?? new Set<string>()
      owners.add(purposeId)
      referenced.set(name, owners)
    }
  }
  return [...referenced.entries()]
    .map(([name, owners]) => ({ name, purposes: [...owners].sort() }))
    .sort((left, right) => (left.name < right.name ? -1 : left.name > right.name ? 1 : 0))
}

/**
 * purposeToolSupportState 回答「这个用途现在能不能真的调用工具」，三态而不是布尔：
 * 没有绑定供应商、绑定的供应商声明不支持工具，是两个完全不同的修复动作。
 */
export function purposeToolSupportState(
  purpose: AIToolSupportPurposeLike,
  settings: AIToolSupportSettingsLike
): 'ok' | 'no-profile' | 'no-tool-support' {
  const costClass = typeof purpose?.costClass === 'string' ? purpose.costClass.trim() : ''
  const profileId = costClass ? (settings?.costClassProfiles?.[costClass] ?? '').trim() : ''
  if (!profileId) return 'no-profile'
  const profile = (settings?.profiles ?? []).find(entry => entry.id === profileId)
  if (!profile) return 'no-profile'
  // 缺省视为支持：只有供应商明确声明 false 时才认为它不能调用工具。
  if (profile.supportsTools === false) return 'no-tool-support'
  return 'ok'
}

/**
 * filterTools 是工具清单的搜索与来源筛选。匹配 name / description / 各语言说明 /
 * 扩展标识，大小写不敏感：运营者通常记得「那个隐藏内容的工具」，而不是精确工具名。
 */
export function filterTools<T extends AIToolDisplayLike>(
  tools: readonly T[],
  query: string,
  origin: AIToolFilterOrigin
): T[] {
  const needle = typeof query === 'string' ? query.trim().toLowerCase() : ''
  return (tools ?? []).filter((tool) => {
    const kind = typeof tool.origin?.kind === 'string' ? tool.origin.kind.trim() : ''
    if (origin === 'core' && kind !== 'core') return false
    if (origin === 'plugin' && kind !== 'plugin') return false
    if (!needle) return true
    const haystack = [
      tool.name,
      tool.description,
      tool.origin?.extensionId,
      ...Object.values(tool.descriptionI18n ?? {})
    ]
    return haystack.some(value => typeof value === 'string' && value.toLowerCase().includes(needle))
  })
}

