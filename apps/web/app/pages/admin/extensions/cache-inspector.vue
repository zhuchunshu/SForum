<script setup lang="ts">
import { useAdminPage } from '~/composables/admin/useAdminPage'
import { useAdminCacheInspector } from '~/composables/admin/useAdminCacheInspector'
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  cacheInspectorErrorKind,
  formatCacheDuration,
  type CacheInspectorMetrics,
  type CacheInspectorOutcome,
  type CacheInspectorPolicy,
  type CacheInspectorSnapshot,
  type CacheInspectorTrace
} from '~/composables/admin/useAdminCacheInspector'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionCacheInspector' })

const { t, te } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/extensions/cache-inspector')
const { inspect } = useAdminCacheInspector()

const limit = ref(100)
const pending = ref(true)
const error = ref('')
const snapshot = ref<CacheInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

const failures = computed(() => {
  const metrics = snapshot.value?.metrics
  return metrics
    ? metrics.denied + metrics.stale + metrics.conflicts + metrics.errors + metrics.deadlines
    : 0
})

const hitRate = computed(() => {
  const metrics = snapshot.value?.metrics
  const reads = (metrics?.hits || 0) + (metrics?.misses || 0)
  return reads ? `${Math.round(((metrics?.hits || 0) / reads) * 100)}%` : '—'
})

function i18nOrRaw(key: string, fallback: string) {
  return te(key) ? t(key) : fallback
}

function formatCount(value: number) {
  return new Intl.NumberFormat().format(value)
}

function shortDigest(value: string) {
  return value.length > 16 ? `${value.slice(0, 12)}…${value.slice(-4)}` : value
}

function outcomeColor(outcome: CacheInspectorOutcome) {
  if (outcome === 'hit' || outcome === 'allowed') return 'primary'
  if (outcome === 'denied' || outcome === 'error' || outcome === 'deadline') return 'error'
  if (outcome === 'stale' || outcome === 'conflict') return 'warning'
  return 'neutral'
}

function policyColor(policy: CacheInspectorPolicy) {
  if (policy === 'public') return 'primary'
  if (policy === 'permission') return 'warning'
  return 'neutral'
}

function operationLabel(operation: string) {
  return i18nOrRaw(`admin.extensions.cacheInspector.operation.${operation}`, operation)
}

function outcomeLabel(outcome: string) {
  return i18nOrRaw(`admin.extensions.cacheInspector.outcome.${outcome}`, outcome)
}

function policyLabel(policy: string) {
  return i18nOrRaw(`admin.extensions.cacheInspector.policy.${policy}`, policy)
}

function traceOwner(trace: CacheInspectorTrace) {
  return trace.extensionId || t('admin.extensions.cacheInspector.host')
}

function traceProvider(trace: CacheInspectorTrace) {
  return trace.providerId || t('admin.extensions.cacheInspector.defaultProvider')
}

function metricsFailureCount(metrics: CacheInspectorMetrics) {
  return metrics.denied + metrics.stale + metrics.conflicts + metrics.errors + metrics.deadlines
}

function mapLoadError(cause: unknown) {
  const kind = cacheInspectorErrorKind(cause)
  if (kind === 'permission') return t('admin.extensions.cacheInspector.permissionDenied')
  if (kind === 'validation') return apiErrorMessage(cause) || t('admin.extensions.cacheInspector.validationFailed')
  if (kind === 'conflict') return apiErrorMessage(cause) || t('admin.extensions.cacheInspector.conflictError')
  if (kind === 'unavailable') return apiErrorMessage(cause) || t('admin.extensions.cacheInspector.unavailable')
  return apiErrorMessage(cause) || t('admin.extensions.cacheInspector.loadFailed')
}

