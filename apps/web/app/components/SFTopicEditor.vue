<script setup lang="ts">
import {
  isForumTagSlug,
  normalizeForumTagSlugInput,
  type ForumCategoryGroup,
  type ForumTopicDetail,
  type ForumTopicTagSummary
} from '~/utils/forumTaxonomy'

// 主题编辑器组件：从详情页 ?edit=1 切入，复用发帖编辑器与字段校验。
// 保存成功后 emit saved（含更新后的 topic），由详情页负责跳转/规范化；
// 取消则 emit cancel。权限/加载由详情页保证，这里只负责编辑表单。
const props = defineProps<{ topic: ForumTopicDetail }>()
const emit = defineEmits<{
  saved: [topic: ForumTopicDetail]
  cancel: []
}>()

const { t } = useI18n()
const forumApi = useForumApi()
const { canEditTopic } = usePermissions()

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
const {
  limits,
  validateTopicTitle,
  validateTopicBody,
  validateTagCount
} = useForumContentLimits()

// 主题加载完成后初始化表单字段（仅在首次为空时填充，避免覆盖用户输入）。
watchEffect(() => {
  if (!props.topic) {
    return
  }
  if (title.value === '') {
    title.value = props.topic.title
  }
  if (selectedCategorySlug.value === '') {
    selectedCategorySlug.value = props.topic.categorySlug
  }
  if (tagDraft.value.length === 0 && props.topic.tags?.length) {
    tagDraft.value = props.topic.tags.map((tag: ForumTopicTagSummary) => tag.slug)
  }
  if (bodyMarkdown.value === '') {
    bodyMarkdown.value = props.topic.content.rawContent
  }
})

const canEdit = computed(() => props.topic ? canEditTopic(props.topic) : false)

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
  if (!canEdit.value || submitState.value === 'submitting' || title.value.trim() === '') {
    return false
  }
  if (limits.value.topicContentMinRunes > 0 && bodyMarkdown.value.trim() === '') {
    return false
  }
  return !validateTopicTitle(title.value) && !validateTopicBody(bodyMarkdown.value) && !validateTagCount(tagDraft.value.length)
})

function addTag() {
  const raw = normalizeForumTagSlugInput(tagInput.value)
  if (!raw) {
    return
  }
  // 标签 slug 支持 Unicode 字母/数字与连字符，便于中文社区直接使用中文标签。
  if (!isForumTagSlug(raw)) {
    fieldErrors.value.tagSlugs = [t('composer.tagInvalid')]
    return
  }
  if (tagDraft.value.includes(raw)) {
    tagInput.value = ''
    return
  }
  if (tagDraft.value.length >= limits.value.tagMaxPerTopic) {
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
  if (!canEdit.value || !props.topic || submitState.value === 'submitting') {
    return
  }
  const markdown = payload?.markdown ?? bodyMarkdown.value
  const nextErrors: Record<string, string[]> = {}
  const titleError = validateTopicTitle(title.value)
  if (titleError === 'titleTooShort') {
    nextErrors.title = [t('composer.titleTooShort', { min: limits.value.topicTitleMinRunes })]
  } else if (titleError === 'titleTooLong') {
    nextErrors.title = [t('composer.titleTooLong', { max: limits.value.topicTitleMaxRunes })]
  }
  const bodyError = validateTopicBody(markdown)
  if (bodyError === 'contentTooShort') {
    nextErrors.content = [t('composer.contentTooShort', { min: limits.value.topicContentMinRunes })]
  } else if (bodyError === 'contentTooLong') {
    nextErrors.content = [t('composer.contentTooLong', { max: limits.value.topicContentMaxRunes })]
  }
  const tagError = validateTagCount(tagDraft.value.length)
  if (tagError === 'tagMin') {
    nextErrors.tagSlugs = [t('composer.tagMinRequired', { min: limits.value.tagMinPerTopic })]
  } else if (tagError === 'tagMax') {
    nextErrors.tagSlugs = [t('composer.tagLimit')]
  }
  if (Object.keys(nextErrors).length) {
    fieldErrors.value = nextErrors
    submitState.value = 'error'
    return
  }

  submitState.value = 'submitting'
  errorMessage.value = ''
  fieldErrors.value = {}

  try {
    const updated = await forumApi.updateTopic(props.topic.id, {
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
    submitState.value = 'idle'
    emit('saved', updated)
  } catch (error) {
    submitState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: { markdown: string }) {
  save({ markdown: payload.markdown })
}

defineExpose({ save })
</script>

<template>
  <div>
    <SFCard v-if="!canEdit" class="p-8">
      <SFEmptyState
        icon-label="LOCK"
        :title="t('composer.permissionDenied.title')"
        :description="t('composer.permissionDenied.description')"
      />
    </SFCard>

    <template v-else>
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
          <LazySFEditor
            v-model="bodyMarkdown"
            :placeholder="t('composer.bodyPlaceholder')"
            :submit-label="submitLabel"
            :disabled="submitState === 'submitting'"
            @submit="onEditorSubmit"
          />
        </div>

        <div class="flex justify-end gap-2">
          <SFButton variant="ghost" @click="emit('cancel')">
            {{ t('topicDetail.cancel') }}
          </SFButton>
        </div>
      </SFCard>
    </template>
  </div>
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
