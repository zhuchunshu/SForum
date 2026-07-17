<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  formatDurationMicros,
  ROUTE_INSPECTOR_METHODS,
  routeInspectorErrorKind,
  routeInspectorMatchedStep,
  routeInspectorQueryFromRoute,
  routeInspectorQueryParams,
  validateRouteInspectorLookup,
  type RouteInspectorMethod,
  type RouteInspectorProvider,
  type RouteInspectorSnapshot,
  type RouteInspectorTrace
} from '~/composables/useAdminRouteInspector'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionRouteInspector'
})

const { t, te } = useI18n()
const { format: formatSiteDateTime } = useSiteDateTime()
const route = useRoute()
const router = useRouter()
const adminPage = useAdminPage('/extensions/route-inspector')
const { inspect } = useAdminRouteInspector()

const method = ref<RouteInspectorMethod>('GET')
const path = ref('')
const pending = ref(false)
const hasInspected = ref(false)
const validationError = ref('')
const error = ref('')
const snapshot = ref<RouteInspectorSnapshot | null>(null)

const methodItems = computed(() =>
  ROUTE_INSPECTOR_METHODS.map(value => ({ label: value, value }))
)

const matched = computed(() => (snapshot.value ? routeInspectorMatchedStep(snapshot.value) : undefined))

function i18nOrRaw(key: string, fallback: string) {
  return te(key) ? t(key) : fallback
}

function resolutionColor(resolution: RouteInspectorSnapshot['resolution']) {
  if (resolution === 'resolved') return 'primary'
  if (resolution === 'stale') return 'error'
  if (resolution === 'ambiguous') return 'warning'
  return 'neutral'
}

function providerStatusColor(status: RouteInspectorSnapshot['provider']['status']) {
  if (status === 'selected') return 'primary'
  if (status === 'stale') return 'error'
  if (status === 'unselected') return 'warning'
  return 'neutral'
}

function outcomeColor(outcome: RouteInspectorTrace['outcome']) {
  if (outcome === 'succeeded' || outcome === 'committed') return 'primary'
  if (outcome === 'fallback_used') return 'warning'
  return 'error'
}

function labelResolution(value: string) {
  return i18nOrRaw(`admin.extensions.routeInspector.resolution.${value}`, value)
}

function labelProviderStatus(value: string) {
  return i18nOrRaw(`admin.extensions.routeInspector.providerStatus.${value}`, value)
}

function labelPhase(value: string) {
  return i18nOrRaw(`admin.extensions.routeInspector.phase.${value}`, value)
}

function labelOutcome(value: string) {
  return i18nOrRaw(`admin.extensions.routeInspector.outcome.${value}`, value)
}

function labelCommit(value: string) {
  return i18nOrRaw(`admin.extensions.routeInspector.commitState.${value}`, value)
}

function labelConflictKind(value: string) {
  return i18nOrRaw(`admin.extensions.routeInspector.conflictKind.${value}`, value)
}

function providerLabel(provider: RouteInspectorProvider | undefined) {
  if (!provider) return '—'
  if (provider.kind === 'core') return t('admin.extensions.routeInspector.providerCore')
  return provider.artifact?.extensionId || t('admin.extensions.routeInspector.providerPlugin')
}

function applyLookupFromQuery() {
  const lookup = routeInspectorQueryFromRoute(route.query as Record<string, unknown>)
  const upper = lookup.method.toUpperCase()
  if ((ROUTE_INSPECTOR_METHODS as readonly string[]).includes(upper)) {
    method.value = upper as RouteInspectorMethod
  }
  if (lookup.path) path.value = lookup.path
  return lookup
}

function syncQuery(nextMethod: string, nextPath: string) {
  const params = routeInspectorQueryParams(nextMethod, nextPath)
  const current = routeInspectorQueryFromRoute(route.query as Record<string, unknown>)
  if (
    current.method.toUpperCase() === params.method
    && current.path === params.path
  ) {
    return
  }
  void router.replace({ query: { ...route.query, method: params.method, path: params.path } })
}

