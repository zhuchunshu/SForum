import type { InjectionKey } from 'vue'

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

export type AdminAISettings = {
  schemaVersion: string
  enabled: boolean
  profiles: AdminAIProfile[]
  costClassProfiles: Record<string, string>
  purposeFailurePosture: Record<string, string>
  // reply.systemPrompt 为空表示使用内置默认提示词；一键恢复就是把这里清空。
  reply: { systemPrompt: string }
  autoAction: { enabled: boolean, confidenceThreshold: number, allowedActions: string[] }
  gates: { rateLimitPerMinute: number, extensionDailyQuota: number, userDailyQuota: number, monthlyBudgetMicros: number, toolCallsPerReply?: number }
  redaction: { enabled: boolean, redactEmail: boolean, redactPhone: boolean, redactIp: boolean }
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

export type AdminAISettingsPayload = {
  settings: AdminAISettings
  warnings: string[]
  encryptionEnabled: boolean
  credentialStatus: Record<string, boolean>
  replyBot: AdminAIReplyBot
  /** 内置回复提示词（不含安全尾注）。编辑器用它预填，让默认值可见。 */
  defaultSystemPrompt: string
  /** 始终附加在提示词末尾的防注入声明，只读展示。 */
  safetyAppendix: string
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
}

export type AdminAIDiagnoseResult = {
  ok: boolean
  providerId?: string
  model?: string
  protocol?: string
  reply?: string
  latencyMs?: number
  inputTokens?: number
  outputTokens?: number
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
    reply: { systemPrompt: '' },
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

  // 请求失败时 data 是 undefined，而不是 default 的返回值：所有下游读取都必须
  // 经由这些经过保护的 computed，否则一次 401/500 就会让整页渲染抛错。
  const settings = computed(() => payload.data.value?.settings ?? emptyAdminAISettings())
  const credentialStatus = computed(() => payload.data.value?.credentialStatus ?? {})
  const warnings = computed(() => payload.data.value?.warnings ?? [])
  const encryptionEnabled = computed(() => payload.data.value?.encryptionEnabled ?? false)
  const replyBot = computed<AdminAIReplyBot>(() => payload.data.value?.replyBot ?? { configured: false })
  const defaultSystemPrompt = computed(() => payload.data.value?.defaultSystemPrompt ?? '')
  const safetyAppendix = computed(() => payload.data.value?.safetyAppendix ?? '')
  const loadFailed = computed(() => Boolean(payload.error.value))
  const loading = computed(() => payload.pending.value)

  const siteUsage = computed(() => usage.data.value?.usage?.site ?? emptyAdminAIUsage())
  const budgetMicros = computed(() => usage.data.value?.budgetMicros ?? 0)
  const executions_ = computed(() => executions.data.value?.items ?? [])
  const executionsPending = computed(() => executions.pending.value)

  async function refresh() {
    await Promise.all([payload.refresh(), usage.refresh(), executions.refresh()])
  }

  async function saveSettings(next: AdminAISettings) {
    busy.value = 'settings'
    try {
      await request('/admin/ai/settings', { method: 'PUT', body: next })
      toast.add({ color: 'success', icon: 'i-lucide-save', title: t('admin.ai.saved'), duration: 10000 })
      await refresh()
      return true
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
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
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
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
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
      return false
    } finally {
      busy.value = ''
    }
  }

  // 诊断结果只保留在内存：它是一次性的探测，不需要持久化，也不该在刷新后
  // 让人误以为是最新结论。
  const diagnoseResult = ref<AdminAIDiagnoseResult | null>(null)

  async function diagnose(message: string) {
    busy.value = 'diagnose'
    diagnoseResult.value = null
    try {
      const result = await request<AdminAIDiagnoseResult>('/admin/ai/diagnose', {
        method: 'POST',
        body: { message }
      })
      diagnoseResult.value = result
      if (result.ok) {
        toast.add({ color: 'success', icon: 'i-lucide-circle-check', title: t('admin.ai.diagnose.okToast'), duration: 10000 })
      }
      await executions.refresh()
      return result
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
      return null
    } finally {
      busy.value = ''
    }
  }

  return {
    settings,
    credentialStatus,
    warnings,
    encryptionEnabled,
    replyBot,
    defaultSystemPrompt,
    safetyAppendix,
    loadFailed,
    loading,
    siteUsage,
    budgetMicros,
    executions: executions_,
    executionsPending,
    diagnoseResult,
    diagnose,
    busy,
    refresh,
    saveSettings,
    resetSettings,
    saveCredential
  }
}
