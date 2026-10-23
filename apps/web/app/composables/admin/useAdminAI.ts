import type { InjectionKey } from 'vue'
import type { AdminAILimits } from '~/utils/admin/adminAI'
import { DEFAULT_AI_LIMITS } from '~/utils/admin/adminAI'

export type AdminAIProtocol = 'openai-chat' | 'anthropic-messages'
export type AdminAICostClass = 'economy' | 'standard' | 'premium'

export type AdminAIProfile = {
  id: string
  label: string
  protocol: AdminAIProtocol
  baseUrl: string
  apiKeyRef: string
  model: string
  costClass: AdminAICostClass
  enabled: boolean
  price: { inputPerMillionMicros: number, outputPerMillionMicros: number }
  defaults: { maxTokens: number, temperature?: number, timeoutMs: number }
  // 端点是否支持工具调用。缺省（undefined）视为支持；只有明确不支持时才写 false。
  supportsTools?: boolean
}

// AdminAIPurposeTools 是某个用途的工具白名单覆盖。字段存在（哪怕空列表）即表示
// 运营者接管了这一项；字段缺失表示沿用该用途声明的默认列表。
export type AdminAIPurposeTools = { allowed: string[] }

// AdminAIPurposeOverride 是某个用途的站点覆盖（与后端 ai.PurposeSettings 对应）。
export type AdminAIPurposeOverride = {
  tools?: AdminAIPurposeTools
  failurePosture?: string
}

// AdminAIValidatedPurpose 是控制台的用途目录条目：声明了什么，以及当前生效什么。
// declaredTools 与 toolsAllowed 并列，让「覆盖成了什么」与「原本是什么」都能看见。
export type AdminAIValidatedPurpose = {
  id: string
  label: { default: string, 'en-US'?: string }
  description: { default: string, 'en-US'?: string }
  costClass: AdminAICostClass
  resourceType: string
  promptVersion: string
  blastRadius: 'single' | 'batch' | 'site' | string
  failurePosture: string
  failurePostureOverridden: boolean
  declaredTools: string[]
  toolsAllowed: string[]
  toolsOverridden: boolean
}

export type AdminAISettings = {
  schemaVersion: string
  enabled: boolean
  profiles: AdminAIProfile[]
  costClassProfiles: Record<string, string>
  purposeFailurePosture: Record<string, string>
  // purposes 是按用途的覆盖。缺省即沿用各用途自己的声明；它是 reply.tools 特例的
  // 替代品（旧字段由服务端在归一化时迁移进来）。
  purposes?: Record<string, AdminAIPurposeOverride>
  // toolDisplayDescriptions 是仅控制台展示的工具说明覆盖（工具名 -> locale -> 文本）。
  // 它不进模型请求：模型读的是工具自己的 description。旧文档可能没有这个字段。
  toolDisplayDescriptions?: Record<string, Record<string, string>>
  // reply.systemPrompt 为空表示使用内置默认提示词；一键恢复就是把这里清空。
  // triggerMode 决定机器人何时回复：mention（默认）/ autonomous（AI 自主判定
  // 是否参与）/ off（停用自动触发）。旧文档可能没有这个字段。
  reply: { systemPrompt: string, triggerMode?: 'mention' | 'autonomous' | 'off' }
  autoAction: { enabled: boolean, confidenceThreshold: number, allowedActions: string[] }
  gates: { rateLimitPerMinute: number, extensionDailyQuota: number, userDailyQuota: number, monthlyBudgetMicros: number, toolCallsPerReply?: number }
  redaction: { enabled: boolean, redactEmail: boolean, redactPhone: boolean, redactIp: boolean }
  // multimodal 是图片理解开关。旧文档可能没有这个字段（引入前保存的）。
  multimodal?: { enabled: boolean }
  revision: number
  updatedAt?: string
}

// ReplyBotInfo 描述当前充当 AI 助手的账号。configured=false 表示站内还没有
// 机器人账号，回复功能不会生效。
export type AdminAIReplyBot = {
  userId?: number
  username?: string
  configured: boolean
}

export type AdminAIToolOrigin = {
  kind: 'core' | 'plugin' | string
  extensionId?: string
  version?: string
}

// AdminAIToolParameterSummary 是参数 schema 的展示摘要（不是完整 JSON Schema）。
export type AdminAIToolParameterSummary = {
  names?: string[]
  required?: string[]
  truncated?: boolean
}

