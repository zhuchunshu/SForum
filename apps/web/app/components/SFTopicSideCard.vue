<script setup lang="ts">
import type { ForumTopicDetail, ForumTopicExtensionSidebarItem } from '~/utils/forumTaxonomy'
import { forumTopicExtensionLabel } from '~/utils/forumTaxonomy'

const props = defineProps<{
  topic: ForumTopicDetail
  authorName: string
  authorTo?: string
  tags: { id: number, name: string, to: string }[]
  categoryTo: string
  firstCommentId?: number
  latestCommentId?: number
  /** forum.topic.sidebar；空时不渲染扩展卡片区 */
  extensionSidebar?: ForumTopicExtensionSidebarItem[]
}>()

const { t, locale } = useI18n()
const localePath = useLocalePath()
const { request } = useApiClient()
const toast = useToast()

const statusLabel = computed(() => {
  if (props.topic.status === 'locked') {
    return t('topicDetail.badge.locked')
  }
  if (props.topic.isPinned) {
    return t('topicDetail.badge.pinned')
  }
  return t('topicDetail.side.statusActive')
})

const sidebarItems = computed(() => props.extensionSidebar || [])
const runningKey = ref('')
const currentCommentIndex = ref(1)
let commentObserver: IntersectionObserver | null = null

const commentProgress = computed(() => {
  const total = Math.max(1, props.topic.commentCount)
  return Math.min(100, Math.max(4, currentCommentIndex.value / total * 100))
})

onMounted(async () => {
  await nextTick()
  const comments = Array.from(document.querySelectorAll<HTMLElement>('.sf-comment-list > .sf-comment'))
  if (!comments.length) return

  commentObserver = new IntersectionObserver((entries) => {
    const visible = entries
      .filter(entry => entry.isIntersecting)
      .sort((a, b) => Math.abs(a.boundingClientRect.top) - Math.abs(b.boundingClientRect.top))[0]
    if (!visible) return
    const index = comments.indexOf(visible.target as HTMLElement)
    if (index >= 0) currentCommentIndex.value = index + 1
  }, { rootMargin: '-20% 0px -60% 0px', threshold: 0 })
  comments.forEach(comment => commentObserver?.observe(comment))
})

onBeforeUnmount(() => commentObserver?.disconnect())

function itemLabel(item: ForumTopicExtensionSidebarItem) {
  return forumTopicExtensionLabel(item, String(locale.value || 'zh-CN')) || item.id
}

function hostLinkTo(item: ForumTopicExtensionSidebarItem) {
  const href = `${item.url || ''}`.trim()
  if (!href.startsWith('/') || href.startsWith('//') || href.includes('://') || href.startsWith('/api')) {
    return ''
  }
  return localePath(href)
}

function extensionRoutePath(item: ForumTopicExtensionSidebarItem) {
  const url = `${item.url || ''}`.trim()
  if (!url.startsWith('/extensions/') || url.includes('://') || url.includes('..') || url.startsWith('/api')) {
    return ''
  }
  return url
}

async function runSidebarExtensionRoute(item: ForumTopicExtensionSidebarItem) {
  const path = extensionRoutePath(item)
  const method = `${item.method || 'GET'}`.toUpperCase()
  if (!path || runningKey.value) {
    return
  }
  const key = `${item.extensionId}:${item.id}`
  runningKey.value = key
  try {
    await request(path, {
      method: method as 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
      body: method === 'GET' ? undefined : { topicId: props.topic.id }
    })
    toast.add({
      color: 'primary',
      icon: item.icon || 'i-lucide-check',
      title: itemLabel(item),
      duration: 10000
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-alert-circle',
      title: apiErrorMessage(error) || t('topicDetail.actionFailed'),
      // 错误 Toast 不自动关闭（项目约定）
      duration: 0
    })
  } finally {
    runningKey.value = ''
  }
}
</script>

