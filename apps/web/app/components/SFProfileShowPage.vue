<script setup lang="ts">
/**
 * 宿主 body 岛：forum.profile.show。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import {
  forumUserProfilePath,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'
import type { PublicProfile, ProfileActivity } from '~/composables/useProfileApi'
import {
  groupProfileActivitiesByDate,
  profileActivityLink,
  profileHasPublicDetails
} from '~/utils/profileActivity'
import { safeUrl } from '~/utils/sfUrl'

const route = useRoute()
const { t, locale } = useI18n()
const localePath = useLocalePath()
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
const activities = computed<ProfileActivity[]>(() => profile.value?.activities || [])
const activityGroups = computed(() => groupProfileActivitiesByDate(activities.value, {
  settings: dateTimeSettings.value,
  locale: String(locale.value || 'zh-CN'),
  topicUrlMode: topicUrlMode.value,
  labels: {
    today: t('profile.timeline.today'),
    yesterday: t('profile.timeline.yesterday')
  },
  now: new Date(renderedAt.value)
}))

function formatDate(value: string) {
  return formatDateOnly(value)
}

function formatDateTime(value: string) {
  return format(value)
}

function activityTo(activity: ProfileActivity) {
  return localePath(profileActivityLink(activity, topicUrlMode.value))
}

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
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
              <div class="sf-profile-intro__meta">
                <span v-if="profile.profile.location">
                  <UIcon name="i-lucide-map-pin" class="size-3.5" aria-hidden="true" />
                  {{ profile.profile.location }}
                </span>
                <a
                  v-if="profile.profile.websiteUrl"
                  :href="safeUrl(profile.profile.websiteUrl)"
                  target="_blank"
                  rel="noopener noreferrer nofollow"
                >
                  <UIcon name="i-lucide-link" class="size-3.5" aria-hidden="true" />
                  {{ profile.profile.websiteUrl.replace(/^https?:\/\//, '') }}
                </a>
                <span>
                  <UIcon name="i-lucide-calendar-days" class="size-3.5" aria-hidden="true" />
                  {{ t('profile.joinedOn', { date: formatDate(profile.joinedAt) }) }}
                </span>
              </div>
            </div>
            <div v-if="isSelf" class="sf-profile-intro__actions">
              <NuxtLink :to="localePath('/settings/profile')" class="sf-profile-edit-link">
                <UIcon name="i-lucide-settings-2" class="size-4" aria-hidden="true" />
                {{ t('profile.editProfile') }}
              </NuxtLink>
            </div>
          </header>

          <div class="sf-profile-summary" aria-label="profile statistics">
            <div class="sf-profile-summary__item">
              <strong>{{ profile.topicCount }}</strong>
              <span>{{ t('profile.topicCount') }}</span>
            </div>
            <div class="sf-profile-summary__item">
              <strong>{{ profile.commentCount }}</strong>
              <span>{{ t('profile.commentCount') }}</span>
            </div>
          </div>

          <section class="sf-profile-section-head">
            <div>
              <h2>{{ t('profile.timeline.title') }}</h2>
              <p>{{ t('profile.timeline.description') }}</p>
            </div>
          </section>

          <div v-if="activityGroups.length" class="sf-profile-timeline">
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
          </div>

          <div v-else class="sf-profile-empty">
            <UIcon name="i-lucide-message-square-off" class="size-7" aria-hidden="true" />
            <strong>{{ t('profile.timeline.emptyTitle') }}</strong>
            <span>{{ t('profile.timeline.emptyDescription') }}</span>
          </div>
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
