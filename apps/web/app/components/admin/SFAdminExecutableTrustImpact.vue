<script setup lang="ts">
import {
  executableDatabaseGrants,
  executableDatabaseMigrationRisk,
  type ExecutableTrustImpact,
  type ExecutableTrustMigrationDeclaration
} from '~/utils/extensions/extensionTrust'

const props = defineProps<{ impact: ExecutableTrustImpact }>()
const { t } = useI18n()
const showBackupGuidance = ref(true)
let backupGuidanceTimer: ReturnType<typeof setTimeout> | undefined

const database = computed(() => props.impact.database)
const databaseGrants = computed(() => executableDatabaseGrants(database.value))
const databaseGrantSummary = computed(() => databaseGrants.value.join(' · '))
const migrationRisk = computed(() => executableDatabaseMigrationRisk(props.impact))
const databaseAuthorityColor = computed(() => {
  if (databaseGrants.value.includes('raw_core') || databaseGrants.value.includes('kernel')) return 'error'
  if (databaseGrants.value.includes('core_views') || databaseGrants.value.includes('host_commands')) return 'warning'
  return 'neutral'
})
const riskDescription = computed(() => [
  migrationRisk.value.nonTransactional.length
    ? t('admin.extensions.trust.databaseRisk.nonTransactionalWarning', { count: migrationRisk.value.nonTransactional.length })
    : '',
  migrationRisk.value.missingRequiredBackup
    ? t('admin.extensions.trust.databaseRisk.missingRequiredBackupWarning')
    : ''
].filter(Boolean).join(' '))

function migrationLabel(migration: ExecutableTrustMigrationDeclaration, index: number) {
  return migration.id || migration.path || t('admin.extensions.trust.databaseRisk.migrationFallback', { index: index + 1 })
}

function migrationTransaction(migration: ExecutableTrustMigrationDeclaration) {
  return migration.transaction || 'auto'
}

function scheduleBackupGuidanceDismissal() {
  showBackupGuidance.value = true
  if (backupGuidanceTimer) clearTimeout(backupGuidanceTimer)
  backupGuidanceTimer = setTimeout(() => {
    showBackupGuidance.value = false
  }, 10000)
}

onMounted(scheduleBackupGuidanceDismissal)
watch(() => props.impact.digest, scheduleBackupGuidanceDismissal)
onBeforeUnmount(() => {
  if (backupGuidanceTimer) clearTimeout(backupGuidanceTimer)
})

