<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import {
  createAdminForumApi,
  createDefaultForumSettings,
  forumSettingsPayload,
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

const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const forumApi = createAdminForumApi(request)
const adminPage = useAdminPage('/forum/settings')
const adminRoutes = useAdminRoutes()
const { can } = usePermissions()

const saving = ref(false)
const restoring = ref(false)
const savedSnapshot = ref('')
const form = reactive(createDefaultForumSettings())

const { data, pending, error, refresh } = await useAsyncData('admin-forum-settings', async () => {
  const [groups, settings] = await Promise.all([
    forumApi.listCategoryGroups(),
    forumApi.getSettings()
  ])
  applySettings(settings)
  return { groups, settings }
}, {
  default: () => ({ groups: [] as ForumCategoryGroup[], settings: createDefaultForumSettings() })
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
const canManagePagination = computed(() => can('settings.manage'))
const paginationError = computed(() => {
  const valid = [form.topicsPerPage, form.commentsPerPage]
    .every(value => Number.isInteger(value) && value >= 1 && value <= 100)
  return valid ? '' : t('admin.forum.settings.paginationRangeError')
})

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
  if (paginationError.value) {
    return
  }
  saving.value = true
  try {
    const payload = forumSettingsPayload(form)
    if (!canManagePagination.value) {
      delete payload.topicsPerPage
      delete payload.commentsPerPage
    }
    const updated = await forumApi.updateSettings(payload)
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

    <UCard
      class="border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900"
      :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }"
    >
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-bold text-slate-900 dark:text-white">
              {{ t('admin.forum.settings.formTitle') }}
            </h3>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.forum.settings.formHelp') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="font-mono">
            forum.*
          </UBadge>
        </div>
      </template>

      <div class="grid max-w-5xl gap-6">
        <section class="space-y-3">
          <!-- 无公开版块：空 select 对运营不友好，改为明确引导 -->
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
                class="rounded-lg border px-3 py-2 text-left text-sm transition"
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
            <UFormField :label="t('admin.forum.settings.topicsPerPage')" :error="paginationError" name="topics-per-page">
              <UInputNumber
                v-model="form.topicsPerPage"
                :min="1"
                :max="100"
                :disabled="!canManagePagination"
                class="w-full"
              />
            </UFormField>
            <UFormField :label="t('admin.forum.settings.commentsPerPage')" :error="paginationError" name="comments-per-page">
              <UInputNumber
                v-model="form.commentsPerPage"
                :min="1"
                :max="100"
                :disabled="!canManagePagination"
                class="w-full"
              />
            </UFormField>
          </div>
          <p v-if="!canManagePagination" class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.forum.settings.paginationPermissionHelp') }}
          </p>
        </section>

        <section class="space-y-3 border-t border-slate-200 pt-5 dark:border-zinc-800">
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
              class="rounded-lg border px-4 py-3 text-left transition"
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
              class="mt-1 size-4 rounded border-slate-300 text-[var(--sf-accent)] focus:ring-[var(--sf-accent)]"
            />
            <span>
              <span class="font-semibold text-slate-900 dark:text-zinc-100">
                {{ t('admin.forum.settings.publicTagPages') }}
              </span>
              <span class="mt-1 block text-xs leading-5 text-slate-500 dark:text-zinc-400">
                {{ t('admin.forum.settings.publicTagPagesHelp') }}
              </span>
            </span>
          </label>

          <UFormField :label="t('admin.forum.settings.maxTagsPerTopic')" name="max-tags">
            <UInput
              v-model.number="form.tagMaxPerTopic"
              icon="i-lucide-tags"
              type="number"
              inputmode="numeric"
              min="0"
              max="10"
              step="1"
              class="w-full"
            />
            <p class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
              {{ t('admin.forum.settings.maxTagsPerTopicHelp') }}
            </p>
          </UFormField>
        </section>

        <section class="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60 md:grid-cols-4">
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.defaultCategory') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.defaultCategorySlug }}</span>
          </div>
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.tagCreationMode') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.tagCreationMode }}</span>
          </div>
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.publicTagPages') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.tagPublicPages ? 'enabled' : 'disabled' }}</span>
          </div>
          <div>
            <span class="block text-xs font-medium text-slate-500 dark:text-zinc-400">{{ t('admin.forum.settings.maxTagsPerTopic') }}</span>
            <span class="mt-1 block font-mono text-slate-900 dark:text-zinc-100">{{ recommended.tagMaxPerTopic }}</span>
          </div>
        </section>
      </div>

      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <UAlert
            v-if="hasChanges"
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
            <UButton type="submit" leading-icon="i-lucide-save" :loading="saving">
              {{ t('admin.common.save') }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </form>
</template>
