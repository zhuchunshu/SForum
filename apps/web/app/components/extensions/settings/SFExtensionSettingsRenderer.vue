<script setup lang="ts">
import SFExtensionSettingsActions from './SFExtensionSettingsActions.vue'
import SFExtensionSettingsCallout from './SFExtensionSettingsCallout.vue'
import SFExtensionSettingsGroup from './SFExtensionSettingsGroup.vue'

import type {
  AdminExtensionSettingValue,
  AdminExtensionSettings,
  AdminExtensionSettingsAction,
  AdminExtensionSettingsActionResult,
  AdminExtensionSettingsGroup
} from '~/utils/adminExtensions'

const props = defineProps<{
  settings: AdminExtensionSettings | null
  values: Record<string, string>
  loading: boolean
  saving: boolean
  recommendedApplied: boolean
  actionLoading: Record<string, boolean>
  actionResults: Record<string, AdminExtensionSettingsActionResult | undefined>
}>()

const emit = defineEmits<{
  save: []
  reset: []
  update: [key: string, value: string]
  action: [action: AdminExtensionSettingsAction]
}>()

const { t } = useI18n()
const activeTab = ref('')
const hasSecretFields = computed(() => (props.settings?.items || []).some(item => item.type === 'secret'))
const groupsById = computed(() => new Map((props.settings?.groups || []).map(group => [group.id, group])))

const presentationValid = computed(() => {
  const settings = props.settings
  if (!settings || settings.renderer.layout !== 'tabs') return true
  return Boolean(settings.tabs?.length && settings.tabs.every(tab => (tab.groups || []).every(id => groupsById.value.has(id))))
})

const useTabs = computed(() => props.settings?.renderer.layout === 'tabs' && presentationValid.value)
const tabs = computed(() => props.settings?.tabs || [])

watch(tabs, (items) => {
  if (!items.some(item => item.id === activeTab.value)) {
    activeTab.value = items[0]?.id || ''
  }
}, { immediate: true })

function itemsForGroup(groupId: string) {
  return (props.settings?.items || []).filter(item => item.groupId === groupId)
}

function updateValue(key: string, value: string) {
  emit('update', key, value)
}

function legacySections() {
  const sections: Array<{ id: string, group?: AdminExtensionSettingsGroup, items: AdminExtensionSettingValue[] }> = []
  const byLabel = new Map<string, AdminExtensionSettingValue[]>()
  for (const item of props.settings?.items || []) {
    const label = item.group || ''
    const groupItems = byLabel.get(label) || []
    groupItems.push(item)
    byLabel.set(label, groupItems)
  }
  let index = 0
  for (const [label, items] of byLabel) {
    sections.push({ id: `legacy-${index++}`, group: label ? { id: label, label } : undefined, items })
  }
  return sections
}

const activeCallouts = computed(() => (props.settings?.callouts || []).filter(callout => !callout.tab || callout.tab === activeTab.value))
</script>

<template>
  <div class="space-y-4">
    <section class="rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 text-sm text-emerald-950 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-100">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 class="text-base font-bold">{{ t('admin.extensions.dynamic.recommendedTitle') }}</h3>
          <p class="mt-1 max-w-3xl text-sm text-emerald-800 dark:text-emerald-200">{{ t('admin.extensions.dynamic.recommendedDescription') }}</p>
          <p v-if="hasSecretFields" class="mt-2 text-xs text-emerald-700 dark:text-emerald-300">{{ t('admin.extensions.dynamic.secretsPreserved') }}</p>
        </div>
        <UBadge color="neutral" variant="subtle">
          {{ settings?.renderer.source === 'legacy_array' ? t('admin.extensions.dynamic.compatBadge') : t('admin.extensions.dynamic.schemaBadge') }}
        </UBadge>
      </div>
    </section>

    <UAlert
      v-if="!presentationValid"
      color="warning"
      variant="subtle"
      icon="i-lucide-layout-template"
      :title="t('admin.extensions.dynamic.presentationFallbackTitle')"
      :description="t('admin.extensions.dynamic.presentationFallbackDescription')"
    />

    <SFExtensionSettingsActions
      :actions="(settings?.actions || []).filter(action => action.placement === 'header')"
      :loading="actionLoading"
      :results="actionResults"
      @execute="emit('action', $event)"
    />

    <template v-if="settings?.items.length">
      <div v-if="useTabs" class="space-y-4">
        <div class="overflow-x-auto border-b border-slate-200 dark:border-zinc-800">
          <div class="flex min-w-max gap-1">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              class="border-b-2 px-4 py-3 text-sm font-medium transition"
              :class="activeTab === tab.id ? 'border-[var(--sf-accent)] text-slate-950 dark:text-white' : 'border-transparent text-slate-500 hover:text-slate-800 dark:text-zinc-400 dark:hover:text-zinc-100'"
              @click="activeTab = tab.id"
            >
              {{ tab.label }}
            </button>
          </div>
        </div>
        <p v-if="tabs.find(tab => tab.id === activeTab)?.description" class="text-sm text-slate-500 dark:text-zinc-400">
          {{ tabs.find(tab => tab.id === activeTab)?.description }}
        </p>
        <SFExtensionSettingsCallout v-for="callout in activeCallouts" :key="callout.id" :callout="callout" />
        <template v-for="groupId in tabs.find(tab => tab.id === activeTab)?.groups || []" :key="groupId">
          <SFExtensionSettingsCallout
            v-for="callout in (settings.callouts || []).filter(item => item.group === groupId)"
            :key="callout.id"
            :callout="callout"
          />
          <SFExtensionSettingsGroup
            v-if="itemsForGroup(groupId).length"
            :group="groupsById.get(groupId)"
            :items="itemsForGroup(groupId)"
            :values="values"
            @update="updateValue"
          />
        </template>
      </div>

      <div v-else class="space-y-4">
        <SFExtensionSettingsCallout v-for="callout in settings.callouts || []" :key="callout.id" :callout="callout" />
        <SFExtensionSettingsGroup
          v-for="section in legacySections()"
          :key="section.id"
          :group="section.group"
          :items="section.items"
          :values="values"
          @update="updateValue"
        />
      </div>

      <SFExtensionSettingsActions
        :actions="(settings.actions || []).filter(action => action.placement === 'footer')"
        :loading="actionLoading"
        :results="actionResults"
        @execute="emit('action', $event)"
      />

      <div class="rounded-lg border border-slate-200 bg-white px-4 py-4 dark:border-zinc-800 dark:bg-zinc-900">
        <SFAdminFormFooter
          :saving="saving"
          :disabled="loading"
          :submit-text="t('admin.extensions.dynamic.saveSettings')"
          :reset-text="t('admin.extensions.dynamic.resetDefaults')"
          @submit="emit('save')"
          @reset="emit('reset')"
        >
          <template #left>
            <span>{{ hasSecretFields ? t('admin.extensions.dynamic.footerSecretHint') : t('admin.extensions.dynamic.footerHint') }}</span>
          </template>
        </SFAdminFormFooter>
      </div>
    </template>

    <div v-else-if="!loading" class="rounded-lg border border-slate-200 bg-white p-10 dark:border-zinc-800 dark:bg-zinc-900">
      <SFEmptyState icon-label="CFG" :title="t('admin.extensions.dynamic.emptySettingsTitle')" :description="t('admin.extensions.dynamic.emptySettingsDescription')" />
    </div>
    <div v-else class="rounded-lg border border-slate-200 bg-white p-8 text-sm text-slate-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-400">
      {{ t('admin.extensions.dynamic.loading') }}
    </div>
  </div>
</template>