// AdminAIToolAvailability 是「站内有哪些 AI 工具」的条目。它与用途白名单是两个
// 问题：这里是全局事实，白名单是某个用途的生效配置。
//
// description 是模型可见文本；descriptionI18n 是作者/内置提供的界面文案。
// 运营者在 AI 设置里的 toolDisplayDescriptions 优先级更高，由控制台自行合并。
export type AdminAIToolAvailability = {
  name: string
  description?: string
  descriptionI18n?: Record<string, string>
  /** 插件声明的单次调用超时上限；Core 内置工具没有这一层声明。 */
  timeoutMs?: number
  /** 插件管理设置页路径（不含 /control-panel）；只有真的声明了设置页才有。 */
  settingsPath?: string
  parameters?: AdminAIToolParameterSummary
  origin: AdminAIToolOrigin
}

export type AdminAISettingsPayload = {
  settings: AdminAISettings
  warnings: string[]
  encryptionEnabled: boolean
  credentialStatus: Record<string, boolean>
  replyBot: AdminAIReplyBot
  /** 站内已声明的 AI 用途清单（声明 + 当前生效参数）。 */
  purposes: AdminAIValidatedPurpose[]
  /** 站内已注册的全部 AI 工具（Core 内置 + 已启用插件贡献）。旧载荷可能没有。 */
  availableTools?: AdminAIToolAvailability[]
  /** 内置回复提示词（不含安全尾注）。编辑器用它预填，让默认值可见。 */
  defaultSystemPrompt: string
  /** 始终附加在提示词末尾的防注入声明，只读展示。 */
  safetyAppendix: string
  /** 服务端当前生效的取值天花板。旧载荷可能没有，此时用前端兜底值。 */
  limits?: AdminAILimits
}

export type AdminAIUsageSnapshot = {
  minuteCalls: number
  dayCalls: number
  dayTokens: number
  monthSpendMicros: number
}

export type AdminAIUsagePayload = {
  usage: { site: AdminAIUsageSnapshot, extension: AdminAIUsageSnapshot, user: AdminAIUsageSnapshot }
  budgetMicros: number
  warnings: string[]
}

export type AdminAIExecution = {
  id: number
  createdAt: string
  purpose: string
  costClass: string
  callerExtensionId: string
  profileId: string
  protocol: string
  model: string
  status: 'succeeded' | 'failed' | 'denied'
  gateReason: string
  gateScope: string
  promptVersion: string
  configRevision: number
  latencyMs: number
  inputTokens: number
  outputTokens: number
  cachedTokens: number
  spendMicros: number
  cacheHit: boolean
  errorSummary: string
  // 归属对象。一次 AI run 的每一次模型调用共享同一个 (type="ai.run", id=runId)，
  // 回放据此把执行记录聚成一条时间线。
  //
  // 两者在 Go 侧都是 `omitempty`：值为空串时字段**不会出现**在 JSON 里，因此这里
  // 必须是可选而不是 `string`——否则类型会声称「一定存在」，而 A3 之前的历史记录
  // 恰恰没有它。`''` 与 `undefined` 都表示「不属于任何 run」，判定统一走
  // `executionExpandable`。
  resourceType?: string
  resourceId?: string
}

// AdminAIReplyOutcome 是一条回复的最终裁决：这条触发最后有没有变成一条评论。
//
// 它与 AdminAIExecution 是两个问题：执行记录记每一次模型调用，这张表记一次生成
// 的结果。status=succeeded 却看不到回复时，答案在这里而不在调用里。
export type AdminAIReplyOutcome = {
  id: number
  createdAt: string
  topicId: number
  parentCommentId?: number
  triggerCommentId?: number
  botUserId?: number
  runId?: string
  commentId?: number
  outcome: 'posted' | 'empty_reply' | 'tool_loop_stuck' | 'unavailable' | 'publish_failed' | 'failed'
  reason?: string
  detail?: string
  modelCalls?: number
  toolCalls?: number
  inputTokens?: number
  outputTokens?: number
}

// AdminAIModelList 是一次只读的模型列表探测结论。供应商层面的失败以 reason + error
// 返回，而不是抛出：地址能连上但密钥无效，是运营者要读到的信息。
export type AdminAIModelList = {
  profileId?: string
  protocol?: string
  models: string[]
  count: number
  /** 供应商还有更多条目未展示（超过上限，或供应商自身分页未取完）。 */
  truncated?: boolean
  reason?: string
  error?: string
}

// AdminAIModelProbe 是探测参数。三个可选字段都只在这一次请求里存在：带上它们，
// 运营者才能在保存之前就看到模型列表；缺省时服务端使用已保存的地址与密钥。
export type AdminAIModelProbe = {
  profileId: string
  baseUrl?: string
  protocol?: AdminAIProtocol
  apiKey?: string
}

