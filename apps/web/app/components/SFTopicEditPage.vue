<script setup lang="ts">
/**
 * 宿主 body 岛：forum.topic.edit。
 * 主题编辑独立页：复用发帖页三栏壳（sforum-home__* + SFTopicComposerPage.css），
 * 按路由 topicId 加载主题，保存后跳回详情页（slug 变了由详情页 canonical 兜底 301）。
 * 跨作者编辑（版主/管理员）按 API 契约必须填写编辑原因。
 */

import {
  forumContentFromEditorPayload,
  forumEditorInitialContent,
  forumTopicPath,
  type ForumCategoryGroup,
  type ForumTag,
  type ForumTopicDetail
} from '~/utils/forumTaxonomy'
import type { SFEditorContentPayload } from '~/utils/sfEditor'
import type { ComposerFocusField } from '~/components/SFTopicComposerLeftRail.vue'
import { onBeforeRouteLeave } from 'vue-router'

const route = useRoute()
const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()
const { canEditTopic } = usePermissions()
const { user } = useAuthSession()
const toast = useToast()
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)

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

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'topic-edit-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)
const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const totalTopics = computed(() => categories.value.reduce((sum, category) => sum + category.topicCount, 0))

const { data: tagOptions, pending: tagsPending } = await useAsyncData(
  'topic-edit-active-tags',
  async () => (await forumApi.listTags()).filter((tag) => tag.status === 'active'),
  { default: () => [] as ForumTag[] }
)

const missingTopic = computed(() => !topicId.value || (!topic.value && topicStatus.value === 'success') || Boolean(topicError.value))
const canEdit = computed(() => Boolean(topic.value && canEditTopic(topic.value)))
// 跨作者编辑：API 强制要求审计原因（validateEditReason）。
const editingAnotherAuthor = computed(() => Boolean(
  topic.value && user.value && topic.value.authorUserId !== user.value.id
))

const topicReturnPath = computed(() => {
  if (!topic.value) {
    return localePath('/')
  }
  return localePath(forumTopicPath(topic.value, topicUrlMode.value))
})

// —— 表单状态（结构与发帖页一致；初值来自已加载主题）——
const title = ref('')
const selectedCategorySlug = ref('')
const tagDraft = ref<string[]>([])
const bodyMarkdown = ref('')
const staffReason = ref('')
const editorPayload = ref<SFEditorContentPayload | null>(null)
const awaitingEditorBaseline = ref(false)
const {
  limits,
  runeLength,
  validateTopicTitle,
  validateTopicBody,
  validateTagCount
} = useForumContentLimits()

type SubmitState = 'idle' | 'submitting' | 'error' | 'success'
const submitState = ref<SubmitState>('idle')
const errorMessage = ref('')
const conflictMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
const editReasonMaxRunes = 500

// 主题加载后填充一次；watch 覆盖路由复用（topicId 变化）场景。
const baselineSignature = ref('')
function applyTopic(next: ForumTopicDetail | null) {
  if (!next) {
    return
  }
  title.value = next.title
  selectedCategorySlug.value = next.categorySlug
  tagDraft.value = (next.tags || []).map(tag => tag.slug)
  // 正文不直接塞 rawContent：editor-document 的 raw 是 Tiptap JSON，须经 initialContent 加载。
  bodyMarkdown.value = ''
  staffReason.value = ''
  // 动态路由复用时不能沿用上一主题的富文本 payload 或提交反馈。
  editorPayload.value = null
  submitState.value = 'idle'
  errorMessage.value = ''
  conflictMessage.value = ''
  fieldErrors.value = {}
  awaitingEditorBaseline.value = true
  closeMobileDrawers()
  baselineSignature.value = currentSignature.value
}
const currentSignature = computed(() => JSON.stringify({
  title: title.value,
  selectedCategorySlug: selectedCategorySlug.value,
  tagDraft: tagDraft.value,
  bodyMarkdown: bodyMarkdown.value,
  staffReason: staffReason.value
}))
applyTopic(topic.value)
watch(topic, (next, prev) => {
  if (next && (next.id !== prev?.id || next.currentRevision !== prev?.currentRevision)) {
    applyTopic(next)
  }
})

