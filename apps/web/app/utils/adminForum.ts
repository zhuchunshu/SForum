import {
  normalizeForumTagCreationMode,
  normalizeForumTagMaxPerTopic,
  parseForumTagPublicPagesOption,
  recommendedForumSettings,
  type ForumCategory,
  type ForumCategoryGroup,
  type ForumDefaultSort,
  type ForumSettings,
  type ForumTag,
  type ForumTagStatus,
  type ForumVisibility
} from '~/utils/forumTaxonomy'

export type AdminForumRequestOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  body?: Record<string, unknown> | null
}

export type AdminForumRequester = <T>(path: string, options?: AdminForumRequestOptions) => Promise<T>

export type AdminForumCategoryGroupPayload = {
  slug: string
  name: string
  description: string
  visibility: ForumVisibility
  position: number
}

export type AdminForumCategoryPayload = {
  groupId: number
  slug: string
  name: string
  description: string
  visibility: ForumVisibility
  position: number
  defaultSort: ForumDefaultSort
}

export type AdminForumTagPayload = {
  slug: string
  name: string
  description: string
  status: ForumTagStatus
}

export type AdminForumSettingsPayload = {
  defaultCategorySlug: string
  tagCreationMode: ForumSettings['tagCreationMode']
  tagPublicPages: boolean
  tagMaxPerTopic: number
}

type ForumSettingsLike = Partial<Record<keyof ForumSettings, unknown>>

export const forumVisibilityChoices: ForumVisibility[] = ['public', 'hidden']
export const forumCategorySortChoices: ForumDefaultSort[] = ['latest', 'hot']
export const forumTagStatusChoices: ForumTagStatus[] = ['active', 'pending', 'disabled']

export function createAdminForumApi(request: AdminForumRequester) {
  return {
    listCategoryGroups: () => request<ForumCategoryGroup[]>('/admin/forum/category-groups'),
    createCategoryGroup: (payload: AdminForumCategoryGroupPayload) => request<ForumCategoryGroup>('/admin/forum/category-groups', {
      method: 'POST',
      body: payload
    }),
    updateCategoryGroup: (id: number, payload: Partial<AdminForumCategoryGroupPayload>) => request<ForumCategoryGroup>(`/admin/forum/category-groups/${id}`, {
      method: 'PATCH',
      body: payload
    }),
    listCategories: () => request<ForumCategory[]>('/admin/forum/categories'),
    createCategory: (payload: AdminForumCategoryPayload) => request<ForumCategory>('/admin/forum/categories', {
      method: 'POST',
      body: payload
    }),
    updateCategory: (id: number, payload: Partial<AdminForumCategoryPayload>) => request<ForumCategory>(`/admin/forum/categories/${id}`, {
      method: 'PATCH',
      body: payload
    }),
    listTags: () => request<ForumTag[]>('/admin/forum/tags'),
    createTag: (payload: AdminForumTagPayload) => request<ForumTag>('/admin/forum/tags', {
      method: 'POST',
      body: payload
    }),
    updateTag: (id: number, payload: Partial<AdminForumTagPayload>) => request<ForumTag>(`/admin/forum/tags/${id}`, {
      method: 'PATCH',
      body: payload
    }),
    getSettings: async () => normalizeForumSettings(await request<ForumSettings>('/admin/forum/settings')),
    updateSettings: async (payload: AdminForumSettingsPayload) => normalizeForumSettings(await request<ForumSettings>('/admin/forum/settings', {
      method: 'PUT',
      body: payload
    })),
    resetSettings: async () => normalizeForumSettings(await request<ForumSettings>('/admin/forum/settings/reset', {
      method: 'POST',
      body: {}
    })),
    reindexSearch: () => request<ReindexRun>('/admin/forum/search/reindex', { method: 'POST', body: {} }),
    getReindexStatus: () => request<ReindexStatus>('/admin/forum/search/reindex'),
    listReindexRuns: () => request<ReindexRun[]>('/admin/forum/search/reindex/runs')
  }
}

// 搜索索引重建运行记录。
export type ReindexRun = {
  id: number
  total: number
  status: 'running' | 'completed' | 'failed'
  startedAt: string
  finishedAt?: string | null
  startedByUserId?: number
  error?: string
}

export type ReindexStatus = ReindexRun & {
  processed: number
  remaining: number
  percent: number
}

export function createDefaultForumSettings(): ForumSettings {
  return { ...recommendedForumSettings }
}

export function normalizeForumSettings(input: ForumSettingsLike | null | undefined): ForumSettings {
  const defaults = createDefaultForumSettings()
  const defaultCategorySlug = normalizeSlug(input?.defaultCategorySlug, defaults.defaultCategorySlug)
  return {
    defaultCategorySlug,
    tagCreationMode: normalizeForumTagCreationMode(stringValue(input?.tagCreationMode)),
    tagPublicPages: normalizeForumTagPublicPages(input?.tagPublicPages, defaults.tagPublicPages),
    tagMaxPerTopic: normalizeForumTagMaxPerTopic(numberLikeValue(input?.tagMaxPerTopic))
  }
}

export function forumSettingsPayload(settings: ForumSettings): AdminForumSettingsPayload {
  return normalizeForumSettings(settings)
}

export function createCategoryGroupPayload(overrides: Partial<AdminForumCategoryGroupPayload> = {}): AdminForumCategoryGroupPayload {
  return {
    slug: '',
    name: '',
    description: '',
    visibility: 'public',
    position: 0,
    ...overrides
  }
}

export function createCategoryPayload(groupId = 0, overrides: Partial<AdminForumCategoryPayload> = {}): AdminForumCategoryPayload {
  return {
    groupId,
    slug: '',
    name: '',
    description: '',
    visibility: 'public',
    position: 0,
    defaultSort: 'latest',
    ...overrides
  }
}

export function createTagPayload(overrides: Partial<AdminForumTagPayload> = {}): AdminForumTagPayload {
  return {
    slug: '',
    name: '',
    description: '',
    status: 'active',
    ...overrides
  }
}

function normalizeForumTagPublicPages(value: unknown, fallback: boolean) {
  if (typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'string') {
    return parseForumTagPublicPagesOption(value, fallback)
  }
  return fallback
}

function normalizeSlug(value: unknown, fallback: string) {
  const normalized = stringValue(value).trim().toLowerCase()
  return /^[a-z0-9]+(?:[-_][a-z0-9]+)*$/.test(normalized) ? normalized : fallback
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function numberLikeValue(value: unknown) {
  return typeof value === 'string' || typeof value === 'number' ? value : undefined
}
