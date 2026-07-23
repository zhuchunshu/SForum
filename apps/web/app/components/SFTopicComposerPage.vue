<script setup lang="ts">
/**
 * 宿主 body 岛：forum.topic.create。主题 L1 挂载；路由页仅 outlet + fail-closed 回退。
 */

import {
  forumContentFromEditorPayload,
  forumTopicPath,
  isForumTagSlug,
  normalizeForumTagSlugInput,
  type ForumCategoryGroup,
  type ForumTag
} from '~/utils/forumTaxonomy'
import type { SFEditorContentPayload } from '~/utils/sfEditor'
import { onBeforeRouteLeave } from 'vue-router'


const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()
const { can } = usePermissions()
const toast = useToast()
const { locale } = useI18n()
const { user } = useAuthSession()
const mobileMenuOpen = useState<boolean>('forum-mobile-menu-open', () => false)
const mobileInfoOpen = useState<boolean>('forum-mobile-info-open', () => false)
const topicDraftKey = computed(() => {
  const actorKey = user.value?.id || user.value?.username || 'authenticated'
  return `sforum.topic-composer.draft.v1:${actorKey}`
})

// F4.3：composer 工具栏扩展动作（登录后拉取；失败静默为空）。
const { data: composerToolbarActions } = await useAsyncData(
  'composer-toolbar-actions',
  async () => {
    try {
      return await forumApi.listComposerToolbarActions()
    } catch {
      return []
    }
  },
  { default: () => [] as import('~/utils/forumTaxonomy').ForumComposerToolbarAction[] }
)

function composerToolbarLabel(action: import('~/utils/forumTaxonomy').ForumComposerToolbarAction) {
  const labels = action.label || {}
  return labels[String(locale.value)] || labels['zh-CN'] || labels['en-US'] || Object.values(labels)[0] || action.id
}

async function runComposerToolbarAction(action: import('~/utils/forumTaxonomy').ForumComposerToolbarAction) {
  if (action.confirm && !window.confirm(composerToolbarLabel(action))) {
    return
  }
  try {
    await forumApi.applyComposerToolbarAction(action)
    toast.add({
      color: 'primary',
      icon: action.icon || 'i-lucide-check',
      title: composerToolbarLabel(action),
      duration: 10000
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('composer.submitFailed'),
      duration: 0
    })
  }
}

useSForumSeo({
  title: () => `${t('composer.metaTitle')} - ${siteName.value}`,
  description: () => t('composer.metaDescription'),
  type: 'website'
})

// 没有发帖权限直接给出提示。
const canCreate = computed(() => can(FORUM_PERMISSIONS.topicCreate))

const { data: categoryGroups, pending: categoriesPending } = await useAsyncData(
  'composer-category-groups',
  () => forumApi.listCategoryGroups(),
  { default: () => [] as ForumCategoryGroup[] }
)

const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const totalTopics = computed(() => categories.value.reduce((sum, category) => sum + category.topicCount, 0))

const { data: tagOptions, pending: tagsPending } = await useAsyncData(
  'composer-active-tags',
  async () => (await forumApi.listTags()).filter((tag) => tag.status === 'active'),
  { default: () => [] as ForumTag[] }
)

const title = ref('')
const selectedCategorySlug = ref('')
const tagInput = ref('')
const tagDraft = ref<string[]>([])
const bodyMarkdown = ref('')
const editorPayload = ref<SFEditorContentPayload | null>(null)
const {
  limits,
  runeLength,
  validateTopicTitle,
  validateTopicBody,
  validateTagCount
} = useForumContentLimits()

// 表单状态。
type SubmitState = 'idle' | 'submitting' | 'error' | 'success'
const submitState = ref<SubmitState>('idle')
const errorMessage = ref('')
const fieldErrors = ref<Record<string, string[]>>({})
const draftSavedAt = ref('')
const draftSaving = ref(false)
const draftError = ref('')
const lastPersistedDraftSignature = ref('')

const submitLabel = computed(() => {
  if (submitState.value === 'submitting') {
    return t('composer.submitting')
  }
  return t('composer.submit')
})

const titleCount = computed(() => runeLength(title.value))
const bodyCount = computed(() => runeLength(bodyMarkdown.value))
const titleHint = computed(() => t('composer.titleHintWithLimit', {
  min: limits.value.topicTitleMinRunes,
  max: limits.value.topicTitleMaxRunes
}))
const bodyHint = computed(() => t('composer.bodyHintWithLimit', {
  max: limits.value.topicContentMaxRunes
}))

