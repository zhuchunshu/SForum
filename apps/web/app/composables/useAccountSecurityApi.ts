// 账号安全 / 登录设备管理 API 客户端。
// opaque 会话标识（id）仅用于指定「下线哪一条」，不是认证凭证，无法用于登录。

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

  return {
    listSessions,
    revokeSession,
    revokeOtherSessions,
    listAPITokens,
    createAPIToken,
    revokeAPIToken,
    rotateAPIToken
  }
}
