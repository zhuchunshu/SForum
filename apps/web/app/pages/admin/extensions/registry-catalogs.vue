<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import {
  type ContentCatalogView,
  type EntityCatalogView,
  type EntityImportExportDryRunView,
  type MediaCatalogView,
  useAdminRegistryCatalogs
} from '~/composables/useAdminRegistryCatalogs'

definePageMeta({ middleware: 'admin', layout: 'admin' })
defineOptions({ name: 'AdminExtensionRegistryCatalogs' })

const { t } = useI18n()
const toast = useToast()
const adminRoutes = useAdminRoutes()
useAdminPage('/extensions/registry-catalogs')
const {
  loadContentCatalog,
  loadEntityCatalog,
  loadMediaCatalog,
  dryRunEntityImportExport
} = useAdminRegistryCatalogs()

const tab = ref<'content' | 'entity' | 'media'>('entity')
const pending = ref(true)
const error = ref('')
const content = ref<ContentCatalogView | null>(null)
const entities = ref<EntityCatalogView | null>(null)
const media = ref<MediaCatalogView | null>(null)

const dryRunEntityId = ref('')
const dryRunAction = ref<'import' | 'export'>('export')
const dryRunPending = ref(false)
const dryRunError = ref('')
const dryRunResult = ref<EntityImportExportDryRunView | null>(null)

const entityOptions = computed(() => {
  const rows = entities.value?.entities || []
  return rows.map(row => ({
    label: row.id,
    value: row.id
  }))
})

const actionItems = computed(() => [
  { label: t('admin.extensions.registryCatalogs.actionExport'), value: 'export' as const },
  { label: t('admin.extensions.registryCatalogs.actionImport'), value: 'import' as const }
])

async function loadAll() {
  pending.value = true
  error.value = ''
  try {
    const [contentCatalog, entityCatalog, mediaCatalog] = await Promise.all([
      loadContentCatalog(),
      loadEntityCatalog(),
      loadMediaCatalog()
    ])
    content.value = contentCatalog
    entities.value = entityCatalog
    media.value = mediaCatalog
    if (!dryRunEntityId.value && entityCatalog.entities.length > 0) {
      dryRunEntityId.value = entityCatalog.entities[0]?.id ?? ''
    }
  } catch (err) {
    error.value = apiErrorMessage(err)
    content.value = null
    entities.value = null
    media.value = null
  } finally {
    pending.value = false
  }
}

async function refresh() {
  await loadAll()
  if (!error.value) {
    toast.add({ title: t('admin.extensions.registryCatalogs.refreshed'), color: 'primary' })
  }
}

async function runDryRun() {
  dryRunPending.value = true
  dryRunError.value = ''
  dryRunResult.value = null
  try {
    dryRunResult.value = await dryRunEntityImportExport(dryRunEntityId.value, dryRunAction.value)
    toast.add({
      title: dryRunResult.value.allowed
        ? t('admin.extensions.registryCatalogs.dryRunAllowed')
        : t('admin.extensions.registryCatalogs.dryRunDenied'),
      color: 'primary'
    })
  } catch (err) {
    dryRunError.value = apiErrorMessage(err)
  } finally {
    dryRunPending.value = false
  }
}

onMounted(loadAll)
useSeoMeta({ title: t('admin.extensions.registryCatalogs.metaTitle') })
</script>

