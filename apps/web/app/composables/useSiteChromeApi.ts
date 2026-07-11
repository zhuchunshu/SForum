/** Site chrome：导航 / 友情链接 / 公告 的公开与管理 API。 */

export type SiteNavItem = {
  id: number
  labelZhCN: string
  labelEnUS: string
  href: string
  openInNewTab: boolean
  position: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export type SiteFriendLink = {
  id: number
  name: string
  url: string
  description: string
  logoUrl: string
  position: number
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export type SiteAnnouncementStyle = 'info' | 'success' | 'warning' | 'danger'

export type SiteAnnouncement = {
  id: number
  titleZhCN: string
  titleEnUS: string
  bodyZhCN: string
  bodyEnUS: string
  style: SiteAnnouncementStyle
  href: string
  dismissible: boolean
  position: number
  enabled: boolean
  startsAt?: string | null
  endsAt?: string | null
  createdAt: string
  updatedAt: string
}

export type SiteNavItemInput = {
  labelZhCN: string
  labelEnUS: string
  href: string
  openInNewTab?: boolean
  position?: number
  enabled?: boolean
}

export type SiteFriendLinkInput = {
  name: string
  url: string
  description?: string
  logoUrl?: string
  position?: number
  enabled?: boolean
}

export type SiteAnnouncementInput = {
  titleZhCN?: string
  titleEnUS?: string
  bodyZhCN?: string
  bodyEnUS?: string
  style?: SiteAnnouncementStyle
  href?: string
  dismissible?: boolean
  position?: number
  enabled?: boolean
  startsAt?: string | null
  endsAt?: string | null
}

export function useSiteChromeApi() {
  const { request } = useApiClient()

  return {
    listPublicNavItems: () => request<SiteNavItem[]>('/site/nav-items'),
    listAdminNavItems: () => request<SiteNavItem[]>('/admin/site/nav-items'),
    createNavItem: (body: SiteNavItemInput) =>
      request<SiteNavItem>('/admin/site/nav-items', { method: 'POST', body }),
    updateNavItem: (id: number, body: Partial<SiteNavItemInput>) =>
      request<SiteNavItem>(`/admin/site/nav-items/${id}`, { method: 'PATCH', body }),
    deleteNavItem: (id: number) =>
      request<{ deleted: boolean }>(`/admin/site/nav-items/${id}`, { method: 'DELETE' }),

    listPublicFriendLinks: () => request<SiteFriendLink[]>('/site/friend-links'),
    listAdminFriendLinks: () => request<SiteFriendLink[]>('/admin/site/friend-links'),
    createFriendLink: (body: SiteFriendLinkInput) =>
      request<SiteFriendLink>('/admin/site/friend-links', { method: 'POST', body }),
    updateFriendLink: (id: number, body: Partial<SiteFriendLinkInput>) =>
      request<SiteFriendLink>(`/admin/site/friend-links/${id}`, { method: 'PATCH', body }),
    deleteFriendLink: (id: number) =>
      request<{ deleted: boolean }>(`/admin/site/friend-links/${id}`, { method: 'DELETE' }),

    listPublicAnnouncements: () => request<SiteAnnouncement[]>('/site/announcements'),
    listAdminAnnouncements: () => request<SiteAnnouncement[]>('/admin/site/announcements'),
    createAnnouncement: (body: SiteAnnouncementInput) =>
      request<SiteAnnouncement>('/admin/site/announcements', { method: 'POST', body }),
    updateAnnouncement: (id: number, body: SiteAnnouncementInput) =>
      request<SiteAnnouncement>(`/admin/site/announcements/${id}`, { method: 'PATCH', body }),
    deleteAnnouncement: (id: number) =>
      request<{ deleted: boolean }>(`/admin/site/announcements/${id}`, { method: 'DELETE' })
  }
}