export type AdminAIDiagnoseResult = {
  ok: boolean
  providerId?: string
  model?: string
  protocol?: string
  /** 实际探测的成本等级绑定。运营者可能只绑定了其中一档，展示它避免误判。 */
  costClass?: string
  /** 模型返回的正文。成功但为空时是空字符串（不是缺失），界面据此提示而非渲染空白条。 */
  reply: string
  latencyMs?: number
  inputTokens?: number
  outputTokens?: number
  /** 协议无关的停止原因。正文为空时它解释「为什么一个字都没有」。 */
  stopReason?: string
  reason?: string
  error?: string
}

// MICRO_PER_UNIT 与后端 ai.MicroPerUnit 保持一致：1 个货币单位 = 1_000_000 micro。
export const MICRO_PER_UNIT = 1_000_000

// ADMIN_AI_KEY 让页面壳成为唯一的数据所有者：三个页签共用一个上下文，
// 而不是各自再调一次 useAsyncData。同一个 key 被多个组件使用时，Nuxt 会在
// 组件卸载时清理数据，页签切换因此可能出现空数据与白屏。
export type AdminAIContext = ReturnType<typeof useAdminAI>
export const ADMIN_AI_KEY: InjectionKey<AdminAIContext> = Symbol('sforum.admin.ai')

export function emptyAdminAISettings(): AdminAISettings {
  return {
    schemaVersion: 'sforum.ai.settings@1',
    enabled: false,
    profiles: [],
    costClassProfiles: {},
    purposeFailurePosture: {},
    reply: { systemPrompt: '', triggerMode: 'mention' },
    autoAction: { enabled: false, confidenceThreshold: 0.9, allowedActions: [] },
    gates: { rateLimitPerMinute: 0, extensionDailyQuota: 0, userDailyQuota: 0, monthlyBudgetMicros: 0, toolCallsPerReply: 3 },
    redaction: { enabled: true, redactEmail: true, redactPhone: true, redactIp: true },
    revision: 0
  }
}

export function emptyAdminAIUsage(): AdminAIUsageSnapshot {
  return { minuteCalls: 0, dayCalls: 0, dayTokens: 0, monthSpendMicros: 0 }
}

