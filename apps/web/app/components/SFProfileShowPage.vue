<script setup lang="ts">
/**
 * 宿主 body 岛：forum.profile.show。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 * 布局对齐 demo B1：intro + 分区 tab（主题 / 回复 / 关于）+ 分日时间线。
 */

import {
  forumUserProfilePath,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'
import { apiErrorMessage } from '~/composables/useApiClient'
import type {
  PublicProfile,
  ProfileActivity,
  ProfileActivityKind,
  ProfileActivityPage
} from '~/composables/useProfileApi'
import {
  groupProfileActivitiesByDate,
  profileActivityLink,
  profileHasPublicDetails
} from '~/utils/profileActivity'
import { safeUrl } from '~/utils/sfUrl'

type ProfileTab = 'topics' | 'replies' | 'about'

type ActivityFeedState = {
  items: ProfileActivity[]
  page: number
  hasMore: boolean
  loaded: boolean
}

const ACTIVITY_PER_PAGE = 20

const route = useRoute()
const { t, locale } = useI18n()
const localePath = useLocalePath()
const toast = useToast()
const { seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const { format, formatDateOnly, settings: dateTimeSettings } = useSiteDateTime()
const profileApi = useProfileApi()
const forumApi = useForumApi()
const { user: authUser } = useAuthSession()
const { can } = usePermissions()
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)

const username = computed(() => String(route.params.username ?? ''))
const renderedAt = useState<number>(`forum-profile-rendered-at-${username.value}`, () => Date.now())
const activeTab = ref<ProfileTab>('topics')
const loadingMore = ref(false)
const switchingTab = ref(false)

function emptyFeed(): ActivityFeedState {
  return { items: [], page: 0, hasMore: false, loaded: false }
}

const topicFeed = ref<ActivityFeedState>(emptyFeed())
const replyFeed = ref<ActivityFeedState>(emptyFeed())

const { data: profile, pending: profilePending, error: profileError } = await useAsyncData(
  () => `forum-profile-${username.value}`,
  () => profileApi.getPublicProfile(username.value),
  { default: () => null as PublicProfile | null }
)

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'forum-profile-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

// SSR/首屏：主题 tab 第一页走独立分页接口，避免内嵌 20 条混合活动截断。
const { data: initialTopicPage } = await useAsyncData(
  () => `forum-profile-activities-topic-${username.value}`,
  async () => {
    if (!username.value) {
      return null as ProfileActivityPage | null
    }
    try {
      return await profileApi.listPublicActivities(username.value, {
        kind: 'topic',
        page: 1,
        perPage: ACTIVITY_PER_PAGE
      })
    } catch {
      // 资料可展示时活动失败不阻断整页；客户端可再试。
      return null as ProfileActivityPage | null
    }
  },
  { default: () => null as ProfileActivityPage | null }
)

if (initialTopicPage.value) {
  topicFeed.value = {
    items: [...initialTopicPage.value.items],
    page: initialTopicPage.value.page,
    hasMore: initialTopicPage.value.hasMore,
    loaded: true
  }
}

useSForumSeo(computed(() => ({
  type: 'profile',
  path: forumUserProfilePath(username.value),
  title: profile.value?.displayName || profile.value?.username || username.value,
  description: profile.value?.profile.bio || t('profile.metaDescription'),
  public: Boolean(profile.value),
  noindex: !profile.value,
  variables: { authorName: profile.value?.displayName || profile.value?.username || username.value },
  authorName: profile.value?.displayName || profile.value?.username || username.value
})))

const displayName = computed(() => profile.value?.displayName || profile.value?.username || username.value)
const isSelf = computed(() => Boolean(profile.value && authUser.value?.id === profile.value.userId))
const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))
const navCategories = computed(() => categoryGroups.value.flatMap(group => group.categories || []))
const navTotalTopics = computed(() => navCategories.value.reduce((sum, category) => sum + category.topicCount, 0))
const hasPublicDetails = computed(() => profileHasPublicDetails(profile.value))
const recentTopics = computed(() => profile.value?.recentTopics || [])

