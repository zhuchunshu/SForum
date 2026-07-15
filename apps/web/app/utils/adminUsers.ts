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

export type AdminUserDetail = AdminUserSummary & {
  permissions: string[]
  permissionOverrides: PermissionOverrides
  profile: AdminUserProfile
}

export type AdminUserList = {
  items: AdminUserSummary[]
  total: number
  page: number
  perPage: number
}
