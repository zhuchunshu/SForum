<script setup lang="ts">
/**
 * 话题列表行（宿主岛子组件）。
 * 结构对齐 tmp/demos/sforum-hybrid-topic-list：头像 | 标题+meta | 分类 | 回复 | 最近活动。
 * 呈现用 Tailwind + 公共 token；主题换肤走 L0 变量，深度定制走 Component Registry / L2。
 */
import type { ForumTopicExtensionBadge, ForumTopicSummary } from '~/utils/forumTaxonomy'
import { forumAuthorName, forumTopicExtensionLabel } from '~/utils/forumTaxonomy'
import {
  forumCategoryChipToneClass,
  forumListBadgeToneClass
} from '~/utils/forumListPresentation'

const props = defineProps<{
  topic: ForumTopicSummary
  to: string
  activityLabel: string
  /** forum.topic.list.badges；列表级一次解析后挂到每行；空时不渲染 */
  extensionListBadges?: ForumTopicExtensionBadge[]
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()

const authorName = computed(() => forumAuthorName(props.topic.author, props.topic.authorUserId))
const listBadges = computed(() => props.extensionListBadges || [])

const categoryChipClass = computed(() => {
  const seed = props.topic.categorySlug || props.topic.categoryName || 'c'
  return forumCategoryChipToneClass(seed)
})

function listBadgeLabel(badge: ForumTopicExtensionBadge) {
  return forumTopicExtensionLabel(badge, String(locale.value || 'zh-CN')) || badge.id
}

function listBadgeHref(badge: ForumTopicExtensionBadge) {
  const href = `${badge.href || ''}`.trim()
  if (!href.startsWith('/') || href.startsWith('//') || href.includes('://') || href.startsWith('/api')) {
    return ''
  }
  return localePath(href)
}

const pillBase =
  'inline-flex h-[18px] shrink-0 items-center rounded-[3px] px-1.5 text-[11px] font-semibold leading-[18px]'
</script>

<template>
  <article
    class="sf-home-topic-row grid min-h-[82px] min-w-0 grid-cols-[42px_minmax(0,1fr)_88px_50px_96px] items-center gap-x-[11px] border-b border-[var(--sf-border-light,#eef0f3)] px-2 py-3 transition-colors duration-100 last:border-b-0 hover:bg-[var(--sf-public-row-hover)] max-[1120px]:grid-cols-[42px_minmax(0,1fr)_50px_96px] max-[720px]:min-h-[72px] max-[720px]:grid-cols-[36px_minmax(0,1fr)] max-[720px]:gap-x-2.5"
    :class="topic.isPinned ? 'bg-[#f3f4f6] hover:bg-[#eceef1] dark:bg-slate-400/10 dark:hover:bg-slate-400/15' : ''"
    data-sf-component="forum.topic_list_row"
  >
    <div class="flex items-center justify-center">
      <NuxtLink
        v-if="topic.author?.username"
        :to="localePath(`/u/${topic.author.username}`)"
        :aria-label="authorName"
        tabindex="-1"
        class="inline-flex"
      >
        <!-- 全局 SFAvatar：只传 AvatarView，list 尺寸；禁止列表侧手写字头/改尺寸 -->
        <SFAvatar
          :name="authorName"
          :avatar="topic.author?.avatar"
          size="list"
        />
      </NuxtLink>
      <SFAvatar
        v-else
        :name="authorName"
        :avatar="topic.author?.avatar"
        size="list"
      />
    </div>

    <div class="sf-home-topic-row__copy min-w-0">
      <div class="sf-home-topic-row__title-line flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1">
        <span
          v-if="topic.isPinned"
          :class="[pillBase, 'bg-[#fff3bf] text-[#9a6700]']"
        >
          {{ t('home.badge.pinned') }}
        </span>
        <span
          v-if="topic.status === 'locked'"
          :class="[pillBase, 'bg-slate-100 text-slate-600 dark:bg-slate-500/20 dark:text-slate-300']"
        >
          {{ t('home.badge.locked') }}
        </span>
        <template v-for="badge in listBadges" :key="`${badge.extensionId}:${badge.id}`">
          <NuxtLink
            v-if="listBadgeHref(badge)"
            :to="listBadgeHref(badge)"
            :class="[pillBase, forumListBadgeToneClass(badge.tone)]"
            data-testid="topic-list-extension-badge"
          >
            {{ listBadgeLabel(badge) }}
          </NuxtLink>
          <span
            v-else
            :class="[pillBase, forumListBadgeToneClass(badge.tone)]"
            data-testid="topic-list-extension-badge"
          >
            {{ listBadgeLabel(badge) }}
          </span>
        </template>
        <h2 class="sf-home-topic-row__title m-0 min-w-0 flex-1 truncate text-sm font-semibold leading-snug text-[var(--sf-public-text)]">
          <NuxtLink
            :to="to"
            class="text-inherit no-underline hover:text-[var(--sf-accent)]"
          >
            {{ topic.title }}
          </NuxtLink>
        </h2>
      </div>

      <!-- demo .topic-meta：作者 · 时间；移动端附回复数 -->
      <div class="sf-home-topic-row__meta mt-[5px] flex flex-wrap items-center gap-x-2 gap-y-1 text-[10px] font-medium text-[var(--sf-public-text-muted)]">
        <span class="inline-flex min-w-0 items-center gap-1">
          <NuxtLink
            v-if="topic.author"
            class="truncate font-medium text-[var(--sf-public-text-muted)] no-underline hover:text-[var(--sf-accent)]"
            :to="localePath(`/u/${topic.author.username}`)"
          >
            {{ authorName }}
          </NuxtLink>
          <span v-else class="truncate">{{ authorName }}</span>
          <span aria-hidden="true">·</span>
          <time :datetime="topic.lastActivityAt || topic.createdAt">
            {{ activityLabel }}
          </time>
        </span>
        <span class="sf-home-topic-row__mobile-replies hidden max-[720px]:inline-flex">
          {{ t('home.feed.replyCount', { count: topic.commentCount }) }}
        </span>
      </div>
    </div>

    <NuxtLink
      :to="localePath(`/c/${topic.categorySlug}`)"
      class="sf-home-topic-row__category inline-flex h-auto min-h-[22px] items-center justify-self-start rounded-[5px] px-2 py-1 text-[9px] font-semibold leading-[14px] no-underline max-[1120px]:hidden"
      :class="categoryChipClass"
    >
      {{ topic.categoryName }}
    </NuxtLink>

    <div class="sf-home-topic-row__replies flex flex-col items-center gap-[3px] text-[10px] leading-snug text-[var(--sf-public-text-muted)] max-[720px]:hidden">
      <strong class="text-sm font-bold tabular-nums text-[var(--sf-public-text)]">
        {{ topic.commentCount }}
      </strong>
      <span>{{ t('home.feed.repliesColumn') }}</span>
    </div>

    <!-- demo .topic-author：最近活动（当前合同仅有 author + lastActivityAt） -->
    <div class="sf-home-topic-row__activity flex flex-col items-start gap-[3px] text-[10px] leading-snug text-[var(--sf-public-text-muted)] max-[720px]:hidden">
      <strong
        v-if="topic.author"
        class="max-w-[94px] truncate font-semibold text-[var(--sf-public-text)]"
      >
        {{ authorName }}
      </strong>
      <strong
        v-else
        class="max-w-[94px] truncate font-semibold text-[var(--sf-public-text)]"
      >
        {{ t('home.feed.activityColumn') }}
      </strong>
      <time :datetime="topic.lastActivityAt || topic.createdAt">{{ activityLabel }}</time>
    </div>
  </article>
</template>
