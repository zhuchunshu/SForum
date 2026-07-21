<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useAdminPage } from '~/composables/useAdminPage'
import { createAdminForumApi, type ReindexRun, type ReindexStatus } from '~/utils/adminForum'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSearch'
})

const { format: formatSiteDateTime } = useSiteDateTime()
const { t } = useI18n()
const toast = useToast()
const { request } = useApiClient()
const adminPage = useAdminPage('/search')
const adminRoutes = useAdminRoutes()

const api = createAdminForumApi(request)

// 当前重建进度。null 表示尚无记录。
const currentStatus = ref<ReindexStatus | null>(null)
// 历史 run 列表。
const runs = ref<ReindexRun[]>([])
// 触发重建中的 loading。
const rebuilding = ref(false)
// 确认弹窗。
const confirmOpen = ref(false)

const isRunning = computed(() => currentStatus.value?.status === 'running')

// 拉取当前进度 + 历史。
async function refresh() {
  try {
    currentStatus.value = await api.getReindexStatus()
  } catch (error: any) {
    // 404 表示尚无 run，置空即可。
    currentStatus.value = null
  }
  try {
    runs.value = await api.listReindexRuns()
  } catch {
    // 历史拉取失败不阻断。
  }
}

// 触发重建。
async function triggerReindex() {
  rebuilding.value = true
  try {
    await api.reindexSearch()
    toast.add({ color: 'success', title: t('admin.search.toast.started'), icon: 'i-lucide-check' })
    await refresh()
    startPolling()
  } catch (error) {
    const msg = apiErrorMessage(error)
    if (msg?.includes('reindex_running')) {
      toast.add({ color: 'warning', title: t('admin.search.toast.alreadyRunning'), icon: 'i-lucide-alert-triangle' })
    } else {
      toast.add({ color: 'error', title: msg || t('admin.search.toast.failed'), icon: 'i-lucide-x' })
    }
  } finally {
    rebuilding.value = false
  }
}

// 轮询：running 时每 2s 刷新进度；完成/失败时停止。
let pollTimer: ReturnType<typeof setInterval> | null = null
function startPolling() {
  if (pollTimer || !import.meta.client) return
  pollTimer = setInterval(async () => {
    await refresh()
    if (currentStatus.value && currentStatus.value.status !== 'running') {
      stopPolling()
      // 完成时给一次 toast。
      if (currentStatus.value.status === 'completed') {
        toast.add({ color: 'success', title: t('admin.search.toast.completed'), icon: 'i-lucide-check' })
      }
    }
  }, 2000)
}
function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 状态徽标颜色。
function statusColor(status: string): string {
  switch (status) {
    case 'running': return 'info'
    case 'completed': return 'success'
    case 'failed': return 'error'
    default: return 'neutral'
  }
}

// 按站点时区与日期时间格式展示。
function formatTime(value?: string | null): string {
  if (!value) return '-'
  return formatSiteDateTime(value) || '-'
}

// 初始化。
await useAsyncData('admin-search-init', async () => {
  await refresh()
  return true
})

onMounted(() => {
  if (isRunning.value) startPolling()
})
onBeforeUnmount(stopPolling)

watch(isRunning, (running) => {
  if (running) startPolling()
  else stopPolling()
})
</script>

