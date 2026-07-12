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

export type PublicProfile = {
  userId: number
  username: string
  displayName: string
  profile: ProfileData
  topicCount: number
  commentCount: number
  recentTopics: import('~/utils/forumTaxonomy').ForumTopicSummary[]
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

  return { getPublicProfile, getMyProfile, updateMyProfile, uploadAvatar, deleteAvatar }
}
