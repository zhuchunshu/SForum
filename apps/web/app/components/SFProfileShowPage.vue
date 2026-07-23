<script setup lang="ts">
/**
 * 宿主 body 岛：forum.profile.show。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import {
  forumTopicPath,
  forumUserProfilePath,
  type ForumCategoryGroup,
  type ForumTopicSummary
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

function topicTo(topic: ForumTopicSummary) {
  return localePath(forumTopicPath(topic, topicUrlMode.value))
}

function activityTo(activity: ProfileActivity) {
  return localePath(profileActivityLink(activity, topicUrlMode.value))
}

function extensionTabLabel(tab: NonNullable<PublicProfile['extensionTabs']>[number]) {
  const labels = tab.label || {}
  return labels[String(locale.value)] || labels['zh-CN'] || labels['en-US'] || Object.values(labels)[0] || tab.id
}

function extensionTabTo(tab: NonNullable<PublicProfile['extensionTabs']>[number]) {
  if (tab.kind === 'hostLink') {
    return localePath(tab.url)
  }
  return tab.url.startsWith('/') ? tab.url : `/${tab.url}`
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

        <aside class="sforum-home__right sf-profile-right" aria-label="profile details">
          <section class="sf-profile-side-section sf-profile-side-identity">
            <div class="sf-profile-side-identity__main">
              <SFAvatar :name="displayName" :avatar="profile.profile.avatar" size="md" />
              <span>
                <strong>{{ displayName }}</strong>
                <small>@{{ profile.username }} · UID {{ profile.userId }}</small>
              </span>
            </div>
            <div class="sf-profile-side-stats">
              <div>
                <strong>{{ profile.topicCount }}</strong>
                <span>{{ t('profile.topicCount') }}</span>
              </div>
              <div>
                <strong>{{ profile.commentCount }}</strong>
                <span>{{ t('profile.commentCount') }}</span>
              </div>
            </div>
          </section>

          <section class="sf-profile-side-section">
            <header class="sf-profile-side-section__head">
              <h3>{{ t('profile.publicDetails') }}</h3>
              <span>{{ t('profile.memberInfo') }}</span>
            </header>
            <ul v-if="hasPublicDetails" class="sf-profile-detail-list">
              <li v-if="profile.profile.bio">
                <UIcon name="i-lucide-user-round" class="size-4" aria-hidden="true" />
                <span>{{ profile.profile.bio }}</span>
              </li>
              <li v-if="profile.profile.signature">
                <UIcon name="i-lucide-quote" class="size-4" aria-hidden="true" />
                <span>{{ profile.profile.signature }}</span>
              </li>
              <li v-if="profile.profile.location">
                <UIcon name="i-lucide-map-pin" class="size-4" aria-hidden="true" />
                <span>{{ profile.profile.location }}</span>
              </li>
              <li v-if="profile.profile.websiteUrl">
                <UIcon name="i-lucide-link" class="size-4" aria-hidden="true" />
                <a :href="safeUrl(profile.profile.websiteUrl)" target="_blank" rel="noopener noreferrer nofollow">
                  {{ profile.profile.websiteUrl.replace(/^https?:\/\//, '') }}
                </a>
              </li>
            </ul>
            <p v-else class="sf-profile-side-empty">{{ t('profile.publicDetailsEmpty') }}</p>
          </section>

          <section class="sf-profile-side-section">
            <header class="sf-profile-side-section__head">
              <h3>{{ t('profile.recentTopics') }}</h3>
              <span>{{ t('profile.publicContent') }}</span>
            </header>
            <ol v-if="recentTopics.length" class="sf-profile-recent-list">
              <li v-for="(topic, index) in recentTopics" :key="topic.id">
                <span class="sf-profile-recent-list__rank">{{ String(index + 1).padStart(2, '0') }}</span>
                <a :href="topicTo(topic)">{{ topic.title }}</a>
                <span>{{ topic.commentCount }}</span>
              </li>
            </ol>
            <p v-else class="sf-profile-side-empty">{{ t('profile.recentTopicsEmpty') }}</p>
          </section>

          <section v-if="profile.extensionTabs?.length" class="sf-profile-side-section">
            <header class="sf-profile-side-section__head">
              <h3>{{ t('profile.extensionLinks') }}</h3>
              <span>{{ t('profile.publicContent') }}</span>
            </header>
            <div class="sf-profile-extension-links">
              <template v-for="tab in profile.extensionTabs" :key="`${tab.extensionId}:${tab.id}`">
                <NuxtLink v-if="tab.kind === 'hostLink'" :to="extensionTabTo(tab)">
                  <UIcon v-if="tab.icon" :name="tab.icon" class="size-4" aria-hidden="true" />
                  <span>{{ extensionTabLabel(tab) }}</span>
                </NuxtLink>
                <a v-else :href="extensionTabTo(tab)">
                  <UIcon v-if="tab.icon" :name="tab.icon" class="size-4" aria-hidden="true" />
                  <span>{{ extensionTabLabel(tab) }}</span>
                </a>
              </template>
            </div>
          </section>
        </aside>
      </div>
    </template>
  </main>
</template>
