/**
 * 公共第三方登录 catalog：只读 Host GET /auth/providers。
 * 可执行状态以 activatedOperations 为准；展示名/图标由插件经 Host catalog 注入。
 * Core 前端不得硬编码 GitHub 等供应商品牌或可执行状态。
 */

import { normalizeAuthReturnPath } from '~/utils/identity/authReturn'

export type AuthProviderKind = 'auth' | 'recovery'

export type AuthProviderOperation = 'login' | 'registration' | 'link'

export type PublicAuthProvider = {
  id: string
  kind: AuthProviderKind | string
  contractVersion: string
  priority: number
  operations: string[]
  activatedOperations: string[]
  ownerExtensionId?: string
  /** Host 已按 Accept-Language 解析的插件展示名。 */
  label?: string
  /** 插件声明的 Iconify 名称；空则 Host 壳用通用图标。 */
  icon?: string
}

export type AuthProviderStartResult = {
  providerId: string
  operation: string
  status: 'continue' | 'redirect' | 'challenge' | string
  correlationId: string
  continueToken?: string
  redirectUrl?: string
  challengeKind?: string
}

export type AuthProviderDisplay = {
  id: string
  /** 已解析的展示名（插件 → Host catalog）。 */
  label: string
  icon: string
  activatedOperations: AuthProviderOperation[]
}

const AUTH_OPERATION_SET = new Set<AuthProviderOperation>(['login', 'registration', 'link'])
/** Host 壳通用图标；品牌图标必须由插件 catalog.icon 提供。 */
const HOST_GENERIC_PROVIDER_ICON = 'i-lucide-key-round'

function asProviderList(value: unknown): PublicAuthProvider[] {
  if (!Array.isArray(value)) {
    return []
  }
  const out: PublicAuthProvider[] = []
  for (const item of value) {
    if (!item || typeof item !== 'object') {
      continue
    }
    const row = item as Record<string, unknown>
    const id = typeof row.id === 'string' ? row.id.trim() : ''
    if (!id) {
      continue
    }
    const operations = Array.isArray(row.operations)
      ? row.operations.filter((op): op is string => typeof op === 'string')
      : []
    const activatedOperations = Array.isArray(row.activatedOperations)
      ? row.activatedOperations.filter((op): op is string => typeof op === 'string')
      : []
    const entry: PublicAuthProvider = {
      id,
      kind: typeof row.kind === 'string' ? row.kind : 'auth',
      contractVersion: typeof row.contractVersion === 'string' ? row.contractVersion : '',
      priority: typeof row.priority === 'number' ? row.priority : 0,
      operations,
      activatedOperations
    }
    if (typeof row.ownerExtensionId === 'string' && row.ownerExtensionId.trim()) {
      entry.ownerExtensionId = row.ownerExtensionId.trim()
    }
    if (typeof row.label === 'string' && row.label.trim()) {
      entry.label = row.label.trim()
    }
    if (typeof row.icon === 'string' && row.icon.trim()) {
      entry.icon = row.icon.trim()
    }
    out.push(entry)
  }
  return out
}

/**
 * 展示元数据：仅消费 Host catalog 注入字段。
 * 禁止按 provider id / ownerExtensionId 猜测 GitHub 等品牌。
 */
export function authProviderDisplayMeta(
  provider: PublicAuthProvider,
  fallbackLabel: string
): AuthProviderDisplay {
  const activatedOperations = provider.activatedOperations
    .map(op => op.trim().toLowerCase())
    .filter((op): op is AuthProviderOperation => AUTH_OPERATION_SET.has(op as AuthProviderOperation))

  const label = (provider.label || '').trim() || fallbackLabel
  const icon = (provider.icon || '').trim() || HOST_GENERIC_PROVIDER_ICON

  return {
    id: provider.id,
    label,
    icon,
    activatedOperations
  }
}

export function providerSupportsOperation(
  provider: PublicAuthProvider,
  operation: AuthProviderOperation
): boolean {
  return provider.activatedOperations
    .map(op => op.trim().toLowerCase())
    .includes(operation)
}

export function newAuthCorrelationId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `corr-${Date.now()}-${Math.random().toString(36).slice(2, 12)}`
}

export function useAuthProviders() {
  const { request } = useApiClient()

  // SSR 与客户端共享同一 catalog；失败时 fail closed 为空列表（密码登录仍可用）。
  const { data, pending, error, refresh } = useAsyncData(
    'auth-public-providers',
    async () => {
      try {
        const list = await request<PublicAuthProvider[]>('/auth/providers')
        return asProviderList(list)
      } catch {
        return [] as PublicAuthProvider[]
      }
    },
    { default: () => [] as PublicAuthProvider[] }
  )

  const providers = computed(() => {
    const list = data.value || []
    return [...list]
      .filter(item => item.kind === 'auth' || !item.kind)
      .sort((a, b) => (b.priority - a.priority) || a.id.localeCompare(b.id))
  })

  const loginProviders = computed(() =>
    providers.value.filter(item => providerSupportsOperation(item, 'login'))
  )

  const registrationProviders = computed(() =>
    providers.value.filter(item => providerSupportsOperation(item, 'registration'))
  )

  /** Host 已对当前站点有效开放 link 操作的提供方（账号安全绑定入口）。 */
  const linkProviders = computed(() =>
    providers.value.filter(item => providerSupportsOperation(item, 'link'))
  )

  async function startProvider(
    providerId: string,
    operation: AuthProviderOperation,
    options: { redirectHint?: unknown } = {}
  ): Promise<AuthProviderStartResult> {
    const safeRedirect = normalizeAuthReturnPath(options.redirectHint) || undefined
    const body: Record<string, unknown> = {
      correlationId: newAuthCorrelationId()
    }
    if (safeRedirect) {
      body.redirectHint = safeRedirect
    }
    return request<AuthProviderStartResult>(
      `/auth/providers/${encodeURIComponent(providerId)}/${encodeURIComponent(operation)}/start`,
      {
        method: 'POST',
        body
      }
    )
  }

  /**
   * 启动 OAuth 并浏览器跳转。仅客户端调用。
   * 返回 null 表示已发起跳转；抛错由调用方展示。
   */
  async function redirectToProvider(
    providerId: string,
    operation: AuthProviderOperation,
    options: { redirectHint?: unknown } = {}
  ): Promise<void> {
    const result = await startProvider(providerId, operation, options)
    const redirectUrl = typeof result.redirectUrl === 'string' ? result.redirectUrl.trim() : ''
    if (result.status === 'redirect' && redirectUrl) {
      if (import.meta.client) {
        window.location.assign(redirectUrl)
      }
      return
    }
    throw new Error('auth.provider_unavailable')
  }

  return {
    providers,
    loginProviders,
    registrationProviders,
    linkProviders,
    pending,
    error,
    refresh,
    startProvider,
    redirectToProvider,
    displayMeta: authProviderDisplayMeta
  }
}