const hasUnsavedChanges = computed(() => (
  submitState.value !== 'success' && currentSignature.value !== baselineSignature.value
))
const editorInitialContent = computed(() => (
  topic.value ? forumEditorInitialContent(topic.value.content) : ''
))

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

// —— 实时校验与检查清单（与发帖页同源）——
const titleCount = computed(() => runeLength(title.value))
const bodyCount = computed(() => runeLength(bodyMarkdown.value))
const titleHint = computed(() => t('composer.titleHintWithLimit', {
  min: limits.value.topicTitleMinRunes,
  max: limits.value.topicTitleMaxRunes
}))
const bodyHint = computed(() => t('composer.bodyHintWithLimit', {
  max: limits.value.topicContentMaxRunes
}))
const effectiveText = computed(() => editorPayload.value?.text ?? bodyMarkdown.value)

const validationState = computed(() => {
  const titleError = validateTopicTitle(title.value)
  const bodyError = validateTopicBody(effectiveText.value)
  const tagError = validateTagCount(tagDraft.value.length)
  return { titleError, bodyError, tagError }
})

const titleCheckText = computed(() => {
  if (!title.value.trim()) return t('composer.checks.title.empty')
  if (validationState.value.titleError === 'titleTooShort') {
    return t('composer.titleTooShort', { min: limits.value.topicTitleMinRunes })
  }
  if (validationState.value.titleError === 'titleTooLong') {
    return t('composer.titleTooLong', { max: limits.value.topicTitleMaxRunes })
  }
  return t('composer.checks.title.ready')
})

const bodyCheckText = computed(() => {
  if (!effectiveText.value.trim()) return t('composer.checks.body.empty')
  if (validationState.value.bodyError === 'contentTooShort') {
    return t('composer.contentTooShort', { min: limits.value.topicContentMinRunes })
  }
  if (validationState.value.bodyError === 'contentTooLong') {
    return t('composer.contentTooLong', { max: limits.value.topicContentMaxRunes })
  }
  return t('composer.checks.body.ready')
})

const selectedCategory = computed(() => categories.value.find(category => category.slug === selectedCategorySlug.value))
// 非公开分类（后台可见）不在公开列表里：保留原 slug 展示，且未改动时不提交该字段。
const summaryCategoryName = computed(() => selectedCategory.value?.name || selectedCategorySlug.value || null)

const categoryCheckText = computed(() => (
  summaryCategoryName.value
    ? t('composer.checks.category.selected', { category: summaryCategoryName.value })
    : t('composer.checks.category.serverDefault')
))

const tagCheckText = computed(() => {
  if (validationState.value.tagError === 'tagMin') {
    return t('composer.tagMinRequired', { min: limits.value.tagMinPerTopic })
  }
  if (validationState.value.tagError === 'tagMax') {
    return t('composer.tagLimit')
  }
  if (tagDraft.value.length) {
    return t('composer.checks.tags.selected', { count: tagDraft.value.length, max: limits.value.tagMaxPerTopic })
  }
  return limits.value.tagMinPerTopic > 0
    ? t('composer.checks.tags.emptyRequired', { min: limits.value.tagMinPerTopic })
    : t('composer.checks.tags.optional', { max: limits.value.tagMaxPerTopic })
})