const profileTabs = computed(() => [
  {
    value: 'topics' as const,
    label: t('profile.tabs.topics'),
    count: profile.value?.topicCount ?? 0
  },
  {
    value: 'replies' as const,
    label: t('profile.tabs.replies'),
    count: profile.value?.commentCount ?? 0
  },
  {
    value: 'about' as const,
    label: t('profile.tabs.about')
  }
])

const activeFeed = computed(() => {
  if (activeTab.value === 'replies') {
    return replyFeed.value
  }
  if (activeTab.value === 'topics') {
    return topicFeed.value
  }
  return emptyFeed()
})

const panelActivities = computed(() => activeFeed.value.items)

const activityGroups = computed(() => groupProfileActivitiesByDate(panelActivities.value, {
  settings: dateTimeSettings.value,
  locale: String(locale.value || 'zh-CN'),
  topicUrlMode: topicUrlMode.value,
  labels: {
    today: t('profile.timeline.today'),
    yesterday: t('profile.timeline.yesterday')
  },
  now: new Date(renderedAt.value)
}))

const showLoadMore = computed(() => {
  if (activeTab.value === 'about') {
    return false
  }
  return activeFeed.value.loaded && activeFeed.value.hasMore
})

const categoryParticipationCount = computed(() => {
  const slugs = new Set<string>()
  for (const activity of [...topicFeed.value.items, ...replyFeed.value.items]) {
    if (activity.topic.categorySlug) {
      slugs.add(activity.topic.categorySlug)
    }
  }
  for (const topic of recentTopics.value) {
    if (topic.categorySlug) {
      slugs.add(topic.categorySlug)
    }
  }
  return slugs.size
})

const websiteHost = computed(() => {
  const raw = profile.value?.profile.websiteUrl?.trim() || ''
  if (!raw) {
    return ''
  }
  return raw.replace(/^https?:\/\//i, '').replace(/\/$/, '')
})

const websiteHref = computed(() => {
  const raw = profile.value?.profile.websiteUrl?.trim() || ''
  return raw ? safeUrl(raw) : ''
})

function formatDate(value: string) {
  return formatDateOnly(value)
}

function formatDateTime(value: string) {
  return format(value)
}

function activityTo(activity: ProfileActivity) {
  return localePath(profileActivityLink(activity, topicUrlMode.value))
}

function activityKindForTab(tab: ProfileTab): ProfileActivityKind | null {
  if (tab === 'topics') {
    return 'topic'
  }
  if (tab === 'replies') {
    return 'comment'
  }
  return null
}

function feedRefForKind(kind: ProfileActivityKind) {
  return kind === 'comment' ? replyFeed : topicFeed
}

async function ensureFeedLoaded(tab: ProfileTab) {
  const kind = activityKindForTab(tab)
  if (!kind || !username.value) {
    return
  }
  const feed = feedRefForKind(kind)
  if (feed.value.loaded) {
    return
  }
  switchingTab.value = true
  try {
    const page = await profileApi.listPublicActivities(username.value, {
      kind,
      page: 1,
      perPage: ACTIVITY_PER_PAGE
    })
    feed.value = {
      items: [...page.items],
      page: page.page,
      hasMore: page.hasMore,
      loaded: true
    }
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-triangle',
      title: apiErrorMessage(error) || t('profile.timeline.loadFailed')
    })
  } finally {
    switchingTab.value = false
  }
}

async function loadMoreActivities() {
  const kind = activityKindForTab(activeTab.value)
  if (!kind || !username.value || loadingMore.value) {
    return
  }
  const feed = feedRefForKind(kind)
  if (!feed.value.hasMore) {
    return
  }
  loadingMore.value = true
  try {
    const nextPage = feed.value.page + 1
    const page = await profileApi.listPublicActivities(username.value, {
      kind,
      page: nextPage,
      perPage: ACTIVITY_PER_PAGE
    })
    const seen = new Set(
      feed.value.items.map(item => `${item.kind}:${item.commentId || item.topic.id}:${item.createdAt}`)
    )
    const appended = page.items.filter((item) => {
      const key = `${item.kind}:${item.commentId || item.topic.id}:${item.createdAt}`
      if (seen.has(key)) {
        return false
      }
      seen.add(key)
      return true
    })
    feed.value = {
      items: [...feed.value.items, ...appended],
      page: page.page,
      hasMore: page.hasMore,
      loaded: true
    }
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-triangle',
      title: apiErrorMessage(error) || t('profile.timeline.loadMoreFailed')
    })
  } finally {
    loadingMore.value = false
  }
}

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