<template>
  <aside class="sforum-topic-page__side" :aria-label="t('topicDetail.side.title')">
    <div class="sf-topic-side-card">
      <h3>{{ t('topicDetail.side.title') }}</h3>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.status') }}</span>
        <strong>{{ statusLabel }}</strong>
      </div>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.category') }}</span>
        <strong>
          <NuxtLink :to="categoryTo">{{ topic.categoryName }}</NuxtLink>
        </strong>
      </div>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.replies') }}</span>
        <strong>{{ topic.commentCount }}</strong>
      </div>
      <div class="sf-topic-side-card__row">
        <span>{{ t('topicDetail.side.views') }}</span>
        <strong>{{ topic.viewCount }}</strong>
      </div>
    </div>

    <div class="sf-topic-side-card">
      <h3>{{ t('topicDetail.side.participants') }}</h3>
      <div class="sf-topic-side-card__participants">
        <NuxtLink v-if="authorTo" :to="authorTo" :aria-label="authorName">
          <SFAvatar :name="authorName" :avatar="topic.author?.avatar" size="md" />
        </NuxtLink>
        <SFAvatar v-else :name="authorName" :avatar="topic.author?.avatar" size="md" />
      </div>
    </div>

    <div v-if="tags.length" class="sf-topic-side-card">
      <h3>{{ t('topicDetail.side.tags') }}</h3>
      <div class="sf-topic-side-card__tags">
        <NuxtLink
          v-for="tag in tags"
          :key="tag.id"
          :to="tag.to"
          class="sf-topic-side-card__tag"
        >
          {{ tag.name }}
        </NuxtLink>
      </div>
    </div>

    <section v-if="topic.commentCount > 0" class="sf-topic-side-card sf-topic-side-card--progress">
      <header class="sf-topic-side-card__progress-head">
        <h3>{{ t('topicDetail.progress.label') }}</h3>
        <span><b>{{ currentCommentIndex }}</b> / {{ topic.commentCount }}</span>
      </header>
      <div class="sf-topic-side-card__progress-track" aria-hidden="true">
        <span :style="{ width: `${commentProgress}%` }" />
      </div>
      <div class="sf-topic-side-card__progress-actions">
        <a
          :href="firstCommentId ? `#comment-${firstCommentId}` : '#topic-latest'"
          :aria-label="t('topicDetail.progress.first')"
        >
          <UIcon name="i-lucide-arrow-up" class="size-4" aria-hidden="true" />
        </a>
        <a
          :href="latestCommentId ? `#comment-${latestCommentId}` : '#topic-latest'"
          :aria-label="t('topicDetail.progress.latest')"
        >
          <UIcon name="i-lucide-arrow-down" class="size-4" aria-hidden="true" />
        </a>
        <a
          class="sf-topic-side-card__latest-link"
          :href="latestCommentId ? `#comment-${latestCommentId}` : '#topic-latest'"
        >
          <UIcon name="i-lucide-corner-right-down" class="size-4" aria-hidden="true" />
          {{ t('topicDetail.progress.latest') }}
        </a>
      </div>
    </section>

    <nav class="sf-topic-side-card sf-topic-side-card--page-nav" :aria-label="t('topicDetail.side.pageNav')">
      <h3>{{ t('topicDetail.side.pageNav') }}</h3>
      <a href="#topic-start" class="is-active">{{ t('topicDetail.side.topicContent') }}</a>
      <a href="#topic-latest">
        {{ t('topicDetail.side.discussion') }}
        <span>{{ topic.commentCount }}</span>
      </a>
      <a href="#topic-reply-editor">{{ t('topicDetail.side.joinDiscussion') }}</a>
      <NuxtLink :to="localePath('/guidelines')">{{ t('home.sidebar.guidelines') }}</NuxtLink>
    </nav>

    <!-- 扩展侧栏卡片：无贡献时整块不渲染，保证空安全；顺序与宿主 order 一致 -->
    <div
      v-if="sidebarItems.length"
      class="sf-topic-side-card sf-topic-side-card--extensions"
      data-testid="topic-extension-sidebar"
    >
      <h3>{{ t('topicDetail.side.extensions') }}</h3>
      <div class="sf-topic-side-card__extensions">
        <template v-for="item in sidebarItems" :key="`${item.extensionId}:${item.id}`">
          <NuxtLink
            v-if="item.kind === 'hostLink' && hostLinkTo(item)"
            :to="hostLinkTo(item)"
            class="sf-topic-side-card__extension-link"
          >
            <UIcon
              v-if="item.icon"
              :name="item.icon"
              class="size-4"
              aria-hidden="true"
            />
            <span>{{ itemLabel(item) }}</span>
          </NuxtLink>
          <button
            v-else-if="item.kind === 'extensionRoute' && extensionRoutePath(item)"
            type="button"
            class="sf-topic-side-card__extension-link"
            :disabled="runningKey === `${item.extensionId}:${item.id}`"
            @click="runSidebarExtensionRoute(item)"
          >
            <UIcon
              v-if="item.icon"
              :name="item.icon"
              class="size-4"
              aria-hidden="true"
            />
            <span>{{ itemLabel(item) }}</span>
          </button>
        </template>
      </div>
    </div>
  </aside>
</template>
