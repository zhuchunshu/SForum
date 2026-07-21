<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  type NavigationInspectorSnapshot,
  useAdminCompositionInspectors
} from '~/composables/useAdminCompositionInspectors'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionNavigationInspector' })

const { t } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/extensions/navigation-inspector')
const { inspectNavigation } = useAdminCompositionInspectors()

const limit = ref(100)
const pending = ref(true)
const error = ref('')
const snapshot = ref<NavigationInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

function shortDigest(value: string) {
  return value.length > 16 ? `${value.slice(0, 12)}…${value.slice(-4)}` : value
}

function formatCount(value: number) {
  return new Intl.NumberFormat().format(value)
}

function mapLoadError(cause: unknown) {
  // plain Error（composable 包装后）没有 API envelope，必须读 message，否则页面空白。
  const fromApi = apiErrorMessage(cause)
  if (fromApi) return fromApi
  if (cause instanceof Error && cause.message.trim()) return cause.message
  return t('admin.extensions.navigationInspector.loadFailed')
}

function traceField(trace: Record<string, unknown>, key: string) {
  const value = trace[key]
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  return '—'
}

function traceArtifactOwner(trace: Record<string, unknown>) {
  const artifact = trace.artifact
  if (artifact && typeof artifact === 'object' && !Array.isArray(artifact)) {
    const extensionId = (artifact as Record<string, unknown>).extensionId
    if (typeof extensionId === 'string' && extensionId) return extensionId
  }
  return t('admin.extensions.navigationInspector.host')
}

function outcomeColor(outcome: string) {
  if (outcome === 'succeeded' || outcome === 'composed') return 'primary'
  if (outcome === 'denied' || outcome === 'failed_closed' || outcome === 'error') return 'error'
  if (outcome === 'fallback' || outcome === 'skipped') return 'warning'
  return 'neutral'
}

async function load(manual = false) {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspectNavigation(limit.value)
    if (manual) {
      toast.add({
        color: 'primary',
        icon: 'i-lucide-check',
        title: t('admin.extensions.navigationInspector.refreshSuccess'),
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
useSeoMeta({ title: t('admin.extensions.navigationInspector.metaTitle') })
void load()
</script>

<template>
  <div data-testid="admin-navigation-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-4 flex min-w-0 flex-col gap-1">
      <h2 class="flex min-w-0 items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <span class="min-w-0 truncate">{{ t('admin.extensions.navigationInspector.title') }}</span>
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.navigationInspector.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-4 rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <UIcon name="i-lucide-compass" class="size-4 shrink-0" />
          <template v-if="snapshot">
            <UBadge color="neutral" variant="subtle">
              {{ t('admin.extensions.navigationInspector.revisionBadge', { revision: snapshot.revision }) }}
            </UBadge>
            <span>{{ t('admin.extensions.navigationInspector.navigationCount', { count: snapshot.navigationCount }) }}</span>
            <span>{{ t('admin.extensions.navigationInspector.regionCount', { count: snapshot.regionCount }) }}</span>
            <span>{{ t('admin.extensions.navigationInspector.traceCount', { count: snapshot.traces.length }) }}</span>
          </template>
        </div>
      </template>
      <template #right>
        <div class="flex min-w-0 items-center gap-2">
          <label class="text-xs text-slate-500 dark:text-zinc-400" for="navigation-inspector-limit">
            {{ t('admin.extensions.navigationInspector.limit') }}
          </label>
          <USelect
            id="navigation-inspector-limit"
            v-model="limit"
            :items="limitItems"
            value-key="value"
            label-key="label"
            class="w-20"
            data-testid="navigation-inspector-limit"
          />
          <UButton
            icon="i-lucide-rotate-cw"
            color="neutral"
            variant="subtle"
            :loading="pending"
            data-testid="navigation-inspector-refresh"
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
      data-testid="navigation-inspector-error"
    />

    <div v-if="pending" class="space-y-3" aria-busy="true" data-testid="navigation-inspector-loading">
      <USkeleton class="h-24 w-full rounded-lg" />
      <USkeleton class="h-44 w-full rounded-lg" />
      <USkeleton class="h-52 w-full rounded-lg" />
    </div>

    <template v-else-if="snapshot">
      <SFAlert
        v-if="snapshot.safeMode"
        variant="warning"
        :title="t('admin.extensions.navigationInspector.safeModeTitle')"
        :description="t('admin.extensions.navigationInspector.safeModeDescription')"
        class="mb-4"
        data-testid="navigation-inspector-safe-mode"
      />

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex min-w-0 flex-col gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.navigationInspector.summaryTitle') }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.redactionHint') }}
          </p>
        </div>
        <dl class="grid grid-cols-2 divide-x divide-y divide-slate-200 dark:divide-zinc-800 sm:grid-cols-3 lg:grid-cols-6">
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.navigationInspector.fields.revision') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.revision) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.navigationInspector.fields.navigation') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.navigationCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.navigationInspector.fields.regions') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.regionCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.navigationInspector.fields.conflicts') }}</dt>
            <dd
              class="mt-1 text-lg font-semibold"
              :class="snapshot.providerConflicts ? 'text-red-700 dark:text-red-300' : 'text-slate-900 dark:text-zinc-100'"
            >
              {{ formatCount(snapshot.providerConflicts) }}
            </dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.navigationInspector.fields.safeMode') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">
              {{ snapshot.safeMode ? t('admin.extensions.navigationInspector.yes') : t('admin.extensions.navigationInspector.no') }}
            </dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.navigationInspector.fields.digest') }}</dt>
            <dd
              class="mt-1 break-all font-mono text-sm font-semibold text-slate-900 dark:text-zinc-100"
              :title="snapshot.digest || undefined"
            >
              {{ snapshot.digest ? shortDigest(snapshot.digest) : '—' }}
            </dd>
          </div>
        </dl>
      </section>

      <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <div>
            <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.navigationInspector.tracesTitle') }}
            </h3>
            <p class="mt-0.5 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.navigationInspector.redactionHint') }}
            </p>
          </div>
          <UBadge color="neutral" variant="subtle">{{ snapshot.traces.length }}</UBadge>
        </div>
        <SFEmptyState
          v-if="snapshot.traces.length === 0"
          icon-label="NAV"
          :title="t('admin.extensions.navigationInspector.tracesEmptyTitle')"
          :description="t('admin.extensions.navigationInspector.tracesEmptyDescription')"
          class="m-6"
          data-testid="navigation-inspector-empty-traces"
        />
        <div v-else class="overflow-x-auto" data-testid="navigation-inspector-traces">
          <table class="min-w-[960px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.sequence') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.family') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.target') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.action') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.outcome') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.owner') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.navigationInspector.fields.duration') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="(item, index) in snapshot.traces" :key="`${traceField(item, 'sequence')}-${index}`">
                <td class="px-3 py-2 font-mono">#{{ traceField(item, 'sequence') }}</td>
                <td class="break-all px-3 py-2 font-mono">{{ traceField(item, 'family') }}</td>
                <td class="break-all px-3 py-2 font-mono">{{ traceField(item, 'targetId') }}</td>
                <td class="break-all px-3 py-2 font-mono">{{ traceField(item, 'action') }}</td>
                <td class="px-3 py-2">
                  <UBadge :color="outcomeColor(traceField(item, 'outcome'))" variant="subtle">
                    {{ traceField(item, 'outcome') }}
                  </UBadge>
                </td>
                <td class="break-all px-3 py-2">{{ traceArtifactOwner(item) }}</td>
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