async function selectTab(tab: ProfileTab) {
  activeTab.value = tab
  if (tab === 'topics' || tab === 'replies') {
    await ensureFeedLoaded(tab)
  }
}

async function shareProfile() {
  if (!import.meta.client) {
    return
  }
  const url = window.location.href
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
    } else {
      const input = document.createElement('input')
      input.value = url
      document.body.appendChild(input)
      input.select()
      document.execCommand('copy')
      document.body.removeChild(input)
    }
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('profile.shareCopied')
    })
  } catch {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-triangle',
      title: t('profile.shareFailed')
    })
  }
}
</script>

<template>
  <main class="sforum-home sf-profile-page" data-layout="fullwidth-3col">
    <template v-if="profileError && !profile">
      <div class="sf-profile-state">
        <SFEmptyState
          :title="t('profile.notFound.title')"
          :description="t('profile.notFound.description')"
        />
      </div>
    </template>

    <template v-else-if="profilePending && !profile">
      <div class="sforum-home__layout sforum-home__layout--with-right sf-profile-layout">
        <aside class="sforum-home__sidebar">
          <SFHomeNavigation
            :categories="navCategories"
            :total-topics="navTotalTopics"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
            navigation-mode="route"
            desktop-only
          />
        </aside>
        <section class="sforum-home__main sf-profile-main">
          <SFSkeleton :lines="6" />
        </section>
      </div>
    </template>

    <template v-else-if="profile">
      <div class="sforum-home__layout sforum-home__layout--with-right sf-profile-layout">
        <aside class="sforum-home__sidebar">
          <SFHomeNavigation
            :categories="navCategories"
            :total-topics="navTotalTopics"
            :pending="categoriesPending"
            :can-create-topic="canCreateTopic"
            navigation-mode="route"
            desktop-only
          />
        </aside>

        <section class="sforum-home__main sf-profile-main" aria-labelledby="profile-name">
          <div class="sforum-home__mobile-nav">
            <SFHomeNavigation
              :categories="navCategories"
              :total-topics="navTotalTopics"
              :pending="categoriesPending"
              :can-create-topic="canCreateTopic"
              navigation-mode="route"
              mobile-only
            />
          </div>

          <header class="sf-profile-intro">
            <SFAvatar :name="displayName" :avatar="profile.profile.avatar" size="lg" class="sf-profile-intro__avatar" />
            <div class="sf-profile-intro__content">
              <div class="sf-profile-intro__name-row">
                <h1 id="profile-name">{{ displayName }}</h1>
                <span class="sf-profile-handle">@{{ profile.username }}</span>
                <span class="sf-profile-uid">UID {{ profile.userId }}</span>
              </div>
              <p v-if="profile.profile.bio" class="sf-profile-intro__bio">{{ profile.profile.bio }}</p>
              <p v-else class="sf-profile-intro__bio sf-profile-intro__bio--empty">{{ t('profile.bioEmpty') }}</p>
              <!-- 简介下 meta：位置 / 网站 / 加入时间，对齐 demo B1 profile-intro__meta -->
              <div class="sf-profile-intro__meta">
                <span v-if="profile.profile.location">
                  <UIcon name="i-lucide-map-pin" class="size-3.5" aria-hidden="true" />
                  {{ profile.profile.location }}
                </span>
                <a
                  v-if="websiteHref"
                  :href="websiteHref"
                  target="_blank"
                  rel="noopener noreferrer nofollow"
                >
                  <UIcon name="i-lucide-link" class="size-3.5" aria-hidden="true" />
                  {{ websiteHost }}
                </a>
                <span>
                  <UIcon name="i-lucide-calendar-days" class="size-3.5" aria-hidden="true" />
                  {{ t('profile.joinedOn', { date: formatDate(profile.joinedAt) }) }}
                </span>
              </div>
            </div>
            <div class="sf-profile-intro__actions">
              <NuxtLink
                v-if="isSelf"
                :to="localePath('/settings/profile')"
                class="sf-button sf-button--primary sf-button--md sf-profile-edit-link"
              >
                <UIcon name="i-lucide-settings-2" class="size-4" aria-hidden="true" />
                {{ t('profile.editProfile') }}
              </NuxtLink>
              <button
                type="button"
                class="sf-profile-share-button"
                :aria-label="t('profile.shareProfile')"
                :title="t('profile.shareProfile')"
                @click="shareProfile"
              >
                <UIcon name="i-lucide-share-2" class="size-4" aria-hidden="true" />
              </button>
            </div>
          </header>

          <!-- 分区 tab：主题 / 回复 / 关于 -->
          <nav class="sf-profile-tabs" :aria-label="t('profile.tabs.ariaLabel')">
            <button
              v-for="tab in profileTabs"
              :key="tab.value"
              type="button"
              class="sf-profile-tab"
              :class="{ 'is-active': activeTab === tab.value }"
              role="tab"
              :aria-selected="activeTab === tab.value ? 'true' : 'false'"
              @click="selectTab(tab.value)"
            >
              {{ tab.label }}
              <span v-if="tab.count !== undefined" class="sf-profile-tab__count">{{ tab.count }}</span>
            </button>
          </nav>

          <template v-if="activeTab === 'about'">
            <section class="sf-profile-about" :aria-label="t('profile.tabs.about')">
              <h2>{{ t('profile.aboutTitle', { name: displayName }) }}</h2>
              <p v-if="profile.profile.bio">{{ profile.profile.bio }}</p>
              <p v-else class="sf-profile-about__empty">{{ t('profile.bioEmpty') }}</p>
              <ul class="sf-profile-detail-list">
                <li v-if="profile.profile.location">
                  <UIcon name="i-lucide-map-pin" class="size-4" aria-hidden="true" />
                  <span>{{ profile.profile.location }}</span>
                </li>
                <li v-if="websiteHref">
                  <UIcon name="i-lucide-link" class="size-4" aria-hidden="true" />
                  <span>
                    <a :href="websiteHref" target="_blank" rel="noopener noreferrer nofollow">
                      {{ websiteHost }}
                    </a>
                  </span>
                </li>
                <li>
                  <UIcon name="i-lucide-calendar-days" class="size-4" aria-hidden="true" />
                  <span>{{ t('profile.joinedOn', { date: formatDate(profile.joinedAt) }) }}</span>
                </li>
                <li v-if="profile.profile.signature">
                  <UIcon name="i-lucide-quote" class="size-4" aria-hidden="true" />
                  <span>{{ profile.profile.signature }}</span>
                </li>
              </ul>
            </section>
          </template>

          <template v-else>
            <div class="sf-profile-summary" aria-label="profile statistics">
              <div class="sf-profile-summary__item">
                <strong>{{ profile.topicCount }}</strong>
                <span>{{ t('profile.topicCount') }}</span>
              </div>
              <div class="sf-profile-summary__item">
                <strong>{{ profile.commentCount }}</strong>
                <span>{{ t('profile.commentCount') }}</span>
              </div>
              <div class="sf-profile-summary__item">
                <strong>{{ categoryParticipationCount }}</strong>
                <span>{{ t('profile.categoryCount') }}</span>
              </div>
            </div>

            <section class="sf-profile-section-head">
              <div>
                <h2>
                  {{ activeTab === 'replies' ? t('profile.timeline.repliesTitle') : t('profile.timeline.title') }}
                </h2>
                <p>
                  {{ activeTab === 'replies' ? t('profile.timeline.repliesDescription') : t('profile.timeline.description') }}
                </p>
              </div>
            </section>

            <div v-if="switchingTab && !activityGroups.length" class="sf-profile-loading">
              <SFSkeleton :lines="4" />
            </div>

            <div v-else-if="activityGroups.length" class="sf-profile-timeline">
              <section v-for="group in activityGroups" :key="group.key" class="sf-profile-activity-day">
                <header class="sf-profile-activity-day__head">
                  <h3>{{ group.label }}</h3>
                  <time :datetime="group.key">{{ group.dateLabel }}</time>
                </header>
                <div class="sf-profile-activity-day__events">
                  <a
                    v-for="activity in group.items"
                    :key="`${activity.kind}:${activity.commentId || activity.topic.id}:${activity.createdAt}`"
                    :href="activityTo(activity)"
                    class="sf-profile-activity"
                    :class="`sf-profile-activity--${activity.kind}`"
                  >
                    <span class="sf-profile-activity__icon" aria-hidden="true">
                      <UIcon
                        :name="activity.kind === 'topic' ? 'i-lucide-square-pen' : 'i-lucide-message-circle-reply'"
                        class="size-4"
                      />
                    </span>
                    <span class="sf-profile-activity__body">
                      <span class="sf-profile-activity__type">
                        {{ activity.kind === 'topic' ? t('profile.timeline.topicAction') : t('profile.timeline.commentAction') }}
                      </span>
                      <strong>{{ activity.topic.title }}</strong>
                      <span v-if="activity.excerpt" class="sf-profile-activity__excerpt">{{ activity.excerpt }}</span>
                      <span class="sf-profile-activity__meta">
                        {{ activity.topic.categoryName }}
                        <template v-if="activity.kind === 'topic'">
                          · {{ t('profile.timeline.replyCount', { count: activity.topic.commentCount }) }}
                        </template>
                        <template v-else-if="activity.commentId">
                          · #{{ activity.commentId }}
                        </template>
                      </span>
                    </span>
                    <time class="sf-profile-activity__time" :datetime="activity.createdAt">
                      {{ activity.timeLabel || formatDateTime(activity.createdAt) }}
                    </time>
                  </a>
                </div>
              </section>

              <div v-if="showLoadMore" class="sf-profile-load-more">
                <button
                  type="button"
                  class="sf-button sf-button--ghost sf-button--md sf-profile-load-more__button"
                  :disabled="loadingMore"
                  :aria-busy="loadingMore ? 'true' : undefined"
                  @click="loadMoreActivities"
                >
                  <UIcon
                    :name="loadingMore ? 'i-lucide-loader-2' : 'i-lucide-chevrons-down'"
                    class="size-4"
                    :class="{ 'sf-profile-load-more__spinner': loadingMore }"
                    aria-hidden="true"
                  />
                  {{ loadingMore ? t('profile.timeline.loadingMore') : t('profile.timeline.loadMore') }}
                </button>
              </div>
            </div>

            <div v-else class="sf-profile-empty">
              <UIcon name="i-lucide-message-square-off" class="size-7" aria-hidden="true" />
              <strong>
                {{ activeTab === 'replies' ? t('profile.timeline.emptyRepliesTitle') : t('profile.timeline.emptyTitle') }}
              </strong>
              <span>
                {{ activeTab === 'replies' ? t('profile.timeline.emptyRepliesDescription') : t('profile.timeline.emptyDescription') }}
              </span>
            </div>
          </template>
        </section>

        <aside class="sforum-home__right sf-profile-right" :aria-label="t('profile.publicDetails')">
          <SFProfileRightRail
            :profile="profile"
            :display-name="displayName"
            :has-public-details="hasPublicDetails"
            :recent-topics="recentTopics"
          />
        </aside>
      </div>
    </template>

    <button
      v-if="profile && (mobileMenuOpen || mobileInfoOpen)"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('common.close')"
      @click="closeMobileDrawers"
    />

    <aside v-if="profile && mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.navTitle') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        :categories="navCategories"
        :total-topics="navTotalTopics"
        :pending="categoriesPending"
        :can-create-topic="canCreateTopic"
        navigation-mode="route"
        desktop-only
      />
    </aside>

    <aside v-if="profile && mobileInfoOpen" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('profile.publicDetails') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFProfileRightRail
        :profile="profile"
        :display-name="displayName"
        :has-public-details="hasPublicDetails"
        :recent-topics="recentTopics"
        drawer
      />
    </aside>
  </main>
</template>
