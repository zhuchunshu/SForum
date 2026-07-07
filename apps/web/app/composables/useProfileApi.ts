export type ProfileData = {
  userId: number
  bio: string
  signature: string
  location: string
  websiteUrl: string
  avatarAttachmentId?: number | null
  createdAt: string
  updatedAt: string
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

  return { getPublicProfile, getMyProfile, updateMyProfile }
}
