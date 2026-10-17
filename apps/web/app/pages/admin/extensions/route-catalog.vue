<script setup lang="ts">
import SFAdminRouteCatalogGroup from '~/components/admin/SFAdminRouteCatalogGroup.vue'
import { useAdminPage } from '~/composables/admin/useAdminPage'
import {
  groupRouteCatalog,
  routeCatalogStats,
  useAdminRouteCatalog,
  type RouteCatalogSnapshot
} from '~/composables/admin/useAdminRouteCatalog'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionRouteCatalog'
})

const { t } = useI18n()
const toast = useToast()
const adminPage = useAdminPage('/extensions/route-catalog')
const { load } = useAdminRouteCatalog()

const pending = ref(false)
const error = ref('')
const snapshot = ref<RouteCatalogSnapshot | null>(null)
const collapsedKeys = ref<Set<string>>(new Set())

const groups = computed(() => (snapshot.value ? groupRouteCatalog(snapshot.value.routes) : []))
const stats = computed(() =>
  snapshot.value
    ? routeCatalogStats(snapshot.value)
    : { total: 0, pluginProviders: 0, pluginRoutes: 0, unadmitted: 0, conflicts: 0 }
)
const allCollapsed = computed(() => groups.value.length > 0 && collapsedKeys.value.size >= groups.value.length)

async function loadCatalog() {
  pending.value = true
  error.value = ''
  try {
    snapshot.value = await load()
  } catch (cause) {
    error.value = apiErrorMessage(cause) || t('admin.extensions.routeCatalog.loadFailed')
  } finally {
    pending.value = false
  }
}

async function refresh() {
  await loadCatalog()
  if (!error.value) {
    toast.add({ title: t('admin.extensions.routeCatalog.refreshed'), color: 'success' })
  }
}

function toggleGroup(key: string) {
  const next = new Set(collapsedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedKeys.value = next
}

function toggleAll() {
  collapsedKeys.value = allCollapsed.value ? new Set() : new Set(groups.value.map(group => group.key))
}

useSeoMeta({ title: t('admin.extensions.routeCatalog.metaTitle') })
void loadCatalog()
</script>

<template>
  <div data-testid="admin-route-catalog" class="min-w-0 shrink-0">
    <div class="mb-4 flex flex-col gap-1">
      <h2 class="flex items-center gap-2 text-xl font-bold text-slate-900 dark:text-zinc-100">
        <UIcon :name="adminPage.icon" class="size-5 text-[var(--sf-accent)] dark:text-[var(--sf-accent-dark)]" />
        {{ t('admin.extensions.routeCatalog.title') }}
      </h2>
      <p class="max-w-4xl text-sm text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.routeCatalog.intro') }}
      </p>
    </div>

    <UDashboardToolbar class="mb-5 rounded-lg border border-slate-200 bg-white px-4 py-2.5 dark:border-zinc-800 dark:bg-zinc-900">
      <template #left>
        <div class="flex min-w-0 flex-wrap items-center gap-2 text-sm text-slate-600 dark:text-zinc-300">
          <span v-if="snapshot">
            {{ t('admin.extensions.routeCatalog.snapshot', { revision: snapshot.revision }) }}
            <UBadge v-if="snapshot.safeMode" color="warning" variant="subtle" size="xs" class="ms-1">
              {{ t('admin.extensions.routeCatalog.safeMode') }}
            </UBadge>
            <span class="mx-1 text-slate-300 dark:text-zinc-700">·</span>
          </span>
          <span>{{ t('admin.extensions.routeCatalog.totalRoutes', { count: stats.total }) }}</span>
          <UBadge color="primary" variant="subtle">
            {{ t('admin.extensions.routeCatalog.pluginProviders', { count: stats.pluginProviders }) }}
          </UBadge>
          <UBadge v-if="stats.unadmitted" color="warning" variant="subtle">
            {{ t('admin.extensions.routeCatalog.unadmittedCount', { count: stats.unadmitted }) }}
          </UBadge>
          <UBadge v-if="stats.conflicts" color="error" variant="subtle">
            {{ t('admin.extensions.routeCatalog.conflictCount', { count: stats.conflicts }) }}
          </UBadge>
        </div>
      </template>
      <template #right>
        <UButton
          icon="i-lucide-fold-vertical"
          color="neutral"
          variant="subtle"
          size="sm"
          :disabled="pending || groups.length === 0"
          @click="toggleAll"
        >
          {{ allCollapsed ? t('admin.extensions.routeCatalog.expandAll') : t('admin.extensions.routeCatalog.collapseAll') }}
        </UButton>
        <UButton
          icon="i-lucide-rotate-cw"
          color="neutral"
          variant="subtle"
          size="sm"
          :loading="pending"
          @click="refresh"
        >
          {{ t('admin.extensions.refresh') }}
        </UButton>
      </template>
    </UDashboardToolbar>

    <SFAlert v-if="error" variant="danger" :title="error" class="mb-4" />

    <div v-if="pending && !snapshot" class="space-y-3" aria-busy="true">
      <USkeleton v-for="index in 3" :key="index" class="h-24 w-full rounded-lg" />
    </div>

    <SFEmptyState
      v-else-if="!error && snapshot && snapshot.routes.length === 0"
      icon-label="RTC"
      :title="t('admin.extensions.routeCatalog.emptyTitle')"
      :description="t('admin.extensions.routeCatalog.emptyDescription')"
    />

    <div v-else-if="!error" class="space-y-3">
      <SFAdminRouteCatalogGroup
        v-for="group in groups"
        :key="group.key"
        :group="group"
        :collapsed="collapsedKeys.has(group.key)"
        @toggle="toggleGroup(group.key)"
      />
      <p class="px-1 text-xs text-slate-500 dark:text-zinc-400">
        {{ t('admin.extensions.routeCatalog.hint') }}
      </p>
    </div>
  </div>
</template>
