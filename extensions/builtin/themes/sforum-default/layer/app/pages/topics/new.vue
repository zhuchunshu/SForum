<script setup lang="ts">
import {
  forumTopicPath,
  type ForumCategory,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'

definePageMeta({
  // 发帖需要登录；路由中间件由全局 auth 守卫处理，这里只声明意图。
  requiresAuth: true
})

const { t } = useI18n()
const localePath = useLocalePath()
const { siteName } = useWebOptions()
const forumApi = useForumApi()
const { can } = usePermissions()

useSForumSeo({
  title: () => `${t('composer.metaTitle')} - ${siteName.value}`,
  description: () => t('composer.metaDescription'),
  type: 'website'
})

// 没有发帖权限直接给出提示。
const canCreate = computed(() => can(FORUM_PERMISSIONS.topicCreate))

const { data: categoryGroups } = await useAsyncData(
  'composer-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))

const title = ref('')
const selectedCategorySlug = ref('')
const tagInput = ref('')
const tagDraft = ref<string[]>([])
const bodyMarkdown = ref('')

// 表单状态。
type SubmitState = 'idle' | 'submitting' | 'error' | 'success'
const submitState = ref<SubmitState>('idle')
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})

const submitLabel = computed(() => {
  if (submitState.value === 'submitting') {
    return t('composer.submitting')
  }
  return t('composer.submit')
})

const canSubmit = computed(() => {
  return canCreate.value && title.value.trim() !== '' && bodyMarkdown.value.trim() !== '' && submitState.value !== 'submitting'
})

function addTag() {
  const raw = tagInput.value.trim().toLowerCase()
  if (!raw) {
    return
  }
  // 简单 slug 校验，与后端 tagSlugPattern 保持一致。
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(raw)) {
    fieldErrors.value.tagSlugs = [t('composer.tagInvalid')]
    return
  }
  if (tagDraft.value.includes(raw)) {
    tagInput.value = ''
    return
  }
  if (tagDraft.value.length >= 5) {
    fieldErrors.value.tagSlugs = [t('composer.tagLimit')]
    return
  }
  delete fieldErrors.value.tagSlugs
  tagDraft.value = [...tagDraft.value, raw]
  tagInput.value = ''
}

function removeTag(slug: string) {
  tagDraft.value = tagDraft.value.filter((item) => item !== slug)
}

function onTagEnter(event: KeyboardEvent) {
  event.preventDefault()
  addTag()
}

async function submit(payload?: { markdown?: string }) {
  if (!canSubmit.value) {
    return
  }
  submitState.value = 'submitting'
  errorMessage.value = ''
  fieldErrors.value = {}

  const markdown = payload?.markdown ?? bodyMarkdown.value
  try {
    const created = await forumApi.createTopic({
      title: title.value.trim(),
      categorySlug: selectedCategorySlug.value || undefined,
      tagSlugs: tagDraft.value.length ? tagDraft.value : undefined,
      rawContent: markdown,
      sourceFormat: 'markdown',
      editorType: 'tiptap',
      editorVersion: 'sf-editor-v1'
    })
    submitState.value = 'success'
    await navigateTo(localePath(forumTopicPath(created)))
  } catch (error) {
    submitState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: { markdown: string }) {
  submit({ markdown: payload.markdown })
}
</script>

<template>
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-3xl mx-auto px-4 sm:px-6">
      <h1 class="text-2xl font-bold text-slate-900 mb-6 dark:text-zinc-50">
        {{ t('composer.title') }}
      </h1>

      <!-- 无权限提示 -->
      <SFCard v-if="!canCreate" class="p-8">
        <SFEmptyState
          icon-label="LOCK"
          :title="t('composer.permissionDenied.title')"
          :description="t('composer.permissionDenied.description')"
        />
      </SFCard>

      <template v-else>
        <!-- 全局错误（不自动消失） -->
        <SFAlert
          v-if="errorMessage"
          variant="danger"
          :title="errorMessage"
          closable
          class="mb-4"
          @close="errorMessage = ''"
        />

        <SFCard class="p-6 space-y-5">
          <!-- 分类选择 -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
              {{ t('composer.categoryLabel') }}
            </label>
            <select
              v-model="selectedCategorySlug"
              class="sf-input w-full"
            >
              <option value="">{{ t('composer.categoryDefault') }}</option>
              <option v-for="cat in categories" :key="cat.id" :value="cat.slug">
                {{ cat.name }}
              </option>
            </select>
          </div>

          <!-- 标题 -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
              {{ t('composer.titleLabel') }}
            </label>
            <input
              v-model="title"
              type="text"
              class="sf-input w-full"
              :placeholder="t('composer.titlePlaceholder')"
            >
            <p v-if="fieldErrors.title" class="text-sm text-red-600 mt-1 dark:text-red-400">
              {{ fieldErrors.title.join(', ') }}
            </p>
          </div>

          <!-- 标签 -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
              {{ t('composer.tagsLabel') }}
            </label>
            <div v-if="tagDraft.length" class="flex flex-wrap gap-2 mb-2">
              <SFBadge v-for="slug in tagDraft" :key="slug" variant="neutral">
                #{{ slug }}
                <button type="button" class="ml-1" :aria-label="t('composer.removeTag')" @click="removeTag(slug)">
                  <UIcon name="i-lucide-x" class="size-3" />
                </button>
              </SFBadge>
            </div>
            <input
              v-model="tagInput"
              type="text"
              class="sf-input w-full"
              :placeholder="t('composer.tagsPlaceholder')"
              @keydown.enter="onTagEnter"
            >
            <p v-if="fieldErrors.tagSlugs" class="text-sm text-red-600 mt-1 dark:text-red-400">
              {{ fieldErrors.tagSlugs.join(', ') }}
            </p>
          </div>

          <!-- 正文编辑器 -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
              {{ t('composer.bodyLabel') }}
            </label>
            <SFEditor
              v-model="bodyMarkdown"
              :placeholder="t('composer.bodyPlaceholder')"
              :submit-label="submitLabel"
              :disabled="submitState === 'submitting'"
              @submit="onEditorSubmit"
            />
          </div>
        </SFCard>
      </template>
    </div>
  </main>
</template>

<style scoped>
.sf-input {
  border: 1px solid #d1d5db;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.95rem;
  background: #ffffff;
  color: #111827;
  outline: none;
  transition: border-color 0.15s;
}
.sf-input:focus {
  border-color: #0f766e;
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.12);
}
:global(.dark) .sf-input {
  background: #18181b;
  border-color: #3f3f46;
  color: #f4f4f5;
}
</style>
