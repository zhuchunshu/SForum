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
const adminPage = useAdminPage('/extensions/template-inspector')
const { inspect } = useAdminTemplateInspector()

const limit = ref(100)
const pending = ref(true)
const error = ref('')
const snapshot = ref<TemplateInspectorSnapshot | null>(null)
const limitItems = [25, 50, 100, 200].map(value => ({ label: String(value), value }))

function shortDigest(value: string) {
  return value.length > 16 ? `${value.slice(0, 12)}…${value.slice(-4)}` : value
}

function formatCount(value: number) {
  return new Intl.NumberFormat().format(value)
}

function mapLoadError(cause: unknown) {
  const fromApi = apiErrorMessage(cause)
  if (fromApi) return fromApi
  if (cause instanceof Error && cause.message.trim()) return cause.message
  return t('admin.extensions.templateInspector.loadFailed')
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
        title: t('admin.extensions.templateInspector.refreshSuccess'),
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
useSeoMeta({ title: t('admin.extensions.templateInspector.metaTitle') })
void load()
</script>

<template>
  <div data-testid="admin-template-inspector" class="min-w-0 w-full max-w-full">
    <div class="mb-4 flex min-w-0 flex-col gap-1">
      <h2 class="flex min-w-0 items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 shrink-0 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        <span class="min-w-0 truncate">{{ t('admin.extensions.templateInspector.title') }}</span>
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.templateInspector.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-4 rounded-lg border border-slate-200 bg-white px-3 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <UIcon name="i-lucide-layout-template" class="size-4 shrink-0" />
          <template v-if="snapshot">
            <UBadge color="neutral" variant="subtle">
              {{ t('admin.extensions.templateInspector.revisionBadge', { revision: snapshot.revision }) }}
            </UBadge>
            <span>{{ t('admin.extensions.templateInspector.snapshotCount', { count: snapshot.snapshotCount }) }}</span>
            <span>{{ t('admin.extensions.templateInspector.overrideCount', { count: snapshot.overrideCount }) }}</span>
          </template>
        </div>
      </template>
      <template #right>
        <div class="flex min-w-0 items-center gap-2">
          <label class="text-xs text-slate-500 dark:text-zinc-400" for="template-inspector-limit">
            {{ t('admin.extensions.templateInspector.limit') }}
          </label>
          <USelect
            id="template-inspector-limit"
            v-model="limit"
            :items="limitItems"
            value-key="value"
            label-key="label"
            class="w-20"
            data-testid="template-inspector-limit"
          />
          <UButton
            icon="i-lucide-rotate-cw"
            color="neutral"
            variant="subtle"
            :loading="pending"
            data-testid="template-inspector-refresh"
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
      data-testid="template-inspector-error"
    />

    <div v-if="pending" class="space-y-3" aria-busy="true" data-testid="template-inspector-loading">
      <USkeleton class="h-24 w-full rounded-lg" />
      <USkeleton class="h-44 w-full rounded-lg" />
      <USkeleton class="h-52 w-full rounded-lg" />
    </div>

    <template v-else-if="snapshot">
      <section class="mb-4 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex min-w-0 flex-col gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800 sm:flex-row sm:items-center sm:justify-between">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.templateInspector.summaryTitle') }}
          </h3>
          <p class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.templateInspector.redactionHint') }}
          </p>
        </div>
        <dl class="grid grid-cols-2 divide-x divide-y divide-slate-200 dark:divide-zinc-800 sm:grid-cols-2 lg:grid-cols-4">
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.templateInspector.fields.revision') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.revision) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.templateInspector.fields.snapshots') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.snapshotCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.templateInspector.fields.overrides') }}</dt>
            <dd class="mt-1 text-lg font-semibold text-slate-900 dark:text-zinc-100">{{ formatCount(snapshot.overrideCount) }}</dd>
          </div>
          <div class="min-w-0 p-3">
            <dt class="text-xs text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.templateInspector.fields.activeTheme') }}</dt>
            <dd
              class="mt-1 break-all text-sm font-semibold text-slate-900 dark:text-zinc-100"
              data-testid="template-inspector-active"
            >
              {{ snapshot.activeTheme || '—' }}
            </dd>
          </div>
        </dl>
        <dl class="grid gap-x-4 gap-y-2 border-t border-slate-200 px-3 py-3 text-xs dark:border-zinc-800 sm:grid-cols-2">
          <div class="min-w-0">
            <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.templateInspector.fields.schema') }}</dt>
            <dd class="break-all font-mono text-slate-700 dark:text-zinc-200">{{ snapshot.schemaVersion }}</dd>
          </div>
          <div class="min-w-0">
            <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.templateInspector.fields.defaultTheme') }}</dt>
            <dd class="break-all text-slate-700 dark:text-zinc-200">{{ snapshot.defaultTheme || '—' }}</dd>
          </div>
        </dl>
      </section>

      <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.templateInspector.snapshotsTitle') }}
          </h3>
          <UBadge color="neutral" variant="subtle">
            {{ snapshot.snapshots.length }}/{{ snapshot.snapshotCount }}
          </UBadge>
        </div>
        <SFEmptyState
          v-if="snapshot.snapshots.length === 0"
          icon-label="TPL"
          :title="t('admin.extensions.templateInspector.snapshotsEmptyTitle')"
          :description="t('admin.extensions.templateInspector.snapshotsEmptyDescription')"
          class="m-6"
          data-testid="template-inspector-empty-snapshots"
        />
        <div v-else class="overflow-x-auto" data-testid="template-inspector-snapshots">
          <table class="min-w-[920px] w-full text-left text-xs">
            <thead class="bg-slate-50 text-slate-500 dark:bg-zinc-950/70 dark:text-zinc-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.templateInspector.fields.extension') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.templateInspector.fields.kind') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.templateInspector.fields.digest') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.templateInspector.fields.contributions') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.templateInspector.fields.overrideTargets') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.extensions.templateInspector.fields.flags') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-200 dark:divide-zinc-800">
              <tr
                v-for="item in snapshot.snapshots"
                :key="item.extensionId + item.packageDigest"
              >
                <td class="px-3 py-2">
                  <p class="break-all font-mono font-medium text-slate-800 dark:text-zinc-100">{{ item.extensionId }}</p>
                  <p class="text-slate-500 dark:text-zinc-400">{{ item.extensionVersion }}</p>
                </td>
                <td class="px-3 py-2">
                  <UBadge color="neutral" variant="subtle">{{ item.kind }}</UBadge>
                </td>
                <td
                  class="break-all px-3 py-2 font-mono text-slate-600 dark:text-zinc-300"
                  :title="item.packageDigest"
                >
                  {{ shortDigest(item.packageDigest) }}
                </td>
                <td class="break-all px-3 py-2 text-slate-600 dark:text-zinc-300">
                  {{ item.contributionIds.length ? item.contributionIds.join(', ') : '—' }}
                </td>
                <td class="break-all px-3 py-2 text-slate-600 dark:text-zinc-300">
                  {{ item.overrideTargets.length ? item.overrideTargets.join(', ') : '—' }}
                </td>
                <td class="px-3 py-2">
                  <div class="flex flex-wrap gap-1">
                    <UBadge v-if="item.active" color="primary" variant="subtle">
                      {{ t('admin.extensions.templateInspector.active') }}
                    </UBadge>
                    <UBadge v-if="item.default" color="neutral" variant="subtle">
                      {{ t('admin.extensions.templateInspector.defaultBadge') }}
                    </UBadge>
                    <span v-if="!item.active && !item.default" class="text-slate-400">—</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </div>
</template>
