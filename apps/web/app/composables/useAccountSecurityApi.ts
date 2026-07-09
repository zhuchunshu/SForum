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

  return { listSessions, revokeSession, revokeOtherSessions }
}
