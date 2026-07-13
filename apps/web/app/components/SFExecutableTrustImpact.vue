<script setup lang="ts">
import type { ExecutableTrustImpact } from '~/utils/extensionTrust'

const props = defineProps<{ impact: ExecutableTrustImpact }>()
const { t } = useI18n()

const sections = computed(() => [
  { key: 'artifactDigests', value: props.impact.artifactDigests },
  { key: 'requestedAuthority', value: props.impact.requestedAuthority },
  { key: 'contracts', value: props.impact.contracts },
  { key: 'binaries', value: props.impact.binaries },
  { key: 'routes', value: props.impact.routes },
  { key: 'guards', value: props.impact.guards },
  { key: 'hooks', value: props.impact.hooks },
  { key: 'events', value: props.impact.events },
  { key: 'migrations', value: props.impact.migrations },
  { key: 'providers', value: props.impact.providers },
  { key: 'jobs', value: props.impact.jobs },
  { key: 'schedules', value: props.impact.schedules },
  { key: 'components', value: props.impact.components },
  { key: 'contributions', value: props.impact.contributions },
  { key: 'capabilities', value: props.impact.capabilities },
  { key: 'permissions', value: props.impact.permissions },
  { key: 'requiredFeatures', value: props.impact.requiredFeatures },
  { key: 'dependencies', value: props.impact.dependencies }
])

function itemCount(value: unknown) {
  if (Array.isArray(value)) return value.length
  if (value && typeof value === 'object') return Object.keys(value).length
  return value ? 1 : 0
}

function formatted(value: unknown) {
  return JSON.stringify(value, null, 2)
}
</script>

<template>
  <div class="space-y-4">
    <dl class="grid gap-3 text-xs sm:grid-cols-2">
      <div>
        <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.schemaVersion') }}</dt>
        <dd class="mt-1 font-mono text-slate-900 dark:text-zinc-100">{{ impact.schemaVersion }}</dd>
      </div>
      <div>
        <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.identity') }}</dt>
        <dd class="mt-1 break-all font-mono text-slate-900 dark:text-zinc-100">
          {{ impact.extensionId }}@{{ impact.extensionVersion }} / {{ impact.extensionType }} / {{ impact.source }}
        </dd>
      </div>
      <div class="sm:col-span-2">
        <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.packageDigest') }}</dt>
        <dd class="mt-1 break-all font-mono text-slate-900 dark:text-zinc-100">{{ impact.packageDigest }}</dd>
      </div>
      <div class="sm:col-span-2">
        <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.impactDigest') }}</dt>
        <dd class="mt-1 break-all font-mono text-slate-900 dark:text-zinc-100">{{ impact.digest }}</dd>
      </div>
    </dl>

    <div class="grid gap-2 sm:grid-cols-2">
      <details
        v-for="section in sections"
        :key="section.key"
        class="group min-w-0 rounded-md border border-slate-200 bg-white open:sm:col-span-2 dark:border-zinc-700 dark:bg-zinc-950"
      >
        <summary class="flex min-h-10 cursor-pointer list-none items-center justify-between gap-3 px-3 py-2 text-sm font-medium text-slate-800 dark:text-zinc-100">
          <span class="flex min-w-0 items-center gap-2">
            <UIcon name="i-lucide-chevron-right" class="size-4 shrink-0 transition-transform group-open:rotate-90" />
            <span class="truncate">{{ t(`admin.extensions.trust.sections.${section.key}`) }}</span>
          </span>
          <UBadge color="neutral" variant="subtle" size="xs">{{ itemCount(section.value) }}</UBadge>
        </summary>
        <pre class="max-h-56 overflow-auto border-t border-slate-200 p-3 text-[11px] leading-5 text-slate-700 dark:border-zinc-800 dark:text-zinc-300">{{ formatted(section.value) }}</pre>
      </details>
    </div>
  </div>
</template>
