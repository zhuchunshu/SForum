import {
  normalizeForumBoundedInt,
  normalizeForumTagCreationMode,
  normalizeForumTagMaxPerTopic,
  normalizeForumTagMinPerTopic,
  normalizeForumPageSize,
  parseForumTagPublicPagesOption,
  recommendedForumSettings,
  type ForumCategory,
  type ForumCategoryGroup,
  type ForumDefaultSort,
  type ForumSettings,
  type ForumTag,
  type ForumTagStatus,
  type ForumVisibility
} from '~/utils/forum/forumTaxonomy'

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
  icon: string
  iconColor: string
  visibility: ForumVisibility
  position: number
  defaultSort: ForumDefaultSort
}

export type AdminForumTagPayload = {
  slug: string
  name: string
  description: string
  icon: string
  iconColor: string
  status: ForumTagStatus
}

export type AdminForumSettingsPayload = {
  defaultCategorySlug?: string
  tagCreationMode?: ForumSettings['tagCreationMode']
  tagPublicPages?: boolean
  tagMinPerTopic?: number
  tagMaxPerTopic?: number
  topicsPerPage?: number
  commentsPerPage?: number
  topicTitleMinRunes?: number
  topicTitleMaxRunes?: number
  topicContentMinRunes?: number
  topicContentMaxRunes?: number
  topicEditWindowMinutes?: number
  topicCooldownSeconds?: number
  dailyTopicLimit?: number
  commentMinRunes?: number
  commentMaxRunes?: number
  commentMaxNestingDepth?: number
  treeDescendantsPerRoot?: number
  commentEditWindowMinutes?: number
  commentCooldownSeconds?: number
  dailyCommentLimit?: number
  excerptRuneLimit?: number
  guestRead?: ForumSettings['guestRead']
  listDefaultSort?: ForumSettings['listDefaultSort']
  listHotWindowDays?: number
  allowAuthorCloseReplies?: boolean
  allowAuthorDelete?: boolean
  autoLockIdleDays?: number
  showTopicEditMark?: boolean
  duplicateTitlePolicy?: ForumSettings['duplicateTitlePolicy']
  showCommentEditMark?: boolean
  softDeleteVisibility?: ForumSettings['softDeleteVisibility']
  mentionsEnabled?: boolean
  mentionsMaxPerPost?: number
}

/** forum.settings.manage 控制的字段；保存时无权限会剔除。 */
export const forumSettingsManageKeys = [
  'topicsPerPage',
  'commentsPerPage',
  'topicTitleMinRunes',
  'topicTitleMaxRunes',
  'topicContentMinRunes',
  'topicContentMaxRunes',
  'topicEditWindowMinutes',
  'topicCooldownSeconds',
  'dailyTopicLimit',
  'commentMinRunes',
  'commentMaxRunes',
  'commentMaxNestingDepth',
  'treeDescendantsPerRoot',
  'commentEditWindowMinutes',
  'commentCooldownSeconds',
  'dailyCommentLimit',
  'excerptRuneLimit',
  'guestRead',
  'listDefaultSort',
  'listHotWindowDays',
  'allowAuthorCloseReplies',
  'allowAuthorDelete',
  'autoLockIdleDays',
  'showTopicEditMark',
  'duplicateTitlePolicy',
  'showCommentEditMark',
  'softDeleteVisibility',
  'mentionsEnabled',
  'mentionsMaxPerPost'
] as const satisfies ReadonlyArray<keyof AdminForumSettingsPayload>

export type AdminForumSettingKey = keyof AdminForumSettingsPayload

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
    listReindexRuns: () => request<ReindexRun[]>('/admin/forum/search/reindex/runs'),
    listSearchProviders: () => request<SearchProvidersState>('/admin/forum/search/providers'),
    selectSearchProvider: (extensionId: string) => request<{ selected: boolean }>('/admin/forum/search/provider', {
      method: 'PUT',
      body: { extensionId }
    }),
    resetSearchProvider: () => request<{ pinned: boolean, defaultExtensionId: string }>('/admin/forum/search/provider/reset', {
      method: 'POST',
      body: {}
    })
  }
}

