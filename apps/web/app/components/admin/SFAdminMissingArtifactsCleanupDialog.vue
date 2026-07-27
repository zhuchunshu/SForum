<script setup lang="ts">
import type {
  AdminExtension,
  AdminMissingArtifactDataMode
} from '~/utils/admin/adminExtensions'

defineProps<{
  extensions: AdminExtension[]
  busy?: boolean
  error?: string
}>()

const emit = defineEmits<{
  cancel: []
  confirm: []
}>()

const open = defineModel<boolean>('open', { required: true })
const dataMode = defineModel<AdminMissingArtifactDataMode>('dataMode', { required: true })
const { t } = useI18n()

const options = computed(() => [
  {
    value: 'preserve' as const,
    label: t('admin.extensions.missingCleanup.modes.preserve.label'),
    description: t('admin.extensions.missingCleanup.modes.preserve.description')
  },
  {
    value: 'discard_settings' as const,
    label: t('admin.extensions.missingCleanup.modes.discardSettings.label'),
    description: t('admin.extensions.missingCleanup.modes.discardSettings.description')
  }
])
</script>

<template>
  <UModal v-model:open="open" :ui="{ content: 'sm:max-w-2xl' }">
    <template #content>
      <div class="p-5 sm:p-6">
        <div class="flex items-start gap-3">
          <div class="flex size-9 shrink-0 items-center justify-center rounded-md bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300">
            <UIcon name="i-lucide-package-x" class="size-4" />
          </div>
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-slate-900 dark:text-zinc-100">
              {{ t('admin.extensions.missingCleanup.title') }}
            </h2>
            <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-zinc-300">
              {{ t('admin.extensions.missingCleanup.description', { count: extensions.length }) }}
            </p>
          </div>
        </div>

        <div class="mt-5">
          <h3 class="text-sm font-medium text-slate-900 dark:text-zinc-100">
            {{ t('admin.extensions.missingCleanup.listTitle') }}
          </h3>
          <ul class="mt-2 max-h-56 divide-y divide-slate-200 overflow-y-auto rounded-md border border-slate-200 dark:divide-zinc-800 dark:border-zinc-800">
            <li v-for="extension in extensions" :key="extension.id" class="flex items-start gap-3 px-3 py-3">
              <UIcon :name="extension.type === 'theme' ? 'i-lucide-palette' : 'i-lucide-plug'" class="mt-0.5 size-4 shrink-0 text-red-500" />
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-medium text-slate-900 dark:text-zinc-100">
                  {{ extension.name || extension.id }}
                </p>
                <p class="mt-0.5 break-all text-xs text-slate-500 dark:text-zinc-400">
                  {{ extension.id }} · {{ t(`admin.extensions.types.${extension.type}`) }} · v{{ extension.version }}
                </p>
              </div>
            </li>
          </ul>
        </div>

        <div class="mt-5 border-t border-slate-200 pt-5 dark:border-zinc-800">
          <URadioGroup
            v-model="dataMode"
            :legend="t('admin.extensions.missingCleanup.dataLegend')"
            :items="options"
            value-key="value"
            color="primary"
          />
          <UAlert
            class="mt-4"
            :color="dataMode === 'discard_settings' ? 'warning' : 'neutral'"
            :icon="dataMode === 'discard_settings' ? 'i-lucide-triangle-alert' : 'i-lucide-database-backup'"
            variant="subtle"
            :title="t(dataMode === 'discard_settings'
              ? 'admin.extensions.missingCleanup.discardSettingsWarning'
              : 'admin.extensions.missingCleanup.preserveWarning')"
          />
          <p class="mt-3 text-xs leading-5 text-slate-500 dark:text-zinc-400">
            {{ t('admin.extensions.missingCleanup.businessDataNotice') }}
          </p>
        </div>

        <UAlert
          v-if="error"
          class="mt-4"
          color="error"
          icon="i-lucide-triangle-alert"
          variant="subtle"
          :title="error"
        />

        <div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="ghost" :disabled="busy" @click="emit('cancel')">
            {{ t('admin.extensions.confirmUninstallCancel') }}
          </UButton>
          <UButton color="error" icon="i-lucide-trash-2" :loading="busy" @click="emit('confirm')">
            {{ t('admin.extensions.missingCleanup.confirm', { count: extensions.length }) }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>
