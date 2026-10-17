<script setup lang="ts">
import {
  routeCatalogDigestShort,
  type RouteCatalogGroup,
  type RouteCatalogRoute
} from '~/composables/admin/useAdminRouteCatalog'

const props = defineProps<{
  group: RouteCatalogGroup
  collapsed: boolean
}>()

const emit = defineEmits<{ toggle: [] }>()

const { t } = useI18n()

const replaceRoutes = computed(() =>
  props.group.routes.filter((route): route is RouteCatalogRoute & { action: 'replace' } => route.action === 'replace')
)

const replaceRouteLabels = computed(() =>
  replaceRoutes.value.map(route => `${route.method} ${route.path}`)
)

const providerName = computed(() =>
  props.group.kind === 'core' ? t('admin.extensions.routeCatalog.coreProvider') : props.group.name
)

const providerMeta = computed(() => {
  if (props.group.kind === 'core') return t('admin.extensions.routeCatalog.coreMeta')
  const parts = [
    props.group.version,
    routeCatalogDigestShort(props.group.packageDigest),
    props.group.runtimeInstanceId
  ].filter(Boolean)
  return parts.join(' · ')
})
</script>

<template>
  <section
    class="overflow-hidden rounded-lg border bg-white dark:border-zinc-800 dark:bg-zinc-900"
    :class="group.unadmittedCount > 0 ? 'border-amber-300 dark:border-amber-500/40' : 'border-slate-200'"
    :data-testid="`route-catalog-group-${group.key}`"
  >
    <button
      type="button"
      class="flex w-full flex-wrap items-center gap-2.5 px-4 py-3 text-left transition-colors hover:bg-slate-50 dark:hover:bg-zinc-800/60"
      :aria-expanded="!collapsed"
      @click="emit('toggle')"
    >
      <UIcon
        name="i-lucide-chevron-down"
        class="size-4 shrink-0 text-slate-400 transition-transform dark:text-zinc-500"
        :class="{ '-rotate-90': collapsed }"
      />
      <UBadge :color="group.kind === 'core' ? 'neutral' : 'primary'" variant="subtle" size="sm">
        {{ group.kind === 'core' ? t('admin.extensions.routeCatalog.coreBadge') : t('admin.extensions.routeCatalog.pluginBadge') }}
      </UBadge>
      <strong class="text-sm text-slate-900 dark:text-zinc-100">{{ providerName }}</strong>
      <span class="break-all font-mono text-xs text-slate-500 dark:text-zinc-400">{{ providerMeta }}</span>
      <span class="ms-auto flex flex-wrap items-center gap-2">
        <span class="font-mono text-xs text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.routeCatalog.routesCount', { count: group.routes.length }) }}
        </span>
        <UBadge v-if="group.unadmittedCount > 0" color="warning" variant="subtle" size="sm">
          {{ t('admin.extensions.routeCatalog.unadmittedBadge', { count: group.unadmittedCount }) }}
        </UBadge>
        <UBadge v-if="group.replaceCount > 0" color="error" variant="subtle" size="sm">
          {{ t('admin.extensions.routeCatalog.replaceBadge', { count: group.replaceCount }) }}
        </UBadge>
      </span>
    </button>

    <div v-if="!collapsed" class="border-t border-slate-200 dark:border-zinc-800">
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-slate-200 text-left text-xs font-medium text-slate-500 dark:border-zinc-800 dark:text-zinc-400">
              <th class="px-4 py-2">{{ t('admin.extensions.routeCatalog.columns.method') }}</th>
              <th class="px-3 py-2">{{ t('admin.extensions.routeCatalog.columns.path') }}</th>
              <th class="px-3 py-2">{{ t('admin.extensions.routeCatalog.columns.action') }}</th>
              <th class="px-3 py-2">{{ t('admin.extensions.routeCatalog.columns.guard') }}</th>
              <th class="px-4 py-2">{{ t('admin.extensions.routeCatalog.columns.admission') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="route in group.routes"
              :key="`${route.routeId}\u0000${route.method}`"
              class="border-b border-slate-100 last:border-b-0 dark:border-zinc-800/60"
            >
              <td class="px-4 py-2">
                <span class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs font-semibold text-slate-700 dark:bg-zinc-800 dark:text-zinc-200">
                  {{ route.method }}
                </span>
              </td>
              <td class="break-all px-3 py-2 font-mono text-xs text-slate-900 dark:text-zinc-100">
                {{ route.path }}
                <span
                  v-if="route.conflictKinds.length > 0"
                  class="ms-1 font-bold text-red-600 dark:text-red-400"
                  :title="t('admin.extensions.routeCatalog.conflictHint')"
                >●</span>
              </td>
              <td class="px-3 py-2">
                <UBadge :color="route.action === 'replace' ? 'error' : 'neutral'" variant="subtle" size="xs">
                  {{ route.action }}
                </UBadge>
              </td>
              <td class="break-all px-3 py-2 font-mono text-xs text-slate-500 dark:text-zinc-400">{{ route.guard || '—' }}</td>
              <td class="px-4 py-2">
                <UBadge v-if="route.admitted" color="success" variant="subtle" size="xs">
                  {{ t('admin.extensions.routeCatalog.admitted') }}
                </UBadge>
                <UBadge
                  v-else
                  color="warning"
                  variant="subtle"
                  size="xs"
                  :title="t('admin.extensions.routeCatalog.notAdmittedHint')"
                >
                  {{ t('admin.extensions.routeCatalog.notAdmitted') }}
                </UBadge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="replaceRoutes.length > 0"
        class="border-t border-red-200 bg-red-50 px-4 py-2.5 text-xs text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
      >
        <p class="font-semibold">{{ t('admin.extensions.routeCatalog.replaceNoteTitle', { count: replaceRoutes.length }) }}</p>
        <p class="mt-1 break-all font-mono">{{ replaceRouteLabels.join('、') }}</p>
        <p class="mt-1">{{ t('admin.extensions.routeCatalog.replaceNoteBody') }}</p>
      </div>
      <div
        v-if="group.unadmittedCount > 0"
        class="border-t border-amber-200 bg-amber-50 px-4 py-2.5 text-xs text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300"
      >
        <p class="font-semibold">{{ t('admin.extensions.routeCatalog.unadmittedNoteTitle', { count: group.unadmittedCount }) }}</p>
        <p class="mt-1">{{ t('admin.extensions.routeCatalog.unadmittedNoteBody') }}</p>
      </div>
    </div>
  </section>
</template>
