<script setup lang="ts">
import type { PublicProfile } from '~/composables/profile/useProfileApi'
import { forumTopicPath, type ForumTopicSummary } from '~/utils/forum/forumTaxonomy'
import { safeUrl } from '~/utils/sfUrl'

const props = defineProps<{
  profile: PublicProfile
  displayName: string
  hasPublicDetails: boolean
  recentTopics: ForumTopicSummary[]
  drawer?: boolean
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()
const { formatDateOnly } = useSiteDateTime()
const { seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)

function topicTo(topic: ForumTopicSummary) {
  return localePath(forumTopicPath(topic, topicUrlMode.value))
}

// 与 SFProfileShowPage 同口径：safeUrl 拒绝非法 scheme 时返回 ''，
// 显隐必须以净化结果为准，否则会渲染出 href 为空的死链接。
const websiteHref = computed(() => {
  const raw = props.profile.profile.websiteUrl?.trim() || ''
  return raw ? safeUrl(raw) : ''
})

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
  <div class="sf-profile-right-content" :class="{ 'sf-profile-right-content--drawer': drawer }">
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
      <!-- 与 demo B1 右侧一致：位置 / 网站 / 加入时间优先；签名可选 -->
      <ul class="sf-profile-detail-list">
        <li v-if="profile.profile.location">
          <UIcon name="i-lucide-map-pin" class="size-4" aria-hidden="true" />
          <span>{{ profile.profile.location }}</span>
        </li>
        <li v-if="websiteHref">
          <UIcon name="i-lucide-link" class="size-4" aria-hidden="true" />
          <a :href="websiteHref" target="_blank" rel="noopener noreferrer nofollow">
            {{ (profile.profile.websiteUrl || '').replace(/^https?:\/\//i, '').replace(/\/$/, '') }}
          </a>
        </li>
        <li>
          <UIcon name="i-lucide-calendar-days" class="size-4" aria-hidden="true" />
          <span>{{ t('profile.joinedOn', { date: formatDateOnly(profile.joinedAt) }) }}</span>
        </li>
        <li v-if="profile.profile.signature">
          <UIcon name="i-lucide-quote" class="size-4" aria-hidden="true" />
          <span>{{ profile.profile.signature }}</span>
        </li>
      </ul>
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
  </div>
</template>
