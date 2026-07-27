// 账号安全 / 登录设备管理 API 客户端。
// opaque 会话标识（id）仅用于指定「下线哪一条」，不是认证凭证，无法用于登录。
// M4B：外部身份列表/解绑、本地密码设置（session-bound recent-auth 由 Host 门控）。

export type LoginSession = {
  id: string
  deviceName: string
  browser: string
  os: string
  ipPrefix: string
  createdAt: string
  lastSeenAt: string
  isCurrent: boolean
  revokedAt?: string | null
  revokeReason?: string
}

export type LoginSessionList = {
  items: LoginSession[]
  total: number
  page: number
  perPage: number
}

export type RevokeOthersResult = {
  revoked: number
}

// 个人访问令牌（PAT）公开视图；明文仅在 create/rotate 响应的 token 字段出现一次。
export type APIToken = {
  id: number
  publicId: string
  name: string
  scopes: string[]
  prefix: string
  lastUsedAt?: string | null
  expiresAt?: string | null
  revokedAt?: string | null
  createdAt: string
}

export type CreatedAPIToken = APIToken & {
  token: string
}

/** Host 返回的 redacted 外部身份；禁止期望 subject/digest/token 字段。 */
export type ExternalIdentityStatus = 'active' | 'inert' | 'erased' | string

export type ExternalIdentityItem = {
  linkId: number
  providerId: string
  status: ExternalIdentityStatus
  linkedAt?: string | null
  ownerExtensionId?: string
}

export function asExternalIdentityList(value: unknown): ExternalIdentityItem[] {
  if (!Array.isArray(value)) {
    return []
  }
  const out: ExternalIdentityItem[] = []
  for (const item of value) {
    if (!item || typeof item !== 'object') {
      continue
    }
    const row = item as Record<string, unknown>
    const linkId = Number(row.linkId)
    const providerId = typeof row.providerId === 'string' ? row.providerId.trim() : ''
    const status = typeof row.status === 'string' ? row.status.trim() : ''
    if (!Number.isFinite(linkId) || linkId <= 0 || !providerId || !status) {
      continue
    }
    const entry: ExternalIdentityItem = {
      linkId,
      providerId,
      status
    }
    if (typeof row.linkedAt === 'string' && row.linkedAt.trim()) {
      entry.linkedAt = row.linkedAt.trim()
    }
    if (typeof row.ownerExtensionId === 'string' && row.ownerExtensionId.trim()) {
      entry.ownerExtensionId = row.ownerExtensionId.trim()
    }
    // 硬拒绝：任何 raw subject / digest / token 字段不得进入前端状态。
    out.push(entry)
  }
  return out
}

export function useAccountSecurityApi() {
  const { request } = useApiClient()

  // 获取当前用户的活跃设备列表；includeHistory=true 含已下线的历史记录。
  function listSessions(opts: { includeHistory?: boolean; page?: number; perPage?: number } = {}) {
    const params = new URLSearchParams()
    if (opts.includeHistory) params.set('includeHistory', 'true')
    if (opts.page) params.set('page', String(opts.page))
    if (opts.perPage) params.set('perPage', String(opts.perPage))
    const query = params.toString()
    return request<LoginSessionList>(`/auth/sessions${query ? `?${query}` : ''}`)
  }

  // 下线单个设备（越权 sid 由后端 user_id 过滤保证，返回 404 不泄漏归属）。
  function revokeSession(sessionId: string) {
    return request<null>(`/auth/sessions/${encodeURIComponent(sessionId)}`, { method: 'DELETE' })
  }

  // 下线除当前设备外的所有其他设备。
  function revokeOtherSessions() {
    return request<RevokeOthersResult>('/auth/sessions/revoke-others', { method: 'POST' })
  }

  function listAPITokens() {
    return request<{ items: APIToken[] }>('/auth/tokens')
  }

  function createAPIToken(input: { name: string, scopes: string[], expiresAt?: string }) {
    return request<CreatedAPIToken>('/auth/tokens', { method: 'POST', body: input })
  }

  function revokeAPIToken(tokenId: number) {
    return request<null>(`/auth/tokens/${tokenId}`, { method: 'DELETE' })
  }

  function rotateAPIToken(tokenId: number) {
    return request<CreatedAPIToken>(`/auth/tokens/${tokenId}/rotate`, { method: 'POST', body: {} })
  }

  /** 当前用户 redacted 外部身份列表（登录会话必需）。 */
  async function listExternalIdentities() {
    const raw = await request<unknown>('/auth/external-identities')
    return asExternalIdentityList(raw)
  }

  /**
   * 解绑外部身份。Host 在同一事务内执行 last-login-method 与 revision 校验；
   * expectedRevision 可省略（Host 用当前 active revision）。
   */
  function unlinkExternalIdentity(
    linkId: number,
    options: { expectedRevision?: number; requestId?: string } = {}
  ) {
    const body: Record<string, unknown> = {}
    if (typeof options.expectedRevision === 'number' && options.expectedRevision > 0) {
      body.expectedRevision = options.expectedRevision
    }
    if (typeof options.requestId === 'string' && options.requestId.trim()) {
      body.requestId = options.requestId.trim()
    }
    return request<null>(
      `/auth/external-identities/${encodeURIComponent(String(linkId))}`,
      {
        method: 'DELETE',
        body: Object.keys(body).length > 0 ? body : {}
      }
    )
  }

  /**
   * 自助设置/更新本地密码。需要 session-bound recent-auth；
   * external-only 用户首次调用会创建 credential 行。
   */
  function setupPassword(password: string) {
    return request<null>('/auth/password', {
      method: 'POST',
      body: { password }
    })
  }

  return {
    listSessions,
    revokeSession,
    revokeOtherSessions,
    listAPITokens,
    createAPIToken,
    revokeAPIToken,
    rotateAPIToken,
    listExternalIdentities,
    unlinkExternalIdentity,
    setupPassword
  }
}
