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
useAdminPage('/extensions/navigation-inspector')
const { inspectNavigation } = useAdminCompositionInspectors()

const limit = ref(50)
const pending = ref(true)
const error = ref('')
const snapshot = ref<NavigationInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

async function load() {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspectNavigation(limit.value)
  } catch (err) {
    error.value = apiErrorMessage(err)
    snapshot.value = null
  } finally {
    pending.value = false
  }
}

async function refresh() {
  await load()
  if (!error.value) {
    toast.add({ title: t('admin.extensions.navigationInspector.refreshed'), color: 'primary' })
  }
}

onMounted(load)
watch(limit, load)
useSeoMeta({ title: t('admin.extensions.navigationInspector.metaTitle') })
</script>

<template>
  <div data-testid="admin-navigation-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-6 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-slate-900 dark:text-white">
          {{ t('admin.extensions.navigationInspector.title') }}
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.navigationInspector.intro') }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <USelect
          v-model="limit"
          :items="limitItems"
          value-key="value"
          class="w-28"
          data-testid="navigation-inspector-limit"
        />
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="soft"
          :loading="pending"
          data-testid="navigation-inspector-refresh"
          @click="refresh"
        >
          {{ t('admin.extensions.navigationInspector.refresh') }}
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      icon="i-tabler-alert-triangle"
      :title="error"
      class="mb-4"
      data-testid="navigation-inspector-error"
      :close-button="{ icon: 'i-tabler-x', color: 'neutral', variant: 'link' }"
      @close="error = ''"
    />

    <div
      v-if="pending && !snapshot"
      class="text-sm text-slate-500 dark:text-zinc-400"
      data-testid="navigation-inspector-loading"
    >
      {{ t('admin.extensions.navigationInspector.loading') }}
    </div>

    <template v-if="snapshot">
      <UAlert
        v-if="snapshot.safeMode"
        color="warning"
        variant="subtle"
        icon="i-tabler-shield-lock"
        class="mb-4"
        data-testid="navigation-inspector-safe-mode"
        :title="t('admin.extensions.navigationInspector.safeModeTitle')"
        :description="t('admin.extensions.navigationInspector.safeModeDescription')"
      />

      <div class="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.revision') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.revision }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.navigation') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.navigationCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.regions') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.regionCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.conflicts') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.providerConflicts }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.safeMode') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">
            {{
              snapshot.safeMode
                ? t('admin.extensions.navigationInspector.yes')
                : t('admin.extensions.navigationInspector.no')
            }}
          </div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.navigationInspector.digest') }}
          </div>
          <div
            class="truncate font-mono text-sm text-slate-900 dark:text-white"
            :title="snapshot.digest || undefined"
          >
            {{ snapshot.digest || '—' }}
          </div>
        </div>
      </div>

      <div class="rounded-lg border border-slate-200 dark:border-zinc-800">
        <div class="border-b border-slate-200 px-4 py-3 font-medium dark:border-zinc-800">
          {{ t('admin.extensions.navigationInspector.traces') }}
          ({{ snapshot.traces.length }})
        </div>
        <div class="p-4">
          <p
            v-if="!snapshot.traces.length"
            class="text-sm text-slate-500 dark:text-zinc-400"
            data-testid="navigation-inspector-empty-traces"
          >
            {{ t('admin.extensions.navigationInspector.noTraces') }}
          </p>
          <pre
            v-else
            class="max-h-96 overflow-x-auto overflow-y-auto break-all rounded-md bg-slate-50 p-3 text-xs dark:bg-zinc-900"
            data-testid="navigation-inspector-traces"
          >{{ JSON.stringify(snapshot.traces, null, 2) }}</pre>
        </div>
      </div>
    </template>
  </div>
</template>
