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
  /** 左侧 meta：发帖时间（createdAt） */
  createdLabel: string
  /** 右侧列：最近活动时间（lastActivityAt） */
  activityLabel: string
  /** forum.topic.list.badges；列表级一次解析后挂到每行；空时不渲染 */
  extensionListBadges?: ForumTopicExtensionBadge[]
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()

const authorName = computed(() => forumAuthorName(props.topic.author, props.topic.authorUserId))
/** 右侧「最近回复」：优先 lastReplyAuthor，否则回退楼主 */
const lastReplyAuthor = computed(() => props.topic.lastReplyAuthor || props.topic.author)
const lastReplyName = computed(() =>
  forumAuthorName(lastReplyAuthor.value, lastReplyAuthor.value?.id || props.topic.authorUserId)
)
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
  'inline-flex h-5 shrink-0 items-center rounded-[3px] px-1.5 text-xs font-semibold leading-5'
</script>

<template>
  <article
    class="sf-home-topic-row grid min-h-[88px] min-w-0 grid-cols-[42px_minmax(0,1fr)_96px_54px_104px] items-center gap-x-3 border-b border-[var(--sf-border-light,#eef0f3)] px-2 py-3.5 transition-colors duration-100 last:border-b-0 hover:bg-[var(--sf-public-row-hover)] max-[1120px]:grid-cols-[42px_minmax(0,1fr)_54px_104px] max-[720px]:min-h-[76px] max-[720px]:grid-cols-[36px_minmax(0,1fr)] max-[720px]:gap-x-2.5"
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
        <h2 class="sf-home-topic-row__title m-0 min-w-0 flex-1 truncate text-base font-semibold leading-snug text-[var(--sf-public-text)]">
          <NuxtLink
            :to="to"
            :prefetch="false"
            class="text-inherit no-underline hover:text-[var(--sf-accent)]"
          >
            {{ topic.title }}
          </NuxtLink>
        </h2>
      </div>

      <!-- 左侧 meta：作者 · 发帖时间；移动端附回复数 -->
      <div class="sf-home-topic-row__meta mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs font-medium text-[var(--sf-public-text-muted)]">
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
          <time :datetime="topic.createdAt">
            {{ createdLabel }}
          </time>
        </span>
        <span class="sf-home-topic-row__mobile-replies hidden max-[720px]:inline-flex">
          {{ t('home.feed.replyCount', { count: topic.commentCount }) }}
        </span>
      </div>
    </div>

    <NuxtLink
      :to="localePath(`/c/${topic.categorySlug}`)"
      class="sf-home-topic-row__category inline-flex h-auto min-h-6 items-center justify-self-start rounded-[5px] px-2.5 py-1 text-[11px] font-semibold leading-4 no-underline max-[1120px]:hidden"
      :class="categoryChipClass"
    >
      {{ topic.categoryName }}
    </NuxtLink>

    <div class="sf-home-topic-row__replies flex flex-col items-center gap-[3px] text-[11px] leading-snug text-[var(--sf-public-text-muted)] max-[720px]:hidden">
      <strong class="text-[15px] font-bold tabular-nums text-[var(--sf-public-text)]">
        {{ topic.commentCount }}
      </strong>
      <span>{{ t('home.feed.repliesColumn') }}</span>
    </div>

    <!-- 最近回复用户 + lastActivityAt -->
    <div class="sf-home-topic-row__activity flex flex-col items-start gap-[3px] text-[11px] leading-snug text-[var(--sf-public-text-muted)] max-[720px]:hidden">
      <NuxtLink
        v-if="lastReplyAuthor?.username"
        :to="localePath(`/u/${lastReplyAuthor.username}`)"
        class="max-w-[100px] truncate text-xs font-semibold text-[var(--sf-public-text)] no-underline hover:text-[var(--sf-accent)]"
      >
        {{ lastReplyName }}
      </NuxtLink>
      <strong
        v-else
        class="max-w-[100px] truncate text-xs font-semibold text-[var(--sf-public-text)]"
      >
        {{ lastReplyName || t('home.feed.activityColumn') }}
      </strong>
      <time :datetime="topic.lastActivityAt || topic.createdAt">{{ activityLabel }}</time>
    </div>
  </article>
</template>