<template>
  <div class="space-y-6">
    <UDashboardToolbar class="mb-6 rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
      <span class="text-base font-semibold text-slate-800 dark:text-slate-100">{{ t('admin.search.title') }}</span>
    </UDashboardToolbar>
    <SFCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-search" class="size-5 text-primary-500" />
          <h2 class="text-base font-semibold text-slate-800 dark:text-slate-100">
            {{ t('admin.search.title') }}
          </h2>
        </div>
      </template>
      <p class="text-sm text-slate-500 dark:text-slate-400">
        {{ t('admin.search.description') }}
      </p>
      <p class="mt-2 text-xs text-slate-500 dark:text-slate-400">
        {{ t('admin.search.providerHint') }}
        <NuxtLink
          :to="adminRoutes.path('/forum/settings') + '?tab=search'"
          class="ml-1 font-medium text-[var(--sf-accent)] hover:underline"
        >
          {{ t('admin.search.providerLink') }}
        </NuxtLink>
      </p>

      <!-- 当前重建进度卡片 -->
      <div v-if="currentStatus" class="mt-5 rounded-lg border border-slate-200 p-4 dark:border-slate-700">
        <div class="mb-3 flex items-center justify-between">
          <span class="text-sm font-medium text-slate-700 dark:text-slate-200">
            {{ t('admin.search.current.title') }}
          </span>
          <SFBadge :color="statusColor(currentStatus.status)">
            {{ t(`admin.search.current.status.${currentStatus.status}`) }}
          </SFBadge>
        </div>
        <SFProgress :value="currentStatus.percent" />
        <div class="mt-3 grid grid-cols-3 gap-4 text-center">
          <div>
            <div class="text-xs text-slate-400">{{ t('admin.search.current.processed') }}</div>
            <div class="text-lg font-semibold text-slate-700 dark:text-slate-200">{{ currentStatus.processed }}</div>
          </div>
          <div>
            <div class="text-xs text-slate-400">{{ t('admin.search.current.remaining') }}</div>
            <div class="text-lg font-semibold text-slate-700 dark:text-slate-200">{{ currentStatus.remaining }}</div>
          </div>
          <div>
            <div class="text-xs text-slate-400">{{ t('admin.search.current.total') }}</div>
            <div class="text-lg font-semibold text-slate-700 dark:text-slate-200">{{ currentStatus.total }}</div>
          </div>
        </div>
        <div class="mt-3 flex justify-between text-xs text-slate-400">
          <span>{{ t('admin.search.current.startedAt') }}: {{ formatTime(currentStatus.startedAt) }}</span>
          <span v-if="currentStatus.finishedAt">{{ t('admin.search.current.finishedAt') }}: {{ formatTime(currentStatus.finishedAt) }}</span>
        </div>
        <p v-if="currentStatus.error" class="mt-2 text-xs text-red-500">{{ currentStatus.error }}</p>
      </div>

      <!-- 重建按钮 -->
      <div class="mt-5 flex justify-end">
        <SFButton
          color="primary"
          :icon="isRunning ? 'i-lucide-loader-2' : 'i-lucide-refresh-cw'"
          :loading="rebuilding || isRunning"
          :disabled="rebuilding || isRunning"
          @click="confirmOpen = true"
        >
          {{ isRunning ? t('admin.search.action.reindexing') : t('admin.search.action.reindex') }}
        </SFButton>
      </div>
    </SFCard>

    <!-- 历史记录 -->
    <SFCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-history" class="size-5 text-slate-500" />
          <h2 class="text-base font-semibold text-slate-800 dark:text-slate-100">
            {{ t('admin.search.history.title') }}
          </h2>
        </div>
      </template>
      <div v-if="runs.length === 0" class="py-6 text-center text-sm text-slate-400">
        {{ t('admin.search.history.empty') }}
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="border-b border-slate-200 text-left text-xs text-slate-400 dark:border-slate-700">
            <th class="py-2 pr-3">{{ t('admin.search.history.columns.id') }}</th>
            <th class="py-2 pr-3">{{ t('admin.search.history.columns.status') }}</th>
            <th class="py-2 pr-3">{{ t('admin.search.history.columns.total') }}</th>
            <th class="py-2 pr-3">{{ t('admin.search.history.columns.startedAt') }}</th>
            <th class="py-2">{{ t('admin.search.history.columns.finishedAt') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="run in runs" :key="run.id" class="border-b border-slate-100 dark:border-slate-800">
            <td class="py-2 pr-3 text-slate-500">#{{ run.id }}</td>
            <td class="py-2 pr-3"><SFBadge :color="statusColor(run.status)">{{ t(`admin.search.current.status.${run.status}`) }}</SFBadge></td>
            <td class="py-2 pr-3 text-slate-600 dark:text-slate-300">{{ run.total }}</td>
            <td class="py-2 pr-3 text-slate-500">{{ formatTime(run.startedAt) }}</td>
            <td class="py-2 text-slate-500">{{ formatTime(run.finishedAt) }}</td>
          </tr>
        </tbody>
      </table>
    </SFCard>

    <!-- 确认弹窗 -->
    <UModal v-model:open="confirmOpen">
      <template #content>
        <div class="space-y-4 p-6">
          <div class="flex items-start gap-3">
            <UIcon name="i-lucide-alert-triangle" class="mt-0.5 size-5 text-amber-500" />
            <div>
              <h3 class="text-base font-semibold text-slate-800 dark:text-slate-100">{{ t('admin.search.action.confirmTitle') }}</h3>
              <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">{{ t('admin.search.action.confirmDescription') }}</p>
            </div>
          </div>
          <div class="flex justify-end gap-2">
            <SFButton color="neutral" variant="ghost" @click="confirmOpen = false">{{ t('admin.search.action.cancel') }}</SFButton>
            <SFButton color="primary" @click="confirmOpen = false; triggerReindex()">{{ t('admin.search.action.confirm') }}</SFButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>