const prePublishChecks = computed(() => {
  const checks = [
    {
      key: 'title',
      ok: Boolean(title.value.trim()) && !validationState.value.titleError && !fieldErrors.value.title,
      icon: 'i-lucide-heading-1',
      label: t('composer.checks.title.label'),
      text: titleCheckText.value
    },
    {
      key: 'body',
      ok: Boolean(effectiveText.value.trim()) && !validationState.value.bodyError && !fieldErrors.value.content,
      icon: 'i-lucide-file-text',
      label: t('composer.checks.body.label'),
      text: bodyCheckText.value
    },
    {
      key: 'category',
      ok: Boolean(summaryCategoryName.value || limits.value.defaultCategorySlug),
      icon: 'i-lucide-folder',
      label: t('composer.checks.category.label'),
      text: categoryCheckText.value
    },
    {
      key: 'tags',
      ok: !validationState.value.tagError && !fieldErrors.value.tagSlugs,
      icon: 'i-lucide-tags',
      label: t('composer.checks.tags.label'),
      text: tagCheckText.value
    }
  ]
  if (editingAnotherAuthor.value) {
    checks.push({
      key: 'reason',
      ok: Boolean(staffReason.value.trim()) && !fieldErrors.value.reason,
      icon: 'i-lucide-user-round-pen',
      label: t('composer.checks.reason.label'),
      text: staffReason.value.trim() ? t('composer.checks.reason.ready') : t('composer.checks.reason.empty')
    })
  }
  return checks
})

const editStateLabel = computed(() => {
  if (submitState.value === 'submitting') return t('composer.editState.saving')
  if (submitState.value === 'success') return t('composer.editState.success')
  if (submitState.value === 'error') return t('composer.publishState.error')
  return hasUnsavedChanges.value ? t('composer.editState.unsaved') : t('composer.editState.clean')
})

const tagPolicyLabel = computed(() => t(`composer.tagPolicy.${limits.value.tagCreationMode}`))
const actorName = computed(() => user.value?.displayName || user.value?.username || t('composer.currentAccount'))
const staffReasonTooLong = computed(() => (
  runeLength(staffReason.value.trim()) > editReasonMaxRunes
))
const staffReasonError = computed(() => (
  staffReasonTooLong.value
    ? t('composer.editReasonTooLong', { max: editReasonMaxRunes })
    : fieldErrors.value.reason?.join(', ')
))

const submitLabel = computed(() => (
  submitState.value === 'submitting' ? t('composer.editState.saving') : t('composer.save')
))

const canSubmit = computed(() => {
  if (
    !canEdit.value
    || !hasUnsavedChanges.value
    || submitState.value === 'submitting'
    || title.value.trim() === ''
  ) {
    return false
  }
  if (limits.value.topicContentMinRunes > 0 && bodyMarkdown.value.trim() === '') {
    return false
  }
  if (
    editingAnotherAuthor.value
    && (
      !staffReason.value.trim()
      || staffReasonTooLong.value
    )
  ) {
    return false
  }
  return !validateTopicTitle(title.value) && !validateTopicBody(effectiveText.value) && !validateTagCount(tagDraft.value.length)
})

watch(title, () => {
  if (!validateTopicTitle(title.value)) {
    delete fieldErrors.value.title
  }
})
watch(bodyMarkdown, () => {
  if (!validateTopicBody(effectiveText.value)) {
    delete fieldErrors.value.content
  }
})
watch(selectedCategorySlug, () => {
  delete fieldErrors.value.categorySlug
})
watch(staffReason, () => {
  if (
    staffReason.value.trim()
    && runeLength(staffReason.value.trim()) <= editReasonMaxRunes
  ) {
    delete fieldErrors.value.reason
  }
})

function onTagDraftUpdate(next: string[]) {
  tagDraft.value = next
  delete fieldErrors.value.tagSlugs
}

function onTagInputInvalid(key: 'tagInvalid' | 'tagUnknownControlled' | 'tagLimit') {
  fieldErrors.value.tagSlugs = [t(`composer.${key}`)]
}

function onEditorContentChange(payload: SFEditorContentPayload) {
  editorPayload.value = payload
  // SFEditor 会把原生 Tiptap JSON 规范化为 Markdown v-model；首次同步属于加载，不是用户修改。
  if (awaitingEditorBaseline.value) {
    baselineSignature.value = currentSignature.value
    awaitingEditorBaseline.value = false
  }
}

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