async function load(manual = false) {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspect(limit.value)
    if (manual) {
      toast.add({
        color: 'primary',
        icon: 'i-lucide-check',
        title: t('admin.extensions.cacheInspector.refreshSuccess'),
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
useSeoMeta({ title: t('admin.extensions.cacheInspector.metaTitle') })
void load()
</script>

<template>
  <div data-testid="admin-cache-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-4 flex min-w-0 flex-col gap-1">
      <h2 class="flex min-w-0 items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <span class="min-w-0 truncate">{{ t('admin.extensions.cacheInspector.title') }}</span>
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.cacheInspector.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-4 rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <UIcon name="i-lucide-database-zap" class="size-4 shrink-0" />
          <template v-if="snapshot">
            <UBadge color="neutral" variant="subtle">
              {{ t('admin.extensions.cacheInspector.revision', { revision: snapshot.registry.revision }) }}
            </UBadge>
            <span>{{ t('admin.extensions.cacheInspector.cacheCount', { count: snapshot.registry.caches.length }) }}</span>
            <span>{{ t('admin.extensions.cacheInspector.traceCount', { count: snapshot.traces.length }) }}</span>
          </template>
        </div>
      </template>
      <template #right>
        <div class="flex min-w-0 items-center gap-2">
          <label class="text-xs text-slate-500 dark:text-zinc-400" for="cache-inspector-limit">
            {{ t('admin.extensions.cacheInspector.limit') }}
          </label>
          <USelect
            id="cache-inspector-limit"
            v-model="limit"
            :items="limitItems"
            value-key="value"
            label-key="label"
            class="w-20"
            data-testid="cache-inspector-limit"
          />
          <UButton
            icon="i-lucide-rotate-cw"
            color="neutral"
            variant="subtle"
            :loading="pending"
            data-testid="cache-inspector-refresh"
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
      data-testid="cache-inspector-error"
    />

    <div v-if="pending" class="space-y-3" aria-busy="true" data-testid="cache-inspector-loading">
      <USkeleton class="h-24 w-full rounded-lg" />
      <USkeleton class="h-44 w-full rounded-lg" />
      <USkeleton class="h-52 w-full rounded-lg" />
    </div>

    <template v-else-if="snapshot">
      <SFAlert
        v-if="snapshot.registry.safeMode"
        variant="warning"
        :title="t('admin.extensions.cacheInspector.safeModeTitle')"
        :description="t('admin.extensions.cacheInspector.safeModeDescription')"
        class="mb-4"
        data-testid="cache-inspector-safe-mode"
      />

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex min-w-0 flex-col gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.cacheInspector.summaryTitle') }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.cacheInspector.redactionHint') }}
          </p>
        </div>
        <dl class="grid grid-cols-2 divide-x divide-y divide-slate-200 dark:divide-zinc-800 sm:grid-cols-3 lg:grid-cols-6">
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.samples') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.metrics.samples) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.hitRate') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ hitRate }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.failures') }}</dt>
            <dd class="mt-1 text-lg font-semibold" :class="failures ? 'text-red-700 dark:text-red-300' : 'text-slate-900 dark:text-zinc-100'">
              {{ formatCount(failures) }}
            </dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.slow') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.metrics.slow) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.average') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCacheDuration(snapshot.metrics.averageDurationMicros) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.p95') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCacheDuration(snapshot.metrics.p95DurationMicros) }}</dd>
          </div>
        </dl>
        <dl class="grid gap-x-4 gap-y-2 border-t border-slate-200 px-3 py-3 text-xs dark:border-zinc-800 sm:grid-cols-2 lg:grid-cols-4">
          <div class="min-w-0"><dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.schema') }}</dt><dd class="break-all font-mono text-slate-700 dark:text-zinc-200">{{ snapshot.schemaVersion }}</dd></div>
          <div class="min-w-0"><dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.digest') }}</dt><dd :title="snapshot.registry.digest" class="break-all font-mono text-slate-700 dark:text-zinc-200">{{ shortDigest(snapshot.registry.digest) }}</dd></div>
          <div class="min-w-0"><dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.publications') }}</dt><dd class="text-slate-700 dark:text-zinc-200">{{ formatCount(snapshot.registry.publications.length) }}</dd></div>
          <div class="min-w-0"><dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.fields.retention') }}</dt><dd class="font-mono text-slate-700 dark:text-zinc-200">{{ snapshot.retainedFromSequence ? `${snapshot.retainedFromSequence}–${snapshot.retainedThroughSequence}` : t('admin.extensions.cacheInspector.retentionEmpty') }}</dd></div>
        </dl>
      </section>

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.extensions.cacheInspector.declarationsTitle') }}</h3>
          <UBadge color="neutral" variant="subtle">{{ snapshot.registry.caches.length }}</UBadge>
        </div>
        <SFEmptyState
          v-if="snapshot.registry.caches.length === 0"
          icon-label="CAC"
          :title="t('admin.extensions.cacheInspector.declarationsEmptyTitle')"
          :description="t('admin.extensions.cacheInspector.declarationsEmptyDescription')"
          class="m-6"
        />
        <div v-else class="overflow-x-auto">
          <table class="min-w-[880px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400">
              <tr><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.cache') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.owner') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.namespace') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.policy') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.provider') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.invalidators') }}</th></tr>
            </thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="cache in snapshot.registry.caches" :key="cache.id">
                <td class="px-3 py-2"><p class="break-all font-mono font-medium text-slate-800 dark:text-zinc-100">{{ cache.id }}</p><p class="break-all text-slate-500 dark:text-zinc-400">{{ cache.contractVersion }}</p></td>
                <td class="px-3 py-2"><p class="break-all text-slate-700 dark:text-zinc-200">{{ cache.artifact.extensionId }} @ {{ cache.artifact.extensionVersion }}</p><p :title="cache.artifact.packageDigest" class="font-mono text-slate-500 dark:text-zinc-400">{{ shortDigest(cache.artifact.packageDigest) }}</p></td>
                <td class="break-all px-3 py-2 font-mono text-slate-700 dark:text-zinc-200">{{ cache.namespace }}</td>
                <td class="px-3 py-2"><UBadge :color="policyColor(cache.policy)" variant="subtle">{{ policyLabel(cache.policy) }}</UBadge></td>
                <td class="break-all px-3 py-2 font-mono text-slate-700 dark:text-zinc-200">{{ cache.provider || t('admin.extensions.cacheInspector.defaultProvider') }}</td>
                <td class="break-all px-3 py-2 text-slate-600 dark:text-zinc-300">{{ cache.invalidators.length ? cache.invalidators.join(', ') : '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.extensions.cacheInspector.metricsTitle') }}</h3>
          <UBadge color="neutral" variant="subtle">{{ snapshot.operations.length }}</UBadge>
        </div>
        <SFEmptyState v-if="snapshot.operations.length === 0" icon-label="OPS" :title="t('admin.extensions.cacheInspector.metricsEmptyTitle')" :description="t('admin.extensions.cacheInspector.metricsEmptyDescription')" class="m-6" />
        <div v-else class="overflow-x-auto">
          <table class="min-w-[760px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400"><tr><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.operation') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.samples') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.hitsMisses') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.failures') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.affected') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.average') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.p95') }}</th></tr></thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="item in snapshot.operations" :key="item.operation"><td class="break-all px-3 py-2 font-mono font-medium text-slate-800 dark:text-zinc-100">{{ operationLabel(item.operation) }}</td><td class="px-3 py-2">{{ formatCount(item.samples) }}</td><td class="px-3 py-2">{{ item.hits }} / {{ item.misses }}</td><td class="px-3 py-2" :class="metricsFailureCount(item) ? 'text-red-700 dark:text-red-300' : ''">{{ formatCount(metricsFailureCount(item)) }}</td><td class="px-3 py-2">{{ formatCount(item.affected) }}</td><td class="px-3 py-2">{{ formatCacheDuration(item.averageDurationMicros) }}</td><td class="px-3 py-2">{{ formatCacheDuration(item.p95DurationMicros) }}</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800"><div><h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.extensions.cacheInspector.tracesTitle') }}</h3><p class="mt-0.5 text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.redactionHint') }}</p></div><UBadge color="neutral" variant="subtle">{{ snapshot.traces.length }}</UBadge></div>
        <SFEmptyState v-if="snapshot.traces.length === 0" icon-label="TRC" :title="t('admin.extensions.cacheInspector.tracesEmptyTitle')" :description="t('admin.extensions.cacheInspector.tracesEmptyDescription')" class="m-6" data-testid="cache-inspector-empty-traces" />
        <div v-else class="overflow-x-auto">
          <table class="min-w-[1060px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400"><tr><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.sequence') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.operation') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.outcome') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.cache') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.owner') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.provider') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.duration') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.revision') }}</th><th class="px-3 py-2 font-medium">{{ t('admin.extensions.cacheInspector.fields.tagDigest') }}</th></tr></thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr v-for="item in snapshot.traces" :key="item.sequence"><td class="px-3 py-2 font-mono">#{{ item.sequence }}</td><td class="break-all px-3 py-2 font-mono">{{ operationLabel(item.operation) }}</td><td class="px-3 py-2"><UBadge :color="outcomeColor(item.outcome)" variant="subtle">{{ outcomeLabel(item.outcome) }}</UBadge></td><td class="break-all px-3 py-2 font-mono">{{ item.cacheId || '—' }}</td><td class="break-all px-3 py-2">{{ traceOwner(item) }}</td><td class="break-all px-3 py-2">{{ traceProvider(item) }}</td><td class="px-3 py-2" :class="item.slow ? 'text-amber-700 dark:text-amber-300' : ''">{{ formatCacheDuration(item.durationMicros) }}</td><td class="px-3 py-2"><UBadge :color="item.registryCurrent ? 'primary' : 'warning'" variant="subtle">{{ item.registryRevision || '—' }}</UBadge></td><td :title="item.tagDigest" class="px-3 py-2 font-mono">{{ item.tagDigest ? shortDigest(item.tagDigest) : '—' }}<span v-if="item.tagCount" class="ml-1 text-slate-500">({{ item.tagCount }})</span></td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800"><h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ t('admin.extensions.cacheInspector.invalidationsTitle') }}</h3><UBadge color="neutral" variant="subtle">{{ snapshot.invalidations.length }}</UBadge></div>
        <SFEmptyState v-if="snapshot.invalidations.length === 0" icon-label="INV" :title="t('admin.extensions.cacheInspector.invalidationsEmptyTitle')" :description="t('admin.extensions.cacheInspector.invalidationsEmptyDescription')" class="m-6" />
        <div v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
          <div v-for="item in snapshot.invalidations" :key="item.sequence" class="grid min-w-0 gap-2 px-3 py-3 text-xs sm:grid-cols-[100px_minmax(0,1fr)_minmax(0,1fr)_auto] sm:items-center"><span class="font-mono text-slate-500">#{{ item.sequence }}</span><span class="break-all font-mono text-slate-800 dark:text-zinc-100">{{ item.invalidatorId || operationLabel(item.operation) }}</span><span class="break-all text-slate-600 dark:text-zinc-300">{{ item.cacheId || traceOwner(item) }}</span><span class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.cacheInspector.affectedCount', { count: item.affected }) }}</span></div>
        </div>
      </section>
    </template>
  </div>
</template>
