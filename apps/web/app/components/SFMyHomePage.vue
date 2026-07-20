<script setup lang="ts">
/**
 * 宿主 body 岛：forum.my.home。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import {
  forumAuthorName,
  forumTopicPath,
  forumUserProfilePath,
  type ForumTopicSummary
} from '~/utils/forumTaxonomy'

// 登录后「我的中心」管理入口（demo 03），与公开 /u/:username 拆分。

const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const { formatDateOnly } = useSiteDateTime()
const profileApi = useProfileApi()
const { can } = usePermissions()

useSForumSeo({
  title: () => `${t('myCenter.metaTitle')} - ${siteName.value}`,
  description: () => t('myCenter.metaDescription'),
  type: 'website',
  noindex: true
})

const { data: profile, pending, error, refresh } = await useAsyncData(
  'my-center-profile',
  () => profileApi.getMyProfile(),
  { default: () => null as PublicProfile | null }
)

const displayName = computed(
  () => profile.value?.displayName || profile.value?.username || ''
)
const username = computed(() => profile.value?.username || '')
const publicPath = computed(() =>
  username.value ? localePath(forumUserProfilePath(username.value)) : localePath('/')
)

// 资料完整度：仅用已有字段，不引入新后端。
const completeness = computed(() => {
  if (!profile.value) {
    return { percent: 0, items: [] as { ok: boolean, label: string }[] }
  }
  const p = profile.value.profile
  const items = [
    {
      ok: p.avatar?.kind === 'uploaded' || Boolean(p.avatarAttachmentId),
      label: t('myCenter.checklist.avatar')
    },
    { ok: Boolean(p.bio?.trim()), label: t('myCenter.checklist.bio') },
    { ok: Boolean(p.location?.trim()), label: t('myCenter.checklist.location') },
    { ok: Boolean(p.websiteUrl?.trim()), label: t('myCenter.checklist.website') },
    { ok: Boolean(p.signature?.trim()), label: t('myCenter.checklist.signature') }
  ]
  const done = items.filter((i) => i.ok).length
  return {
    percent: Math.round((done / items.length) * 100),
    items
  }
})

const canCreateTopic = computed(() => can(FORUM_PERMISSIONS.topicCreate))

const activeTab = ref('overview')

const meTabs = computed(() => [
  { value: 'overview', label: t('myCenter.tabs.overview') },
  {
    value: 'topics',
    label: t('myCenter.tabs.topics'),
    badge: profile.value?.topicCount
  }
])

function formatDate(value: string) {
  return formatDateOnly(value)
}

function topicAuthor(topic: ForumTopicSummary) {
  return forumAuthorName(topic.author, topic.authorUserId)
}

const recentTopics = computed(() => profile.value?.recentTopics || [])
</script>

<template>

<main class="sf-public-page min-h-screen">
    <div class="sf-me-page">
      <div class="sf-me-top">
        <div>
          <h1>{{ t('myCenter.title') }}</h1>
          <p>{{ t('myCenter.subtitle') }}</p>
        </div>
        <div class="sf-me-top__actions">
          <NuxtLink v-if="username" :to="publicPath">
            <SFButton variant="ghost" size="sm">
              <UIcon name="i-lucide-external-link" class="size-4" />
              <span>{{ t('myCenter.viewPublic') }}</span>
            </SFButton>
          </NuxtLink>
          <NuxtLink :to="localePath('/settings/profile')">
            <SFButton variant="primary" size="sm">
              <UIcon name="i-lucide-settings" class="size-4" />
              <span>{{ t('myCenter.editProfile') }}</span>
            </SFButton>
          </NuxtLink>
        </div>
      </div>

      <div v-if="error && !profile" class="mb-4 space-y-2">
        <SFAlert variant="danger" :title="t('myCenter.loadFailed')" />
        <SFButton variant="ghost" size="sm" @click="() => refresh()">
          {{ t('myCenter.retry') }}
        </SFButton>
      </div>

      <div class="sf-me-cover" aria-hidden="true" />

      <div class="sf-me-card">
        <SFAvatar
          :name="displayName || t('myCenter.fallbackName')"
          :avatar="profile?.profile.avatar"
          size="lg"
        />
        <div class="sf-me-card__text">
          <h2>{{ pending && !profile ? '…' : displayName }}</h2>
          <p v-if="username">
            @{{ username }}
          </p>
        </div>
        <div v-if="profile" class="sf-me-card__stats">
          <div>
            <span class="n">{{ profile.topicCount }}</span>
            <span class="l">{{ t('profile.topicCount') }}</span>
          </div>
          <div>
            <span class="n">{{ profile.commentCount }}</span>
            <span class="l">{{ t('profile.commentCount') }}</span>
          </div>
          <div>
            <span class="n">{{ formatDate(profile.joinedAt) }}</span>
            <span class="l">{{ t('profile.joinedAt') }}</span>
          </div>
        </div>
      </div>

      <div class="sf-me-grid">
        <section class="sf-me-block">
          <h3>{{ t('myCenter.completenessTitle') }}</h3>
          <div class="sf-me-progress">
            <span>{{ t('myCenter.completenessLabel') }}</span>
            <b>{{ completeness.percent }}%</b>
          </div>
          <div class="sf-me-bar" role="progressbar" :aria-valuenow="completeness.percent" aria-valuemin="0" aria-valuemax="100">
            <i :style="{ width: `${completeness.percent}%` }" />
          </div>
          <ul class="sf-me-checklist">
            <li v-for="item in completeness.items" :key="item.label">
              <span
                class="sf-me-check"
                :class="item.ok ? 'sf-me-check--on' : 'sf-me-check--off'"
                aria-hidden="true"
              >{{ item.ok ? '✓' : '!' }}</span>
              {{ item.label }}
            </li>
          </ul>
          <div class="mt-3">
            <NuxtLink :to="localePath('/settings/profile')">
              <SFButton variant="secondary" size="sm" block>
                {{ t('myCenter.completeProfile') }}
              </SFButton>
            </NuxtLink>
          </div>
        </section>

        <section class="sf-me-block">
          <h3>{{ t('myCenter.shortcutsTitle') }}</h3>
          <div class="sf-me-quick">
            <NuxtLink v-if="canCreateTopic" :to="localePath('/topics/new')">
              <span class="t">{{ t('myCenter.shortcuts.newTopic') }}</span>
              <span class="d">{{ t('myCenter.shortcuts.newTopicHint') }}</span>
            </NuxtLink>
            <NuxtLink :to="localePath('/settings/profile')">
              <span class="t">{{ t('myCenter.shortcuts.profile') }}</span>
              <span class="d">{{ t('myCenter.shortcuts.profileHint') }}</span>
            </NuxtLink>
            <NuxtLink :to="localePath('/settings/security')">
              <span class="t">{{ t('myCenter.shortcuts.security') }}</span>
              <span class="d">{{ t('myCenter.shortcuts.securityHint') }}</span>
            </NuxtLink>
            <NuxtLink :to="localePath('/my/content-review')">
              <span class="t">{{ t('myCenter.shortcuts.review') }}</span>
              <span class="d">{{ t('myCenter.shortcuts.reviewHint') }}</span>
            </NuxtLink>
            <NuxtLink :to="localePath('/notifications')">
              <span class="t">{{ t('myCenter.shortcuts.notifications') }}</span>
              <span class="d">{{ t('myCenter.shortcuts.notificationsHint') }}</span>
            </NuxtLink>
            <NuxtLink v-if="username" :to="publicPath">
              <span class="t">{{ t('myCenter.shortcuts.public') }}</span>
              <span class="d">{{ t('myCenter.shortcuts.publicHint') }}</span>
            </NuxtLink>
          </div>
        </section>

        <section class="sf-me-block">
          <h3>{{ t('myCenter.tipsTitle') }}</h3>
          <p class="sf-me-hint">
            {{ t('myCenter.tipsBody') }}
          </p>
          <div class="mt-4 flex flex-col gap-2">
            <NuxtLink :to="localePath('/settings/profile')">
              <SFButton variant="secondary" size="sm" block>
                {{ t('profileSettings.title') }}
              </SFButton>
            </NuxtLink>
            <NuxtLink :to="localePath('/settings/security')">
              <SFButton variant="ghost" size="sm" block>
                {{ t('accountSecurity.title') }}
              </SFButton>
            </NuxtLink>
          </div>
        </section>
      </div>

      <div class="sf-me-tabs-card">
        <div class="sf-me-tabs-card__bar">
          <SFTabs
            v-model="activeTab"
            :items="meTabs"
            :aria-label="t('myCenter.tabsAria')"
          />
        </div>
        <div class="sf-me-tabs-card__body">
          <div v-if="activeTab === 'overview'">
            <p v-if="profile?.profile.bio" class="sf-profile-bio !max-w-none !mb-4">
              {{ profile.profile.bio }}
            </p>
            <p v-else class="text-sm text-slate-500 dark:text-zinc-400 py-6 text-center">
              {{ t('myCenter.noBio') }}
            </p>
            <div v-if="profile?.profile.signature" class="text-sm text-slate-600 dark:text-zinc-300 border-t border-slate-100 dark:border-zinc-800 pt-4">
              <span class="text-xs font-semibold uppercase tracking-wide text-slate-400 dark:text-zinc-500">
                {{ t('profileSettings.signature') }}
              </span>
              <p class="mt-1 whitespace-pre-wrap">
                {{ profile.profile.signature }}
              </p>
            </div>
          </div>

          <div v-else>
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
              :action-label="canCreateTopic ? t('myCenter.shortcuts.newTopic') : undefined"
              @action="() => navigateTo(localePath('/topics/new'))"
            />
          </div>
        </div>
      </div>
    </div>
  </main>
</template>