/** search.provider 运营状态（论坛设置「搜索服务」Tab）。 */
export type SearchProviderItem = {
  extensionId: string
  label: string
  healthy: boolean
  isDefault?: boolean
}

export type SearchProvidersState = {
  items: SearchProviderItem[]
  selected: SearchProviderItem
  pinned: boolean
  defaultExtensionId: string
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
  let tagMinPerTopic = normalizeForumTagMinPerTopic(numberLikeValue(input?.tagMinPerTopic))
  let tagMaxPerTopic = normalizeForumTagMaxPerTopic(numberLikeValue(input?.tagMaxPerTopic))
  if (tagMinPerTopic > tagMaxPerTopic) {
    tagMinPerTopic = defaults.tagMinPerTopic
    tagMaxPerTopic = defaults.tagMaxPerTopic
  }

  const pair = (
    minKey: keyof ForumSettings,
    maxKey: keyof ForumSettings,
    minBound: number,
    maxBound: number,
    allowZeroMin: boolean
  ) => {
    const minFallback = defaults[minKey] as number
    const maxFallback = defaults[maxKey] as number
    let min = normalizeForumBoundedInt(numberLikeValue(input?.[minKey]), allowZeroMin ? 0 : minBound, maxBound, minFallback)
    let max = normalizeForumBoundedInt(numberLikeValue(input?.[maxKey]), minBound === 0 ? 1 : minBound, maxBound, maxFallback)
    if (min > max) {
      min = minFallback
      max = maxFallback
    }
    return [min, max] as const
  }

  const [topicTitleMinRunes, topicTitleMaxRunes] = pair('topicTitleMinRunes', 'topicTitleMaxRunes', 1, 200, false)
  const [topicContentMinRunes, topicContentMaxRunes] = pair('topicContentMinRunes', 'topicContentMaxRunes', 0, 200000, true)
  const [commentMinRunes, commentMaxRunes] = pair('commentMinRunes', 'commentMaxRunes', 0, 50000, true)

  return {
    defaultCategorySlug,
    tagCreationMode: normalizeForumTagCreationMode(stringValue(input?.tagCreationMode)),
    tagPublicPages: normalizeForumTagPublicPages(input?.tagPublicPages, defaults.tagPublicPages),
    tagMinPerTopic,
    tagMaxPerTopic,
    topicsPerPage: normalizeForumPageSize(numberLikeValue(input?.topicsPerPage), defaults.topicsPerPage),
    commentsPerPage: normalizeForumPageSize(numberLikeValue(input?.commentsPerPage), defaults.commentsPerPage),
    topicTitleMinRunes,
    topicTitleMaxRunes,
    topicContentMinRunes,
    topicContentMaxRunes,
    topicEditWindowMinutes: normalizeForumBoundedInt(numberLikeValue(input?.topicEditWindowMinutes), 0, 10080, defaults.topicEditWindowMinutes),
    topicCooldownSeconds: normalizeForumBoundedInt(numberLikeValue(input?.topicCooldownSeconds), 0, 86400, defaults.topicCooldownSeconds),
    dailyTopicLimit: normalizeForumBoundedInt(numberLikeValue(input?.dailyTopicLimit), 0, 10000, defaults.dailyTopicLimit),
    commentMinRunes,
    commentMaxRunes,
    commentMaxNestingDepth: normalizeForumBoundedInt(numberLikeValue(input?.commentMaxNestingDepth), 0, 20, defaults.commentMaxNestingDepth),
    treeDescendantsPerRoot: normalizeForumBoundedInt(numberLikeValue(input?.treeDescendantsPerRoot), 1, 100, defaults.treeDescendantsPerRoot),
    commentEditWindowMinutes: normalizeForumBoundedInt(numberLikeValue(input?.commentEditWindowMinutes), 0, 10080, defaults.commentEditWindowMinutes),
    commentCooldownSeconds: normalizeForumBoundedInt(numberLikeValue(input?.commentCooldownSeconds), 0, 86400, defaults.commentCooldownSeconds),
    dailyCommentLimit: normalizeForumBoundedInt(numberLikeValue(input?.dailyCommentLimit), 0, 10000, defaults.dailyCommentLimit),
    excerptRuneLimit: normalizeForumBoundedInt(numberLikeValue(input?.excerptRuneLimit), 40, 500, defaults.excerptRuneLimit),
    guestRead: normalizeGuestRead(stringValue(input?.guestRead), defaults.guestRead),
    listDefaultSort: normalizeListSort(stringValue(input?.listDefaultSort), defaults.listDefaultSort),
    listHotWindowDays: normalizeForumBoundedInt(numberLikeValue(input?.listHotWindowDays), 1, 90, defaults.listHotWindowDays),
    allowAuthorCloseReplies: normalizeForumTagPublicPages(input?.allowAuthorCloseReplies, defaults.allowAuthorCloseReplies),
    allowAuthorDelete: normalizeForumTagPublicPages(input?.allowAuthorDelete, defaults.allowAuthorDelete),
    autoLockIdleDays: normalizeForumBoundedInt(numberLikeValue(input?.autoLockIdleDays), 0, 3650, defaults.autoLockIdleDays),
    showTopicEditMark: normalizeForumTagPublicPages(input?.showTopicEditMark, defaults.showTopicEditMark),
    duplicateTitlePolicy: normalizeDuplicateTitlePolicy(stringValue(input?.duplicateTitlePolicy), defaults.duplicateTitlePolicy),
    showCommentEditMark: normalizeForumTagPublicPages(input?.showCommentEditMark, defaults.showCommentEditMark),
    softDeleteVisibility: normalizeSoftDeleteVisibility(stringValue(input?.softDeleteVisibility), defaults.softDeleteVisibility),
    mentionsEnabled: normalizeForumTagPublicPages(input?.mentionsEnabled, defaults.mentionsEnabled),
    mentionsMaxPerPost: normalizeForumBoundedInt(numberLikeValue(input?.mentionsMaxPerPost), 0, 50, defaults.mentionsMaxPerPost)
  }
}