const canSubmit = computed(() => {
  if (!canCreate.value || submitState.value === 'submitting') {
    return false
  }
  // 允许 contentMin=0 时正文可为空字符串以外的空白由后端 RenderContent 再判。
  if (title.value.trim() === '') {
    return false
  }
  if (limits.value.topicContentMinRunes > 0 && bodyMarkdown.value.trim() === '') {
    return false
  }
  return !validateTopicTitle(title.value) && !validateTopicBody(bodyMarkdown.value) && !validateTagCount(tagDraft.value.length)
})

const actorName = computed(() => user.value?.displayName || user.value?.username || t('composer.currentAccount'))
const selectedCategory = computed(() => categories.value.find(category => category.slug === selectedCategorySlug.value))
const defaultCategory = computed(() => categories.value.find(category => category.slug === limits.value.defaultCategorySlug))
const summaryCategory = computed(() => selectedCategory.value || defaultCategory.value || null)
const activeTagMap = computed(() => new Map(tagOptions.value.map(tag => [tag.slug, tag])))
const effectiveText = computed(() => editorPayload.value?.text ?? bodyMarkdown.value)
const currentDraftSignature = computed(() => JSON.stringify({
  title: title.value,
  selectedCategorySlug: selectedCategorySlug.value,
  tagDraft: tagDraft.value,
  bodyMarkdown: bodyMarkdown.value
}))
const hasComposerContent = computed(() => Boolean(
  title.value.trim()
  || tagDraft.value.length
  || bodyMarkdown.value.trim()
))
const hasUnsavedChanges = computed(() => (
  submitState.value !== 'success'
  && hasComposerContent.value
  && currentDraftSignature.value !== lastPersistedDraftSignature.value
))

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

const categoryCheckText = computed(() => {
  if (selectedCategory.value) return t('composer.checks.category.selected', { category: selectedCategory.value.name })
  if (defaultCategory.value) return t('composer.checks.category.default', { category: defaultCategory.value.name })
  return t('composer.checks.category.serverDefault')
})

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

const prePublishChecks = computed(() => [
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
    ok: Boolean(summaryCategory.value || limits.value.defaultCategorySlug),
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
])

const draftStatusLabel = computed(() => {
  if (draftSaving.value) return t('composer.draft.saving')
  if (draftError.value) return draftError.value
  if (draftSavedAt.value) return t('composer.draft.savedAt', { time: draftSavedAt.value })
  if (hasUnsavedChanges.value) return t('composer.draft.unsaved')
  return t('composer.draft.ready')
})

const publishVisibilityLabel = computed(() => {
  if (submitState.value === 'success') return t('composer.publishState.success')
  if (submitState.value === 'error') return t('composer.publishState.error')
  return t('composer.publishState.public')
})

const tagPolicyLabel = computed(() => t(`composer.tagPolicy.${limits.value.tagCreationMode}`))

watch([categories, limits], () => {
  if (selectedCategorySlug.value || !limits.value.defaultCategorySlug) {
    return
  }
  if (categories.value.some(category => category.slug === limits.value.defaultCategorySlug)) {
    selectedCategorySlug.value = limits.value.defaultCategorySlug
  }
}, { immediate: true })

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
  if (
    limits.value.tagCreationMode === 'controlled'
    && !activeTagMap.value.has(raw)
  ) {
    fieldErrors.value.tagSlugs = [t('composer.tagUnknownControlled')]
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
  if (!validateTagCount(tagDraft.value.length)) {
    delete fieldErrors.value.tagSlugs
  }
}

function onTagKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ',') {
    return
  }
  event.preventDefault()
  addTag()
}

function tagDisplayName(slug: string) {
  return activeTagMap.value.get(slug)?.name || slug
}

function onEditorContentChange(payload: SFEditorContentPayload) {
  editorPayload.value = payload
}

function closeMobileDrawers() {
  mobileMenuOpen.value = false
  mobileInfoOpen.value = false
}

