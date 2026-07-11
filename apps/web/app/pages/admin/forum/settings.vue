<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  createAdminForumApi,
  createDefaultForumSettings,
  forumSettingsManageKeys,
  forumSettingsPayload,
  forumSettingsValidationError,
  normalizeForumSettings
} from '~/utils/adminForum'
import type { ForumCategory, ForumCategoryGroup, ForumSettings, ForumTagCreationMode } from '~/utils/forumTaxonomy'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminForumSettings'
})

type ForumSettingsTab = 'general' | 'topics' | 'comments' | 'tags' | 'reading'

const { t } = useI18n()
const toast = useToast()
const route = useRoute()
const router = useRouter()
const { request } = useApiClient()
const forumApi = createAdminForumApi(request)
const adminPage = useAdminPage('/forum/settings')
const adminRoutes = useAdminRoutes()
const { can } = usePermissions()

const saving = ref(false)
const restoring = ref(false)
const savedSnapshot = ref('')
const form = reactive(createDefaultForumSettings())

const tabIds: ForumSettingsTab[] = ['general', 'topics', 'comments', 'tags', 'reading']
const activeTab = ref<ForumSettingsTab>(normalizeTab(route.query.tab))

// useAsyncData 在 SSR 水合时不会重跑 handler；表单副作用必须用 watch 同步，
// 否则默认版块 select 等会卡在 createDefaultForumSettings() 的值。
const { data, pending, error, refresh } = await useAsyncData('admin-forum-settings', async () => {
  const [groups, settings] = await Promise.all([
    forumApi.listCategoryGroups(),
    forumApi.getSettings()
  ])
  return { groups, settings }
}, {
  default: () => ({ groups: [] as ForumCategoryGroup[], settings: createDefaultForumSettings() })
})

watch(data, (payload) => {
  if (payload?.settings) {
    applySettings(payload.settings)
  }
}, { immediate: true })

watch(() => route.query.tab, (value) => {
  activeTab.value = normalizeTab(value)
})

useSeoMeta({
  title: t('admin.forum.settings.metaTitle')
})

const categoryGroups = computed(() => data.value.groups)
const categories = computed(() => categoryGroups.value.flatMap((group) => group.categories || []))
const publicCategories = computed(() => categories.value.filter((category) => category.visibility === 'public'))
// 无公开版块时不渲染空 select，引导先去版块分类页创建。
const hasPublicCategories = computed(() => publicCategories.value.length > 0)
const hasChanges = computed(() => JSON.stringify(forumSettingsPayload(form)) !== savedSnapshot.value)
const recommended = createDefaultForumSettings()
const canManageSettings = computed(() => can('settings.manage'))
const canManageTags = computed(() => can('tag.manage'))
const canManageCategories = computed(() => can('category.manage'))
const validationKey = computed(() => forumSettingsValidationError(form))

const tabs = computed(() => tabIds.map((id) => ({
  id,
  label: t(`admin.forum.settings.tabs.${id}`),
  icon: tabIcon(id)
})))

const modeOptions = computed<Array<{ value: ForumTagCreationMode, label: string, description: string }>>(() => [
  {
    value: 'controlled',
    label: t('admin.forum.settings.modes.controlled.label'),
    description: t('admin.forum.settings.modes.controlled.description')
  },
  {
    value: 'review',
    label: t('admin.forum.settings.modes.review.label'),
    description: t('admin.forum.settings.modes.review.description')
  },
  {
    value: 'open',
    label: t('admin.forum.settings.modes.open.label'),
    description: t('admin.forum.settings.modes.open.description')
  }
])

