<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  type ComponentCompositionInspectorSnapshot,
  useAdminCompositionInspectors
} from '~/composables/useAdminCompositionInspectors'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionComponentInspector' })

const { t } = useI18n()
const toast = useToast()
useAdminPage('/extensions/component-inspector')
const { inspectComponents } = useAdminCompositionInspectors()

const limit = ref(50)
const pending = ref(true)
const error = ref('')
const snapshot = ref<ComponentCompositionInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

async function load() {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspectComponents(limit.value)
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
    toast.add({ title: t('admin.extensions.componentInspector.refreshed'), color: 'primary' })
  }
}

onMounted(load)
watch(limit, load)
useSeoMeta({ title: t('admin.extensions.componentInspector.metaTitle') })
</script>

<template>
  <div data-testid="admin-component-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-6 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-slate-900 dark:text-white">
          {{ t('admin.extensions.componentInspector.title') }}
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.componentInspector.intro') }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <USelect
          v-model="limit"
          :items="limitItems"
          value-key="value"
          class="w-28"
          data-testid="component-inspector-limit"
        />
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="soft"
          :loading="pending"
          data-testid="component-inspector-refresh"
          @click="refresh"
        >
          {{ t('admin.extensions.componentInspector.refresh') }}
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
      data-testid="component-inspector-error"
      :close-button="{ icon: 'i-tabler-x', color: 'neutral', variant: 'link' }"
      @close="error = ''"
    />

    <div
      v-if="pending && !snapshot"
      class="text-sm text-slate-500 dark:text-zinc-400"
      data-testid="component-inspector-loading"
    >
      {{ t('admin.extensions.componentInspector.loading') }}
    </div>

    <template v-if="snapshot">
      <UAlert
        v-if="snapshot.safeMode"
        color="warning"
        variant="subtle"
        icon="i-tabler-shield-lock"
        class="mb-4"
        data-testid="component-inspector-safe-mode"
        :title="t('admin.extensions.componentInspector.safeModeTitle')"
        :description="t('admin.extensions.componentInspector.safeModeDescription')"
      />

      <div class="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.componentInspector.revision') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.revision }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.componentInspector.targets') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.targetCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.componentInspector.contributions') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.contributionCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.componentInspector.safeMode') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">
            {{
              snapshot.safeMode
                ? t('admin.extensions.componentInspector.yes')
                : t('admin.extensions.componentInspector.no')
            }}
          </div>
        </div>
      </div>

      <div class="mb-4 rounded-lg border border-slate-200 dark:border-zinc-800">
        <div class="border-b border-slate-200 px-4 py-3 font-medium dark:border-zinc-800">
          {{ t('admin.extensions.componentInspector.conflicts') }}
          ({{ snapshot.conflicts.length }})
        </div>
        <div class="p-4">
          <p
            v-if="!snapshot.conflicts.length"
            class="text-sm text-slate-500 dark:text-zinc-400"
            data-testid="component-inspector-empty-conflicts"
          >
            {{ t('admin.extensions.componentInspector.noConflicts') }}
          </p>
          <pre
            v-else
            class="max-h-80 overflow-auto break-all rounded-md bg-slate-50 p-3 text-xs dark:bg-zinc-900"
            data-testid="component-inspector-conflicts"
          >{{ JSON.stringify(snapshot.conflicts, null, 2) }}</pre>
        </div>
      </div>

      <div class="rounded-lg border border-slate-200 dark:border-zinc-800">
        <div class="border-b border-slate-200 px-4 py-3 font-medium dark:border-zinc-800">
          {{ t('admin.extensions.componentInspector.traces') }}
          ({{ snapshot.traces.length }})
        </div>
        <div class="p-4">
          <p
            v-if="!snapshot.traces.length"
            class="text-sm text-slate-500 dark:text-zinc-400"
            data-testid="component-inspector-empty-traces"
          >
            {{ t('admin.extensions.componentInspector.noTraces') }}
          </p>
          <pre
            v-else
            class="max-h-96 overflow-x-auto overflow-y-auto break-all rounded-md bg-slate-50 p-3 text-xs dark:bg-zinc-900"
            data-testid="component-inspector-traces"
          >{{ JSON.stringify(snapshot.traces, null, 2) }}</pre>
        </div>
      </div>
    </template>
  </div>
</template>
