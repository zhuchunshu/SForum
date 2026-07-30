<script setup lang="ts">
import { usePermissions } from '~/composables/identity/usePermissions'
import { useForumContentLimits } from '~/composables/forum/useForumContentLimits'
import { useForumApi } from '~/composables/forum/useForumApi'
import {
  forumContentFromEditorPayload,
  forumEditorInitialContent,
  isForumTagSlug,
  normalizeForumTagSlugInput,
  type ForumCategoryGroup,
  type ForumRenderedContent,
  type ForumTopicDetail,
  type ForumTopicTagSummary
} from '~/utils/forum/forumTaxonomy'
import type { SFEditorContentPayload } from '~/utils/sfEditor'

// 主题编辑器组件：由编辑独立页（/topics/:id/edit）与后台内容管理复用，含字段校验。
// 保存成功后 emit saved（含更新后的 topic），由宿主页面负责跳转/规范化；
// 取消则 emit cancel。权限/加载由宿主页面与 API 保证，这里只负责编辑表单。
type EditableTopic = {
  id: number
  authorUserId: number
  title: string
  categorySlug: string
  tags?: ForumTopicTagSummary[]
  content: ForumRenderedContent
  currentRevision: number
}

const props = withDefaults(defineProps<{
  topic: EditableTopic
  staffReason?: string
  requireStaffReason?: boolean
  editingAnotherAuthor?: boolean
}>(), {
  staffReason: '',
  requireStaffReason: false,
  editingAnotherAuthor: false
})
const emit = defineEmits<{
  saved: [topic: ForumTopicDetail]
  cancel: []
  'update:staffReason': [value: string]
  conflict: []
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
// v-model 仅同步 Markdown；editor-document 经 initialContent 加载，禁止 rawContent 直灌。
const bodyMarkdown = ref('')
const editorPayload = shallowRef<SFEditorContentPayload | null>(null)
const tagInput = ref('')
const editorInitialContent = computed(() => forumEditorInitialContent(props.topic.content))
const editorKey = computed(() => `${props.topic.id}-${props.topic.currentRevision}`)
const {
  limits,
  validateTopicTitle,
  validateTopicBody,
  validateTagCount
} = useForumContentLimits()

// 主题切换时重置表单；正文留给 SFEditor initialContent + 首次 Markdown 回写。
watch(
  () => [props.topic.id, props.topic.currentRevision] as const,
  () => {
    title.value = props.topic.title
    selectedCategorySlug.value = props.topic.categorySlug
    tagDraft.value = (props.topic.tags || []).map((tag: ForumTopicTagSummary) => tag.slug)
    bodyMarkdown.value = ''
  },
  { immediate: true }
)

const canEdit = computed(() => props.topic ? canEditTopic(props.topic) : false)
// 后台可加载非公开主题；保留不在公开分类列表中的当前 slug，避免一次正文编辑意外清空分类。
const selectedCategoryMissing = computed(() => Boolean(
  selectedCategorySlug.value && !categories.value.some(category => category.slug === selectedCategorySlug.value)
))

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
  if (
    !canEdit.value
    || submitState.value === 'submitting'
    || title.value.trim() === ''
    || (editorPayload.value?.pendingUploadCount || 0) > 0
  ) {
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

async function save(payload?: Pick<SFEditorContentPayload, 'markdown' | 'native' | 'text' | 'attachmentIds' | 'pendingUploadCount'>) {
  const currentPayload = payload || editorPayload.value || undefined
  if (
    !canEdit.value
    || !props.topic
    || submitState.value === 'submitting'
    || (currentPayload?.pendingUploadCount || 0) > 0
  ) {
    return
  }
  const markdown = currentPayload?.markdown ?? bodyMarkdown.value
  const text = currentPayload?.text ?? markdown
  const content = forumContentFromEditorPayload({
    markdown,
    native: currentPayload?.native,
    text,
    attachmentIds: currentPayload?.attachmentIds
  })
  const nextErrors: Record<string, string[]> = {}
  const titleError = validateTopicTitle(title.value)
  if (titleError === 'titleTooShort') {
    nextErrors.title = [t('composer.titleTooShort', { min: limits.value.topicTitleMinRunes })]
  } else if (titleError === 'titleTooLong') {
    nextErrors.title = [t('composer.titleTooLong', { max: limits.value.topicTitleMaxRunes })]
  }
  const bodyError = validateTopicBody(text)
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
  const reason = props.staffReason.trim()
  if (props.requireStaffReason && !reason) {
    nextErrors.reason = [t('admin.forum.content.reasonRequired')]
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
      expectedRevision: props.topic.currentRevision,
      reason: reason || undefined,
      title: title.value.trim(),
      categorySlug: selectedCategorySlug.value || undefined,
      tagSlugs: tagDraft.value,
      content
    })
    submitState.value = 'idle'
    emit('saved', updated)
  } catch (error) {
    submitState.value = 'error'
    if (apiErrorReason(error) === 'forum.revision_conflict') {
      emit('conflict')
      return
    }
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: SFEditorContentPayload) {
  void save(payload)
}

function onEditorContentChange(payload: SFEditorContentPayload) {
  editorPayload.value = payload
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
        <UAlert
          v-if="editingAnotherAuthor"
          color="warning"
          variant="soft"
          icon="i-lucide-user-round-pen"
          :title="t('admin.forum.content.editingAnotherAuthor')"
          class="mb-1"
        />

        <UFormField v-if="requireStaffReason" :label="t('admin.forum.content.reason')" :error="fieldErrors.reason?.join(', ')">
          <UTextarea
            :model-value="staffReason"
            :placeholder="t('admin.forum.content.reasonPlaceholder')"
            :disabled="submitState === 'submitting'"
            :rows="3"
            class="w-full"
            @update:model-value="emit('update:staffReason', String($event))"
          />
        </UFormField>

        <div>
          <label class="block text-sm font-semibold text-slate-700 mb-2 dark:text-zinc-300">
            {{ t('composer.categoryLabel') }}
          </label>
          <select v-model="selectedCategorySlug" class="sf-input w-full">
            <option v-if="selectedCategoryMissing" :value="selectedCategorySlug">
              {{ selectedCategorySlug }}
            </option>
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
            :key="editorKey"
            v-model="bodyMarkdown"
            :initial-content="editorInitialContent"
            :placeholder="t('composer.bodyPlaceholder')"
            :submit-label="submitLabel"
            :disabled="submitState === 'submitting'"
            :submit-disabled="(editorPayload?.pendingUploadCount || 0) > 0"
            @content-change="onEditorContentChange"
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
