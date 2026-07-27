<script setup lang="ts">
import SFExtensionSettingsField from './SFExtensionSettingsField.vue'

import type { AdminExtensionSettingValue, AdminExtensionSettingsGroup } from '~/utils/admin/adminExtensions'

defineProps<{
  group?: AdminExtensionSettingsGroup
  items: AdminExtensionSettingValue[]
  values: Record<string, string>
}>()

const emit = defineEmits<{ update: [key: string, value: string] }>()
</script>

<template>
  <section class="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <header v-if="group" class="border-b border-slate-200 bg-slate-50 px-4 py-3 dark:border-zinc-800 dark:bg-zinc-950">
      <h4 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ group.label }}</h4>
      <p v-if="group.description" class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ group.description }}</p>
    </header>
    <div :class="group?.columns === 2 ? 'grid lg:grid-cols-2' : 'divide-y divide-slate-200 dark:divide-zinc-800'">
      <SFExtensionSettingsField
        v-for="item in items"
        :key="item.key"
        :item="item"
        :model-value="values[item.key] ?? ''"
        :class="group?.columns === 2 ? 'border-b border-slate-200 dark:border-zinc-800' : ''"
        @update:model-value="emit('update', item.key, $event)"
      />
    </div>
  </section>
</template>
