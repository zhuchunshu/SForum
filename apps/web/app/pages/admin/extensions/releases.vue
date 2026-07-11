<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  webReleaseCanRetry,
  webReleaseCanRollback,
  webReleaseProgress,
  type AdminWebReleaseStatus
} from '~/utils/adminWebReleases'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionReleases' })

const { t, te } = useI18n()
const { format: formatSiteDateTime } = useSiteDateTime()
const adminPage = useAdminPage('/extensions/releases')
const { data, pending, error, page, perPage, selected, commandId, load, select, command } = useAdminWebReleases()
const pages = computed(() => Math.max(1, Math.ceil(data.value.total / perPage)))

function closeDetail() {
  selected.value = null
}

function statusLabel(status: AdminWebReleaseStatus | string) {
  const key = `admin.extensions.releases.statusLabels.${status}`
  return te(key) ? t(key) : status
}
</script>

<template>
  <div class="mb-5 flex flex-wrap items-start justify-between gap-3">
    <div>
      <h2 class="flex items-center gap-2 text-xl font-bold">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)]" />
        {{ t('admin.extensions.releases.title') }}
      </h2>
      <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.releases.intro') }}
      </p>
    </div>
  </div>

  <UDashboardToolbar class="mb-5 rounded-lg border border-slate-200 bg-white px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
    <template #left>
      <span class="text-sm text-slate-500">{{ t('admin.extensions.releases.intro') }}</span>
    </template>
    <template #right>
      <UButton icon="i-lucide-rotate-cw" color="neutral" variant="subtle" :loading="pending" @click="load">
        {{ t('admin.extensions.refresh') }}
      </UButton>
    </template>
  </UDashboardToolbar>

  <UAlert
    v-if="error"
    color="error"
    icon="i-lucide-triangle-alert"
    :title="apiErrorMessage(error)"
    class="mb-4"
  />

  <div class="overflow-hidden border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <table class="w-full text-left text-sm">
      <thead class="bg-slate-50 text-xs text-slate-500 dark:bg-zinc-950">
        <tr>
          <th class="px-4 py-3">ID</th>
          <th class="px-4 py-3">{{ t('admin.extensions.releases.trigger') }}</th>
          <th class="px-4 py-3">{{ t('admin.extensions.releases.theme') }}</th>
          <th class="px-4 py-3">{{ t('admin.extensions.releases.status') }}</th>
          <th class="px-4 py-3 text-right">{{ t('admin.extensions.releases.actions') }}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
        <tr v-for="release in data.items" :key="release.id">
          <td class="px-4 py-3 font-mono">#{{ release.id }}</td>
          <td class="px-4 py-3">
            <span>{{ release.triggerKind }}</span>
            <span v-if="release.triggerExtensionId" class="mt-0.5 block text-xs text-slate-500">
              {{ release.triggerExtensionId }}
            </span>
          </td>
          <td class="px-4 py-3">
            {{ release.activeThemeId }}
            <span class="text-slate-500">v{{ release.themeVersion }}</span>
          </td>
          <td class="px-4 py-3">
            <UBadge variant="subtle">{{ statusLabel(release.status) }}</UBadge>
            <UProgress
              v-if="webReleaseProgress(release.status) < 100"
              class="mt-2 w-28"
              size="xs"
              :model-value="webReleaseProgress(release.status)"
            />
          </td>
          <td class="px-4 py-3">
            <div class="flex justify-end gap-1">
              <UButton
                size="xs"
                icon="i-lucide-scroll-text"
                color="neutral"
                variant="ghost"
                :aria-label="t('admin.extensions.releases.viewDetail')"
                @click="select(release.id)"
              >
                {{ t('admin.extensions.releases.viewDetail') }}
              </UButton>
              <UButton
                v-if="webReleaseCanRetry(release.status)"
                size="xs"
                icon="i-lucide-refresh-cw"
                :loading="commandId === release.id"
                @click="command(release.id, 'retry')"
              />
              <UButton
                v-if="webReleaseCanRollback(release.status)"
                size="xs"
                icon="i-lucide-undo-2"
                color="warning"
                variant="subtle"
                :loading="commandId === release.id"
                @click="command(release.id, 'rollback')"
              />
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-if="!data.items.length && !pending" class="p-10 text-center text-sm text-slate-500">
      {{ t('admin.extensions.releases.empty') }}
    </div>
  </div>

  <UPagination
    v-if="pages > 1"
    v-model:page="page"
    class="mt-4 justify-end"
    :total="data.total"
    :items-per-page="perPage"
  />

  <UModal :open="Boolean(selected)" @update:open="value => { if (!value) closeDetail() }">
    <template #content>
      <div v-if="selected" class="max-h-[80vh] overflow-y-auto p-5">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-base font-semibold">
              Release #{{ selected.id }}
            </h3>
            <p class="mt-1 text-xs text-slate-500">
              {{ statusLabel(selected.status) }}
              <span v-if="selected.triggerKind"> · {{ selected.triggerKind }}</span>
            </p>
          </div>
          <UButton icon="i-lucide-x" color="neutral" variant="ghost" @click="closeDetail" />
        </div>

        <UAlert
          v-if="selected.publicMessage"
          color="error"
          class="mt-4"
          :title="selected.publicReason"
          :description="selected.publicMessage"
        />

        <div class="mt-5">
          <h4 class="text-sm font-semibold text-slate-700 dark:text-zinc-200">
            {{ t('admin.extensions.releases.events') }}
          </h4>
          <div v-if="selected.events?.length" class="mt-3 space-y-3">
            <div
              v-for="event in selected.events"
              :key="event.id"
              class="border-l-2 border-slate-200 pl-3 text-sm dark:border-zinc-700"
            >
              <p class="font-medium">
                {{ statusLabel(event.nextStatus) }} · {{ event.reason }}
              </p>
              <p class="text-xs text-slate-500">
                {{ formatSiteDateTime(event.createdAt) }}
              </p>
              <p v-if="event.message" class="mt-1 text-xs">
                {{ event.message }}
              </p>
            </div>
          </div>
          <p v-else class="mt-2 text-xs text-slate-500">
            —
          </p>
        </div>

        <div class="mt-5">
          <h4 class="text-sm font-semibold text-slate-700 dark:text-zinc-200">
            {{ t('admin.extensions.releases.buildLog') }}
          </h4>
          <pre
            v-if="selected.buildLog?.trim()"
            class="mt-2 max-h-72 overflow-auto rounded-md bg-zinc-950 p-3 text-xs text-zinc-200"
          >{{ selected.buildLog }}</pre>
          <p v-else class="mt-2 text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.releases.emptyBuildLog') }}
          </p>
        </div>
      </div>
    </template>
  </UModal>
</template>
