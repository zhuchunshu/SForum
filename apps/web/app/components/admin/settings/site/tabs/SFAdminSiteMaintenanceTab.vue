<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { adminOptionMap, useAdminOptionTab } from '~/composables/admin/settings/useAdminOptionTab'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const toast = useToast()
const section = useSettingsSection()
const { saveOptions } = useAdminOptionTab(items => emit('saved', items))
const map = computed(() => adminOptionMap(props.items))
const form = reactive({ enabled: false, message: '' })
const initial = computed(() => ({
  enabled: normalizeEnabledOption(map.value['site.maintenance.enabled']?.value, false),
  message: map.value['site.maintenance.message']?.value || ''
}))
const hasChanges = computed(() => form.enabled !== initial.value.enabled || form.message.trim() !== initial.value.message.trim())

watch(() => props.items, resetFromItems, { immediate: true })

function resetFromItems() {
  Object.assign(form, initial.value)
}

async function save() {
  await section.runSave({
    successTitle: t('admin.settings.saved'),
    failureTitle: t('admin.settings.saveFailed'),
    save: () => saveOptions([
      { name: 'site.maintenance.enabled', value: enabledOptionValue(form.enabled) },
      { name: 'site.maintenance.message', value: form.message.trim() }
    ])
  })
}

function resetChanges() {
  resetFromItems()
  toast.add({ color: 'neutral', icon: 'i-lucide-rotate-ccw', title: t('admin.settings.maintenance.resetChanges'), duration: 10000 })
}
</script>

<template>
  <form class="flex flex-col" @submit.prevent="save">
    <UCard class="border-slate-200 bg-white text-slate-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 backdrop-blur-sm border-t border-slate-200 dark:border-zinc-800 p-4 sm:px-6' }">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div><h2 class="text-base font-bold">{{ t('admin.settings.maintenance.title') }}</h2><p class="mt-1 text-xs text-muted">{{ t('admin.settings.maintenance.description') }}</p></div>
          <UBadge color="neutral" variant="soft" class="font-mono">site.maintenance.*</UBadge>
        </div>
      </template>
      <div class="space-y-5">
        <UAlert color="warning" variant="soft" icon="i-lucide-construction" :title="t('admin.settings.maintenance.warning')" />
        <UFormField :label="t('admin.settings.maintenance.enabled')" name="maintenance-enabled">
          <div class="flex items-center justify-between gap-3 rounded-md border border-slate-200 bg-white p-3 dark:border-zinc-700 dark:bg-zinc-950">
            <div><p class="text-sm">{{ form.enabled ? t('admin.settings.maintenance.enabledOn') : t('admin.settings.maintenance.enabledOff') }}</p><p class="mt-1 text-xs text-muted">{{ t('admin.settings.maintenance.enabledHint') }}</p></div>
            <USwitch v-model="form.enabled" />
          </div>
        </UFormField>
        <UFormField :label="t('admin.settings.maintenance.message')" :description="t('admin.settings.maintenance.messageHint')" name="maintenance-message">
          <UTextarea v-model="form.message" :rows="3" class="w-full" :placeholder="t('admin.settings.maintenance.messagePlaceholder')" maxlength="500" />
        </UFormField>
      </div>
      <template #footer>
        <SFAdminFormFooter :saving="section.saving.value" :show-unsaved-alert="hasChanges" :submit-text="t('admin.settings.save')" @reset="resetChanges" />
      </template>
    </UCard>
  </form>
</template>
