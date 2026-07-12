<script setup lang="ts">
import {
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'
import { safeUrl } from '~/utils/sfUrl'

definePageMeta({ public: true })

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
  // extensionRoute：公开资料页以链接形式打开宿主代理路径（GET）或展示动作按钮。
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
const isSelf = computed(() => {
  return Boolean(profile.value && authUser.value?.id === profile.value.userId)
})

type ProfileTab = 'works' | 'topics' | 'comments' | 'following' | 'followers'
const activeTab = ref<ProfileTab>('topics')

const tabItems = computed(() => [
  { value: 'works' as const, label: t('profile.tabs.works') },
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
  { value: 'followers' as const, label: t('profile.tabs.followers') }
])

function formatDate(value: string) {
  return formatDateOnly(value)
}

function topicAuthor(topic: ForumTopicSummary) {
  return forumAuthorName(topic.author, topic.authorUserId)
}

// 关注 API 尚未落地：仅 UI 反馈，避免假数据误导。
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

const recentTopics = computed(() => profile.value?.recentTopics || [])
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
      <!-- 封面：默认主题渐变，上传封面后续再接附件 -->
      <div class="sf-profile-cover" role="img" :aria-label="t('profile.coverLabel')" />

      <div class="sf-profile-shell">
        <div class="sf-profile-head">
          <div class="sf-profile-avatar-ring">
            <SFAvatar :name="displayName" :avatar="profile.profile.avatar" size="lg" />
          </div>

          <div class="sf-profile-head__main">
            <h1 class="sf-profile-head__name">
              <span>{{ displayName }}</span>
            </h1>
            <p class="sf-profile-head__uname">
              @{{ profile.username }}
            </p>
          </div>

          <div class="sf-profile-head__actions">
            <template v-if="isSelf">
              <NuxtLink :to="localePath('/my')">
                <SFButton variant="secondary" size="sm">
                  <UIcon name="i-lucide-layout-dashboard" class="size-4" />
                  <span>{{ t('profile.goToMyCenter') }}</span>
                </SFButton>
              </NuxtLink>
              <NuxtLink :to="localePath('/settings/profile')">
                <SFButton variant="ghost" size="sm">
                  <UIcon name="i-lucide-settings" class="size-4" />
                  <span>{{ t('profile.editProfile') }}</span>
                </SFButton>
              </NuxtLink>
            </template>
            <template v-else>
              <SFButton variant="primary" size="sm" @click="onFollowClick">
                <UIcon name="i-lucide-user-plus" class="size-4" />
                <span>{{ t('profile.follow') }}</span>
              </SFButton>
            </template>
          </div>
        </div>

        <div class="sf-profile-stats">
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">{{ profile.topicCount }}</span>
            <span class="sf-profile-stats__label">{{ t('profile.topicCount') }}</span>
          </div>
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">{{ profile.commentCount }}</span>
            <span class="sf-profile-stats__label">{{ t('profile.commentCount') }}</span>
          </div>
          <!-- 关注关系 API 未就绪：占位展示，避免空白布局 -->
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">—</span>
            <span class="sf-profile-stats__label">{{ t('profile.followingCount') }}</span>
          </div>
          <div class="sf-profile-stats__item">
            <span class="sf-profile-stats__num">—</span>
            <span class="sf-profile-stats__label">{{ t('profile.followersCount') }}</span>
          </div>
        </div>

        <p v-if="profile.profile.bio" class="sf-profile-bio">
          {{ profile.profile.bio }}
        </p>

        <div class="sf-profile-meta">
          <span v-if="profile.profile.location" class="sf-profile-meta__item">
            <UIcon name="i-lucide-map-pin" class="size-3.5" />
            {{ profile.profile.location }}
          </span>
          <span v-if="profile.profile.websiteUrl" class="sf-profile-meta__item">
            <UIcon name="i-lucide-link" class="size-3.5" />
            <a
              :href="safeUrl(profile.profile.websiteUrl)"
              target="_blank"
              rel="noopener noreferrer nofollow"
            >{{ profile.profile.websiteUrl }}</a>
          </span>
          <span class="sf-profile-meta__item">
            <UIcon name="i-lucide-calendar" class="size-3.5" />
            {{ t('profile.joinedAt') }} {{ formatDate(profile.joinedAt) }}
          </span>
        </div>

        <!-- 扩展资料入口（宿主渲染，无插件 HTML） -->
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
          <!-- 作品集：领域模型未落地，先空状态引导 -->
          <div v-if="activeTab === 'works'" role="tabpanel">
            <SFEmptyState
              class="py-10"
              :title="t('profile.worksEmpty.title')"
              :description="t('profile.worksEmpty.description')"
            />
          </div>

          <div v-else-if="activeTab === 'topics'" role="tabpanel">
            <template v-if="recentTopics.length">
              <NuxtLink
                v-for="topic in recentTopics"
                :key="topic.id"
                :to="localePath(forumTopicPath(topic, topicUrlMode))"
                class="sf-profile-topic"
              >
                <p class="sf-profile-topic__title">
                  {{ topic.title }}
                </p>
                <p class="sf-profile-topic__meta">
                  {{ topicAuthor(topic) }}
                  · {{ formatDate(topic.lastActivityAt || topic.createdAt) }}
                  · {{ topic.commentCount }} {{ t('profile.replies') }}
                </p>
              </NuxtLink>
            </template>
            <SFEmptyState
              v-else
              class="py-10"
              :title="t('profile.topicsEmpty.title')"
              :description="t('profile.topicsEmpty.description')"
            />
          </div>

          <div v-else-if="activeTab === 'comments'" role="tabpanel">
            <SFEmptyState
              class="py-10"
              :title="t('profile.commentsEmpty.title')"
              :description="t('profile.commentsEmpty.description')"
            />
          </div>

          <div v-else-if="activeTab === 'following'" role="tabpanel">
            <SFEmptyState
              class="py-10"
              :title="t('profile.socialEmpty.followingTitle')"
              :description="t('profile.socialEmpty.followingDescription')"
            />
          </div>

          <div v-else role="tabpanel">
            <SFEmptyState
              class="py-10"
              :title="t('profile.socialEmpty.followersTitle')"
              :description="t('profile.socialEmpty.followersDescription')"
            />
          </div>
        </div>
      </div>
    </template>
  </main>
</template>