function mapLoadError(cause: unknown) {
  const kind = routeInspectorErrorKind(cause)
  if (kind === 'permission') return t('admin.extensions.routeInspector.permissionDenied')
  if (kind === 'validation') return apiErrorMessage(cause) || t('admin.extensions.routeInspector.validationFailed')
  if (kind === 'unavailable') return apiErrorMessage(cause) || t('admin.extensions.routeInspector.unavailable')
  if (kind === 'conflict') return apiErrorMessage(cause) || t('admin.extensions.routeInspector.conflictError')
  return apiErrorMessage(cause) || t('admin.extensions.routeInspector.loadFailed')
}

async function runInspect(options: { syncUrl?: boolean } = {}) {
  const validation = validateRouteInspectorLookup({ method: method.value, path: path.value })
  if (!validation.ok) {
    snapshot.value = null
    hasInspected.value = false
    error.value = ''
    if (validation.reason === 'empty') {
      validationError.value = t('admin.extensions.routeInspector.validationEmpty')
    } else if (validation.reason === 'method') {
      validationError.value = t('admin.extensions.routeInspector.validationMethod')
    } else {
      validationError.value = t('admin.extensions.routeInspector.validationPath')
    }
    return
  }

  validationError.value = ''
  if (options.syncUrl !== false) {
    syncQuery(validation.method, validation.path)
  }

  pending.value = true
  error.value = ''
  hasInspected.value = true
  try {
    snapshot.value = await inspect(validation.method, validation.path)
  } catch (cause) {
    snapshot.value = null
    error.value = mapLoadError(cause)
  } finally {
    pending.value = false
  }
}

function onInspectClick() {
  void runInspect({ syncUrl: true })
}

// 仅在 URL 查询本身有效时自动检查，避免分享坏链接时触发畸形请求。
function hydrateFromQuery() {
  const lookup = applyLookupFromQuery()
  const validation = validateRouteInspectorLookup(lookup)
  if (!validation.ok) {
    if (lookup.method || lookup.path) {
      validationError.value = validation.reason === 'method'
        ? t('admin.extensions.routeInspector.validationMethod')
        : validation.reason === 'path'
          ? t('admin.extensions.routeInspector.validationPath')
          : t('admin.extensions.routeInspector.validationEmpty')
    }
    return
  }
  method.value = validation.method
  path.value = validation.path
  void runInspect({ syncUrl: false })
}

watch(
  () => [route.query.method, route.query.path] as const,
  (next, prev) => {
    if (!prev) return
    if (next[0] === prev[0] && next[1] === prev[1]) return
    hydrateFromQuery()
  }
)

useSeoMeta({ title: t('admin.extensions.routeInspector.metaTitle') })
hydrateFromQuery()
</script>

