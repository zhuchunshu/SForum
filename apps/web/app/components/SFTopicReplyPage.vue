<script setup lang="ts">
/**
 * 宿主 body 岛：forum.topic.reply。
 * 高级回复独立页：完整 SFEditor，提交后回话题详情。
 */

import {
  advancedReplyDraftStorageKey,
  forumContentFromEditorPayload,
  forumTopicPath,
  type ForumTopicDetail
} from '~/utils/forumTaxonomy'

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()
const { can } = usePermissions()
const toast = useToast()

const topicId = computed(() => {
  const raw = Array.isArray(route.query.topic) ? route.query.topic[0] : route.query.topic
  const id = Number(raw)
  return Number.isInteger(id) && id > 0 ? id : null
})

const parentId = computed(() => {
  const raw = Array.isArray(route.query.parent) ? route.query.parent[0] : route.query.parent
  const id = Number(raw)
  return Number.isInteger(id) && id > 0 ? id : null
})

const canReply = computed(() => can(FORUM_PERMISSIONS.postCreate))

const {
  data: topic,
  error: topicError,
  status: topicStatus
} = await useAsyncData(
  () => `topic-advanced-reply:${topicId.value || 'missing'}`,
  async () => {
    if (!topicId.value) {
      return null
    }
    return await forumApi.getTopic(topicId.value)
  },
  { default: () => null as ForumTopicDetail | null, watch: [topicId] }
)

const bodyMarkdown = ref('')
const submitState = ref<'idle' | 'submitting' | 'error'>('idle')
const errorMessage = ref('')

// 从紧凑评论框交接草稿（若有）。
onMounted(() => {
  if (!import.meta.client || !topicId.value) {
    return
  }
  try {
    const key = advancedReplyDraftStorageKey(topicId.value)
    const draft = sessionStorage.getItem(key)
    if (draft != null && draft !== '' && !bodyMarkdown.value) {
      bodyMarkdown.value = draft
    }
  } catch {
    // sessionStorage 不可用时忽略
  }
})

const topicReturnPath = computed(() => {
  if (!topic.value) {
    return localePath('/')
  }
  return localePath(forumTopicPath(topic.value, topicUrlMode.value))
})

const locked = computed(() => topic.value?.status === 'locked')
const missingTopic = computed(() => !topicId.value || (!topic.value && topicStatus.value === 'success') || Boolean(topicError.value))

useSForumSeo({
  title: () => {
    if (topic.value?.title) {
      return `${t('topicDetail.advancedReplyMetaTitle', { title: topic.value.title })} - ${siteName.value}`
    }
    return `${t('topicDetail.advancedReply')} - ${siteName.value}`
  },
  description: () => t('topicDetail.advancedReplyMetaDescription'),
  type: 'website'
})

const submitLabel = computed(() => {
  if (submitState.value === 'submitting') {
    return t('topicDetail.submitting')
  }
  return t('topicDetail.submitReply')
})

async function submit(payload?: { markdown?: string, native?: unknown, text?: string }) {
  if (!topic.value || submitState.value === 'submitting' || locked.value || !canReply.value) {
    return
  }
  const markdown = payload?.markdown ?? bodyMarkdown.value
  if (!(payload?.text || markdown).trim()) {
    return
  }
  const content = forumContentFromEditorPayload({
    markdown,
    native: payload?.native,
    text: payload?.text
  })
  submitState.value = 'submitting'
  errorMessage.value = ''
  try {
    const created = await forumApi.createTopicComment(topic.value.id, content, parentId.value)
    if (import.meta.client) {
      try {
        sessionStorage.removeItem(advancedReplyDraftStorageKey(topic.value.id))
      } catch {
        // ignore
      }
    }
    if (created.status === 'pending') {
      toast.add({
        color: 'primary',
        icon: 'i-lucide-clock-3',
        title: t('topicDetail.replySubmittedForReview'),
        duration: 10000
      })
    } else {
      toast.add({
        color: 'success',
        icon: 'i-lucide-check',
        title: t('topicDetail.replyPosted'),
        duration: 10000
      })
    }
    const anchor = created.id ? `#comment-${created.id}` : '#topic-latest'
    await navigateTo(`${topicReturnPath.value}${anchor}`)
  } catch (error) {
    submitState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('topicDetail.replyFailed')
  } finally {
    if (submitState.value === 'submitting') {
      submitState.value = 'idle'
    }
  }
}