export function useAdminAI() {
  const { request } = useApiClient()
  const toast = useToast()
  const { t } = useI18n()
  const busy = ref('')

  const payload = useAsyncData<AdminAISettingsPayload>('admin-ai-settings', () => request('/admin/ai/settings'))
  const usage = useAsyncData<AdminAIUsagePayload>('admin-ai-usage', () => request('/admin/ai/usage'))
  const executions = useAsyncData<{ items: AdminAIExecution[] }>('admin-ai-executions', () => request('/admin/ai/executions?limit=50'))
  // 回复结果与执行记录并列拉取：页面壳是唯一数据所有者，页签只渲染。
  const replyOutcomePayload = useAsyncData<{ items: AdminAIReplyOutcome[] }>('admin-ai-reply-outcomes', () => request('/admin/ai/reply-outcomes?limit=50'))
  // 页面壳必须在 SSR 渲染前等这三个请求完成：useAdminAI 是同步组合式函数，若不
  // await，SSR 首渲染拿到的 settings/purposes/usage 全是空兜底，而客户端水合时
  // payload 已从 __NUXT_DATA__ 同步恢复——两边的分支（v-if/v-else、v-for 数量）
  // 全部错位，hydration mismatch 由此而来（所有 AI 页签都受影响，不只是用途页签）。
  const ready = Promise.all([payload, usage, executions, replyOutcomePayload]).then(() => undefined)

  // 请求失败时 data 是 undefined，而不是 default 的返回值：所有下游读取都必须
  // 经由这些经过保护的 computed，否则一次 401/500 就会让整页渲染抛错。
  const settings = computed(() => payload.data.value?.settings ?? emptyAdminAISettings())
  const credentialStatus = computed(() => payload.data.value?.credentialStatus ?? {})
  const warnings = computed(() => payload.data.value?.warnings ?? [])
  const encryptionEnabled = computed(() => payload.data.value?.encryptionEnabled ?? false)
  const replyBot = computed<AdminAIReplyBot>(() => payload.data.value?.replyBot ?? { configured: false })
  const purposes = computed<AdminAIValidatedPurpose[]>(() => payload.data.value?.purposes ?? [])
  // 旧载荷没有 availableTools：空清单让用途页签退化成「手动输入工具名」，
  // 那是可用但更笨的界面，不该让整页打不开。
  const availableTools = computed<AdminAIToolAvailability[]>(() => payload.data.value?.availableTools ?? [])
  const defaultSystemPrompt = computed(() => payload.data.value?.defaultSystemPrompt ?? '')
  // 上限的唯一权威是服务端：界面据此渲染 min/max，不再各自硬编码一份数字。
  const limits = computed<AdminAILimits>(() => payload.data.value?.limits ?? DEFAULT_AI_LIMITS)
  const safetyAppendix = computed(() => payload.data.value?.safetyAppendix ?? '')
  // 保存失败时的字段级消息（服务端 422 的 data.fields）。它按载荷路径索引，
  // 例如 profiles[0].defaults.maxTokens / gates.rateLimitPerMinute。
  const fieldErrors = ref<Record<string, string[]>>({})
  const loadFailed = computed(() => Boolean(payload.error.value))
  const loading = computed(() => payload.pending.value)

  const siteUsage = computed(() => usage.data.value?.usage?.site ?? emptyAdminAIUsage())
  const budgetMicros = computed(() => usage.data.value?.budgetMicros ?? 0)
  const executions_ = computed(() => executions.data.value?.items ?? [])
  const executionsPending = computed(() => executions.pending.value)
  const replyOutcomes_ = computed(() => replyOutcomePayload.data.value?.items ?? [])
  const replyOutcomesPending = computed(() => replyOutcomePayload.pending.value)

  async function refresh() {
    await Promise.all([payload.refresh(), usage.refresh(), executions.refresh(), replyOutcomePayload.refresh()])
  }

  // reportFailure 把失败说清楚：优先用服务端 envelope 的 message，其次按
  // reason 兜底。以前这里直接 `${error}`，运营者看到的只是 ofetch 的
  // 「FetchError: ... 422」，完全不知道是哪个字段。
  function reportFailure(error: unknown, fallbackKey: string) {
    fieldErrors.value = apiErrorFields(error)
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t(fallbackKey),
      duration: 0
    })
  }

  // 任意一次成功写入都清空上一次的字段级错误：留着会让已修好的字段继续报错。
  function clearFieldErrors() {
    fieldErrors.value = {}
  }

  async function saveSettings(next: AdminAISettings) {
    busy.value = 'settings'
    clearFieldErrors()
    try {
      await request('/admin/ai/settings', { method: 'PUT', body: next })
      toast.add({ color: 'success', icon: 'i-lucide-save', title: t('admin.ai.saved'), duration: 10000 })
      await refresh()
      return true
    } catch (error) {
      reportFailure(error, 'admin.ai.saveFailed')
      return false
    } finally {
      busy.value = ''
    }
  }

  async function resetSettings() {
    busy.value = 'reset'
    try {
      await request('/admin/ai/settings/reset', { method: 'POST', body: {} })
      toast.add({ color: 'success', icon: 'i-lucide-rotate-ccw', title: t('admin.ai.resetDone'), duration: 10000 })
      await refresh()
    } catch (error) {
      reportFailure(error, 'admin.ai.resetFailed')
    } finally {
      busy.value = ''
    }
  }

  // 密钥只在此处提交一次；提交后界面只显示「已配置」，永远读不回明文。
  async function saveCredential(profileId: string, apiKey: string) {
    busy.value = `credential:${profileId}`
    try {
      const result = await request<{ profileId: string, version: number }>(`/admin/ai/profiles/${encodeURIComponent(profileId)}/credential`, {
        method: 'PUT',
        body: { apiKey }
      })
      toast.add({ color: 'success', icon: 'i-lucide-key-round', title: t('admin.ai.credentialSaved', { version: result.version }), duration: 10000 })
      await refresh()
      return true
    } catch (error) {
      reportFailure(error, 'admin.ai.credentialFailed')
      return false
    } finally {
      busy.value = ''
    }
  }

  // 诊断结果只保留在内存：它是一次性的探测，不需要持久化，也不该在刷新后
  // 让人误以为是最新结论。
  const diagnoseResult = ref<AdminAIDiagnoseResult | null>(null)

  // fetchModels 用当前的地址与密钥探测供应商的模型列表。它是一次只读探测：
  // 不产生补全调用、不计费、不写执行记录。请求里携带的密钥明文只用于这一次探测，
  // 不经过任何持久化路径（写入密钥仍然只走 saveCredential）。
  //
  // HTTP 层失败（401/403/422/网络）在这里提示并返回 null；供应商层面的失败会作为
  // 结果返回，交给调用方就地在字段旁解释——那是运营者要读的信息，不是一个弹窗。
  async function fetchModels(probe: AdminAIModelProbe): Promise<AdminAIModelList | null> {
    busy.value = `models:${probe.profileId}`
    try {
      return await request<AdminAIModelList>(`/admin/ai/profiles/${encodeURIComponent(probe.profileId)}/models`, {
        method: 'POST',
        body: { baseUrl: probe.baseUrl, protocol: probe.protocol, apiKey: probe.apiKey }
      })
    } catch (error) {
      reportFailure(error, 'admin.ai.modelListFailed')
      return null
    } finally {
      busy.value = ''
    }
  }

  async function diagnose(message: string) {
    busy.value = 'diagnose'
    diagnoseResult.value = null
    try {
      const result = await request<AdminAIDiagnoseResult>('/admin/ai/diagnose', {
        method: 'POST',
        body: { message }
      })
      diagnoseResult.value = result
      // 调用成功但正文为空不是「连接正常」：推理模型的预算被思考耗尽时就会这样。
      // 提示语必须区分，否则运营者看到成功提示却找不到回复内容。
      if (result.ok && !result.reason) {
        toast.add({ color: 'success', icon: 'i-lucide-circle-check', title: t('admin.ai.diagnose.okToast'), duration: 10000 })
      } else if (result.reason === 'empty_reply') {
        toast.add({ color: 'warning', icon: 'i-lucide-message-square-warning', title: t('admin.ai.diagnose.emptyReplyToast'), duration: 10000 })
      }
      await executions.refresh()
      return result
    } catch (error) {
      reportFailure(error, 'admin.ai.diagnose.failed')
      return null
    } finally {
      busy.value = ''
    }
  }

  return {
    ready,
    settings,
    credentialStatus,
    warnings,
    encryptionEnabled,
    replyBot,
    purposes,
    availableTools,
    defaultSystemPrompt,
    safetyAppendix,
    limits,
    fieldErrors,
    loadFailed,
    loading,
    siteUsage,
    budgetMicros,
    executions: executions_,
    executionsPending,
    replyOutcomes: replyOutcomes_,
    replyOutcomesPending,
    diagnoseResult,
    diagnose,
    fetchModels,
    busy,
    refresh,
    saveSettings,
    resetSettings,
    saveCredential
  }
}

