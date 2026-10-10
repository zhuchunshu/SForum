<script setup lang="ts">
/**
 * 公开徽章墙。
 *
 * 数据来源不是硬编码的插件地址：先从公开资料接口拿到该用户的
 * forum.profile.tabs 扩展标签（宿主渲染的描述符），再用其中徽章墙
 * 条目的 URL 拉取数据。插件未启用时 extensionTabs 为空，页面走空态
 * （失败关闭），Core 永远不知道「徽章」这个业务概念。
 */
import SFBadgeWall from '~/components/profile/SFBadgeWall.vue'
import { useProfileApi } from '~/composables/profile/useProfileApi'
import { useSForumSeo } from '~/composables/seo/useSForumSeo'

definePageMeta({ public: true })

const route = useRoute()
const { t, locale } = useI18n()
const { request } = useApiClient()
const profileApi = useProfileApi()

const username = computed(() => String(route.params.username ?? ''))

const { data: profile, pending, error } = await useAsyncData(
  () => `badge-wall-profile-${username.value}`,
  () => profileApi.getPublicProfile(username.value),
  { default: () => null as Awaited<ReturnType<typeof profileApi.getPublicProfile>> | null }
)

/** 徽章墙条目：由插件贡献声明，宿主只认 URL。 */
const badgeWallTab = computed(() => {
  const tabs = profile.value?.extensionTabs ?? []
  return tabs.find(tab => tab.extensionId === 'sforum.gamification') ?? null
})

const wallUrl = computed(() => {
  const tab = badgeWallTab.value
  if (!tab?.url) return ''
  const path = tab.url.replace('{userId}', String(profile.value?.userId ?? ''))
  return path.startsWith('/') ? path : `/${path}`
})

const { data: wall, pending: wallPending } = await useAsyncData(
  () => `badge-wall-${username.value}`,
  async () => {
    if (!wallUrl.value) return null
    const response = await request<BadgeWallResponse>(wallUrl.value)
    return normalizeWall(response)
  },
  { default: () => null as BadgeWall | null, watch: [wallUrl] }
)

const displayName = computed(() => profile.value?.displayName || username.value)

useSForumSeo({
  title: () => t('badges.metaTitle', { name: displayName.value }),
  path: () => `/badges/${username.value}`,
  public: true
})

type BadgeWallResponse = {
  userId?: number
  points?: number
  level?: string | { title?: string, icon?: string, minPoints?: number }
  badges?: string | Array<Record<string, unknown>>
}

type BadgeWall = {
  userId: number
  points: number
  level: { title: string, icon: string }
  badges: Array<{ slug: string, name: string, description: string, rarity: string, icon: string, grantedAt: string }>
}

/** 结构化数组/对象以 JSON 字符串回传（TypedDocument 值域为原始类型）。 */
function normalizeWall(response: BadgeWallResponse | null): BadgeWall | null {
  if (!response) return null
  const level = typeof response.level === 'string'
    ? safeParse<{ title?: string, icon?: string }>(response.level)
    : response.level
  const badges = typeof response.badges === 'string'
    ? safeParse<Array<Record<string, unknown>>>(response.badges)
    : response.badges
  return {
    userId: Number(response.userId ?? 0),
    points: Number(response.points ?? 0),
    level: { title: String(level?.title ?? ''), icon: String(level?.icon ?? '') },
    badges: (badges ?? []).map(item => ({
      slug: String(item.slug ?? ''),
      name: String(item.name ?? ''),
      description: String(item.description ?? ''),
      rarity: String(item.rarity ?? 'common'),
      icon: String(item.icon ?? ''),
      grantedAt: String(item.grantedAt ?? '')
    }))
  }
}

function safeParse<T>(value: string): T | null {
  try {
    return JSON.parse(value) as T
  } catch {
    return null
  }
}

function tabLabel(tab: { label?: Record<string, string>, id: string } | null) {
  if (!tab) return t('badges.title')
  const labels = tab.label || {}
  return labels[String(locale.value)] || labels['zh-CN'] || labels['en-US'] || Object.values(labels)[0] || tab.id
}
</script>

<template>
  <SFPageOutlet page="forum.profile.show">
    <div class="sf-badge-wall-page">
      <header class="sf-badge-wall-page__head">
        <h1 class="text-xl font-bold text-slate-900 dark:text-zinc-100">
          {{ tabLabel(badgeWallTab) }}
        </h1>
        <p class="mt-1 text-sm text-muted">{{ t('badges.subtitle', { name: displayName }) }}</p>
      </header>

      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :title="t('badges.loadFailed')"
      />

      <!-- 插件未启用：宿主拿不到贡献条目，失败关闭给空态而不是报错。 -->
      <UAlert
        v-else-if="!pending && !badgeWallTab"
        color="neutral"
        variant="soft"
        icon="i-lucide-plug"
        :title="t('badges.unavailableTitle')"
        :description="t('badges.unavailable')"
      />

      <SFBadgeWall
        v-else
        :wall="wall"
        :pending="pending || wallPending"
        :display-name="displayName"
      />
    </div>
  </SFPageOutlet>
</template>
