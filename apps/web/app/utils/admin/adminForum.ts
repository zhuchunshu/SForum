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

/**
 * 版块分类页的删除确认目标。
 * blockedReason 非空表示服务端会拒绝（分组内仍有分类 / 分类内仍有主题），
 * 前端据此直接禁用确认按钮，避免无意义请求与不可读的错误码。
 */
export type AdminForumTaxonomyDeleteTarget = {
  kind: 'group' | 'category'
  id: number
  name: string
  blockedReason: string
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
  externalInterstitial?: boolean
  trustedDomains?: string
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
  'mentionsMaxPerPost',
  'externalInterstitial',
  'trustedDomains'
] as const satisfies ReadonlyArray<keyof AdminForumSettingsPayload>

export type AdminForumSettingKey = keyof AdminForumSettingsPayload

type ForumSettingsLike = Partial<Record<keyof ForumSettings, unknown>>

export const forumVisibilityChoices: ForumVisibility[] = ['public', 'hidden']
export const forumCategorySortChoices: ForumDefaultSort[] = ['latest', 'hot']
export const forumTagStatusChoices: ForumTagStatus[] = ['active', 'pending', 'disabled']

/**
 * 分类/分组删除的服务端拒绝原因码 → `admin.forum.categories.delete.errors.*` 后缀。
 * 未收录的原因码返回空串，由调用方按原样展示，避免静默失败。
 */
export const forumTaxonomyDeleteErrorKeys: Record<string, string> = {
  'forum.category_not_empty': 'categoryNotEmpty',
  'forum.category_group_not_empty': 'groupNotEmpty',
  'forum.category_is_default': 'categoryIsDefault',
  'forum.category_not_found': 'notFound',
  'forum.category_group_not_found': 'notFound'
}

export function forumTaxonomyDeleteErrorKey(reason: string) {
  return forumTaxonomyDeleteErrorKeys[reason] || ''
}

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
    deleteCategoryGroup: (id: number) => request<ForumCategoryGroup>(`/admin/forum/category-groups/${id}`, {
      method: 'DELETE'
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
    deleteCategory: (id: number) => request<ForumCategory>(`/admin/forum/categories/${id}`, {
      method: 'DELETE'
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
    mentionsMaxPerPost: normalizeForumBoundedInt(numberLikeValue(input?.mentionsMaxPerPost), 0, 50, defaults.mentionsMaxPerPost),
    externalInterstitial: normalizeForumTagPublicPages(input?.externalInterstitial, defaults.externalInterstitial),
    trustedDomains: normalizeTrustedDomainsInput(stringValue(input?.trustedDomains), defaults.trustedDomains)
  }
}

/**
 * 归一免中转域名白名单：逐行/逗号分隔，去 scheme 与路径，小写去重保序。
 * 与后端 options.normalizeTrustedDomainList 保持同一语义（非法条目在这里丢弃，
 * 由服务端对最终提交值再做一次 fail-closed 校验）。
 */
function normalizeTrustedDomainsInput(value: string | undefined, _fallback: string) {
  const raw = typeof value === 'string' ? value : ''
  if (!raw.trim()) {
    return ''
  }
  const hosts: string[] = []
  const seen = new Set<string>()
  for (const entry of raw.split(/[\n,]/)) {
    const trimmed = entry.trim()
    if (!trimmed) {
      continue
    }
    let host = ''
    try {
      host = new URL(trimmed.includes('://') ? trimmed : `https://${trimmed}`).host.toLowerCase()
    } catch {
      continue
    }
    if (!host || seen.has(host)) {
      continue
    }
    seen.add(host)
    hosts.push(host)
  }
  return hosts.join('\n')
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
  // 免中转白名单条目上限与后端 options 归一化保持一致（50 条）。
  if (trustedDomainEntries(settings.trustedDomains).length > 50) {
    return 'trustedDomainsTooMany'
  }
  return null
}

/** 白名单有效条目（去空行/去重），供校验与计数共用。 */
export function trustedDomainEntries(value: string | undefined) {
  const seen = new Set<string>()
  const entries: string[] = []
  for (const entry of String(value || '').split(/[\n,]/)) {
    const host = entry.trim().toLowerCase()
    if (!host || seen.has(host)) {
      continue
    }
    seen.add(host)
    entries.push(host)
  }
  return entries
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