// ADMIN_AI_RUN_NONE 是「没有选中任何 run」时的键后缀。它让「没选」与「选了一个
// 空串 runId」共用一个缓存槽，而不是各自开一个永远不会再用的槽。
const ADMIN_AI_RUN_NONE = 'none'

/**
 * useAdminAIRunTimeline 读取一次 AI run 的全部执行记录，按时间正序。
 *
 * 它是 A4 的读面：`GET /admin/ai/runs/:runId` 返回 `{ items: AIExecution[] }`，与
 * `listExecutions` 同形。回放是查询的组合，不是新的持久化面——执行记录里没有工具
 * 调用明细（那些在日志里），所以这里也只返回执行记录，界面不得补造。
 *
 * `runId` 为空时不发请求。A3 之前的历史记录没有 run 归属，服务端对空 runId 也只会
 * 返回空列表；与其让一次必然为空的往返出现在网络面板里，不如在这里就地短路。
 */
export function useAdminAIRunTimeline(runId: Ref<string | null> | ComputedRef<string | null>) {
  const { request } = useApiClient()

  const normalizedRunId = computed(() => (runId.value ?? '').trim())

  const payload = useAsyncData<{ items: AdminAIExecution[] }>(
    () => `admin-ai-run-timeline:${normalizedRunId.value || ADMIN_AI_RUN_NONE}`,
    async () => {
      const id = normalizedRunId.value
      if (!id) return { items: [] }
      // runId 来自 URL/界面状态，只作为路径段参与请求（服务端参数化查询），
      // 这里做一次编码以免 id 里的字符改变路径结构。
      return await request<{ items: AdminAIExecution[] }>(`/admin/ai/runs/${encodeURIComponent(id)}`)
    },
    { watch: [normalizedRunId], default: () => ({ items: [] }) }
  )

  // 请求失败时 data 是 undefined：下游一律经由这个受保护的 computed 读取，
  // 一次 401/500 不会让页签渲染抛错。
  const items = computed<AdminAIExecution[]>(() => payload.data.value?.items ?? [])
  const pending = computed(() => payload.pending.value)
  const loadFailed = computed(() => Boolean(payload.error.value))

  return {
    items,
    pending,
    loadFailed,
    refresh: payload.refresh
  }
}