const sections = computed(() => [
  { key: 'artifactDigests', value: props.impact.artifactDigests },
  { key: 'requestedAuthority', value: props.impact.requestedAuthority },
  { key: 'contracts', value: props.impact.contracts },
  { key: 'binaries', value: props.impact.binaries },
  { key: 'backend', value: props.impact.backend },
  { key: 'routes', value: props.impact.routes },
  { key: 'guards', value: props.impact.guards },
  { key: 'guardDeclarations', value: props.impact.guardDeclarations },
  { key: 'hooks', value: props.impact.hooks },
  { key: 'events', value: props.impact.events },
  { key: 'migrations', value: props.impact.migrations },
  { key: 'migrationDeclarations', value: props.impact.migrationDeclarations },
  { key: 'providers', value: props.impact.providers },
  { key: 'jobs', value: props.impact.jobs },
  { key: 'schedules', value: props.impact.schedules },
  { key: 'components', value: props.impact.components },
  { key: 'registryComponents', value: props.impact.registryComponents },
  { key: 'templates', value: props.impact.templates },
  { key: 'assets', value: props.impact.assets },
  { key: 'content', value: props.impact.content },
  { key: 'database', value: props.impact.database },
  { key: 'cache', value: props.impact.cache },
  { key: 'services', value: props.impact.services },
  { key: 'commands', value: props.impact.commands },
  { key: 'adminSurfaces', value: props.impact.adminSurfaces },
  { key: 'queries', value: props.impact.queries },
  { key: 'queryResultFilters', value: props.impact.queryResultFilters },
  { key: 'identity', value: props.impact.identity },
  { key: 'permissionDefinitions', value: props.impact.permissionDefinitions },
  { key: 'media', value: props.impact.media },
  { key: 'navigation', value: props.impact.navigation },
  { key: 'regions', value: props.impact.regions },
  { key: 'contributions', value: props.impact.contributions },
  { key: 'capabilities', value: props.impact.capabilities },
  { key: 'permissions', value: props.impact.permissions },
  { key: 'requiredFeatures', value: props.impact.requiredFeatures },
  { key: 'dependencies', value: props.impact.dependencies },
  { key: 'lifecycle', value: props.impact.lifecycle },
  { key: 'openapi', value: props.impact.openapi },
  { key: 'packageFiles', value: props.impact.packageFiles }
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
        <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.manifestContract') }}</dt>
        <dd class="mt-1 font-mono text-slate-900 dark:text-zinc-100">{{ impact.manifestContract }}</dd>
      </div>
      <div class="sm:col-span-2">
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

    <section
      v-if="migrationRisk.hasDatabaseChanges"
      class="space-y-4 border-y border-slate-200 py-4 dark:border-zinc-800"
      data-testid="extension-database-risk"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex min-w-0 items-center gap-2">
          <UIcon name="i-lucide-database" class="size-5 shrink-0 text-slate-500 dark:text-zinc-400" />
          <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.trust.databaseRisk.title') }}
          </h3>
        </div>
        <UBadge v-if="database" :color="databaseAuthorityColor" variant="subtle" class="max-w-full whitespace-normal break-words font-mono">
          {{ databaseGrantSummary }}
        </UBadge>
      </div>

      <UAlert
        v-if="riskDescription"
        color="error"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        :title="t('admin.extensions.trust.databaseRisk.highRiskTitle')"
        :description="riskDescription"
        data-testid="extension-database-high-risk"
      />
      <UAlert
        v-if="migrationRisk.hasMigrations && showBackupGuidance"
        color="warning"
        variant="subtle"
        icon="i-lucide-database-backup"
        :title="t('admin.extensions.trust.databaseRisk.backupGuidanceTitle')"
        :description="t('admin.extensions.trust.databaseRisk.backupGuidanceDescription')"
        data-testid="extension-database-backup-guidance"
      />

      <dl v-if="database" class="grid gap-x-5 gap-y-3 text-xs sm:grid-cols-2 lg:grid-cols-4">
        <div class="min-w-0">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.databaseRisk.grants') }}</dt>
          <dd class="mt-1 break-all font-mono text-slate-900 dark:text-zinc-100">{{ databaseGrantSummary }}</dd>
        </div>
        <div class="min-w-0">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.databaseRisk.coreCompatibility') }}</dt>
          <dd class="mt-1 break-all font-mono text-slate-900 dark:text-zinc-100">
            {{ database.coreCompatibility || t('admin.extensions.trust.databaseRisk.notDeclared') }}
          </dd>
        </div>
        <div class="min-w-0">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.databaseRisk.backup') }}</dt>
          <dd class="mt-1 flex flex-wrap items-center gap-2">
            <UBadge :color="database.backup.required ? 'warning' : 'neutral'" variant="subtle" size="xs">
              {{ database.backup.required
                ? t('admin.extensions.trust.databaseRisk.backupRequired')
                : t('admin.extensions.trust.databaseRisk.backupNotRequired') }}
            </UBadge>
            <span class="break-all font-mono text-slate-900 dark:text-zinc-100">
              {{ database.backup.strategy || t('admin.extensions.trust.databaseRisk.strategyNotDeclared') }}
            </span>
          </dd>
        </div>
        <div class="min-w-0">
          <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.databaseRisk.retention') }}</dt>
          <dd class="mt-1 space-y-1 text-slate-900 dark:text-zinc-100">
            <div>{{ t('admin.extensions.trust.databaseRisk.onDisable') }}: <span class="font-mono">{{ database.retention.onDisable }}</span></div>
            <div>
              {{ t('admin.extensions.trust.databaseRisk.onUninstall') }}:
              <span class="font-mono">{{ database.retention.onUninstall }}</span>
              <span v-if="database.retention.days"> · {{ t('admin.extensions.trust.databaseRisk.retentionDays', { count: database.retention.days }) }}</span>
            </div>
          </dd>
        </div>
      </dl>

      <div v-if="migrationRisk.hasMigrations" class="space-y-2">
        <h4 class="text-xs font-semibold text-slate-700 dark:text-zinc-300">
          {{ t('admin.extensions.trust.databaseRisk.migrationPlan', { count: impact.migrationDeclarations.length }) }}
        </h4>
        <ul class="grid gap-2 lg:grid-cols-2">
          <li
            v-for="(migration, index) in impact.migrationDeclarations"
            :key="migration.id || migration.path"
            class="min-w-0 rounded-md border border-slate-200 px-3 py-2.5 dark:border-zinc-700"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <p class="truncate text-xs font-medium text-slate-900 dark:text-zinc-100">{{ migrationLabel(migration, index) }}</p>
                <p class="mt-1 break-all font-mono text-[11px] text-slate-500 dark:text-zinc-400">{{ migration.path }}</p>
              </div>
              <UBadge
                :color="migrationTransaction(migration) === 'forbidden' ? 'error' : migrationTransaction(migration) === 'required' ? 'primary' : 'neutral'"
                variant="subtle"
                size="xs"
              >
                {{ t(`admin.extensions.trust.databaseRisk.transactions.${migrationTransaction(migration)}`) }}
              </UBadge>
            </div>
            <dl class="mt-2 grid gap-2 text-[11px] sm:grid-cols-2">
              <div class="min-w-0">
                <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.databaseRisk.contractVersion') }}</dt>
                <dd class="mt-0.5 break-all font-mono text-slate-700 dark:text-zinc-300">{{ migration.contractVersion || t('admin.extensions.trust.databaseRisk.notDeclared') }}</dd>
              </div>
              <div class="min-w-0">
                <dt class="text-slate-500 dark:text-zinc-400">{{ t('admin.extensions.trust.databaseRisk.digest') }}</dt>
                <dd class="mt-0.5 break-all font-mono text-slate-700 dark:text-zinc-300">{{ migration.digest || t('admin.extensions.trust.databaseRisk.notDeclared') }}</dd>
              </div>
            </dl>
          </li>
        </ul>
      </div>
    </section>

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