/** 左栏进度项：滚到主列对应字段并尝试 focus。 */
function focusComposerField(field: ComposerFocusField) {
  if (!import.meta.client) {
    return
  }
  const idByField: Record<typeof field, string> = {
    title: 'topic-edit-title-input',
    body: 'topic-edit-body',
    category: 'topic-edit-category',
    tags: 'topic-edit-tags'
  }
  const target = document.getElementById(idByField[field])
  if (!target) {
    return
  }
  target.scrollIntoView({ behavior: 'smooth', block: 'center' })
  const focusable = target.matches('input, select, textarea, button, [contenteditable="true"]')
    ? target
    : target.querySelector<HTMLElement>('input, select, textarea, button, [contenteditable="true"], .ProseMirror')
  if (focusable && typeof focusable.focus === 'function') {
    window.setTimeout(() => focusable.focus(), 180)
  }
  closeMobileDrawers()
}

async function submit(payload?: { markdown?: string; native?: unknown; text?: string }) {
  if (
    !topic.value
    || !canEdit.value
    || !hasUnsavedChanges.value
    || submitState.value === 'submitting'
  ) {
    return
  }
  const markdown = payload?.markdown ?? bodyMarkdown.value
  const content = forumContentFromEditorPayload({
    markdown,
    native: payload?.native,
    text: payload?.text
  })
  const nextErrors: Record<string, string[]> = {}
  const titleError = validateTopicTitle(title.value)
  if (titleError === 'titleTooShort') {
    nextErrors.title = [t('composer.titleTooShort', { min: limits.value.topicTitleMinRunes })]
  } else if (titleError === 'titleTooLong') {
    nextErrors.title = [t('composer.titleTooLong', { max: limits.value.topicTitleMaxRunes })]
  }
  const bodyError = validateTopicBody(payload?.text || markdown)
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
  const reason = staffReason.value.trim()
  if (editingAnotherAuthor.value && !reason) {
    nextErrors.reason = [t('admin.forum.content.reasonRequired')]
  } else if (runeLength(reason) > editReasonMaxRunes) {
    nextErrors.reason = [t('composer.editReasonTooLong', { max: editReasonMaxRunes })]
  }
  if (Object.keys(nextErrors).length) {
    fieldErrors.value = nextErrors
    submitState.value = 'error'
    return
  }

  submitState.value = 'submitting'
  errorMessage.value = ''
  conflictMessage.value = ''
  fieldErrors.value = {}

  try {
    const updated = await forumApi.updateTopic(topic.value.id, {
      expectedRevision: topic.value.currentRevision,
      reason: reason || undefined,
      title: title.value.trim(),
      // 分类未改动时不提交，避免把非公开分类误改为默认分类。
      categorySlug: selectedCategorySlug.value !== topic.value.categorySlug
        ? (selectedCategorySlug.value || undefined)
        : undefined,
      tagSlugs: tagDraft.value,
      content
    })
    submitState.value = 'success'
    baselineSignature.value = currentSignature.value
    toast.add({ color: 'success', icon: 'i-lucide-check', title: t('topicDetail.topicUpdated'), duration: 10000 })
    await navigateTo(localePath(forumTopicPath(updated, topicUrlMode.value)))
  } catch (error) {
    submitState.value = 'error'
    if (apiErrorReason(error) === 'forum.revision_conflict') {
      conflictMessage.value = t('composer.editConflict')
      return
    }
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: { markdown: string; native?: unknown; text?: string }) {
  void submit(payload)
}

function submitCurrent() {
  void submit(editorPayload.value || undefined)
}

function onCancel() {
  void navigateTo(topicReturnPath.value)
}

// —— 未保存修改守卫（离开页面 / 关闭标签页）——
let beforeUnloadHandler: ((event: BeforeUnloadEvent) => void) | null = null

onMounted(() => {
  beforeUnloadHandler = (event: BeforeUnloadEvent) => {
    if (!hasUnsavedChanges.value) {
      return
    }
    event.preventDefault()
    event.returnValue = ''
  }
  window.addEventListener('beforeunload', beforeUnloadHandler)
})

onBeforeUnmount(() => {
  if (beforeUnloadHandler) {
    window.removeEventListener('beforeunload', beforeUnloadHandler)
  }
})

onBeforeRouteLeave(() => {
  if (!import.meta.client || !hasUnsavedChanges.value) {
    return true
  }
  return window.confirm(t('composer.editLeaveConfirm'))
})
</script>

<template>
  <main
    class="sforum-home sforum-topic-composer sforum-topic-edit"
    data-sforum-island-body="forum.component.topic_editor"
    data-layout="fullwidth-3col"
  >
    <div
      class="sforum-home__layout"
      :class="{ 'sforum-home__layout--with-right': canEdit }"
    >
      <!-- 左栏：与发帖页一致 — 无类别列表，挂编辑进度与写作要点 -->
      <div class="sforum-home__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          selected-category-slug=""
          :total-topics="totalTopics"
          :pending="categoriesPending"
          :can-create-topic="canEdit"
          :show-categories="false"
        >
          <template #after-navigation>
            <SFTopicComposerLeftRail
              :checks="prePublishChecks"
              :draft-status-label="editStateLabel"
              :can-create="canEdit"
              :show-draft-action="false"
              @focus-field="focusComposerField"
            />
          </template>
        </SFHomeNavigation>
      </div>

      <section
        class="sforum-home__main sforum-topic-composer__main"
        :aria-labelledby="canEdit ? 'topic-edit-title' : undefined"
      >
        <!-- 未找到 / 无权限 -->
        <SFCard v-if="missingTopic" class="p-8">
          <SFEmptyState
            icon-label="?"
            :title="t('topicDetail.notFound.title')"
            :description="t('topicDetail.notFound.description')"
          />
        </SFCard>

        <SFCard v-else-if="!canEdit" class="p-8">
          <SFEmptyState
            icon-label="LOCK"
            :title="t('composer.permissionDenied.title')"
            :description="t('composer.permissionDenied.description')"
          />
        </SFCard>

        <template v-else>
          <div class="sforum-topic-composer__inner">
            <nav class="sforum-topic-composer__breadcrumbs" :aria-label="t('composer.breadcrumbLabel')">
              <NuxtLink :to="localePath('/')" class="hover:text-[color:var(--sf-accent)]">
                {{ t('composer.breadcrumbHome') }}
              </NuxtLink>
              <UIcon name="i-lucide-chevron-right" class="size-3" aria-hidden="true" />
              <NuxtLink
                v-if="topic"
                :to="topicReturnPath"
                class="min-w-0 truncate hover:text-[color:var(--sf-accent)]"
              >
                {{ topic.title }}
              </NuxtLink>
              <UIcon v-if="topic" name="i-lucide-chevron-right" class="size-3" aria-hidden="true" />
              <span>{{ t('composer.editTitle') }}</span>
            </nav>

            <header class="sforum-topic-composer__head">
              <div class="sforum-topic-composer__head-copy">
                <h1 id="topic-edit-title">{{ t('composer.editTitle') }}</h1>
                <p>{{ t('composer.editSubtitle') }}</p>
              </div>
              <div class="sforum-topic-composer__head-actions">
                <span class="sforum-topic-composer__draft-state" :class="{ 'is-error': submitState === 'error' }">
                  <UIcon
                    :name="hasUnsavedChanges ? 'i-lucide-file-pen-line' : 'i-lucide-file-check-2'"
                    class="size-4"
                    aria-hidden="true"
                  />
                  {{ editStateLabel }}
                </span>
                <!-- 中窄屏：打开完整右栏（与发帖页 panel-right 一致） -->
                <button
                  type="button"
                  class="sforum-topic-composer__icon-button sforum-topic-composer__desktop-hidden"
                  :aria-label="t('composer.rightRail.open')"
                  @click="mobileInfoOpen = true"
                >
                  <UIcon name="i-lucide-panel-right" class="size-[18px]" aria-hidden="true" />
                </button>
              </div>
            </header>

            <!-- 版本冲突 / 全局错误（不自动消失） -->
            <SFAlert
              v-if="conflictMessage"
              variant="danger"
              :title="conflictMessage"
              closable
              class="sforum-topic-composer__alert"
              @close="conflictMessage = ''"
            />
            <SFAlert
              v-if="errorMessage"
              variant="danger"
              :title="errorMessage"
              closable
              class="sforum-topic-composer__alert"
              @close="errorMessage = ''"
            />

            <SFRegionOutlet page="forum.topic.edit" region="content_before" />

            <form class="sforum-topic-composer__form" @submit.prevent="submitCurrent">
              <!-- 跨作者编辑：警示 + 必填原因 -->
              <template v-if="editingAnotherAuthor">
                <UAlert
                  color="warning"
                  variant="soft"
                  icon="i-lucide-user-round-pen"
                  :title="t('admin.forum.content.editingAnotherAuthor')"
                />
                <div id="topic-edit-reason" class="sforum-topic-composer__field">
                  <div class="sforum-topic-composer__field-head">
                    <span class="sforum-topic-composer__field-label">{{ t('admin.forum.content.reason') }}</span>
                  </div>
                  <UTextarea
                    v-model="staffReason"
                    :placeholder="t('admin.forum.content.reasonPlaceholder')"
                    :disabled="submitState === 'submitting'"
                    :rows="3"
                    class="w-full"
                  />
                  <p v-if="staffReasonError" class="sforum-topic-composer__error">
                    {{ staffReasonError }}
                  </p>
                </div>
              </template>

              <!-- 分类 / 标签：与发帖页同款控件 -->
              <div class="sforum-topic-composer__taxonomy">
                <div class="sforum-topic-composer__field">
                  <div class="sforum-topic-composer__field-head">
                    <span class="sforum-topic-composer__field-label">{{ t('composer.categoryLabel') }}</span>
                  </div>
                  <SFCategorySelect
                    id="topic-edit-category"
                    v-model="selectedCategorySlug"
                    :categories="categories"
                    :empty-label="summaryCategoryName || t('composer.categoryDefault')"
                    :hint="t('composer.categoryHint')"
                    :error="fieldErrors.categorySlug?.join(', ')"
                    :disabled="submitState === 'submitting'"
                    :pending="categoriesPending"
                  />
                </div>

                <div class="sforum-topic-composer__field">
                  <div class="sforum-topic-composer__field-head">
                    <span class="sforum-topic-composer__field-label">{{ t('composer.tagsLabel') }}</span>
                    <span>{{ t('composer.tagLimitSummary', { min: limits.tagMinPerTopic, max: limits.tagMaxPerTopic }) }}</span>
                  </div>
                  <SFTagInput
                    id="topic-edit-tags"
                    :model-value="tagDraft"
                    :options="tagOptions"
                    :hint="tagsPending ? t('composer.tagsLoading') : t('composer.tagsHint')"
                    :error="fieldErrors.tagSlugs?.join(', ')"
                    :disabled="submitState === 'submitting'"
                    :max="limits.tagMaxPerTopic"
                    :creation-mode="limits.tagCreationMode"
                    @update:model-value="onTagDraftUpdate"
                    @invalid="onTagInputInvalid"
                  />
                </div>
              </div>

              <!-- 标题 -->
              <div id="topic-edit-title-field" class="sforum-topic-composer__field">
                <SFInput
                  id="topic-edit-title-input"
                  v-model="title"
                  :label="t('composer.titleLabel')"
                  :placeholder="t('composer.titlePlaceholder')"
                  :hint="`${titleHint} (${t('composer.charCount', { count: titleCount, max: limits.topicTitleMaxRunes })})`"
                  :error="fieldErrors.title?.join(', ')"
                  :disabled="submitState === 'submitting'"
                  required
                />
              </div>

              <!-- 正文编辑器 -->
              <div id="topic-edit-body" class="sforum-topic-composer__field">
                <div class="sforum-topic-composer__field-head">
                  <label>{{ t('composer.bodyLabel') }}</label>
                  <span>{{ t('composer.charCount', { count: bodyCount, max: limits.topicContentMaxRunes }) }}</span>
                </div>
                <p class="sforum-topic-composer__hint">{{ bodyHint }}</p>
                <LazySFEditor
                  :key="`${topic?.id}-${topic?.currentRevision}`"
                  v-model="bodyMarkdown"
                  :initial-content="editorInitialContent"
                  :placeholder="t('composer.bodyPlaceholder')"
                  :submit-label="submitLabel"
                  :disabled="submitState === 'submitting'"
                  :max-characters="limits.topicContentMaxRunes"
                  :error="fieldErrors.content?.join(', ')"
                  :rows="14"
                  @content-change="onEditorContentChange"
                  @submit="onEditorSubmit"
                />
                <p v-if="fieldErrors.content" class="sforum-topic-composer__error">
                  {{ fieldErrors.content.join(', ') }}
                </p>
              </div>
            </form>

            <SFRegionOutlet page="forum.topic.edit" region="content_after" />
          </div>
        </template>

        <footer v-if="canEdit" class="sforum-topic-composer__dock" aria-live="polite">
          <div class="sforum-topic-composer__dock-status">
            <SFAvatar size="sm" :name="actorName" :avatar="user?.avatar" />
            <span>
              <strong>{{ actorName }}</strong>
              {{ editStateLabel }}
            </span>
          </div>
          <div class="sforum-topic-composer__dock-actions">
            <SFButton
              type="button"
              variant="ghost"
              :disabled="submitState === 'submitting'"
              @click="onCancel"
            >
              <UIcon name="i-lucide-x" class="size-4" aria-hidden="true" />
              {{ t('topicDetail.cancel') }}
            </SFButton>
            <SFButton
              type="button"
              :disabled="!canSubmit"
              @click="submitCurrent"
            >
              <UIcon name="i-lucide-check" class="size-4" aria-hidden="true" />
              {{ submitLabel }}
            </SFButton>
          </div>
        </footer>
      </section>

      <aside
        v-if="canEdit"
        class="sforum-home__right"
        :aria-label="t('composer.rightRail.label')"
      >
        <SFTopicComposerRightRail
          :category-name="summaryCategoryName"
          :title-max="limits.topicTitleMaxRunes"
          :tag-policy-label="tagPolicyLabel"
          :actor-name="actorName"
          :publish-visibility-label="editStateLabel"
          :checks="prePublishChecks"
          :body-max="limits.topicContentMaxRunes"
          :permission-value-label="t('composer.settings.editPermissionValue')"
        />
      </aside>
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('common.close')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="categories"
        selected-category-slug=""
        :total-topics="totalTopics"
        :pending="categoriesPending"
        :can-create-topic="canEdit"
        :show-categories="false"
      >
        <template #after-navigation>
          <SFTopicComposerLeftRail
            :checks="prePublishChecks"
            :draft-status-label="editStateLabel"
            :can-create="canEdit"
            :show-draft-action="false"
            @focus-field="focusComposerField"
          />
        </template>
      </SFHomeNavigation>
    </aside>

    <aside v-if="mobileInfoOpen && canEdit" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('composer.rightRail.label') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <!-- 抽屉内复用完整右栏（与发帖页一致） -->
      <aside class="sforum-home__right" :aria-label="t('composer.rightRail.label')">
        <SFTopicComposerRightRail
          :category-name="summaryCategoryName"
          :title-max="limits.topicTitleMaxRunes"
          :tag-policy-label="tagPolicyLabel"
          :actor-name="actorName"
          :publish-visibility-label="editStateLabel"
          :checks="prePublishChecks"
          :body-max="limits.topicContentMaxRunes"
          :permission-value-label="t('composer.settings.editPermissionValue')"
        />
      </aside>
    </aside>
  </main>
</template>

<style scoped src="./SFTopicComposerPage.css"></style>
