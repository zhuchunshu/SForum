export type ProfileData = {
  userId: number
  bio: string
  signature: string
  location: string
  websiteUrl: string
  avatarAttachmentId?: number | null
  avatar: AvatarView
  createdAt: string
  updatedAt: string
}

export type AvatarView = {
  kind: 'uploaded' | 'initials' | 'gravatar' | 'static'
  url: string
  attachmentId?: number | null
  alt: string
}

export type ProfileExtensionTab = {
  extensionId: string
  id: string
  order: number
  label?: Record<string, string>
  icon?: string
  kind: 'extensionRoute' | 'hostLink'
  method?: string
  url: string
}

export type ProfileActivityTopic = {
  id: number
  slug: string
  title: string
  status: string
  categorySlug: string
  categoryName: string
  commentCount: number
  createdAt: string
  updatedAt: string
  lastActivityAt: string
}

export type ProfileActivity = {
  kind: 'topic' | 'comment'
  topic: ProfileActivityTopic
  commentId?: number | null
  /** 回复在主题 flat 分页中的页码；用于 /page/N#comment-{id} 深链。 */
  commentPage?: number | null
  excerpt: string
  createdAt: string
}

export type ProfileActivityKind = 'topic' | 'comment'

export type ProfileActivityPage = {
  items: ProfileActivity[]
  page: number
  perPage: number
  total: number
  hasMore: boolean
  kind: ProfileActivityKind
}

export type ListProfileActivitiesInput = {
  kind?: ProfileActivityKind
  page?: number
  perPage?: number
}

export type PublicProfile = {
  userId: number
  username: string
  displayName: string
  profile: ProfileData
  topicCount: number
  commentCount: number
  recentTopics: import('~/utils/forumTaxonomy').ForumTopicSummary[]
  activities: ProfileActivity[]
  joinedAt: string
  extensionTabs?: ProfileExtensionTab[]
}

export type UpdateProfileInput = {
  bio?: string
  signature?: string
  location?: string
  websiteUrl?: string
  avatarAttachmentId?: number | null
}

export function useProfileApi() {
  const { request } = useApiClient()

  function getPublicProfile(username: string) {
    return request<PublicProfile>(`/profiles/${encodeURIComponent(username)}`)
  }

  function listPublicActivities(username: string, input: ListProfileActivitiesInput = {}) {
    const query = new URLSearchParams()
    if (input.kind) {
      query.set('kind', input.kind)
    }
    if (input.page && input.page > 0) {
      query.set('page', String(input.page))
    }
    if (input.perPage && input.perPage > 0) {
      query.set('perPage', String(input.perPage))
    }
    const suffix = query.toString()
    return request<ProfileActivityPage>(
      `/profiles/${encodeURIComponent(username)}/activities${suffix ? `?${suffix}` : ''}`
    )
  }

  function getMyProfile() {
    return request<PublicProfile>('/profile')
  }

  function updateMyProfile(input: UpdateProfileInput) {
    return request<ProfileData>('/profile', { method: 'PUT', body: input })
  }

  function uploadAvatar(file: File) {
    const body = new FormData()
    body.append('file', file)
    return request<ProfileData>('/profile/avatar', { method: 'POST', body })
  }

  function deleteAvatar() {
    return request<ProfileData>('/profile/avatar', { method: 'DELETE' })
  }

  return {
    getPublicProfile,
    listPublicActivities,
    getMyProfile,
    updateMyProfile,
    uploadAvatar,
    deleteAvatar
  }
}