<template>
  <div data-testid="admin-route-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-3 flex min-w-0 flex-col gap-1">
      <h2 class="flex min-w-0 items-center gap-2 text-lg font-semibold text-slate-900 dark:text-zinc-100 sm:text-xl">
        <UIcon :name="adminPage.icon" class="size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <span class="min-w-0 truncate">{{ t('admin.extensions.routeInspector.title') }}</span>
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.routeInspector.intro') }}
      </p>
    </div>

    <!-- 查找表单：显式检查，不在输入时自动请求 -->
    <form
      class="mb-4 rounded-lg border border-slate-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-900"
      data-testid="route-inspector-lookup"
      @submit.prevent="onInspectClick"
    >
      <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-end">
        <div class="w-full shrink-0 sm:w-36">
          <label class="mb-1 block text-xs font-medium text-slate-600 dark:text-zinc-300" for="route-inspector-method">
            {{ t('admin.extensions.routeInspector.method') }}
          </label>
          <USelect
            id="route-inspector-method"
            v-model="method"
            :items="methodItems"
            value-key="value"
            label-key="label"
            class="w-full"
            data-testid="route-inspector-method"
          />
        </div>
        <div class="min-w-0 flex-1">
          <label class="mb-1 block text-xs font-medium text-slate-600 dark:text-zinc-300" for="route-inspector-path">
            {{ t('admin.extensions.routeInspector.path') }}
            <UTooltip :text="t('admin.extensions.routeInspector.pathHint')">
              <UIcon name="i-lucide-info" class="ml-1 inline size-3.5 align-text-bottom text-slate-400" />
            </UTooltip>
          </label>
          <UInput
            id="route-inspector-path"
            v-model="path"
            class="w-full font-mono"
            :placeholder="t('admin.extensions.routeInspector.pathPlaceholder')"
            autocomplete="off"
            data-testid="route-inspector-path"
          />
        </div>
        <UButton
          type="submit"
          icon="i-lucide-search"
          color="primary"
          class="w-full shrink-0 sm:w-auto"
          :loading="pending"
          data-testid="route-inspector-submit"
        >
          {{ t('admin.extensions.routeInspector.inspect') }}
        </UButton>
      </div>
      <p class="mt-2 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.routeInspector.readOnlyHint') }}
      </p>
    </form>

    <SFAlert
      v-if="validationError"
      variant="warning"
      :title="validationError"
      class="mb-4"
      data-testid="route-inspector-validation"
    />
    <SFAlert
      v-else-if="error"
      variant="danger"
      :title="error"
      class="mb-4"
      data-testid="route-inspector-error"
    />

    <div v-if="pending" class="space-y-2" aria-busy="true" data-testid="route-inspector-loading">
      <USkeleton class="h-10 w-full rounded-lg" />
      <USkeleton class="h-28 w-full rounded-lg" />
      <USkeleton class="h-40 w-full rounded-lg" />
    </div>

    <SFEmptyState
      v-else-if="!hasInspected && !validationError && !error"
      icon-label="INS"
      :title="t('admin.extensions.routeInspector.firstUseTitle')"
      :description="t('admin.extensions.routeInspector.firstUseDescription')"
      data-testid="route-inspector-first-use"
    />

    <template v-else-if="snapshot && !pending">
      <!-- 快照摘要：revision / Safe Mode / resolution（无 registry digest 字段） -->
      <section
        class="mb-4 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
        data-testid="route-inspector-summary"
      >
        <div class="flex min-w-0 flex-col gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 flex-wrap items-center gap-2 text-xs sm:text-sm">
            <UBadge :color="resolutionColor(snapshot.resolution)" variant="subtle">
              {{ labelResolution(snapshot.resolution) }}
            </UBadge>
            <UBadge color="neutral" variant="subtle">
              {{ t('admin.extensions.routeInspector.revision', { revision: snapshot.revision }) }}
            </UBadge>
            <UBadge v-if="snapshot.safeMode" color="warning" variant="subtle" data-testid="route-inspector-safe-mode">
              {{ t('admin.extensions.routeInspector.safeMode') }}
            </UBadge>
            <span class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs font-semibold text-slate-700 dark:bg-zinc-800 dark:text-zinc-200">
              {{ snapshot.method }}
            </span>
          </div>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.routeInspector.snapshotHint') }}
          </p>
        </div>

        <SFAlert
          v-if="snapshot.safeMode"
          variant="warning"
          compact
          :title="t('admin.extensions.routeInspector.safeModeTitle')"
          :description="t('admin.extensions.routeInspector.safeModeDescription')"
          class="m-3"
        />

        <SFEmptyState
          v-if="snapshot.resolution === 'not_found'"
          icon-label="404"
          :title="t('admin.extensions.routeInspector.notFoundTitle')"
          :description="t('admin.extensions.routeInspector.notFoundDescription')"
          class="p-6"
          data-testid="route-inspector-not-found"
        />

        <SFAlert
          v-else-if="snapshot.resolution === 'ambiguous'"
          variant="warning"
          :title="t('admin.extensions.routeInspector.ambiguousTitle')"
          :description="t('admin.extensions.routeInspector.ambiguousDescription')"
          class="m-3"
          data-testid="route-inspector-ambiguous"
        />

        <SFAlert
          v-else-if="snapshot.resolution === 'stale'"
          variant="danger"
          :title="t('admin.extensions.routeInspector.staleTitle')"
          :description="t('admin.extensions.routeInspector.staleDescription')"
          class="m-3"
          data-testid="route-inspector-stale"
        />

        <dl
          v-if="matched"
          class="grid min-w-0 gap-x-4 gap-y-2 px-3 py-3 text-xs sm:grid-cols-2 lg:grid-cols-3"
          data-testid="route-inspector-matched"
        >
          <div class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.routeId') }}</dt>
            <dd class="break-all font-mono text-slate-900 dark:text-zinc-100">{{ matched.routeId }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.contract') }}</dt>
            <dd class="break-all font-mono">{{ matched.contractVersion }}</dd>
          </div>
          <div v-if="matched.targetRouteId" class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.target') }}</dt>
            <dd class="break-all font-mono">{{ matched.targetRouteId }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.declaredPath') }}</dt>
            <dd class="break-all font-mono">{{ matched.path }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.pathSignature') }}</dt>
            <dd class="break-all font-mono">{{ matched.pathSignature }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.action') }}</dt>
            <dd class="break-all font-mono">{{ matched.action }}</dd>
          </div>
          <div
            v-if="matched.mutableRequestFields?.length"
            class="min-w-0 sm:col-span-2 lg:col-span-3"
            data-testid="route-inspector-matched-request-fields"
          >
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.mutableRequestFields') }}</dt>
            <dd class="mt-1 flex min-w-0 flex-wrap gap-1">
              <code
                v-for="field in matched.mutableRequestFields"
                :key="field"
                class="max-w-full break-all rounded bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-700 dark:bg-zinc-800 dark:text-zinc-200"
              >{{ field }}</code>
            </dd>
          </div>
          <div
            v-if="matched.mutableResponseFields?.length"
            class="min-w-0 sm:col-span-2 lg:col-span-3"
            data-testid="route-inspector-matched-response-fields"
          >
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.mutableResponseFields') }}</dt>
            <dd class="mt-1 flex min-w-0 flex-wrap gap-1">
              <code
                v-for="field in matched.mutableResponseFields"
                :key="field"
                class="max-w-full break-all rounded bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-700 dark:bg-zinc-800 dark:text-zinc-200"
              >{{ field }}</code>
            </dd>
          </div>
        </dl>
      </section>

      <!-- 提供者解析（只读，不提供选择操作） -->
      <section
        class="mb-4 rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
        data-testid="route-inspector-provider"
      >
        <header class="flex min-w-0 flex-wrap items-center gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.routeInspector.providerTitle') }}
          </h3>
          <UBadge :color="providerStatusColor(snapshot.provider.status)" variant="subtle" size="xs">
            {{ labelProviderStatus(snapshot.provider.status) }}
          </UBadge>
        </header>
        <dl class="grid min-w-0 gap-x-4 gap-y-2 px-3 py-3 text-xs sm:grid-cols-2 lg:grid-cols-3">
          <div v-if="snapshot.provider.live" class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.liveProvider') }}</dt>
            <dd class="break-all font-mono">
              {{ snapshot.provider.live.kind }}
              <template v-if="snapshot.provider.live.artifact">
                · {{ snapshot.provider.live.artifact.extensionId }}@{{ snapshot.provider.live.artifact.extensionVersion }}
              </template>
            </dd>
          </div>
          <div v-if="snapshot.provider.live?.artifact" class="min-w-0 sm:col-span-2">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.packageDigest') }}</dt>
            <dd class="break-all font-mono" data-testid="route-inspector-digest">
              {{ snapshot.provider.live.artifact.packageDigest }}
            </dd>
          </div>
          <div v-if="snapshot.provider.live?.artifact" class="min-w-0">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.runtime') }}</dt>
            <dd class="break-all font-mono">{{ snapshot.provider.live.artifact.runtimeInstanceId }}</dd>
          </div>
          <div v-if="snapshot.provider.desired" class="min-w-0 sm:col-span-2 lg:col-span-3">
            <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.desiredSelection') }}</dt>
            <dd class="break-all font-mono">
              {{ snapshot.provider.desired.extensionId }}@{{ snapshot.provider.desired.extensionVersion }}
              · {{ snapshot.provider.desired.packageDigest }}
              · rev {{ snapshot.provider.desired.revision }}
            </dd>
          </div>
          <div v-if="!snapshot.provider.live && !snapshot.provider.desired" class="min-w-0 text-slate-500 sm:col-span-2">
            {{ t('admin.extensions.routeInspector.providerEmpty') }}
          </div>
        </dl>
      </section>

      <!-- 执行链 -->
      <section
        class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
        data-testid="route-inspector-chain"
      >
        <header class="flex min-w-0 flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.routeInspector.chainTitle') }}
          </h3>
          <span class="text-xs text-slate-500">{{ t('admin.extensions.routeInspector.chainCount', { count: snapshot.chain.length }) }}</span>
        </header>

        <SFEmptyState
          v-if="snapshot.chain.length === 0"
          icon-label="CHN"
          :title="t('admin.extensions.routeInspector.chainEmptyTitle')"
          :description="t('admin.extensions.routeInspector.chainEmptyDescription')"
          class="p-6"
        />

        <ol v-else class="divide-y divide-slate-100 dark:divide-zinc-800">
          <li
            v-for="step in snapshot.chain"
            :key="`${step.index}-${step.routeId}-${step.pathSignature}`"
            class="min-w-0 px-3 py-3"
            :data-testid="`route-inspector-step-${step.index}`"
          >
            <div class="flex min-w-0 flex-wrap items-center gap-2">
              <UBadge color="neutral" variant="subtle" size="xs">#{{ step.index }}</UBadge>
              <UBadge color="neutral" variant="outline" size="xs">{{ labelPhase(step.phase) }}</UBadge>
              <span class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[11px] font-semibold dark:bg-zinc-800">{{ step.method }}</span>
              <code class="min-w-0 break-all text-xs font-semibold text-slate-900 dark:text-zinc-100">{{ step.path }}</code>
              <UBadge color="primary" variant="subtle" size="xs">{{ step.action }}</UBadge>
            </div>
            <p class="mt-1 break-all font-mono text-[11px] text-slate-500 dark:text-zinc-400">
              {{ step.routeId }} · {{ step.contractVersion }}
              <template v-if="step.targetRouteId"> · → {{ step.targetRouteId }}</template>
            </p>
            <dl class="mt-2 grid min-w-0 gap-x-4 gap-y-1.5 text-[11px] sm:grid-cols-2 lg:grid-cols-4">
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.provider') }}</dt>
                <dd class="break-all font-mono">{{ providerLabel(step.provider) }}</dd>
              </div>
              <div v-if="step.provider.artifact" class="min-w-0 sm:col-span-2">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.packageDigest') }}</dt>
                <dd class="break-all font-mono">{{ step.provider.artifact.packageDigest }}</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.guard') }}</dt>
                <dd class="break-all font-mono">{{ step.guard || '—' }}</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.access') }}</dt>
                <dd class="break-all font-mono">{{ step.access || '—' }}</dd>
              </div>
              <div v-if="step.permission" class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.permission') }}</dt>
                <dd class="break-all font-mono">{{ step.permission }}</dd>
              </div>
              <div v-if="step.pluginGuard" class="min-w-0 sm:col-span-2">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.pluginGuard') }}</dt>
                <dd class="break-all font-mono">
                  {{ step.pluginGuard.kind }} · {{ step.pluginGuard.id }} · {{ step.pluginGuard.entry }}
                  <template v-if="step.pluginGuard.permissions?.length">
                    · {{ step.pluginGuard.permissions.join(', ') }}
                  </template>
                </dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.mode') }}</dt>
                <dd class="break-all font-mono">{{ step.mode || '—' }}</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.fallback') }}</dt>
                <dd class="break-all font-mono">{{ step.fallback || '—' }}</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.timeout') }}</dt>
                <dd>{{ step.timeoutMs }} ms</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.priority') }}</dt>
                <dd>{{ step.priority }}</dd>
              </div>
              <div v-if="step.handler || step.destination" class="min-w-0 sm:col-span-2">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.handler') }}</dt>
                <dd class="break-all font-mono">{{ step.handler || step.destination || '—' }}</dd>
              </div>
              <div v-if="step.requestSchema || step.responseSchema" class="min-w-0 sm:col-span-2 lg:col-span-4">
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.contracts') }}</dt>
                <dd class="break-all font-mono">{{ step.requestSchema || '—' }} → {{ step.responseSchema || '—' }}</dd>
              </div>
              <div
                v-if="step.mutableRequestFields?.length"
                class="min-w-0 sm:col-span-2 lg:col-span-4"
                :data-testid="`route-inspector-step-${step.index}-request-fields`"
              >
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.mutableRequestFields') }}</dt>
                <dd class="mt-1 flex min-w-0 flex-wrap gap-1">
                  <code
                    v-for="field in step.mutableRequestFields"
                    :key="field"
                    class="max-w-full break-all rounded bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-700 dark:bg-zinc-800 dark:text-zinc-200"
                  >{{ field }}</code>
                </dd>
              </div>
              <div
                v-if="step.mutableResponseFields?.length"
                class="min-w-0 sm:col-span-2 lg:col-span-4"
                :data-testid="`route-inspector-step-${step.index}-response-fields`"
              >
                <dt class="text-slate-500">{{ t('admin.extensions.routeInspector.fields.mutableResponseFields') }}</dt>
                <dd class="mt-1 flex min-w-0 flex-wrap gap-1">
                  <code
                    v-for="field in step.mutableResponseFields"
                    :key="field"
                    class="max-w-full break-all rounded bg-slate-100 px-1.5 py-0.5 text-[10px] text-slate-700 dark:bg-zinc-800 dark:text-zinc-200"
                  >{{ field }}</code>
                </dd>
              </div>
            </dl>
          </li>
        </ol>
      </section>

      <!-- 冲突（只读） -->
      <section
        class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
        data-testid="route-inspector-conflicts"
      >
        <header class="flex min-w-0 flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.routeInspector.conflictsTitle') }}
          </h3>
          <span class="text-xs text-slate-500">{{ t('admin.extensions.routeInspector.conflictsCount', { count: snapshot.conflicts.length }) }}</span>
        </header>

        <p v-if="snapshot.conflicts.length === 0" class="px-3 py-4 text-xs text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.routeInspector.conflictsEmpty') }}
        </p>

        <div
          v-for="(conflict, conflictIndex) in snapshot.conflicts"
          :key="`${conflict.kind}-${conflict.method}-${conflict.pathSignature}-${conflictIndex}`"
          class="border-b border-slate-100 px-3 py-3 last:border-b-0 dark:border-zinc-800"
        >
          <div class="flex min-w-0 flex-wrap items-center gap-2">
            <UBadge color="warning" variant="subtle" size="xs">{{ labelConflictKind(conflict.kind) }}</UBadge>
            <span class="font-mono text-[11px] font-semibold">{{ conflict.method }}</span>
            <code class="min-w-0 break-all text-xs">{{ conflict.pathSignature }}</code>
            <UBadge
              v-if="conflict.selectionStatus"
              :color="providerStatusColor(conflict.selectionStatus)"
              variant="subtle"
              size="xs"
            >
              {{ labelProviderStatus(conflict.selectionStatus) }}
            </UBadge>
          </div>
          <p v-if="conflict.routeId" class="mt-1 break-all font-mono text-[11px] text-slate-500">
            {{ conflict.routeId }}
            <template v-if="conflict.contractVersion"> · {{ conflict.contractVersion }}</template>
          </p>
          <ul class="mt-2 space-y-1.5">
            <li
              v-for="candidate in conflict.candidates"
              :key="`${candidate.routeId}-${candidate.contractVersion}-${candidate.provider.artifact?.packageDigest || candidate.provider.kind}`"
              class="min-w-0 rounded border border-slate-100 px-2 py-1.5 text-[11px] dark:border-zinc-800"
            >
              <div class="flex min-w-0 flex-wrap items-center gap-1.5">
                <strong class="break-all font-mono">{{ candidate.routeId }}</strong>
                <span class="text-slate-500">{{ candidate.action }}</span>
                <span class="break-all font-mono text-slate-500">{{ providerLabel(candidate.provider) }}</span>
              </div>
              <p class="mt-0.5 break-all font-mono text-slate-500">
                {{ candidate.path }} · guard {{ candidate.guard || '—' }}
                <template v-if="candidate.permission"> · {{ candidate.permission }}</template>
              </p>
            </li>
          </ul>
        </div>
      </section>

      <!-- 有界脱敏轨迹 -->
      <section
        class="mb-2 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
        data-testid="route-inspector-traces"
      >
        <header class="flex min-w-0 flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.routeInspector.tracesTitle') }}
            </h3>
            <p class="mt-0.5 text-[11px] text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.routeInspector.tracesHint') }}
            </p>
          </div>
          <span class="text-xs text-slate-500">{{ t('admin.extensions.routeInspector.tracesCount', { count: snapshot.traces.length }) }}</span>
        </header>

        <SFEmptyState
          v-if="snapshot.traces.length === 0"
          icon-label="TRC"
          :title="t('admin.extensions.routeInspector.tracesEmptyTitle')"
          :description="t('admin.extensions.routeInspector.tracesEmptyDescription')"
          class="p-6"
          data-testid="route-inspector-traces-empty"
        />

        <div v-else class="overflow-x-auto">
          <table class="min-w-full text-left text-[11px]">
            <thead class="border-b border-slate-200 bg-slate-50 text-slate-500 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.seq') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.observedAt') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.phase') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.outcome') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.duration') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.commitState') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.routeInspector.fields.routeId') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-100 dark:divide-zinc-800">
              <tr
                v-for="trace in snapshot.traces"
                :key="trace.sequence"
                class="align-top"
                :data-testid="`route-inspector-trace-${trace.sequence}`"
              >
                <td class="whitespace-nowrap px-3 py-2 font-mono">{{ trace.sequence }}</td>
                <td class="min-w-[8rem] px-3 py-2">{{ formatSiteDateTime(trace.observedAt) }}</td>
                <td class="px-3 py-2">
                  <UBadge color="neutral" variant="subtle" size="xs">{{ labelPhase(trace.phase) }}</UBadge>
                  <span class="ml-1 font-mono text-slate-500">#{{ trace.stepIndex }}</span>
                </td>
                <td class="px-3 py-2">
                  <UBadge :color="outcomeColor(trace.outcome)" variant="subtle" size="xs">{{ labelOutcome(trace.outcome) }}</UBadge>
                </td>
                <td class="whitespace-nowrap px-3 py-2 font-mono">{{ formatDurationMicros(trace.durationMicros) }}</td>
                <td class="px-3 py-2 font-mono">{{ labelCommit(trace.commitState) }}</td>
                <td class="min-w-0 max-w-[14rem] break-all px-3 py-2 font-mono">
                  {{ trace.routeId }}
                  <span class="block text-slate-500">{{ trace.pathSignature }} · {{ trace.action }}</span>
                  <span v-if="trace.provider.artifact" class="block text-slate-500">
                    {{ trace.provider.artifact.extensionId }} · {{ trace.provider.artifact.packageDigest.slice(0, 12) }}…
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>
