<script setup lang="ts">
import { useForumApi } from '~/composables/forum/useForumApi'
import { forumTopicPath, type ForumTopicDetail } from '~/utils/forum/forumTaxonomy'

/**
 * 旧 forum.topic.reply 兼容入口。
 * 高级回复已迁回话题页统一抽屉；历史链接只负责定位话题与可选父评论。
 */
const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { seoSettings } = useWebOptions()
const forumApi = useForumApi()

const topicId = computed(() => {
  const raw = Array.isArray(route.query.topic) ? route.query.topic[0] : route.query.topic
  const id = Number(raw)
  return Number.isInteger(id) && id > 0 ? id : 0
})

const parentId = computed(() => {
  const raw = Array.isArray(route.query.parent) ? route.query.parent[0] : route.query.parent
  const id = Number(raw)
  return Number.isInteger(id) && id > 0 ? id : 0
})

if (!topicId.value) {
  throw createError({ statusCode: 404, statusMessage: t('topicDetail.notFound.title') })
}

const { data: topic } = await useAsyncData(
  () => `topic-reply-redirect:${topicId.value}`,
  () => forumApi.getTopic(topicId.value),
  { default: () => null as ForumTopicDetail | null }
)

if (!topic.value) {
  throw createError({ statusCode: 404, statusMessage: t('topicDetail.notFound.title') })
}

const query: Record<string, string> = { compose: 'advanced' }
if (parentId.value > 0) query.parent = String(parentId.value)

await navigateTo({
  path: localePath(forumTopicPath(topic.value, seoSettings.value.topicUrlMode)),
  query,
  hash: parentId.value > 0 ? `#comment-${parentId.value}` : ''
}, { redirectCode: 302, replace: true })
</script>

<template>
  <main class="sf-public-page min-h-screen py-8">
    <div class="mx-auto max-w-4xl px-4 text-sm text-[color:var(--sf-public-text-muted)]">
      {{ t('topicDetail.composerDrawer.redirecting') }}
    </div>
  </main>
</template>
