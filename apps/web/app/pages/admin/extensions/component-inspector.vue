<script setup lang="ts">
import { useAdminPage } from '~/composables/admin/useAdminPage'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  type ComponentCompositionInspectorSnapshot,
  useAdminCompositionInspectors
} from '~/composables/admin/useAdminCompositionInspectors'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionComponentInspector' })

const { t } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/extensions/component-inspector')
const { inspectComponents } = useAdminCompositionInspectors()

const limit = ref(100)
const pending = ref(true)
const error = ref('')
const snapshot = ref<ComponentCompositionInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

function formatCount(value: number) {
  return new Intl.NumberFormat().format(value)
}

function mapLoadError(cause: unknown) {
  const fromApi = apiErrorMessage(cause)
  if (fromApi) return fromApi
  if (cause instanceof Error && cause.message.trim()) return cause.message
  return t('admin.extensions.componentInspector.loadFailed')
}

function field(row: Record<string, unknown>, key: string) {
  const value = row[key]
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  return '—'
}

function outcomeColor(status: string) {
  if (status === 'succeeded' || status === 'composed' || status === 'ok') return 'primary'
  if (status === 'denied' || status === 'failed' || status === 'error') return 'error'
  if (status === 'fallback' || status === 'skipped') return 'warning'
  return 'neutral'
}

async function load(manual = false) {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspectComponents(limit.value)
    if (manual) {
      toast.add({
        color: 'primary',
        icon: 'i-lucide-check',
        title: t('admin.extensions.componentInspector.refreshSuccess'),
        duration: 10000
      })
    }
  } catch (cause) {
    snapshot.value = null
    error.value = mapLoadError(cause)
  } finally {
    pending.value = false
  }
}

watch(limit, () => void load())
useSeoMeta({ title: t('admin.extensions.componentInspector.metaTitle') })
void load()
</script>

