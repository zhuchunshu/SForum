<script setup lang="ts">
import {
  forumTopicPath,
  isForumTagSlug,
  normalizeForumTagSlugInput,
  type ForumCategory,
  type ForumCategoryGroup
} from '~/utils/forumTaxonomy'

definePageMeta({
  // 发帖需要登录；路由中间件由全局 auth 守卫处理，这里只声明意图。
  requiresAuth: true
})

const { t } = useI18n()
const localePath = useLocalePath()
const { siteName, seoSettings } = useWebOptions()
const topicUrlMode = computed(() => seoSettings.value.topicUrlMode)
const forumApi = useForumApi()
const { can } = usePermissions()
const toast = useToast()
const { locale } = useI18n()

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
      title: apiErrorMessage(error) || t('composer.submitFailed')
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

async function submit(payload?: { markdown?: string }) {
  if (!canCreate.value || submitState.value === 'submitting') {
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
    if (created.status === 'pending') {
      toast.add({ color: 'primary', icon: 'i-lucide-clock-3', title: t('composer.submittedForReview'), duration: 10000 })
      await navigateTo(localePath('/my/content-review'))
      return
    }
    await navigateTo(localePath(forumTopicPath(created, topicUrlMode.value)))
  } catch (error) {
    submitState.value = 'error'
    errorMessage.value = apiErrorMessage(error) || t('composer.submitFailed')
    fieldErrors.value = apiErrorFields(error)
  }
}

function onEditorSubmit(payload: { markdown: string }) {
  submit({ markdown: payload.markdown })
}

// 侧栏:发帖要点与 Markdown 速查(走 i18n,中英文都支持)。
const composerTips = computed(() => [
  t('composer.tip1'),
  t('composer.tip2'),
  t('composer.tip3'),
  t('composer.tip4')
])
const markdownCheatsheet = computed(() => [
  { label: t('composer.mdHeading'), syntax: '# 标题' },
  { label: t('composer.mdBold'), syntax: '**粗体**' },
  { label: t('composer.mdList'), syntax: '- 项目' },
  { label: t('composer.mdCode'), syntax: '```go```' },
  { label: t('composer.mdQuote'), syntax: '> 引用' },
  { label: t('composer.mdLink'), syntax: '[文字](url)' }
])
</script>