function onEditorSubmit(payload: { markdown: string, native?: unknown, text?: string }) {
  void submit(payload)
}

function onCancel() {
  void navigateTo(topicReturnPath.value)
}
</script>

<template>
  <main class="sf-public-page min-h-screen py-8">
    <div class="sf-public-page__container mx-auto max-w-4xl px-4 sm:px-6">
      <div class="mb-5">
        <nav class="flex items-center gap-1.5 text-sm text-slate-400 dark:text-zinc-500">
          <NuxtLink :to="localePath('/')" class="hover:text-[color:var(--sf-accent)]">
            {{ t('topicDetail.breadcrumbHome') }}
          </NuxtLink>
          <UIcon name="i-lucide-chevron-right" class="size-3" />
          <NuxtLink
            v-if="topic"
            :to="topicReturnPath"
            class="min-w-0 truncate hover:text-[color:var(--sf-accent)]"
          >
            {{ topic.title }}
          </NuxtLink>
          <UIcon v-if="topic" name="i-lucide-chevron-right" class="size-3" />
          <span>{{ t('topicDetail.advancedReply') }}</span>
        </nav>
        <h1 class="mt-3 text-xl font-bold text-slate-900 dark:text-zinc-50">
          {{ t('topicDetail.advancedReply') }}
        </h1>
        <p v-if="topic" class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('topicDetail.advancedReplySubtitle', { title: topic.title }) }}
        </p>
      </div>

      <SFRegionOutlet page="forum.topic.reply" region="content_before" />

      <SFCard v-if="missingTopic" class="p-8">
        <SFEmptyState
          icon-label="?"
          :title="t('topicDetail.notFound.title')"
          :description="t('topicDetail.notFound.description')"
        />
      </SFCard>

      <SFCard v-else-if="!canReply" class="p-8">
        <SFEmptyState
          icon-label="LOCK"
          :title="t('topicDetail.advancedReplyDenied.title')"
          :description="t('topicDetail.advancedReplyDenied.description')"
        />
      </SFCard>

      <SFCard v-else-if="locked" class="p-8">
        <SFAlert variant="warning" :title="t('topicDetail.lockedNotice')" />
      </SFCard>

      <template v-else-if="topic">
        <SFAlert
          v-if="errorMessage"
          variant="danger"
          :title="errorMessage"
          closable
          class="mb-4"
          @close="errorMessage = ''"
        />

        <div
          v-if="parentId"
          class="mb-4 flex items-center gap-2 rounded-md border border-[color:var(--sf-border)] bg-[color:var(--sf-accent-soft)] px-3 py-2 text-sm text-slate-600 dark:text-zinc-300"
        >
          <UIcon name="i-lucide-corner-up-left" class="size-4 shrink-0" aria-hidden="true" />
          <span>{{ t('topicDetail.advancedReplyParent', { id: parentId }) }}</span>
        </div>

        <SFCard class="p-4 sm:p-6">
          <LazySFEditor
            v-model="bodyMarkdown"
            :rows="14"
            :placeholder="t('topicDetail.replyPlaceholder')"
            :submit-label="submitLabel"
            :cancel-label="t('topicDetail.cancel')"
            :disabled="submitState === 'submitting'"
            @cancel="onCancel"
            @submit="onEditorSubmit"
          />
        </SFCard>
      </template>

      <SFRegionOutlet page="forum.topic.reply" region="content_after" />
    </div>
  </main>
</template>