<template>
  <div data-testid="admin-registry-catalogs" class="min-w-0 w-full max-w-full">
    <div class="mb-6 flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-slate-900 dark:text-white">
          {{ t('admin.extensions.registryCatalogs.title') }}
        </h1>
        <p class="mt-1 text-sm text-slate-500 dark:text-zinc-400">
          {{ t('admin.extensions.registryCatalogs.intro') }}
        </p>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <UButton
          icon="i-tabler-arrow-left"
          color="neutral"
          variant="soft"
          :to="adminRoutes.path('/extensions/plugins')"
        >
          {{ t('admin.extensions.registryCatalogs.back') }}
        </UButton>
        <UButton
          icon="i-tabler-refresh"
          color="neutral"
          variant="soft"
          :loading="pending"
          data-testid="registry-catalogs-refresh"
          @click="refresh"
        >
          {{ t('admin.extensions.registryCatalogs.refresh') }}
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
      data-testid="registry-catalogs-error"
      :close-button="{ icon: 'i-tabler-x', color: 'neutral', variant: 'link' }"
      @close="error = ''"
    />

    <div
      v-if="pending && !content && !entities && !media"
      class="text-sm text-slate-500 dark:text-zinc-400"
      data-testid="registry-catalogs-loading"
    >
      {{ t('admin.extensions.registryCatalogs.loading') }}
    </div>

    <template v-if="content || entities || media">
      <div class="mb-4 grid gap-3 sm:grid-cols-3" data-testid="registry-catalogs-summary">
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.registryCatalogs.contentCount') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">
            {{ content?.entryCount ?? 0 }}
          </div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.registryCatalogs.entityCount') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">
            {{ entities?.entryCount ?? 0 }}
          </div>
        </div>
        <div class="rounded-lg border border-slate-200 p-3 dark:border-zinc-800">
          <div class="text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.registryCatalogs.mediaCount') }}
          </div>
          <div class="text-lg font-semibold text-slate-900 dark:text-white">
            {{ media?.entryCount ?? 0 }}
          </div>
        </div>
      </div>

      <div class="mb-4 flex flex-wrap gap-2" data-testid="registry-catalogs-tabs">
        <UButton
          :variant="tab === 'entity' ? 'solid' : 'soft'"
          color="neutral"
          data-testid="registry-catalogs-tab-entity"
          @click="() => { tab = 'entity' }"
        >
          {{ t('admin.extensions.registryCatalogs.tabEntity') }}
        </UButton>
        <UButton
          :variant="tab === 'content' ? 'solid' : 'soft'"
          color="neutral"
          data-testid="registry-catalogs-tab-content"
          @click="() => { tab = 'content' }"
        >
          {{ t('admin.extensions.registryCatalogs.tabContent') }}
        </UButton>
        <UButton
          :variant="tab === 'media' ? 'solid' : 'soft'"
          color="neutral"
          data-testid="registry-catalogs-tab-media"
          @click="() => { tab = 'media' }"
        >
          {{ t('admin.extensions.registryCatalogs.tabMedia') }}
        </UButton>
      </div>

      <div
        v-if="tab === 'entity' && entities"
        class="space-y-4"
        data-testid="registry-catalogs-entity"
      >
        <div class="rounded-lg border border-slate-200 p-4 dark:border-zinc-800">
          <div class="mb-2 text-sm font-medium text-slate-900 dark:text-white">
            {{ t('admin.extensions.registryCatalogs.dryRunTitle') }}
          </div>
          <p class="mb-3 text-xs text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.registryCatalogs.dryRunIntro') }}
          </p>
          <div class="flex flex-col gap-2 sm:flex-row sm:items-end">
            <div class="min-w-0 flex-1">
              <label class="mb-1 block text-xs text-slate-600 dark:text-zinc-300">
                {{ t('admin.extensions.registryCatalogs.entityId') }}
              </label>
              <USelect
                v-model="dryRunEntityId"
                :items="entityOptions"
                value-key="value"
                class="w-full"
                data-testid="registry-catalogs-dry-run-entity"
              />
            </div>
            <div class="w-full sm:w-40">
              <label class="mb-1 block text-xs text-slate-600 dark:text-zinc-300">
                {{ t('admin.extensions.registryCatalogs.action') }}
              </label>
              <USelect
                v-model="dryRunAction"
                :items="actionItems"
                value-key="value"
                class="w-full"
                data-testid="registry-catalogs-dry-run-action"
              />
            </div>
            <UButton
              icon="i-tabler-player-play"
              color="primary"
              :loading="dryRunPending"
              :disabled="!dryRunEntityId"
              data-testid="registry-catalogs-dry-run-submit"
              @click="runDryRun"
            >
              {{ t('admin.extensions.registryCatalogs.runDryRun') }}
            </UButton>
          </div>
          <UAlert
            v-if="dryRunError"
            color="error"
            variant="subtle"
            class="mt-3"
            :title="dryRunError"
            data-testid="registry-catalogs-dry-run-error"
            :close-button="{ icon: 'i-tabler-x', color: 'neutral', variant: 'link' }"
            @close="dryRunError = ''"
          />
          <div
            v-if="dryRunResult"
            class="mt-3 rounded-md border border-slate-200 p-3 text-sm dark:border-zinc-800"
            data-testid="registry-catalogs-dry-run-result"
          >
            <div class="font-medium text-slate-900 dark:text-white">
              {{ dryRunResult.allowed
                ? t('admin.extensions.registryCatalogs.dryRunAllowed')
                : t('admin.extensions.registryCatalogs.dryRunDenied') }}
            </div>
            <div class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ t('admin.extensions.registryCatalogs.executes') }}:
              {{ dryRunResult.executes
                ? t('admin.extensions.registryCatalogs.yes')
                : t('admin.extensions.registryCatalogs.no') }}
              · {{ dryRunResult.action }}
              <span v-if="dryRunResult.permissionKey"> · {{ dryRunResult.permissionKey }}</span>
            </div>
            <div v-if="dryRunResult.reason" class="mt-1 text-xs text-slate-600 dark:text-zinc-300">
              {{ dryRunResult.reason }}
            </div>
          </div>
        </div>

        <div
          v-if="entities.entities.length === 0"
          class="text-sm text-slate-500 dark:text-zinc-400"
          data-testid="registry-catalogs-entity-empty"
        >
          {{ t('admin.extensions.registryCatalogs.emptyEntity') }}
        </div>
        <ul v-else class="space-y-2" data-testid="registry-catalogs-entity-list">
          <li
            v-for="row in entities.entities"
            :key="row.id"
            class="rounded-lg border border-slate-200 p-3 text-sm dark:border-zinc-800"
          >
            <div class="font-mono font-medium text-slate-900 dark:text-white">{{ row.id }}</div>
            <div class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ row.extensionId }}
              <span v-if="row.importExportPolicy"> · {{ row.importExportPolicy }}</span>
            </div>
          </li>
        </ul>
      </div>

      <div
        v-else-if="tab === 'content' && content"
        data-testid="registry-catalogs-content"
      >
        <div
          v-if="content.content.length === 0"
          class="text-sm text-slate-500 dark:text-zinc-400"
          data-testid="registry-catalogs-content-empty"
        >
          {{ t('admin.extensions.registryCatalogs.emptyContent') }}
        </div>
        <ul v-else class="space-y-2" data-testid="registry-catalogs-content-list">
          <li
            v-for="row in content.content"
            :key="row.id"
            class="rounded-lg border border-slate-200 p-3 text-sm dark:border-zinc-800"
          >
            <div class="font-mono font-medium text-slate-900 dark:text-white">{{ row.id }}</div>
            <div class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
              {{ row.kind }} · {{ row.extensionId }}
            </div>
          </li>
        </ul>
      </div>

      <div
        v-else-if="tab === 'media' && media"
        data-testid="registry-catalogs-media"
      >
        <div
          v-if="media.entryCount === 0"
          class="text-sm text-slate-500 dark:text-zinc-400"
          data-testid="registry-catalogs-media-empty"
        >
          {{ t('admin.extensions.registryCatalogs.emptyMedia') }}
        </div>
        <template v-else>
          <h2 class="mb-2 text-sm font-medium text-slate-900 dark:text-white">
            {{ t('admin.extensions.registryCatalogs.policies') }}
          </h2>
          <ul class="mb-4 space-y-2" data-testid="registry-catalogs-media-policies">
            <li
              v-for="row in media.policies"
              :key="row.id"
              class="rounded-lg border border-slate-200 p-3 text-sm dark:border-zinc-800"
            >
              <div class="font-mono font-medium text-slate-900 dark:text-white">{{ row.id }}</div>
              <div class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ row.purpose }} · {{ row.extensionId }}
              </div>
            </li>
          </ul>
          <h2 class="mb-2 text-sm font-medium text-slate-900 dark:text-white">
            {{ t('admin.extensions.registryCatalogs.processors') }}
          </h2>
          <ul class="space-y-2" data-testid="registry-catalogs-media-processors">
            <li
              v-for="row in media.processors"
              :key="row.id"
              class="rounded-lg border border-slate-200 p-3 text-sm dark:border-zinc-800"
            >
              <div class="font-mono font-medium text-slate-900 dark:text-white">{{ row.id }}</div>
              <div class="mt-1 text-xs text-slate-500 dark:text-zinc-400">
                {{ row.stage }} · {{ row.extensionId }}
              </div>
            </li>
          </ul>
        </template>
      </div>
    </template>
  </div>
</template>
