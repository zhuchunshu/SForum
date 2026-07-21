<script setup lang="ts">
/**
 * 话题列表行（宿主岛子组件）。
 * 呈现用 Tailwind + 公共 token；主题换肤走 L0 变量，深度定制走 Component Registry / L2，
 * 不要在 theme.css 镜像本组件 class。
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
/** 列表 meta 最多 2 个标签，避免行高失控 */
const previewTags = computed(() => (props.topic.tags || []).slice(0, 2))

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
const chipBase =
  'inline-flex h-[18px] items-center rounded-[3px] px-1.5 text-[11px] font-medium leading-[18px] no-underline'
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

    <div class="min-w-0">
      <div class="flex flex-wrap items-center gap-x-1.5 gap-y-1">
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
        <h2 class="m-0 min-w-0 flex-1 text-sm font-semibold leading-snug text-[var(--sf-public-text)] [overflow-wrap:anywhere]">
          <NuxtLink
            :to="to"
            class="text-inherit no-underline hover:text-[var(--sf-accent)]"
          >
            {{ topic.title }}
          </NuxtLink>
        </h2>
      </div>

      <div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-[var(--sf-public-text-muted)]">
        <NuxtLink
          v-for="tag in previewTags"
          :key="tag.id"
          :to="localePath(`/tags/${tag.slug}`)"
          :class="[chipBase, 'border border-[var(--sf-public-border)] bg-transparent text-[var(--sf-public-text-muted)]']"
        >
          {{ tag.name }}
        </NuxtLink>
        <NuxtLink
          v-if="topic.author"
          class="font-medium text-[var(--sf-public-text-muted)] no-underline hover:text-[var(--sf-accent)]"
          :to="localePath(`/u/${topic.author.username}`)"
        >
          {{ authorName }}
        </NuxtLink>
        <span class="text-[#d0d4db]" aria-hidden="true">·</span>
        <time :datetime="topic.lastActivityAt || topic.createdAt">
          {{ activityLabel }}
        </time>
        <span class="sf-home-topic-row__mobile-replies hidden max-[720px]:inline-flex">
          {{ t('home.feed.replyCount', { count: topic.commentCount }) }}
        </span>
      </div>
    </div>

    <NuxtLink
      :to="localePath(`/c/${topic.categorySlug}`)"
      :class="[chipBase, categoryChipClass, 'sf-home-topic-row__category justify-self-start max-[1120px]:hidden']"
    >
      {{ topic.categoryName }}
    </NuxtLink>

    <div class="sf-home-topic-row__replies text-right text-[11px] leading-snug text-[var(--sf-public-text-muted)] max-[720px]:hidden">
      <b class="block text-sm font-semibold tabular-nums text-[var(--sf-public-text-secondary)]">
        {{ topic.commentCount }}
      </b>
      <span>{{ t('home.feed.repliesColumn') }}</span>
    </div>

    <div class="text-right text-[11px] leading-snug whitespace-nowrap text-[var(--sf-public-text-muted)] max-[720px]:hidden">
      <span
        v-if="topic.author"
        class="block font-medium text-[var(--sf-public-text-secondary)]"
      >
        {{ authorName }}
      </span>
      <time :datetime="topic.lastActivityAt || topic.createdAt">{{ activityLabel }}</time>
    </div>
  </article>
</template>
