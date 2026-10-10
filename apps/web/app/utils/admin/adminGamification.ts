/**
 * 徽章/成就（sforum.gamification 受保护内置插件）管理面数据层。
 *
 * 所有请求都打到插件声明的 admin 路由；插件未启用时 Host 返回 404/503，
 * 调用方据此展示「插件未启用」空态而不是报错。
 */

export type GamificationBadgeRule = {
  kind: 'threshold' | 'manual'
  metric?: 'topic_count' | 'comment_count' | 'registered_days'
  threshold?: number
}

export type GamificationBadge = {
  slug: string
  name: string
  description: string
  rarity: 'common' | 'rare' | 'epic' | 'legendary'
  rule: GamificationBadgeRule
  icon: string
  grantsPoints: number
  sort: number
  disabled: boolean
}

export type GamificationLevel = {
  minPoints: number
  title: string
  icon: string
}

export type GamificationGrant = {
  userId: number
  badgeSlug: string
  source: 'auto' | 'manual'
  actorId: number
  grantedAt: string
}

export type GamificationBadgePayload = {
  slug: string
  name: string
  description?: string
  rarity: GamificationBadge['rarity']
  rule: GamificationBadgeRule
  icon?: string
  grantsPoints?: number
  sort?: number
}

export type GamificationLevelPayload = {
  minPoints: number
  title: string
  icon?: string
}

export type GamificationSeedResult = {
  badges: number
  levels: number
}

export type GamificationBackfillPayload = {
  users: Array<{ userId: number, topicCount?: number, commentCount?: number }>
}

export type GamificationBackfillResult = {
  users: number
  points: number
}

export type GamificationGrantResult = {
  granted: boolean
  already: boolean
  badgeSlug: string
  userId: number
}

export type GamificationRequester = <T>(
  path: string,
  options?: { method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE', body?: Record<string, unknown> | null }
) => Promise<T>

const ADMIN = '/admin/gamification'

export function createAdminGamificationApi(request: GamificationRequester) {
  return {
    listBadges: () => request<{ badges: string }>(`${ADMIN}/badges`).then(parseBadges),
    createBadge: (payload: GamificationBadgePayload) =>
      request<{ slug: string }>(`${ADMIN}/badges`, { method: 'POST', body: payload as unknown as Record<string, unknown> }),
    updateBadge: (slug: string, payload: Partial<GamificationBadgePayload>) =>
      request<{ slug: string }>(`${ADMIN}/badges/${slug}`, { method: 'PATCH', body: payload as unknown as Record<string, unknown> }),
    disableBadge: (slug: string) =>
      request<{ slug: string, disabled: boolean }>(`${ADMIN}/badges/${slug}`, { method: 'DELETE' }),
    listLevels: () => request<{ levels: string }>(`${ADMIN}/levels`).then(parseLevels),
    upsertLevel: (payload: GamificationLevelPayload) =>
      request<{ minPoints: number }>(`${ADMIN}/levels`, { method: 'POST', body: payload as unknown as Record<string, unknown> }),
    grantBadge: (userId: number, badgeSlug: string) =>
      request<GamificationGrantResult>(`${ADMIN}/grants`, { method: 'POST', body: { userId, badgeSlug } }),
    revokeBadge: (userId: number, badgeSlug: string) =>
      request<{ revoked: boolean }>(`${ADMIN}/users/${userId}/badges/${badgeSlug}`, { method: 'DELETE' }),
    listUserBadges: (userId: number) =>
      request<{ grants: string }>(`${ADMIN}/users/${userId}/badges`).then(parseGrants),
    seedDefaults: () => request<GamificationSeedResult>(`${ADMIN}/seed-defaults`, { method: 'POST', body: {} }),
    backfill: (payload: GamificationBackfillPayload) =>
      request<GamificationBackfillResult>(`${ADMIN}/backfill`, { method: 'POST', body: payload as unknown as Record<string, unknown> })
  }
}

// 结构化数组以 JSON 字符串回传（TypedDocument 值域为原始类型）。
function parseBadges(response: { badges: string }): GamificationBadge[] {
  return decodeArray<GamificationBadge>(response?.badges)
}

function parseLevels(response: { levels: string }): GamificationLevel[] {
  return decodeArray<GamificationLevel>(response?.levels)
}

function parseGrants(response: { grants: string }): GamificationGrant[] {
  return decodeArray<GamificationGrant>(response?.grants)
}

function decodeArray<T>(value: unknown): T[] {
  if (typeof value !== 'string' || value === '') return []
  try {
    const parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed as T[] : []
  } catch {
    return []
  }
}

export type AdminGamificationApi = ReturnType<typeof createAdminGamificationApi>

/** 推荐默认（与插件端 defaultBadges/defaultLevels 一致，用于展示说明）。 */
export const recommendedBadgeSlugs = [
  'first-topic', 'first-comment', 'ten-topics', 'fifty-topics',
  'two-hundred-topics', 'hundred-comments', 'veteran', 'outstanding'
]

export const recommendedLevels: GamificationLevel[] = [
  { minPoints: 0, title: '新芽', icon: 'i-tabler-seedling' },
  { minPoints: 20, title: '幼苗', icon: 'i-tabler-plant' },
  { minPoints: 60, title: '小树', icon: 'i-tabler-tree' },
  { minPoints: 150, title: '大树', icon: 'i-tabler-trees' },
  { minPoints: 400, title: '守望者', icon: 'i-tabler-shield-check' }
]

/** 规则文案：把 JSON 规则翻译成运营能读的一句话。 */
export function describeBadgeRule(rule: GamificationBadgeRule | undefined, t: (key: string, params?: Record<string, unknown>) => string): string {
  if (!rule) return '—'
  if (rule.kind === 'manual') return t('admin.gamification.rule.manual')
  const metricKey = rule.metric === 'topic_count'
    ? 'admin.gamification.rule.topicCount'
    : rule.metric === 'comment_count'
      ? 'admin.gamification.rule.commentCount'
      : 'admin.gamification.rule.registeredDays'
  return t(metricKey, { count: rule.threshold ?? 0 })
}

export const badgeRarityColors: Record<GamificationBadge['rarity'], 'neutral' | 'info' | 'warning' | 'error'> = {
  common: 'neutral',
  rare: 'info',
  epic: 'warning',
  legendary: 'error'
}
