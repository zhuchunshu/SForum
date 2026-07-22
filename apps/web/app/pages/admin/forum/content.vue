<script setup lang="ts">
import type {
  AdminForumContentDetail,
  AdminForumContentFilters,
  AdminForumContentKind,
  AdminForumContentRow,
  AdminForumTopicDetail
} from '~/utils/adminForumContent'
import { useAdminPage } from '~/composables/useAdminPage'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminForumContent' })

const { t } = useI18n()
const { can, user } = usePermissions()
const adminPage = useAdminPage('/forum/content')
const contentApi = useAdminForumContent()
const { format: formatSiteDateTime } = useSiteDateTime()

const canTopics = computed(() => can('topic.edit_any') || can('topic.revision.view_any'))
const canComments = computed(() => can('post.edit_any') || can('post.revision.view_any'))
const canEditTopics = computed(() => can('topic.edit_any'))
const canEditComments = computed(() => can('post.edit_any'))
const canViewTopicHistory = computed(() => can('topic.revision.view_any'))
const canViewCommentHistory = computed(() => can('post.revision.view_any'))
const availableTabs = computed<AdminForumContentKind[]>(() => [
  ...(canTopics.value ? ['topics' as const] : []),
  ...(canComments.value ? ['comments' as const] : [])
])
const activeTab = ref<AdminForumContentKind>(canTopics.value ? 'topics' : 'comments')
const filters = reactive<AdminForumContentFilters>({ perPage: 20 })
const list = ref<AdminForumContentRow[]>([])
const pending = ref(false)
const errorMessage = ref('')
const cursors = ref<string[]>([''])
const cursorIndex = ref(0)
const hasMore = ref(false)
const nextCursor = ref('')
const selected = ref<AdminForumContentDetail | null>(null)
const detailPending = ref(false)
const detailError = ref('')
const staffReason = ref('')
const conflict = ref<AdminForumContentDetail | null>(null)

const selectedIsDeleted = computed(() => selected.value?.status === 'deleted')
const selectedCanEdit = computed(() => {
  if (!selected.value || selectedIsDeleted.value) return false
  return selected.value.targetType === 'topic' ? canEditTopics.value : canEditComments.value
})
const selectedCanViewHistory = computed(() => selected.value?.targetType === 'topic'
  ? canViewTopicHistory.value
  : canViewCommentHistory.value)
const editingAnotherAuthor = computed(() => Boolean(selected.value && selected.value.authorUserId !== user.value?.id))
const requiresReason = computed(() => selectedCanEdit.value && editingAnotherAuthor.value)
const pageNumber = computed(() => cursorIndex.value + 1)

const statusOptions = computed(() => [
  { label: t('admin.forum.content.allStatuses'), value: '' },
  ...['active', 'locked', 'pending', 'rejected', 'hidden', 'deleted'].map(value => ({
    label: t(`admin.forum.content.status.${value}`), value
  }))
])
const perPageOptions = [20, 50, 100].map(value => ({ label: `${value}`, value }))

function isCurrentTabAllowed() {
  return activeTab.value === 'topics' ? canTopics.value : canComments.value
}

async function load() {
  if (!isCurrentTabAllowed()) {
    list.value = []
    return
  }
  pending.value = true
  errorMessage.value = ''
  try {
    const response = await contentApi.list(activeTab.value, requestFilters(), cursors.value[cursorIndex.value])
    list.value = response.items
    hasMore.value = response.hasMore
    nextCursor.value = response.nextCursor || ''
  } catch (cause) {
    list.value = []
    errorMessage.value = apiErrorMessage(cause) || t('admin.forum.content.loadFailed')
  } finally {
    pending.value = false
  }
}

function requestFilters(): AdminForumContentFilters {
  return {
    ...filters,
    updatedFrom: normalizeFilterDate(filters.updatedFrom),
    updatedTo: normalizeFilterDate(filters.updatedTo)
  }
}

function normalizeFilterDate(value?: string) {
  if (!value) return ''
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toISOString()
}

function resetPagination() {
  cursors.value = ['']
  cursorIndex.value = 0
}

async function applyFilters() {
  resetPagination()
  await load()
}

async function resetFilters() {
  Object.assign(filters, { status: '', authorUserID: undefined, authorUsername: '', updatedFrom: '', updatedTo: '', topicID: undefined, titlePrefix: '', categorySlug: '', perPage: 20 })
  await applyFilters()
}

