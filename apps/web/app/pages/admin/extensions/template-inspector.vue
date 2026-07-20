<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  type TemplateInspectorSnapshot,
  useAdminTemplateInspector
} from '~/composables/useAdminTemplateInspector'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionTemplateInspector' })

const { t } = useI18n()
const toast = useToast()
useAdminPage('/extensions/template-inspector')
const { inspect } = useAdminTemplateInspector()

const limit = ref(50)
const pending = ref(true)
const error = ref('')
const snapshot = ref<TemplateInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

function shortDigest(value: string) {
  return value.length > 16 ? `${value.slice(0, 12)}…${value.slice(-4)}` : value
}

async function load() {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await inspect(limit.value)
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
    toast.add({ title: t('admin.extensions.templateInspector.refreshed'), color: 'primary', duration: 10000 })
  }
}

onMounted(load)
watch(limit, load)
useSeoMeta({ title: t('admin.extensions.templateInspector.metaTitle') })
</script>

<template>
  <div data-testid="admin-template-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-6 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-slate-900 dark:text-white">
          {{ t('admin.extensions.templateInspector.title') }}
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.templateInspector.intro') }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <USelect
          v-model="limit"
          :items="limitItems"
          value-key="value"
          class="w-28"
          data-testid="template-inspector-limit"
        />
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="soft"
          :loading="pending"
          data-testid="template-inspector-refresh"
          @click="refresh"
        >
          {{ t('admin.extensions.templateInspector.refresh') }}
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
      data-testid="template-inspector-error"
      :close-button="{ icon: 'i-tabler-x', color: 'neutral', variant: 'link' }"
      @close="error = ''"
    />

    <div
      v-if="pending && !snapshot"
      class="text-sm text-slate-500 dark:text-zinc-400"
      data-testid="template-inspector-loading"
    >
      {{ t('admin.extensions.templateInspector.loading') }}
    </div>

    <template v-if="snapshot">
      <div class="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.templateInspector.revision') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.revision }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.templateInspector.snapshots') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.snapshotCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.templateInspector.overrides') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">{{ snapshot.overrideCount }}</div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.templateInspector.activeTheme') }}
          </div>
          <div class="break-all text-sm font-semibold text-slate-900 dark:text-white" data-testid="template-inspector-active">
            {{ snapshot.activeTheme || '—' }}
          </div>
        </div>
      </div>

      <p class="mb-4 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.templateInspector.redactionHint') }}
      </p>

      <div class="mb-4 flex flex-wrap gap-3 text-sm text-slate-600 dark:text-zinc-300">
        <span>
          {{ t('admin.extensions.templateInspector.defaultTheme') }}:
          <strong class="break-all text-slate-900 dark:text-white">{{ snapshot.defaultTheme || '—' }}</strong>
        </span>
      </div>

      <div class="rounded-lg border border-slate-200 dark:border-zinc-800">
        <div class="border-b border-slate-200 px-4 py-3 font-medium dark:border-zinc-800">
          {{ t('admin.extensions.templateInspector.snapshotsTitle') }}
          ({{ snapshot.snapshots.length }}/{{ snapshot.snapshotCount }})
        </div>
        <div class="p-4">
          <p
            v-if="!snapshot.snapshots.length"
            class="text-sm text-slate-500 dark:text-zinc-400"
            data-testid="template-inspector-empty-snapshots"
          >
            {{ t('admin.extensions.templateInspector.noSnapshots') }}
          </p>
          <div v-else class="space-y-4" data-testid="template-inspector-snapshots">
            <div
              v-for="item in snapshot.snapshots"
              :key="item.extensionId + item.packageDigest"
              class="rounded-md border border-slate-100 p-3 dark:border-zinc-800"
            >
              <div class="mb-2 flex min-w-0 flex-wrap items-center gap-2">
                <span class="break-all font-medium text-slate-900 dark:text-white">
                  {{ item.extensionId }}
                </span>
                <UBadge color="neutral" variant="subtle">{{ item.extensionVersion }}</UBadge>
                <UBadge color="neutral" variant="outline">{{ item.kind }}</UBadge>
                <UBadge v-if="item.active" color="primary" variant="subtle">
                  {{ t('admin.extensions.templateInspector.active') }}
                </UBadge>
                <UBadge v-if="item.default" color="neutral" variant="subtle">
                  {{ t('admin.extensions.templateInspector.defaultBadge') }}
                </UBadge>
              </div>
              <div class="mb-2 break-all text-xs text-slate-500 dark:text-zinc-400">
                {{ t('admin.extensions.templateInspector.packageDigest') }}:
                {{ shortDigest(item.packageDigest) }}
              </div>
              <div class="grid gap-2 text-xs sm:grid-cols-2">
                <div>
                  <div class="mb-1 font-medium text-slate-500 dark:text-zinc-400">
                    {{ t('admin.extensions.templateInspector.contributions') }}
                  </div>
                  <div class="break-all text-slate-700 dark:text-zinc-200">
                    {{ item.contributionIds.length ? item.contributionIds.join(', ') : '—' }}
                  </div>
                </div>
                <div>
                  <div class="mb-1 font-medium text-slate-500 dark:text-zinc-400">
                    {{ t('admin.extensions.templateInspector.overrideTargets') }}
                  </div>
                  <div class="break-all text-slate-700 dark:text-zinc-200">
                    {{ item.overrideTargets.length ? item.overrideTargets.join(', ') : '—' }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