<template>
  <SFPageOutlet page="forum.topic.create">
  <main class="sf-public-page min-h-screen py-8">
    <div class="sf-public-page__container mx-auto px-4 sm:px-6">
      <!-- 无权限提示 -->
      <SFCard v-if="!canCreate" class="p-8">
        <SFEmptyState
          icon-label="LOCK"
          :title="t('composer.permissionDenied.title')"
          :description="t('composer.permissionDenied.description')"
        />
      </SFCard>

      <template v-else>
        <!-- 双栏:表单 + 侧边栏 -->
        <div class="grid grid-cols-1 lg:grid-cols-[1fr_300px] gap-8 items-start">

        <!-- 主栏 -->
        <div class="min-w-0">
        <!-- 轻量页头:面包屑 + 副标题,不再放大标题(navbar 已在) -->
        <div class="mb-5">
          <nav class="text-sm text-slate-400 dark:text-zinc-500 flex items-center gap-1.5">
            <NuxtLink :to="localePath('/')" class="hover:text-[color:var(--sf-accent)]">
              {{ t('composer.breadcrumbHome') }}
            </NuxtLink>
            <UIcon name="i-lucide-chevron-right" class="size-3" />
            <span>{{ t('composer.title') }}</span>
          </nav>
          <p class="text-sm text-slate-500 dark:text-zinc-400 mt-2">{{ t('composer.subtitle') }}</p>
        </div>

        <!-- 全局错误（不自动消失） -->
        <SFAlert
          v-if="errorMessage"
          variant="danger"
          :title="errorMessage"
          closable
          class="mb-4"
          @close="errorMessage = ''"
        />

        <SFCard class="p-6 sm:p-8 space-y-6">
          <!-- 分类选择(SFInput 不支持 select,保留原生但复用全局 .sf-input__control 样式) -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-1.5 dark:text-zinc-300">
              {{ t('composer.categoryLabel') }}
              <span class="text-rose-500">*</span>
            </label>
            <p class="text-xs text-slate-400 dark:text-zinc-500 mb-2">{{ t('composer.categoryHint') }}</p>
            <div class="relative">
              <select
                v-model="selectedCategorySlug"
                class="sf-input__control sf-input__control--md w-full appearance-none pr-9"
              >
                <option value="">{{ t('composer.categoryDefault') }}</option>
                <option v-for="cat in categories" :key="cat.id" :value="cat.slug">
                  {{ cat.name }}
                </option>
              </select>
              <UIcon name="i-lucide-chevron-down" class="size-4 absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
            </div>
          </div>

          <!-- 标题(复用 SFInput 组件) -->
          <SFInput
            v-model="title"
            :label="t('composer.titleLabel')"
            :placeholder="t('composer.titlePlaceholder')"
            :hint="`${titleHint} (${t('composer.charCount', { count: titleCount, max: limits.topicTitleMaxRunes })})`"
            :error="fieldErrors.title?.join(', ')"
            required
          />

          <!-- 标签 -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-1.5 dark:text-zinc-300">
              {{ t('composer.tagsLabel') }}
            </label>
            <p class="text-xs text-slate-400 dark:text-zinc-500 mb-2">{{ t('composer.tagsHint') }}</p>
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
              class="sf-input__control sf-input__control--md w-full"
              :placeholder="t('composer.tagsPlaceholder')"
              @keydown.enter="onTagEnter"
            >
            <p v-if="fieldErrors.tagSlugs" class="text-sm text-red-600 mt-1 dark:text-red-400">
              {{ fieldErrors.tagSlugs.join(', ') }}
            </p>
          </div>

          <!-- 正文编辑器 -->
          <div>
            <label class="block text-sm font-semibold text-slate-700 mb-1.5 dark:text-zinc-300">
              {{ t('composer.bodyLabel') }}
              <span class="text-rose-500">*</span>
            </label>
            <p class="text-xs text-slate-400 dark:text-zinc-500 mb-2">
              {{ bodyHint }} ({{ t('composer.charCount', { count: bodyCount, max: limits.topicContentMaxRunes }) }})
            </p>
            <!-- F4.3：扩展 composer 工具栏（宿主渲染按钮，执行走扩展路由） -->
            <div
              v-if="composerToolbarActions.length"
              class="mb-2 flex flex-wrap items-center gap-2"
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
              @submit="onEditorSubmit"
            />
            <p v-if="fieldErrors.content" class="text-sm text-red-600 mt-1 dark:text-red-400">
              {{ fieldErrors.content.join(', ') }}
            </p>
          </div>
        </SFCard>
        </div><!-- /主栏 -->

        <!-- 侧边栏:sticky 跟随 -->
        <aside class="hidden lg:block lg:sticky lg:top-20 space-y-4">
          <!-- 发帖要点 -->
          <SFCard class="p-4">
            <h3 class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wide mb-3 flex items-center gap-1.5">
              <UIcon name="i-lucide-lightbulb" class="size-3.5 text-[color:var(--sf-accent)]" />
              {{ t('composer.tipsTitle') }}
            </h3>
            <ul class="space-y-2.5 text-sm">
              <li v-for="tip in composerTips" :key="tip" class="flex gap-2">
                <UIcon name="i-lucide-circle-dot" class="size-3.5 text-[color:var(--sf-accent)] mt-0.5 flex-none" />
                <span class="text-slate-600 dark:text-zinc-300">{{ tip }}</span>
              </li>
            </ul>
          </SFCard>

          <!-- Markdown 速查 -->
          <SFCard class="p-4">
            <h3 class="text-xs font-semibold text-slate-500 dark:text-zinc-400 uppercase tracking-wide mb-3">
              {{ t('composer.markdownTitle') }}
            </h3>
            <dl class="space-y-2 text-xs">
              <div v-for="item in markdownCheatsheet" :key="item.label" class="flex justify-between gap-2">
                <dt class="text-slate-400 dark:text-zinc-500">{{ item.label }}</dt>
                <dd class="font-mono text-slate-600 dark:text-zinc-300">{{ item.syntax }}</dd>
              </div>
            </dl>
          </SFCard>
        </aside>

        </div><!-- /双栏 grid -->
      </template>
    </div>
  </main>

  </SFPageOutlet>
</template>
