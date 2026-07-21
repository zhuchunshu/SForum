<script setup lang="ts">
/**
 * 宿主 body 岛：forum.profile.show。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import {
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'
import { safeUrl } from '~/utils/sfUrl'


const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const toast = useToast()
const { seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const { formatDateOnly } = useSiteDateTime()
const profileApi = useProfileApi()
const { user: authUser } = useAuthSession()

const username = computed(() => String(route.params.username ?? ''))

const { data: profile, error: profileError } = await useAsyncData(
  () => `forum-profile-${username.value}`,
  () => profileApi.getPublicProfile(username.value),
  { default: () => null as PublicProfile | null }
)

const { locale } = useI18n()
const extensionTabs = computed(() => profile.value?.extensionTabs || [])

function profileTabLabel(tab: NonNullable<PublicProfile['extensionTabs']>[number]) {
  const labels = tab.label || {}
  return labels[String(locale.value)] || labels['zh-CN'] || labels['en-US'] || Object.values(labels)[0] || tab.id
}

function profileTabTo(tab: NonNullable<PublicProfile['extensionTabs']>[number]) {
  if (tab.kind === 'hostLink') {
    return localePath(tab.url)
  }
  return tab.url.startsWith('/') ? tab.url : `/${tab.url}`
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

// 对齐 demo：默认落在作品集
type ProfileTab = 'works' | 'topics' | 'comments' | 'following' | 'followers' | 'likes'
const activeTab = ref<ProfileTab>('works')

const recentTopics = computed(() => profile.value?.recentTopics || [])

const tabItems = computed(() => [
  {
    value: 'works' as const,
    label: t('profile.tabs.works'),
    // 作品域未独立：用最近主题数作展示计数（有则显示）
    count: recentTopics.value.length || undefined
  },
  {
    value: 'topics' as const,
    label: t('profile.tabs.topics'),
    count: profile.value?.topicCount
  },
  {
    value: 'comments' as const,
    label: t('profile.tabs.comments'),
    count: profile.value?.commentCount
  },
  { value: 'following' as const, label: t('profile.tabs.following') },
  { value: 'followers' as const, label: t('profile.tabs.followers') },
  { value: 'likes' as const, label: t('profile.tabs.likes') }
])

function formatDate(value: string) {
  return formatDateOnly(value)
}

function topicAuthor(topic: ForumTopicSummary) {
  return forumAuthorName(topic.author, topic.authorUserId)
}

function workThumbClass(index: number) {
  return `sf-profile-work-thumb--${(index % 5) + 1}`
}

function onFollowClick() {
  if (!authUser.value) {
    return navigateTo({
      path: localePath('/login'),
      query: { redirect: route.fullPath }
    })
  }
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-info',
    title: t('profile.followComingSoon')
  })
}

function onShare() {
  const path = forumUserProfilePath(username.value)
  const url = import.meta.client ? `${window.location.origin}${localePath(path)}` : path
  if (import.meta.client && navigator.clipboard?.writeText) {
    void navigator.clipboard.writeText(url).then(() => {
      toast.add({
        color: 'success',
        icon: 'i-lucide-check',
        title: t('profile.linkCopied')
      })
    }).catch(() => {
      toast.add({ color: 'neutral', icon: 'i-lucide-link', title: url })
    })
    return
  }
  toast.add({ color: 'neutral', icon: 'i-lucide-link', title: url })
}

function onCoverChange() {
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-image',
    title: t('profile.coverComingSoon')
  })
}

function onMore() {
  toast.add({
    color: 'neutral',
    icon: 'i-lucide-info',
    title: t('profile.moreComingSoon')
  })
}

const bioText = computed(() => {
  const bio = profile.value?.profile.bio?.trim()
  if (bio) return bio
  return ''
})
</script>

<template>

<main class="sf-public-page sf-profile-page min-h-screen">
    <template v-if="profileError && !profile">
      <div class="sf-profile-shell py-10">
        <SFCard class="p-10">
          <SFEmptyState
            :title="t('profile.notFound.title')"
            :description="t('profile.notFound.description')"
          />
        </SFCard>
      </div>
    </template>

    <template v-else-if="profile">
      <!-- 全宽封面 + 右上操作（对齐 demo） -->
      <div class="sf-profile-cover" role="img" :aria-label="t('profile.coverLabel')">
        <div class="sf-profile-cover__actions">
          <button
            v-if="isSelf"
            type="button"
            class="sf-profile-cover__btn"
            @click="onCoverChange"
          >
            {{ t('profile.changeCover') }}
          </button>
          <button
            type="button"
            class="sf-profile-cover__btn sf-profile-cover__btn--icon"
            :aria-label="t('profile.share')"
            @click="onShare"
          >
            <UIcon name="i-lucide-share-2" class="size-4" />
          </button>
        </div>
      </div>

      <div class="sf-profile-shell">
        <div class="sf-profile-head">
          <div class="sf-profile-avatar-ring">
            <SFAvatar :name="displayName" :avatar="profile.profile.avatar" size="lg" />
          </div>

          <div class="sf-profile-head__main">
            <h1 class="sf-profile-head__name">
              <span>{{ displayName }}</span>
              <span class="sf-profile-badge sf-profile-badge--teal">{{ t('profile.memberBadge') }}</span>
              <span
                v-if="profile.userId"
                class="sf-profile-badge sf-profile-badge--slate"
              >UID {{ profile.userId }}</span>
            </h1>
            <p class="sf-profile-head__uname">
              @{{ profile.username }}
            </p>
          </div>

          <div class="sf-profile-head__actions">
            <template v-if="isSelf">
              <NuxtLink :to="localePath('/settings/profile')">
                <SFButton variant="secondary" size="sm">
                  <UIcon name="i-lucide-settings" class="size-4" />
                  <span>{{ t('profile.editProfile') }}</span>
                </SFButton>
              </NuxtLink>
            </template>
            <template v-else>
              <button
                type="button"
                class="sf-profile-icon-btn"
                :aria-label="t('profile.messageComingSoon')"
                @click="onMore"
              >
                <UIcon name="i-lucide-message-circle" class="size-4" />
              </button>
              <button
                type="button"
                class="sf-profile-icon-btn"
                :aria-label="t('profile.more')"
                @click="onMore"
              >
                <UIcon name="i-lucide-ellipsis" class="size-4" />
              </button>
              <SFButton variant="primary" size="sm" @click="onFollowClick">
                {{ t('profile.follow') }}
              </SFButton>
            </template>
          </div>
        </div>

        <!-- 统计顺序对齐 demo：关注 / 粉丝 / 主题 / 获赞 -->
        <div class="sf-profile-stats">
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">0</span>
            <span class="sf-profile-stats__label">{{ t('profile.followingCount') }}</span>
          </div>
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">0</span>
            <span class="sf-profile-stats__label">{{ t('profile.followersCount') }}</span>
          </div>
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">{{ profile.topicCount }}</span>
            <span class="sf-profile-stats__label">{{ t('profile.topicCount') }}</span>
          </div>
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">0</span>
            <span class="sf-profile-stats__label">{{ t('profile.likesCount') }}</span>
          </div>
        </div>

        <p v-if="bioText" class="sf-profile-bio">
          {{ bioText }}
        </p>
        <p v-else class="sf-profile-bio sf-profile-bio--empty">
          {{ t('profile.bioEmpty') }}
        </p>

        <div class="sf-profile-meta">
          <span v-if="profile.profile.location" class="sf-profile-meta__item">
            <UIcon name="i-lucide-map-pin" class="size-3.5" />
            {{ profile.profile.location }}
          </span>
          <span v-if="profile.profile.websiteUrl" class="sf-profile-meta__item">
            <UIcon name="i-lucide-globe" class="size-3.5" />
            <a
              :href="safeUrl(profile.profile.websiteUrl)"
              target="_blank"
              rel="noopener noreferrer nofollow"
            >{{ profile.profile.websiteUrl.replace(/^https?:\/\//, '') }}</a>
          </span>
          <span class="sf-profile-meta__item">
            <UIcon name="i-lucide-calendar" class="size-3.5" />
            {{ t('profile.joinedOn', { date: formatDate(profile.joinedAt) }) }}
          </span>
        </div>

        <div v-if="extensionTabs.length" class="sf-profile-ext">
          <template v-for="tab in extensionTabs" :key="`${tab.extensionId}:${tab.id}`">
            <NuxtLink
              v-if="tab.kind === 'hostLink'"
              :to="profileTabTo(tab)"
            >
              <UIcon v-if="tab.icon" :name="tab.icon" class="size-4" />
              <span>{{ profileTabLabel(tab) }}</span>
            </NuxtLink>
            <a
              v-else
              :href="profileTabTo(tab)"
            >
              <UIcon v-if="tab.icon" :name="tab.icon" class="size-4" />
              <span>{{ profileTabLabel(tab) }}</span>
            </a>
          </template>
        </div>

        <div class="sf-profile-tabs-bar">
          <nav class="sf-profile-tabs" role="tablist" :aria-label="t('profile.tabsAria')">
            <button
              v-for="item in tabItems"
              :key="item.value"
              type="button"
              role="tab"
              class="sf-profile-tabs__item"
              :class="{ 'is-active': activeTab === item.value }"
              :aria-selected="activeTab === item.value ? 'true' : 'false'"
              @click="activeTab = item.value"
            >
              {{ item.label }}
              <span v-if="item.count != null" class="sf-profile-tabs__count">{{ item.count }}</span>
            </button>
          </nav>
        </div>

        <div class="sf-profile-panel">
          <!-- 作品集：有主题时用卡片栅格呈现（demo 视觉）；无则空状态 -->
          <div v-if="activeTab === 'works'" role="tabpanel">
            <div class="sf-profile-section-head">
              <h3>{{ t('profile.featuredWorks') }}</h3>
              <div class="sf-profile-chips" aria-hidden="true">
                <span class="sf-profile-chip is-active">{{ t('profile.filterAll') }}</span>
                <span class="sf-profile-chip">{{ t('profile.filterTutorial') }}</span>
                <span class="sf-profile-chip">{{ t('profile.filterOpenSource') }}</span>
              </div>
            </div>

            <div v-if="recentTopics.length" class="sf-profile-work-grid">
              <NuxtLink
                v-for="(topic, index) in recentTopics"
                :key="topic.id"
                :to="localePath(forumTopicPath(topic, topicUrlMode))"
                class="sf-profile-work-card"
              >
                <div class="sf-profile-work-thumb" :class="workThumbClass(index)">
                  <div class="sf-profile-work-thumb__play" aria-hidden="true">▸</div>
                </div>
                <div class="sf-profile-work-body">
                  <h4>{{ topic.title }}</h4>
                  <p>
                    {{ topic.commentCount }} {{ t('profile.replies') }}
                    · {{ formatDate(topic.lastActivityAt || topic.createdAt) }}
                  </p>
                </div>
              </NuxtLink>
            </div>
            <div v-else class="sf-profile-empty-hint">
              <strong>{{ t('profile.worksEmpty.title') }}</strong>
              {{ t('profile.worksEmpty.description') }}
            </div>
          </div>

          <div v-else-if="activeTab === 'topics'" role="tabpanel">
            <template v-if="recentTopics.length">
              <NuxtLink
                v-for="topic in recentTopics"
                :key="topic.id"
                :to="localePath(forumTopicPath(topic, topicUrlMode))"
                class="sf-profile-topic"
              >
                <SFAvatar
                  :name="displayName"
                  :avatar="profile.profile.avatar"
                  size="sm"
                />
                <div class="sf-profile-topic__body">
                  <p class="sf-profile-topic__title">
                    {{ topic.title }}
                  </p>
                  <p v-if="topic.excerpt" class="sf-profile-topic__excerpt">
                    {{ topic.excerpt }}
                  </p>
                  <div class="sf-profile-topic__meta">
                    <span>{{ topicAuthor(topic) }}</span>
                    <span>{{ formatDate(topic.lastActivityAt || topic.createdAt) }}</span>
                    <span>{{ topic.commentCount }} {{ t('profile.replies') }}</span>
                  </div>
                </div>
              </NuxtLink>
            </template>
            <div v-else class="sf-profile-empty-hint">
              <strong>{{ t('profile.topicsEmpty.title') }}</strong>
              {{ t('profile.topicsEmpty.description') }}
            </div>
          </div>

          <div v-else-if="activeTab === 'comments'" role="tabpanel">
            <div class="sf-profile-empty-hint">
              <strong>{{ t('profile.commentsEmpty.title') }}</strong>
              {{ t('profile.commentsEmpty.description') }}
            </div>
          </div>

          <div v-else-if="activeTab === 'following'" role="tabpanel">
            <div class="sf-profile-empty-hint">
              <strong>{{ t('profile.socialEmpty.followingTitle') }}</strong>
              {{ t('profile.socialEmpty.followingDescription') }}
            </div>
          </div>

          <div v-else-if="activeTab === 'followers'" role="tabpanel">
            <div class="sf-profile-empty-hint">
              <strong>{{ t('profile.socialEmpty.followersTitle') }}</strong>
              {{ t('profile.socialEmpty.followersDescription') }}
            </div>
          </div>

          <div v-else role="tabpanel">
            <div class="sf-profile-empty-hint">
              <strong>{{ t('profile.likesEmpty.title') }}</strong>
              {{ t('profile.likesEmpty.description') }}
            </div>
          </div>
        </div>
      </div>
    </template>
  </main>
</template>