async function selectTab(tab: AdminForumContentKind) {
  if (activeTab.value === tab || (tab === 'topics' ? !canTopics.value : !canComments.value)) return
  activeTab.value = tab
  selected.value = null
  conflict.value = null
  resetPagination()
  await load()
}

async function nextPage() {
  if (!hasMore.value || !nextCursor.value) return
  cursors.value = [...cursors.value.slice(0, cursorIndex.value + 1), nextCursor.value]
  cursorIndex.value += 1
  await load()
}

async function previousPage() {
  if (cursorIndex.value === 0) return
  cursorIndex.value -= 1
  await load()
}

async function openDetail(row: AdminForumContentRow) {
  detailPending.value = true
  detailError.value = ''
  selected.value = null
  conflict.value = null
  staffReason.value = ''
  try {
    selected.value = row.targetType === 'topic'
      ? await contentApi.getTopic(row.id)
      : await contentApi.getComment(row.id)
  } catch (cause) {
    detailError.value = apiErrorMessage(cause) || t('admin.forum.content.detailLoadFailed')
  } finally {
    detailPending.value = false
  }
}

function closeDetail() {
  selected.value = null
  detailError.value = ''
  conflict.value = null
  staffReason.value = ''
}

function showConflict() {
  conflict.value = selected.value
}

async function reloadLatest() {
  if (!conflict.value) return
  const target = conflict.value
  conflict.value = null
  await openDetail(target)
  await load()
}

// M6 owns the timeline and raw revision detail. Keep the M5 conflict action
// explicit without starting a timeline, diff, restore, or redaction workflow.
function viewHistory() {
  conflict.value = null
  detailError.value = t('admin.forum.content.historyAvailable')
}

async function saved() {
  if (selected.value) await openDetail(selected.value)
  await load()
}

watch(availableTabs, (tabs) => {
  if (!tabs.includes(activeTab.value) && tabs[0]) void selectTab(tabs[0])
}, { immediate: true })

void load()
useSeoMeta({ title: t('admin.forum.content.metaTitle') })
</script>

