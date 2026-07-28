<script setup lang="ts">
import { useDragAndDrop } from '@formkit/drag-and-drop/vue'
import type { SiteNavigationLocation } from '~/composables/admin/useSiteChromeApi'
import type { NavigationEditorItem } from '~/utils/admin/navigationDocument'

const props = defineProps<{
  location: SiteNavigationLocation
  label: string
  supported: boolean
  items: NavigationEditorItem[]
  locations: Array<{ value: SiteNavigationLocation, label: string }>
}>()

const { t } = useI18n()

const emit = defineEmits<{
  create: []
  reorder: [sourceKeys: string[]]
  move: [sourceKey: string, index: number]
  edit: [sourceKey: string]
  remove: [sourceKey: string]
  toggle: [sourceKey: string]
  transfer: [sourceKey: string, target: SiteNavigationLocation, copy: boolean]
}>()

const destination = reactive<Record<string, SiteNavigationLocation>>({})
const [parent, dragItems] = useDragAndDrop<NavigationEditorItem>(props.items, {
  dragHandle: '.sf-navigation-drag-handle',
  onSort: ({ values }) => emit('reorder', values.map(item => item.definition.sourceKey))
})

watch(() => props.items, items => {
  dragItems.value = items
})

function sourceTone(source: NavigationEditorItem['definition']['sourceKind']) {
  return source === 'operator' ? 'primary' : source === 'extension' ? 'warning' : 'neutral'
}

function sourceLabel(source: NavigationEditorItem['definition']['sourceKind']) {
  return t(`admin.navigationEditor.sources.${source}`)
}

function disabledTransferOptions(item: NavigationEditorItem) {
  return props.locations.filter(option => option.value !== item.placement.location)
}
</script>

<template>
  <section class="border border-slate-200 dark:border-zinc-800" :data-navigation-location="location">
    <header class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200 px-3 py-2.5 dark:border-zinc-800">
      <div class="min-w-0"><h3 class="text-sm font-semibold">{{ label }}</h3><p v-if="!supported" class="mt-0.5 text-xs text-amber-700 dark:text-amber-300">{{ t('admin.navigationEditor.notRendered') }}</p></div>
      <div class="flex items-center gap-2">
        <UBadge :color="supported ? 'primary' : 'neutral'" variant="subtle">{{ supported ? t('admin.navigationEditor.activeTheme') : t('admin.navigationEditor.storedOnly') }}</UBadge>
        <UButton size="sm" icon="i-lucide-plus" @click="emit('create')">{{ t('admin.navigationEditor.add') }}</UButton>
      </div>
    </header>
    <ul ref="parent" class="divide-y divide-slate-200 dark:divide-zinc-800">
      <li v-for="(item, index) in dragItems" :key="item.definition.sourceKey" class="grid min-w-0 gap-2 px-3 py-3 lg:grid-cols-[auto_minmax(0,1fr)_auto] lg:items-center">
        <UButton class="sf-navigation-drag-handle cursor-grab active:cursor-grabbing" size="xs" color="neutral" variant="ghost" icon="i-lucide-grip-vertical" :aria-label="`Drag ${item.definition.sourceKey}`" :title="`Drag ${item.definition.sourceKey}`" />
        <div class="flex min-w-0 items-center gap-2.5">
          <UIcon v-if="item.placement.icon || item.definition.icon" :name="item.placement.icon || item.definition.icon" class="size-5 shrink-0 text-muted" aria-hidden="true" />
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2"><strong class="truncate">{{ item.definition.labelZhCN || item.definition.labelEnUS || item.definition.sourceKey }}</strong><UBadge :color="sourceTone(item.definition.sourceKind)" variant="subtle">{{ sourceLabel(item.definition.sourceKind) }}</UBadge><UBadge v-if="!item.placement.enabled" color="neutral" variant="subtle">{{ t('admin.navigationEditor.hidden') }}</UBadge><UBadge v-if="item.definition.sourceKind === 'extension'" color="warning" variant="subtle">{{ t('admin.navigationEditor.extensionUnavailable') }}</UBadge></div>
            <p class="truncate font-mono text-xs text-muted">{{ item.definition.href || item.definition.sourceKey }}</p>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-1">
          <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-chevrons-up" :disabled="index === 0" :aria-label="`Move ${item.definition.sourceKey} to top`" :title="`Move ${item.definition.sourceKey} to top`" @click="emit('move', item.definition.sourceKey, 0)" />
          <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-chevron-up" :disabled="index === 0" :aria-label="`Move ${item.definition.sourceKey} up`" :title="`Move ${item.definition.sourceKey} up`" @click="emit('move', item.definition.sourceKey, index - 1)" />
          <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-chevron-down" :disabled="index === dragItems.length - 1" :aria-label="`Move ${item.definition.sourceKey} down`" :title="`Move ${item.definition.sourceKey} down`" @click="emit('move', item.definition.sourceKey, index + 1)" />
          <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-chevrons-down" :disabled="index === dragItems.length - 1" :aria-label="`Move ${item.definition.sourceKey} to bottom`" :title="`Move ${item.definition.sourceKey} to bottom`" @click="emit('move', item.definition.sourceKey, dragItems.length - 1)" />
          <UButton size="xs" color="neutral" variant="ghost" :icon="item.placement.enabled ? 'i-lucide-eye-off' : 'i-lucide-eye'" :aria-label="`Toggle ${item.definition.sourceKey}`" :title="`Toggle ${item.definition.sourceKey}`" @click="emit('toggle', item.definition.sourceKey)" />
          <UButton v-if="item.definition.sourceKind === 'operator'" size="xs" color="neutral" variant="ghost" icon="i-lucide-pencil" :aria-label="`Edit ${item.definition.sourceKey}`" :title="`Edit ${item.definition.sourceKey}`" @click="emit('edit', item.definition.sourceKey)" />
          <UButton v-if="item.definition.sourceKind === 'operator'" size="xs" color="error" variant="ghost" icon="i-lucide-trash-2" :aria-label="`Delete ${item.definition.sourceKey}`" :title="`Delete ${item.definition.sourceKey}`" @click="emit('remove', item.definition.sourceKey)" />
          <USelect v-model="destination[item.definition.sourceKey]" class="w-36" :items="disabledTransferOptions(item)" value-key="value" label-key="label" :placeholder="t('admin.navigationEditor.moveTo')" />
          <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-right" :disabled="!destination[item.definition.sourceKey]" :aria-label="`Move ${item.definition.sourceKey} to selected location`" :title="`Move ${item.definition.sourceKey} to selected location`" @click="destination[item.definition.sourceKey] && emit('transfer', item.definition.sourceKey, destination[item.definition.sourceKey]!, false)" />
          <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-copy" :disabled="!destination[item.definition.sourceKey]" :aria-label="`Copy ${item.definition.sourceKey} to selected location`" :title="`Copy ${item.definition.sourceKey} to selected location`" @click="destination[item.definition.sourceKey] && emit('transfer', item.definition.sourceKey, destination[item.definition.sourceKey]!, true)" />
        </div>
      </li>
    </ul>
    <p v-if="dragItems.length === 0" class="px-3 py-8 text-center text-sm text-muted">{{ t('admin.navigationEditor.emptyLocation') }}</p>
  </section>
</template>