async function submit(payload?: { markdown?: string; native?: unknown; text?: string }) {
  if (!canCreate.value || submitState.value === 'submitting') {
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
  // 字数校验仍按 markdown/text 预估；最终权威在 Host Accept/Render。
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
  if (Object.keys(nextErrors).length) {
    fieldErrors.value = nextErrors
    submitState.value = 'error'
    return
  }

  submitState.value = 'submitting'
  errorMessage.value = ''
  fieldErrors.value = {}

  try {
    const created = await forumApi.createTopic({
      title: title.value.trim(),
      categorySlug: selectedCategorySlug.value || undefined,
      tagSlugs: tagDraft.value.length ? tagDraft.value : undefined,
      ...content
    })
    submitState.value = 'success'
    clearDraft()
    if (created.status === 'pending') {
      // 待审主题无公开落地页；Toast 提示后回首页，审核结果走通知。
      toast.add({ color: 'primary', icon: 'i-lucide-clock-3', title: t('composer.submittedForReview'), duration: 10000 })
      await navigateTo(localePath('/'))
      return
    }
    await navigateTo(localePath(forumTopicPath(created, topicUrlMode.value)))
  } catch (error) {
    submitState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: { markdown: string; native?: unknown; text?: string }) {
  submit(payload)
}

function submitCurrentDraft() {
  void submit(editorPayload.value || undefined)
}

function formatDraftTime(timestamp: number) {
  return new Date(timestamp).toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
}

function saveDraft({ quiet = false } = {}) {
  if (!import.meta.client || draftSaving.value) {
    return
  }
  draftSaving.value = true
  draftError.value = ''
  try {
    const savedAt = Date.now()
    sessionStorage.setItem(topicDraftKey.value, JSON.stringify({
      title: title.value,
      selectedCategorySlug: selectedCategorySlug.value,
      tagDraft: tagDraft.value,
      bodyMarkdown: bodyMarkdown.value,
      savedAt
    }))
    lastPersistedDraftSignature.value = currentDraftSignature.value
    draftSavedAt.value = formatDraftTime(savedAt)
    if (!quiet) {
      toast.add({
        color: 'primary',
        icon: 'i-lucide-save',
        title: t('composer.draft.saved'),
        duration: 10000
      })
    }
  } catch {
    draftError.value = t('composer.draft.saveFailed')
    if (!quiet) {
      toast.add({
        color: 'error',
        icon: 'i-lucide-triangle-alert',
        title: draftError.value,
        duration: 0
      })
    }
  } finally {
    draftSaving.value = false
  }
}

function clearDraft() {
  lastPersistedDraftSignature.value = currentDraftSignature.value
  if (!import.meta.client) {
    return
  }
  try {
    sessionStorage.removeItem(topicDraftKey.value)
  } catch {
    // 草稿清理失败不影响已提交的正式发布。
  }
}

function restoreDraft() {
  if (!import.meta.client) {
    return
  }
  try {
    const raw = sessionStorage.getItem(topicDraftKey.value)
    if (!raw) {
      lastPersistedDraftSignature.value = currentDraftSignature.value
      return
    }
    const draft = JSON.parse(raw) as {
      title?: string
      selectedCategorySlug?: string
      tagDraft?: string[]
      bodyMarkdown?: string
      savedAt?: number
    }
    title.value = draft.title || ''
    selectedCategorySlug.value = draft.selectedCategorySlug || selectedCategorySlug.value
    tagDraft.value = Array.isArray(draft.tagDraft) ? draft.tagDraft.filter(isForumTagSlug) : []
    bodyMarkdown.value = draft.bodyMarkdown || ''
    lastPersistedDraftSignature.value = currentDraftSignature.value
    if (draft.savedAt) {
      draftSavedAt.value = formatDraftTime(draft.savedAt)
    }
  } catch {
    draftError.value = t('composer.draft.restoreFailed')
  }
}

let beforeUnloadHandler: ((event: BeforeUnloadEvent) => void) | null = null

onMounted(() => {
  restoreDraft()
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

onBeforeRouteLeave((_to, _from, next) => {
  if (!import.meta.client || !hasUnsavedChanges.value) {
    next()
    return
  }
  if (window.confirm(t('composer.draft.leaveConfirm'))) {
    next()
  } else {
    next(false)
  }
})
</script>

<template>
  <main
    class="sforum-topic-composer"
    data-sforum-island-body="forum.component.topic_composer"
    data-layout="fullwidth-3col"
  >
    <div
      class="sforum-topic-composer__layout"
      :class="{ 'sforum-topic-composer__layout--with-right': canCreate }"
    >
      <div class="sforum-topic-composer__sidebar">
        <SFHomeNavigation
          desktop-only
          navigation-mode="route"
          :categories="categories"
          :selected-category-slug="selectedCategorySlug"
          :total-topics="totalTopics"
          :pending="categoriesPending"
          :can-create-topic="canCreate"
        />
      </div>

      <section class="sforum-topic-composer__main" :aria-labelledby="canCreate ? 'topic-composer-title' : undefined">
        <div class="sforum-topic-composer__mobile-nav">
          <SFHomeNavigation
            mobile-only
            navigation-mode="route"
            :categories="categories"
            :selected-category-slug="selectedCategorySlug"
            :total-topics="totalTopics"
            :pending="categoriesPending"
            :can-create-topic="canCreate"
          />
        </div>

      <!-- 无权限提示 -->
      <SFCard v-if="!canCreate" class="p-8">
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
            <span>{{ t('composer.title') }}</span>
          </nav>

          <header class="sforum-topic-composer__head">
            <div>
              <h1 id="topic-composer-title">{{ t('composer.title') }}</h1>
              <p>{{ t('composer.subtitle') }}</p>
            </div>
            <span class="sforum-topic-composer__draft-state" :class="{ 'is-error': draftError }">
              <UIcon :name="draftError ? 'i-lucide-cloud-off' : 'i-lucide-cloud-check'" class="size-4" aria-hidden="true" />
              {{ draftStatusLabel }}
            </span>
          </header>

        <!-- 全局错误（不自动消失） -->
        <SFAlert
          v-if="errorMessage"
          variant="danger"
          :title="errorMessage"
          closable
          class="sforum-topic-composer__alert"
          @close="errorMessage = ''"
        />

        <form class="sforum-topic-composer__form" @submit.prevent="submitCurrentDraft">
          <!-- 分类选择(SFInput 不支持 select,保留原生但复用全局 .sf-input__control 样式) -->
          <div class="sforum-topic-composer__taxonomy">
            <div class="sforum-topic-composer__field">
              <div class="sforum-topic-composer__field-head">
                <label for="topic-composer-category">{{ t('composer.categoryLabel') }}</label>
                <span>{{ t('composer.categoryOptionalDefault') }}</span>
              </div>
              <div class="sforum-topic-composer__select">
              <select
                id="topic-composer-category"
                v-model="selectedCategorySlug"
                class="sf-input__control sf-input__control--md w-full appearance-none pr-9"
              >
                <option value="">{{ t('composer.categoryDefault') }}</option>
                <option v-for="cat in categories" :key="cat.id" :value="cat.slug">
                  {{ cat.name }}
                </option>
              </select>
                <UIcon name="i-lucide-chevron-down" class="size-4" aria-hidden="true" />
              </div>
              <p class="sforum-topic-composer__hint">{{ t('composer.categoryHint') }}</p>
              <p v-if="fieldErrors.categorySlug" class="sforum-topic-composer__error">
                {{ fieldErrors.categorySlug.join(', ') }}
              </p>
            </div>

            <div class="sforum-topic-composer__field">
              <div class="sforum-topic-composer__field-head">
                <label for="topic-composer-tags">{{ t('composer.tagsLabel') }}</label>
                <span>{{ t('composer.tagLimitSummary', { min: limits.tagMinPerTopic, max: limits.tagMaxPerTopic }) }}</span>
              </div>
              <div class="sforum-topic-composer__tag-control" :class="{ 'is-invalid': fieldErrors.tagSlugs }">
                <span v-for="slug in tagDraft" :key="slug" class="sforum-topic-composer__tag">
                  #{{ tagDisplayName(slug) }}
                  <button type="button" :aria-label="t('composer.removeTag')" @click="removeTag(slug)">
                    <UIcon name="i-lucide-x" class="size-3" aria-hidden="true" />
                  </button>
                </span>
                <input
                  id="topic-composer-tags"
                  v-model="tagInput"
                  type="text"
                  :list="tagOptions.length ? 'topic-composer-tag-options' : undefined"
                  :placeholder="tagDraft.length ? t('composer.tagsPlaceholderMore') : t('composer.tagsPlaceholder')"
                  :disabled="submitState === 'submitting' || tagDraft.length >= limits.tagMaxPerTopic"
                  @keydown="onTagKeydown"
                  @blur="addTag"
                >
                <datalist id="topic-composer-tag-options">
                  <option v-for="tag in tagOptions" :key="tag.slug" :value="tag.slug">
                    {{ tag.name }}
                  </option>
                </datalist>
              </div>
              <p class="sforum-topic-composer__hint">{{ tagsPending ? t('composer.tagsLoading') : t('composer.tagsHint') }}</p>
              <p v-if="fieldErrors.tagSlugs" class="sforum-topic-composer__error">
                {{ fieldErrors.tagSlugs.join(', ') }}
              </p>
            </div>
          </div>

          <!-- 标题(复用 SFInput 组件) -->
          <div class="sforum-topic-composer__field">
            <SFInput
              v-model="title"
              :label="t('composer.titleLabel')"
              :placeholder="t('composer.titlePlaceholder')"
              :hint="`${titleHint} (${t('composer.charCount', { count: titleCount, max: limits.topicTitleMaxRunes })})`"
              :error="fieldErrors.title?.join(', ')"
              required
            />
          </div>

          <!-- 正文编辑器 -->
          <div class="sforum-topic-composer__field">
            <div class="sforum-topic-composer__field-head">
              <label>{{ t('composer.bodyLabel') }}</label>
              <span>{{ t('composer.charCount', { count: bodyCount, max: limits.topicContentMaxRunes }) }}</span>
            </div>
            <p class="sforum-topic-composer__hint">{{ bodyHint }}</p>
            <!-- F4.3：扩展 composer 工具栏（宿主渲染按钮，执行走扩展路由） -->
            <div
              v-if="composerToolbarActions.length"
              class="sforum-topic-composer__extension-toolbar"
            >
              <SFButton
                v-for="action in composerToolbarActions"
                :key="`${action.extensionId}:${action.id}`"
                type="button"
                size="sm"
                variant="ghost"
                :disabled="submitState === 'submitting'"
                @click="runComposerToolbarAction(action)"
              >
                <UIcon v-if="action.icon" :name="action.icon" class="size-4" />
                <span>{{ composerToolbarLabel(action) }}</span>
              </SFButton>
            </div>
            <LazySFEditor
              v-model="bodyMarkdown"
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
        </div>
      </template>

        <footer v-if="canCreate" class="sforum-topic-composer__dock" aria-live="polite">
          <div class="sforum-topic-composer__dock-status">
            <SFAvatar size="sm" :name="actorName" :avatar="user?.avatar" />
            <span>
              <strong>{{ t('composer.publishAs', { name: actorName }) }}</strong>
              {{ publishVisibilityLabel }} · {{ draftStatusLabel }}
            </span>
          </div>
          <div class="sforum-topic-composer__dock-actions">
            <SFButton
              type="button"
              variant="ghost"
              :disabled="draftSaving || submitState === 'submitting'"
              @click="saveDraft()"
            >
              <UIcon name="i-lucide-save" class="size-4" aria-hidden="true" />
              {{ draftSaving ? t('composer.draft.saving') : t('composer.draft.save') }}
            </SFButton>
            <SFButton
              type="button"
              :disabled="!canSubmit"
              @click="submitCurrentDraft"
            >
              <UIcon name="i-lucide-send" class="size-4" aria-hidden="true" />
              {{ submitLabel }}
            </SFButton>
          </div>
        </footer>
      </section>

      <aside v-if="canCreate" class="sforum-topic-composer__right" :aria-label="t('composer.rightRail.label')">
        <section class="sforum-topic-composer__right-section">
          <h2>
            {{ t('composer.summary.title') }}
            <span>{{ publishVisibilityLabel }}</span>
          </h2>
          <div class="sforum-topic-composer__summary-list">
            <div class="sforum-topic-composer__summary-row">
              <UIcon name="i-lucide-folder" class="size-4" aria-hidden="true" />
              <div>
                <strong>{{ summaryCategory?.name || t('composer.categoryDefaultShort') }}</strong>
                <span>{{ summaryCategory ? t('composer.summary.category') : t('composer.summary.categoryDefault') }}</span>
              </div>
            </div>
            <div class="sforum-topic-composer__summary-row">
              <UIcon name="i-lucide-heading-1" class="size-4" aria-hidden="true" />
              <div>
                <strong>{{ title.trim() || t('composer.summary.untitled') }}</strong>
                <span>{{ t('composer.charCount', { count: titleCount, max: limits.topicTitleMaxRunes }) }}</span>
              </div>
            </div>
            <div class="sforum-topic-composer__summary-row">
              <UIcon name="i-lucide-tags" class="size-4" aria-hidden="true" />
              <div>
                <strong>{{ tagDraft.length ? t('composer.summary.tagCount', { count: tagDraft.length }) : t('composer.summary.noTags') }}</strong>
                <span>{{ tagPolicyLabel }}</span>
              </div>
            </div>
            <div class="sforum-topic-composer__summary-row">
              <UIcon name="i-lucide-user-round" class="size-4" aria-hidden="true" />
              <div>
                <strong>{{ actorName }}</strong>
                <span>{{ t('composer.summary.actor') }}</span>
              </div>
            </div>
          </div>
        </section>

        <section class="sforum-topic-composer__right-section">
          <h2>{{ t('composer.settings.title') }}</h2>
          <dl class="sforum-topic-composer__settings">
            <div>
              <dt>{{ t('composer.settings.permission') }}</dt>
              <dd>{{ t('composer.settings.permissionValue') }}</dd>
            </div>
            <div>
              <dt>{{ t('composer.settings.category') }}</dt>
              <dd>{{ summaryCategory?.name || t('composer.categoryDefaultShort') }}</dd>
            </div>
            <div>
              <dt>{{ t('composer.settings.tags') }}</dt>
              <dd>{{ tagPolicyLabel }}</dd>
            </div>
            <div>
              <dt>{{ t('composer.settings.limits') }}</dt>
              <dd>{{ t('composer.settings.limitValue', { titleMax: limits.topicTitleMaxRunes, bodyMax: limits.topicContentMaxRunes }) }}</dd>
            </div>
          </dl>
        </section>

        <section class="sforum-topic-composer__right-section">
          <h2>{{ t('composer.checks.heading') }}</h2>
          <ul class="sforum-topic-composer__checks">
            <li
              v-for="item in prePublishChecks"
              :key="item.key"
              :class="{ 'is-ok': item.ok }"
            >
              <UIcon :name="item.ok ? 'i-lucide-circle-check' : 'i-lucide-circle'" class="size-4" aria-hidden="true" />
              <span>
                <strong>{{ item.label }}</strong>
                {{ item.text }}
              </span>
            </li>
          </ul>
        </section>

        <section class="sforum-topic-composer__right-section">
          <div class="sforum-topic-composer__tip">
            <UIcon name="i-lucide-lightbulb" class="size-4" aria-hidden="true" />
            <span>{{ t('composer.rightRail.tip') }}</span>
          </div>
        </section>
      </aside>
    </div>

    <button
      v-if="mobileMenuOpen || mobileInfoOpen"
      type="button"
      class="sforum-mobile-drawer__backdrop"
      :aria-label="t('topicDetail.cancel')"
      @click="closeMobileDrawers"
    />

    <aside v-if="mobileMenuOpen" class="sforum-mobile-drawer sforum-mobile-drawer--left">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.navTitle') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFHomeNavigation
        desktop-only
        navigation-mode="route"
        :categories="categories"
        :selected-category-slug="selectedCategorySlug"
        :total-topics="totalTopics"
        :pending="categoriesPending"
        :can-create-topic="canCreate"
      />
    </aside>

    <aside v-if="mobileInfoOpen && canCreate" class="sforum-mobile-drawer sforum-mobile-drawer--right">
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('composer.rightRail.label') }}</strong>
        <button type="button" :aria-label="t('topicDetail.cancel')" @click="closeMobileDrawers">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <div class="sforum-topic-composer__mobile-summary">
        <ul class="sforum-topic-composer__checks">
          <li
            v-for="item in prePublishChecks"
            :key="item.key"
            :class="{ 'is-ok': item.ok }"
          >
            <UIcon :name="item.ok ? 'i-lucide-circle-check' : 'i-lucide-circle'" class="size-4" aria-hidden="true" />
            <span>
              <strong>{{ item.label }}</strong>
              {{ item.text }}
            </span>
          </li>
        </ul>
      </div>
    </aside>
  </main>
</template>

<style scoped src="./SFTopicComposerPage.css"></style>