function normalizeGuestRead(value: string | undefined, fallback: ForumSettings['guestRead']): ForumSettings['guestRead'] {
  return value === 'login_required' ? 'login_required' : value === 'public' ? 'public' : fallback
}

function normalizeListSort(value: string | undefined, fallback: ForumSettings['listDefaultSort']): ForumSettings['listDefaultSort'] {
  return value === 'active' || value === 'hot' || value === 'latest' ? value : fallback
}

function normalizeDuplicateTitlePolicy(value: string | undefined, fallback: ForumSettings['duplicateTitlePolicy']): ForumSettings['duplicateTitlePolicy'] {
  return value === 'off' || value === 'warn' || value === 'block' ? value : fallback
}

function normalizeSoftDeleteVisibility(value: string | undefined, fallback: ForumSettings['softDeleteVisibility']): ForumSettings['softDeleteVisibility'] {
  return value === 'staff_only' || value === 'hidden' || value === 'author_and_staff' ? value : fallback
}

export function forumSettingsPayload(settings: ForumSettings): AdminForumSettingsPayload {
  return normalizeForumSettings(settings)
}

export function forumSettingsPartialPayload(
  settings: ForumSettings,
  keys: readonly AdminForumSettingKey[]
): AdminForumSettingsPayload {
  return Object.fromEntries(keys.map(key => [key, settings[key]])) as AdminForumSettingsPayload
}

export function forumSettingsValidationError(settings: ForumSettings): string | null {
  if (settings.tagMinPerTopic > settings.tagMaxPerTopic) {
    return 'tagMinMax'
  }
  if (settings.topicTitleMinRunes > settings.topicTitleMaxRunes) {
    return 'topicTitleMinMax'
  }
  if (settings.topicContentMinRunes > settings.topicContentMaxRunes) {
    return 'topicContentMinMax'
  }
  if (settings.commentMinRunes > settings.commentMaxRunes) {
    return 'commentMinMax'
  }
  if (![settings.topicsPerPage, settings.commentsPerPage].every(value => Number.isInteger(value) && value >= 1 && value <= 100)) {
    return 'pagination'
  }
  return null
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
    icon: '',
    iconColor: '',
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
    icon: '',
    iconColor: '',
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