async function saveSettings() {
  if (validationKey.value) {
    return
  }
  saving.value = true
  try {
    const payload = forumSettingsPayload(form) as Record<string, unknown>
    if (!canManageSettings.value) {
      for (const key of forumSettingsManageKeys) {
        delete payload[key]
      }
    }
    if (!canManageTags.value) {
      delete payload.tagCreationMode
      delete payload.tagPublicPages
      delete payload.tagMinPerTopic
      delete payload.tagMaxPerTopic
    }
    if (!canManageCategories.value) {
      delete payload.defaultCategorySlug
    }
    const updated = await forumApi.updateSettings(payload as ReturnType<typeof forumSettingsPayload>)
    applySettings(updated)
    successToast(t('admin.forum.settings.saved'))
  } catch (error) {
    errorToast(error, t('admin.forum.settings.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function restoreRecommended() {
  restoring.value = true
  try {
    const updated = await forumApi.resetSettings()
    applySettings(updated)
    successToast(t('admin.forum.settings.restored'))
  } catch (error) {
    errorToast(error, t('admin.forum.settings.restoreFailed'))
  } finally {
    restoring.value = false
  }
}

function applySettings(settings: ForumSettings) {
  Object.assign(form, normalizeForumSettings(settings))
  savedSnapshot.value = JSON.stringify(forumSettingsPayload(form))
}

function chooseDefaultCategory(category: ForumCategory) {
  form.defaultCategorySlug = category.slug
}

function setActiveTab(tab: ForumSettingsTab) {
  activeTab.value = tab
  router.replace({ query: { ...route.query, tab } })
}

function normalizeTab(value: unknown): ForumSettingsTab {
  const raw = Array.isArray(value) ? value[0] : value
  return tabIds.includes(raw as ForumSettingsTab) ? raw as ForumSettingsTab : 'general'
}

function tabIcon(id: ForumSettingsTab) {
  switch (id) {
    case 'general':
      return 'i-lucide-sliders-horizontal'
    case 'topics':
      return 'i-lucide-file-pen-line'
    case 'comments':
      return 'i-lucide-messages-square'
    case 'tags':
      return 'i-lucide-tags'
    case 'reading':
      return 'i-lucide-book-open-text'
  }
}

function successToast(title: string) {
  toast.add({
    color: 'success',
    icon: 'i-lucide-check',
    title,
    duration: 10000
  })
}

function errorToast(error: unknown, fallback: string) {
  toast.add({
    color: 'error',
    icon: 'i-lucide-triangle-alert',
    title: apiErrorMessage(error) || fallback
  })
}
</script>

<template>
  <div class="mb-4">
    <h2 class="text-xl font-bold flex items-center gap-2 text-slate-900 dark:text-zinc-100">
      <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
      {{ t('admin.forum.settings.title') }}
    </h2>
  </div>

  <UDashboardToolbar class="border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded-lg px-4 py-2.5 mb-6 text-slate-500 dark:text-zinc-400">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-slate-500 dark:text-zinc-400">
        <UIcon name="i-lucide-sliders-horizontal" class="size-4" />
        <span class="truncate">{{ t('admin.forum.settings.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pending"
        class="border-slate-200 dark:border-zinc-700"
        @click="refresh()"
      >
        {{ t('admin.common.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <form class="grid gap-4" @submit.prevent="saveSettings">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.forum.settings.loadFailed')"
    />

    <UAlert
      color="primary"
      variant="soft"
      icon="i-lucide-sparkles"
      :title="t('admin.forum.settings.recommendedTitle')"
      :description="t('admin.forum.settings.recommendedDescription')"
    />

    <div
      role="tablist"
      :aria-label="t('admin.forum.settings.tabs.label')"
      class="flex flex-wrap gap-2 border-b border-slate-200 pb-3 dark:border-zinc-800"
    >
      <UButton
        v-for="tab in tabs"
        :key="tab.id"
        :color="activeTab === tab.id ? 'primary' : 'neutral'"
        :variant="activeTab === tab.id ? 'solid' : 'ghost'"
        :leading-icon="tab.icon"
        role="tab"
        :aria-selected="activeTab === tab.id"
        @click="setActiveTab(tab.id)"
      >
        {{ tab.label }}
      </UButton>
    </div>

    <UCard
      class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900"
      :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t(`admin.forum.settings.sections.${activeTab}.title`) }}
            </h3>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t(`admin.forum.settings.sections.${activeTab}.description`) }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="font-mono">
            forum.*
          </UBadge>
        </div>
      </template>

      <div class="grid max-w-5xl gap-6">
        <!-- 常规 -->
        <template v-if="activeTab === 'general'">
          <section class="space-y-3">
            <div
              v-if="!pending && !hasPublicCategories"
              class="rounded-lg border border-dashed border-slate-200 bg-slate-50/80 p-6 dark:border-zinc-700 dark:bg-zinc-950/40"
            >
              <SFEmptyState
                icon-label="CAT"
                :title="t('admin.forum.settings.noPublicCategoriesTitle')"
                :description="t('admin.forum.settings.noPublicCategoriesDescription')"
              />
              <div class="mt-5 flex flex-wrap justify-center gap-2">
                <UButton icon="i-lucide-folder-tree" color="primary" :to="adminRoutes.path('/forum/categories')">
                  {{ t('admin.forum.settings.openCategories') }}
                </UButton>
                <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="refresh()">
                  {{ t('admin.common.refresh') }}
                </UButton>
              </div>
            </div>

            <template v-else>
              <UFormField :label="t('admin.forum.settings.defaultCategory')" name="default-category">
                <select
                  v-model="form.defaultCategorySlug"
                  :disabled="!canManageCategories"
                  class="h-10 w-full max-w-xl rounded-md border border-slate-200 bg-white px-3 text-sm text-slate-900 outline-none transition focus:border-[var(--sf-accent)] focus:ring-2 focus:ring-[var(--sf-accent-focus)] dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
                >
                  <option v-if="!form.defaultCategorySlug" value="" disabled>
                    {{ t('admin.forum.settings.defaultCategoryPlaceholder') }}
                  </option>
                  <option v-for="category in publicCategories" :key="category.slug" :value="category.slug">
                    {{ category.name }} (/c/{{ category.slug }})
                  </option>
                </select>
              </UFormField>

              <div class="grid gap-2 md:grid-cols-2">
                <button
                  v-for="category in publicCategories"
                  :key="category.id"
                  type="button"
                  :disabled="!canManageCategories"
                  class="rounded-lg border px-3 py-2 text-left text-sm transition disabled:cursor-not-allowed disabled:opacity-60"
                  :class="form.defaultCategorySlug === category.slug ? 'border-[#0F766E] bg-[#E6F4F1] text-[#0F766E] dark:border-teal-700 dark:bg-teal-950/40 dark:text-teal-300' : 'border-slate-200 text-slate-700 hover:border-[#0F766E] dark:border-zinc-700 dark:text-zinc-300'"
                  @click="chooseDefaultCategory(category)"
                >
                  <span class="block font-semibold">{{ category.name }}</span>
                  <span class="mt-1 block text-xs opacity-70">/c/{{ category.slug }}</span>
                </button>
              </div>
            </template>
          </section>

          <section class="space-y-3 border-t border-slate-200 pt-5 dark:border-zinc-800">
            <div>
              <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.forum.settings.paginationTitle') }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.forum.settings.paginationHelp') }}
              </p>
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="t('admin.forum.settings.topicsPerPage')" name="topics-per-page">
                <UInputNumber
                  v-model="form.topicsPerPage"
                  :min="1"
                  :max="100"
                  :disabled="!canManageSettings"
                  class="w-full"
                />
              </UFormField>
              <UFormField :label="t('admin.forum.settings.commentsPerPage')" name="comments-per-page">
                <UInputNumber
                  v-model="form.commentsPerPage"
                  :min="1"
                  :max="100"
                  :disabled="!canManageSettings"
                  class="w-full"
                />
              </UFormField>
            </div>
            <p v-if="!canManageSettings" class="text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.forum.settings.paginationPermissionHelp') }}
            </p>
          </section>
        </template>

        <!-- 发帖 -->
        <template v-else-if="activeTab === 'topics'">
          <section class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.forum.settings.topicTitleMin')" name="topic-title-min">
              <UInputNumber v-model="form.topicTitleMinRunes" :min="1" :max="200" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.topicTitleMinHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.topicTitleMax')" name="topic-title-max">
              <UInputNumber v-model="form.topicTitleMaxRunes" :min="1" :max="200" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.recommendedHint', { value: recommended.topicTitleMaxRunes }) }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.topicContentMin')" name="topic-content-min">
              <UInputNumber v-model="form.topicContentMinRunes" :min="0" :max="200000" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.topicContentMinHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.topicContentMax')" name="topic-content-max">
              <UInputNumber v-model="form.topicContentMaxRunes" :min="1" :max="200000" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.recommendedHint', { value: recommended.topicContentMaxRunes }) }}</p>
            </UFormField>
          </section>

          <section class="grid gap-4 border-t border-slate-200 pt-5 dark:border-zinc-800 md:grid-cols-3">
            <UFormField :label="t('admin.forum.settings.topicEditWindow')" name="topic-edit-window">
              <UInputNumber v-model="form.topicEditWindowMinutes" :min="0" :max="10080" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.zeroUnlimitedHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.topicCooldown')" name="topic-cooldown">
              <UInputNumber v-model="form.topicCooldownSeconds" :min="0" :max="86400" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.zeroUnlimitedHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.dailyTopicLimit')" name="daily-topic-limit">
              <UInputNumber v-model="form.dailyTopicLimit" :min="0" :max="10000" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.zeroUnlimitedHelp') }}</p>
            </UFormField>
          </section>
        </template>

        <!-- 评论 -->
        <template v-else-if="activeTab === 'comments'">
          <section class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.forum.settings.commentMin')" name="comment-min">
              <UInputNumber v-model="form.commentMinRunes" :min="0" :max="50000" :disabled="!canManageSettings" class="w-full" />
            </UFormField>
            <UFormField :label="t('admin.forum.settings.commentMax')" name="comment-max">
              <UInputNumber v-model="form.commentMaxRunes" :min="1" :max="50000" :disabled="!canManageSettings" class="w-full" />
            </UFormField>
            <UFormField :label="t('admin.forum.settings.commentNesting')" name="comment-nesting">
              <UInputNumber v-model="form.commentMaxNestingDepth" :min="0" :max="20" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.commentNestingHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.commentEditWindow')" name="comment-edit-window">
              <UInputNumber v-model="form.commentEditWindowMinutes" :min="0" :max="10080" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.zeroUnlimitedHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.commentCooldown')" name="comment-cooldown">
              <UInputNumber v-model="form.commentCooldownSeconds" :min="0" :max="86400" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.zeroUnlimitedHelp') }}</p>
            </UFormField>
            <UFormField :label="t('admin.forum.settings.dailyCommentLimit')" name="daily-comment-limit">
              <UInputNumber v-model="form.dailyCommentLimit" :min="0" :max="10000" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.zeroUnlimitedHelp') }}</p>
            </UFormField>
          </section>
        </template>

        <!-- 标签 -->
        <template v-else-if="activeTab === 'tags'">
          <section class="space-y-3">
            <div>
              <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.forum.settings.tagCreationMode') }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.forum.settings.tagCreationModeHelp') }}
              </p>
            </div>

            <div class="grid gap-3 md:grid-cols-3">
              <button
                v-for="mode in modeOptions"
                :key="mode.value"
                type="button"
                :disabled="!canManageTags"
                class="rounded-lg border px-4 py-3 text-left transition disabled:cursor-not-allowed disabled:opacity-60"
                :class="form.tagCreationMode === mode.value ? 'border-[#0F766E] bg-[#E6F4F1] text-[#0F766E] dark:border-teal-700 dark:bg-teal-950/40 dark:text-teal-300' : 'border-slate-200 text-slate-700 hover:border-[#0F766E] dark:border-zinc-700 dark:text-zinc-300'"
                @click="form.tagCreationMode = mode.value"
              >
                <span class="block text-sm font-bold">{{ mode.label }}</span>
                <span class="mt-1 block text-xs leading-5 opacity-75">{{ mode.description }}</span>
              </button>
            </div>
          </section>

          <section class="grid gap-4 border-t border-slate-200 pt-5 dark:border-zinc-800 md:grid-cols-2">
            <label class="flex cursor-pointer items-start gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-3 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
              <input
                v-model="form.tagPublicPages"
                type="checkbox"
                :disabled="!canManageTags"
                class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
              >
              <span>
                <span class="font-semibold text-slate-900 dark:text-zinc-100">
                  {{ t('admin.forum.settings.publicTagPages') }}
                </span>
                <span class="mt-1 block text-xs leading-5 text-slate-500 dark:text-zinc-400">
                  {{ t('admin.forum.settings.publicTagPagesHelp') }}
                </span>
              </span>
            </label>

            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('admin.forum.settings.minTagsPerTopic')" name="min-tags">
                <UInputNumber v-model="form.tagMinPerTopic" :min="0" :max="10" :disabled="!canManageTags" class="w-full" />
                <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.minTagsPerTopicHelp') }}</p>
              </UFormField>
              <UFormField :label="t('admin.forum.settings.maxTagsPerTopic')" name="max-tags">
                <UInputNumber v-model="form.tagMaxPerTopic" :min="0" :max="10" :disabled="!canManageTags" class="w-full" />
                <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.maxTagsPerTopicHelp') }}</p>
              </UFormField>
            </div>
          </section>
        </template>

        <!-- 阅读 -->
        <template v-else>
          <section class="grid gap-4 md:grid-cols-2">
            <UFormField :label="t('admin.forum.settings.excerptRuneLimit')" name="excerpt-limit">
              <UInputNumber v-model="form.excerptRuneLimit" :min="40" :max="500" :disabled="!canManageSettings" class="w-full" />
              <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.excerptRuneLimitHelp') }}</p>
            </UFormField>
          </section>
          <section class="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60">
            <p class="font-medium text-slate-900 dark:text-zinc-100">{{ t('admin.forum.settings.readingNoteTitle') }}</p>
            <p class="mt-1 text-xs leading-5 text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.readingNoteBody') }}</p>
          </section>
        </template>

        <!-- 推荐值摘要 -->
        <section class="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60 md:grid-cols-4">
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.defaultCategory') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.defaultCategorySlug }}</span>
          </div>
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.topicTitleMax') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.topicTitleMinRunes }}–{{ recommended.topicTitleMaxRunes }}</span>
          </div>
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.commentNesting') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.commentMaxNestingDepth }}</span>
          </div>
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.maxTagsPerTopic') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.tagMinPerTopic }}–{{ recommended.tagMaxPerTopic }}</span>
          </div>
        </section>
      </div>

      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <UAlert
            v-if="validationKey"
            color="error"
            variant="soft"
            icon="i-lucide-triangle-alert"
            :title="t(`admin.forum.settings.validation.${validationKey}`)"
            class="flex-1"
          />
          <UAlert
            v-else-if="hasChanges"
            color="warning"
            variant="soft"
            icon="i-lucide-pencil"
            :title="t('admin.forum.settings.unsaved')"
            class="flex-1"
          />
          <div class="ml-auto flex flex-wrap gap-2">
            <UButton
              type="button"
              color="neutral"
              variant="outline"
              leading-icon="i-lucide-rotate-ccw"
              :loading="restoring"
              @click="restoreRecommended"
            >
              {{ t('admin.forum.settings.restoreRecommended') }}
            </UButton>
            <UButton type="submit" leading-icon="i-lucide-save" :loading="saving" :disabled="Boolean(validationKey)">
              {{ t('admin.common.save') }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </form>
</template>
