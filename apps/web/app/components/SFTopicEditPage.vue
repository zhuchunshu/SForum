<script setup lang="ts">
/**
 * 宿主 body 岛：forum.topic.edit。
 * 主题编辑独立页：按路由 topicId 加载主题，复用 SFTopicEditor；
 * 保存后跳回详情页（slug 变了由详情页 canonical 兜底 301）。
 */

import {
  forumTopicPath,
  type ForumTopicDetail
} from '~/utils/forumTaxonomy'

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()

const topicId = computed(() => {
  const raw = Array.isArray(route.params.topicId) ? route.params.topicId[0] : route.params.topicId
  const id = Number(raw)
  return Number.isInteger(id) && id > 0 ? id : null
})

const {
  data: topic,
  error: topicError,
  status: topicStatus
} = await useAsyncData(
  () => `topic-edit:${topicId.value || 'missing'}`,
  async () => {
    if (!topicId.value) {
      return null
    }
    return await forumApi.getTopic(topicId.value)
  },
  { default: () => null as ForumTopicDetail | null, watch: [topicId] }
)

const topicReturnPath = computed(() => {
  if (!topic.value) {
    return localePath('/')
  }
  return localePath(forumTopicPath(topic.value, topicUrlMode.value))
})

const missingTopic = computed(() => !topicId.value || (!topic.value && topicStatus.value === 'success') || Boolean(topicError.value))

// 编辑期间他人已保存（revision 冲突）时的阻断提示；不自动关闭。
const conflictMessage = ref('')

useSForumSeo({
  title: () => {
    if (topic.value?.title) {
      return `${t('composer.editTitle')}: ${topic.value.title} - ${siteName.value}`
    }
    return `${t('composer.editTitle')} - ${siteName.value}`
  },
  description: () => t('composer.editMetaDescription'),
  type: 'website'
})

const toast = useToast()

async function onTopicSaved(updated: ForumTopicDetail) {
  toast.add({ color: 'success', icon: 'i-lucide-check', title: t('topicDetail.topicUpdated'), duration: 10000 })
  // 用更新后的 topic 生成路径：标题变化可能产生新 slug。
  await navigateTo(localePath(forumTopicPath(updated, topicUrlMode.value)))
}

function onCancel() {
  void navigateTo(topicReturnPath.value)
}

function onConflict() {
  conflictMessage.value = t('composer.editConflict')
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
          <span>{{ t('composer.editTitle') }}</span>
        </nav>
        <h1 class="mt-3 text-xl font-bold text-slate-900 dark:text-zinc-50">
          {{ t('composer.editTitle') }}
        </h1>
      </div>

      <SFRegionOutlet page="forum.topic.edit" region="content_before" />

      <SFCard v-if="missingTopic" class="p-8">
        <SFEmptyState
          icon-label="?"
          :title="t('topicDetail.notFound.title')"
          :description="t('topicDetail.notFound.description')"
        />
      </SFCard>

      <template v-else-if="topic">
        <SFAlert
          v-if="conflictMessage"
          variant="danger"
          :title="conflictMessage"
          closable
          class="mb-4"
          @close="conflictMessage = ''"
        />

        <!-- 权限（作者/staff）校验由 SFTopicEditor 的 canEditTopic 与 API 双层负责。 -->
        <SFTopicEditor
          :topic="topic"
          @saved="onTopicSaved"
          @cancel="onCancel"
          @conflict="onConflict"
        />
      </template>

      <SFRegionOutlet page="forum.topic.edit" region="content_after" />
    </div>
  </main>
</template>