<template>
  <div data-testid="admin-forum-content" class="min-w-0 w-full space-y-4">
    <header class="flex flex-col gap-1">
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.forum.content.title') }}
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.intro') }}</p>
    </header>

    <UAlert
      v-if="!availableTabs.length"
      color="warning"
      variant="soft"
      icon="i-lucide-shield-alert"
      :title="t('admin.forum.content.permissionDenied')"
    />

    <template v-else>
      <UDashboardToolbar class="rounded-lg border border-slate-200 bg-white px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
        <template #left>
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              v-for="tab in availableTabs"
              :key="tab"
              size="sm"
              :color="activeTab === tab ? 'primary' : 'neutral'"
              :variant="activeTab === tab ? 'solid' : 'outline'"
              @click="selectTab(tab)"
            >
              {{ t(`admin.forum.content.tabs.${tab}`) }}
            </UButton>
          </div>
        </template>
        <template #right>
          <UButton icon="i-lucide-refresh-cw" color="neutral" variant="subtle" :loading="pending" @click="load">
            {{ t('admin.common.refresh') }}
          </UButton>
        </template>
      </UDashboardToolbar>

      <section class="grid gap-3 rounded-lg border border-slate-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900 md:grid-cols-2 xl:grid-cols-4">
        <UFormField :label="t('admin.forum.content.filters.status')">
          <USelect v-model="filters.status" :items="statusOptions" class="w-full" />
        </UFormField>
        <UFormField :label="t('admin.forum.content.filters.author')">
          <UInput v-model="filters.authorUsername" :placeholder="t('admin.forum.content.filters.authorPlaceholder')" class="w-full" />
        </UFormField>
        <UFormField :label="t('admin.forum.content.filters.authorId')">
          <UInput v-model.number="filters.authorUserID" type="number" min="1" class="w-full" />
        </UFormField>
        <UFormField v-if="activeTab === 'topics'" :label="t('admin.forum.content.filters.titlePrefix')">
          <UInput v-model="filters.titlePrefix" class="w-full" />
        </UFormField>
        <UFormField v-if="activeTab === 'topics'" :label="t('admin.forum.content.filters.categorySlug')">
          <UInput v-model="filters.categorySlug" class="w-full" />
        </UFormField>
        <UFormField v-else :label="t('admin.forum.content.filters.topicId')">
          <UInput v-model.number="filters.topicID" type="number" min="1" class="w-full" />
        </UFormField>
        <UFormField :label="t('admin.forum.content.filters.updatedFrom')">
          <UInput v-model="filters.updatedFrom" type="datetime-local" class="w-full" />
        </UFormField>
        <UFormField :label="t('admin.forum.content.filters.updatedTo')">
          <UInput v-model="filters.updatedTo" type="datetime-local" class="w-full" />
        </UFormField>
        <UFormField :label="t('admin.forum.content.filters.perPage')">
          <USelect v-model="filters.perPage" :items="perPageOptions" class="w-full" />
        </UFormField>
        <div class="flex flex-wrap items-end gap-2">
          <UButton icon="i-lucide-filter" @click="applyFilters">{{ t('admin.forum.content.applyFilters') }}</UButton>
          <UButton icon="i-lucide-rotate-ccw" color="neutral" variant="outline" @click="resetFilters">{{ t('admin.common.reset') }}</UButton>
        </div>
      </section>

      <UAlert v-if="errorMessage" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="errorMessage" />

      <div class="overflow-x-auto rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <table class="w-full min-w-[760px] text-left text-sm">
          <thead class="border-b border-slate-200 bg-slate-50 text-xs text-slate-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
            <tr>
              <th class="px-3 py-3 font-medium">{{ t('admin.forum.content.columns.content') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('admin.forum.content.columns.author') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('admin.forum.content.columns.status') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('admin.forum.content.columns.revision') }}</th>
              <th class="px-3 py-3 font-medium">{{ t('admin.forum.content.columns.updatedAt') }}</th>
              <th class="px-3 py-3 text-right font-medium">{{ t('admin.forum.content.columns.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="pending" v-for="index in 5" :key="`skeleton-${index}`" class="border-b border-slate-100 dark:border-zinc-800">
              <td v-for="cell in 6" :key="cell" class="px-3 py-3"><USkeleton class="h-4 w-full" /></td>
            </tr>
            <tr v-else-if="!list.length">
              <td colspan="6" class="px-4 py-12 text-center text-sm text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.empty') }}</td>
            </tr>
            <tr v-for="row in list" :key="`${row.targetType}-${row.id}`" class="border-b border-slate-100 last:border-0 dark:border-zinc-800">
              <td class="max-w-md px-3 py-3">
                <button type="button" class="block max-w-full text-left" @click="openDetail(row)">
                  <span class="block truncate font-medium text-slate-900 hover:text-[var(--sf-accent)] dark:text-zinc-100">{{ row.title || row.topicTitle || `#${row.id}` }}</span>
                  <span class="mt-0.5 block truncate text-xs text-slate-500 dark:text-zinc-400">{{ row.excerpt || t('admin.forum.content.noExcerpt') }}</span>
                </button>
              </td>
              <td class="px-3 py-3 text-xs text-slate-600 dark:text-zinc-300">{{ row.author?.displayName || row.author?.username || `#${row.authorUserId}` }}</td>
              <td class="px-3 py-3"><UBadge color="neutral" variant="soft">{{ t(`admin.forum.content.status.${row.status}`) }}</UBadge></td>
              <td class="px-3 py-3 tabular-nums">{{ row.currentRevision }}</td>
              <td class="px-3 py-3 whitespace-nowrap text-xs text-slate-500 dark:text-zinc-400">{{ formatSiteDateTime(row.updatedAt) }}</td>
              <td class="px-3 py-3 text-right">
                <div class="flex justify-end gap-1">
                  <UTooltip :text="t('admin.forum.content.inspect')"><UButton icon="i-lucide-eye" color="neutral" variant="ghost" size="sm" :aria-label="t('admin.forum.content.inspect')" @click="openDetail(row)" /></UTooltip>
                  <UTooltip v-if="row.targetType === 'topic' ? canEditTopics : canEditComments" :text="t('admin.common.edit')"><UButton icon="i-lucide-pencil" color="neutral" variant="ghost" size="sm" :aria-label="t('admin.common.edit')" @click="openDetail(row)" /></UTooltip>
                  <UTooltip v-if="row.targetType === 'topic' ? canViewTopicHistory : canViewCommentHistory" :text="t('admin.forum.content.historyDeferred')"><UButton icon="i-lucide-history" color="neutral" variant="ghost" size="sm" :aria-label="t('admin.forum.content.historyDeferred')" @click="openDetail(row)" /></UTooltip>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="flex items-center justify-between gap-3">
        <span class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.page', { page: pageNumber }) }}</span>
        <div class="flex gap-2"><UButton color="neutral" variant="outline" :disabled="cursorIndex === 0 || pending" @click="previousPage">{{ t('admin.forum.content.previous') }}</UButton><UButton color="neutral" variant="outline" :disabled="!hasMore || pending" @click="nextPage">{{ t('admin.forum.content.next') }}</UButton></div>
      </div>
    </template>

    <USlideover :open="Boolean(selected || detailPending || detailError)" :ui="{ content: 'w-full max-w-3xl' }" @update:open="value => !value && closeDetail()">
      <template #content>
        <div class="flex h-full min-w-0 flex-col">
          <header class="flex items-start justify-between gap-3 border-b border-slate-200 px-5 py-4 dark:border-zinc-800">
            <div class="min-w-0"><h3 class="truncate text-base font-semibold text-slate-900 dark:text-zinc-100">{{ selected?.title || selected?.topicTitle || t('admin.forum.content.detailTitle') }}</h3><p v-if="selected" class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.authorLine', { author: selected.author?.displayName || selected.author?.username || `#${selected.authorUserId}`, revision: selected.currentRevision }) }}</p></div>
            <UButton icon="i-lucide-x" color="neutral" variant="ghost" :aria-label="t('common.close')" @click="closeDetail" />
          </header>
          <main class="min-h-0 flex-1 overflow-y-auto p-5">
            <div v-if="detailPending" class="space-y-4"><USkeleton class="h-6 w-1/2" /><USkeleton class="h-32 w-full" /><USkeleton class="h-64 w-full" /></div>
            <UAlert v-else-if="detailError" color="error" variant="soft" icon="i-lucide-triangle-alert" :title="detailError" />
            <template v-else-if="selected">
              <UAlert v-if="conflict" color="error" variant="soft" icon="i-lucide-refresh-cw" :title="t('admin.forum.content.conflictTitle')" :description="t('admin.forum.content.conflictDescription')" class="mb-4">
                <template #actions><UButton size="sm" color="neutral" variant="outline" @click="reloadLatest">{{ t('admin.forum.content.reloadLatest') }}</UButton><UButton v-if="selectedCanViewHistory" size="sm" color="neutral" variant="ghost" @click="viewHistory">{{ t('admin.forum.content.viewHistory') }}</UButton></template>
              </UAlert>
              <UAlert v-if="editingAnotherAuthor && selectedCanEdit" color="warning" variant="soft" icon="i-lucide-user-round-pen" :title="t('admin.forum.content.editingAnotherAuthor')" class="mb-4" />
              <UAlert v-if="selectedIsDeleted" color="neutral" variant="soft" icon="i-lucide-lock" :title="t('admin.forum.content.deletedInspectionOnly')" class="mb-4" />
              <div v-if="selectedCanEdit" class="mb-4 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.forum.content.canonicalSaveHint') }}</div>
              <SFTopicEditor
                v-if="selected.targetType === 'topic' && selectedCanEdit"
                :topic="selected as AdminForumTopicDetail"
                v-model:staff-reason="staffReason"
                :require-staff-reason="requiresReason"
                :editing-another-author="editingAnotherAuthor"
                @saved="saved"
                @conflict="showConflict"
                @cancel="closeDetail"
              />
              <SFAdminForumCommentEditor
                v-else-if="selected.targetType === 'comment' && selectedCanEdit"
                :comment="selected"
                v-model:reason="staffReason"
                :require-reason="requiresReason"
                @saved="saved"
                @conflict="showConflict"
              />
              <div v-else class="space-y-3"><p class="whitespace-pre-wrap break-words text-sm text-slate-700 dark:text-zinc-200">{{ selected.content.rawContent }}</p></div>
            </template>
          </main>
        </div>
      </template>
    </USlideover>
  </div>
</template>