<template>
  <div data-testid="admin-component-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-4 flex min-w-0 flex-col gap-1">
      <h2 class="flex min-w-0 items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <span class="min-w-0 truncate">{{ t('admin.extensions.componentInspector.title') }}</span>
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.componentInspector.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-4 rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <UIcon name="i-lucide-blocks" class="size-4 shrink-0" />
          <template v-if="snapshot">
            <UBadge color="neutral" variant="subtle">
              {{ t('admin.extensions.componentInspector.revisionBadge', { revision: snapshot.revision }) }}
            </UBadge>
            <span>{{ t('admin.extensions.componentInspector.targetCount', { count: snapshot.targetCount }) }}</span>
            <span>{{ t('admin.extensions.componentInspector.contributionCount', { count: snapshot.contributionCount }) }}</span>
            <span>{{ t('admin.extensions.componentInspector.traceCount', { count: snapshot.traces.length }) }}</span>
          </template>
        </div>
      </template>
      <template #right>
        <div class="flex min-w-0 items-center gap-2">
          <label class="text-xs text-slate-500 dark:text-zinc-400" for="component-inspector-limit">
            {{ t('admin.extensions.componentInspector.limit') }}
          </label>
          <USelect
            id="component-inspector-limit"
            v-model="limit"
            :items="limitItems"
            value-key="value"
            label-key="label"
            class="w-20"
            data-testid="component-inspector-limit"
          />
          <UButton
            icon="i-lucide-rotate-cw"
            color="neutral"
            variant="subtle"
            :loading="pending"
            data-testid="component-inspector-refresh"
            @click="load(true)"
          >
            {{ t('admin.extensions.refresh') }}
          </UButton>
        </div>
      </template>
    </UDashboardToolbar>

    <SFAlert
      v-if="error"
      variant="danger"
      :title="error"
      class="mb-4"
      data-testid="component-inspector-error"
    />

    <div v-if="pending" class="space-y-3" aria-busy="true" data-testid="component-inspector-loading">
      <USkeleton class="h-24 w-full rounded-lg" />
      <USkeleton class="h-44 w-full rounded-lg" />
      <USkeleton class="h-52 w-full rounded-lg" />
    </div>

    <template v-else-if="snapshot">
      <SFAlert
        v-if="snapshot.safeMode"
        variant="warning"
        :title="t('admin.extensions.componentInspector.safeModeTitle')"
        :description="t('admin.extensions.componentInspector.safeModeDescription')"
        class="mb-4"
        data-testid="component-inspector-safe-mode"
      />

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex min-w-0 flex-col gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.componentInspector.summaryTitle') }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.componentInspector.redactionHint') }}
          </p>
        </div>
        <dl class="grid grid-cols-2 divide-x divide-y divide-slate-200 dark:divide-zinc-800 sm:grid-cols-2 lg:grid-cols-4">
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.componentInspector.fields.revision') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.revision) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.componentInspector.fields.targets') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.targetCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.componentInspector.fields.contributions') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.contributionCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.componentInspector.fields.safeMode') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">
              {{ snapshot.safeMode ? t('admin.extensions.componentInspector.yes') : t('admin.extensions.componentInspector.no') }}
            </dd>
          </div>
        </dl>
      </section>

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.componentInspector.conflictsTitle') }}
          </h3>
          <UBadge color="neutral" variant="subtle">{{ snapshot.conflicts.length }}</UBadge>
        </div>
        <SFEmptyState
          v-if="snapshot.conflicts.length === 0"
          icon-label="CFL"
          :title="t('admin.extensions.componentInspector.conflictsEmptyTitle')"
          :description="t('admin.extensions.componentInspector.conflictsEmptyDescription')"
          class="m-6"
          data-testid="component-inspector-empty-conflicts"
        />
        <div v-else class="overflow-x-auto" data-testid="component-inspector-conflicts">
          <table class="min-w-[720px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.target') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.contract') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.explicit') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="(item, index) in snapshot.conflicts" :key="`${field(item, 'targetId')}-${index}`">
                <td class="break-all px-3 py-2 font-mono font-medium text-slate-800 dark:text-zinc-100">
                  {{ field(item, 'targetId') }}
                </td>
                <td class="break-all px-3 py-2 font-mono text-slate-600 dark:text-zinc-300">
                  {{ field(item, 'targetContractVersion') }}
                </td>
                <td class="px-3 py-2">
                  {{
                    item.explicitSelection === true
                      ? t('admin.extensions.componentInspector.yes')
                      : t('admin.extensions.componentInspector.no')
                  }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <div>
            <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.componentInspector.tracesTitle') }}
            </h3>
            <p class="mt-0.5 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.componentInspector.redactionHint') }}
            </p>
          </div>
          <UBadge color="neutral" variant="subtle">{{ snapshot.traces.length }}</UBadge>
        </div>
        <SFEmptyState
          v-if="snapshot.traces.length === 0"
          icon-label="TRC"
          :title="t('admin.extensions.componentInspector.tracesEmptyTitle')"
          :description="t('admin.extensions.componentInspector.tracesEmptyDescription')"
          class="m-6"
          data-testid="component-inspector-empty-traces"
        />
        <div v-else class="overflow-x-auto" data-testid="component-inspector-traces">
          <table class="min-w-[880px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.id') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.target') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.status') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.revision') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.componentInspector.fields.duration') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="(item, index) in snapshot.traces" :key="`${field(item, 'id')}-${index}`">
                <td class="break-all px-3 py-2 font-mono">{{ field(item, 'id') }}</td>
                <td class="break-all px-3 py-2 font-mono">{{ field(item, 'targetId') }}</td>
                <td class="px-3 py-2">
                  <UBadge :color="outcomeColor(field(item, 'status'))" variant="subtle">
                    {{ field(item, 'status') }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 font-mono">{{ field(item, 'revision') }}</td>
                <td class="px-3 py-2 font-mono">
                  {{
                    typeof item.durationMicros === 'number'
                      ? `${item.durationMicros} µs`
                      : '—'
                  }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>
