export type UserStatus = 'active' | 'disabled' | 'banned'

export type Role = {
  id: number
  key: string
  alias: string
  description: string
  isSystem: boolean
  isDefault: boolean
  isDeletable: boolean
  isEnabled: boolean
  permissionKeys: string[]
}

export type Permission = {
  key: string
  module: string
  label: string
  description: string
}

export type AdminUserProfile = {
  bio: string
  signature: string
  location: string
  websiteUrl: string
}

export type AdminUserSummary = {
  id: number
  username: string
  email: string
  displayName: string
  locale: string
  status: UserStatus
  isInitialSuperAdmin: boolean
  roleKeys: string[]
  createdAt?: string
  updatedAt?: string
}

export type PermissionOverrides = {
  allow: string[]
  deny: string[]
}

export type AdminUserActivity = {
  topicCount: number
  commentCount: number
  activeSessionCount: number
  totalSessionCount: number
  lastLoginAt?: string | null
  lastLoginIP: string
  lastLoginUserAgent: string
  lastSeenAt?: string | null
}

export type AdminSessionInspect = {
  id: string
  deviceName: string
  browser: string
  os: string
  ipPrefix: string
  ipAddress: string
  userAgent: string
  createdAt: string
  lastSeenAt: string
  isActive: boolean
  revokedAt?: string | null
  revokeReason: string
}

export type AdminAuthEvent = {
  id: number
  action: string
  ipAddress: string
  userAgent: string
  sessionHash: string
  createdAt: string
}

export type AdminUserDetail = AdminUserSummary & {
  permissions: string[]
  permissionOverrides: PermissionOverrides
  profile: AdminUserProfile
  /** 管理端预览：活跃度与最近登录线索（含完整 IP / UA） */
  activity?: AdminUserActivity
  /** 管理端预览：最近会话（完整 IP + 原始 UA） */
  sessions?: AdminSessionInspect[]
  /** 管理端预览：最近登录/注册审计 */
  recentAuthEvents?: AdminAuthEvent[]
  passwordChangedAt?: string | null
}

export type AdminUserList = {
  items: AdminUserSummary[]
  total: number
  page: number
  perPage: number
}
