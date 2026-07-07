<script setup lang="ts">
import {
  forumTopicPath,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'

definePageMeta({
  requiresAuth: true
})

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { siteName } = useWebOptions()
const forumApi = useForumApi()
const { canEditTopic } = usePermissions()

const topicID = computed(() => Number(route.params.topicID))
const topicSlug = computed(() => String(route.params.topicSlug ?? ''))

useSForumSeo({
  title: () => `${t('composer.editTitle')} - ${siteName.value}`,
  description: () => t('composer.metaDescription'),
  type: 'website'
})

const { data: topic, error: topicError } = await useAsyncData(
  () => `forum-topic-edit-${topicID.value}`,
  () => forumApi.getTopic(topicID.value),
  { default: () => null }
)

const { data: categoryGroups } = await useAsyncData(
  'composer-edit-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))

const title = ref('')
const selectedCategorySlug = ref('')
const tagDraft = ref<string[]>([])
const bodyMarkdown = ref('')
const tagInput = ref('')

// 主题加载完成后初始化表单字段。
watchEffect(() => {
  if (!topic.value) {
    return
  }
  if (title.value === '') {
    title.value = topic.value.title
  }
  if (selectedCategorySlug.value === '') {
    selectedCategorySlug.value = topic.value.categorySlug
  }
  if (tagDraft.value.length === 0 && topic.value.tags?.length) {
    tagDraft.value = topic.value.tags.map((tag) => tag.slug)
  }
  if (bodyMarkdown.value === '') {
    bodyMarkdown.value = topic.value.content.rawContent
  }
})

const canEdit = computed(() => topic.value ? canEditTopic(topic.value) : false)

type SubmitState = 'idle' | 'submitting' | 'error'
const submitState = ref<SubmitState>('idle')
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})

const submitLabel = computed(() => {
  if (submitState.value === 'submitting') {
    return t('composer.submitting')
  }
  return t('composer.save')
})

const canSubmit = computed(() => {
  return canEdit.value && title.value.trim() !== '' && bodyMarkdown.value.trim() !== '' && submitState.value !== 'submitting'
})

function addTag() {
  const raw = tagInput.value.trim().toLowerCase()
  if (!raw) {
    return
  }
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

async function save(payload?: { markdown?: string }) {
  if (!canSubmit.value || !topic.value) {
    return
  }
  submitState.value = 'submitting'
  errorMessage.value = ''
  fieldErrors.value = {}

  const markdown = payload?.markdown ?? bodyMarkdown.value
  try {
    await forumApi.updateTopic(topic.value.id, {
      title: title.value.trim(),
      categorySlug: selectedCategorySlug.value || undefined,
      tagSlugs: tagDraft.value,
      content: {
        rawContent: markdown,
        sourceFormat: 'markdown',
        editorType: 'tiptap',
        editorVersion: 'sf-editor-v1'
      }
    })
    await navigateTo(localePath(forumTopicPath({ id: topic.value.id, slug: topicSlug.value })))
  } catch (error) {
    submitState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: { markdown: string }) {
  save({ markdown: payload.markdown })
}
</script>

<template>
  <main class="min-h-screen py-8" style="background-color: var(--sf-surface)">
    <div class="max-w-3xl mx-auto px-4 sm:px-6">
      <h1 class="text-2xl font-bold text-slate-900 mb-6 dark:text-zinc-50">
        {{ t('composer.editTitle') }}
      </h1>

      <SFCard v-if="topicError && !topic" class="p-8">
        <SFEmptyState
          :title="t('topicDetail.notFound.title')"
          :description="t('topicDetail.notFound.description')"
        />
      </SFCard>

      <SFCard v-else-if="topic && !canEdit" class="p-8">
        <SFEmptyState
          icon-label="LOCK"
          :title="t('composer.permissionDenied.title')"
          :description="t('composer.permissionDenied.description')"
        />
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

        <SFCard class="p-6 space-y-5">
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
              {{ t('composer.categoryLabel') }}
            </label>
            <select v-model="selectedCategorySlug" class="sf-input w-full">
              <option v-for="cat in categories" :key="cat.id" :value="cat.slug">
                {{ cat.name }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
              {{ t('composer.titleLabel') }}
            </label>
            <input v-model="title" type="text" class="sf-input w-full">
            <p v-if="fieldErrors.title" class="text-sm text-red-600 mt-1 dark:text-red-400">
              {{ fieldErrors.title.join(', ') }}
            </p>
          </div>

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
