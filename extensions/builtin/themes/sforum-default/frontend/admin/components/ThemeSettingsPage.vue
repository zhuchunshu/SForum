<script setup lang="ts">
import type { AdminSlotProps } from '@sforum/admin-sdk'
import { computed, ref } from 'vue'
import { useSForumAdminHost } from '@sforum/admin-sdk'

const props = defineProps<AdminSlotProps<'admin.extension.settings.page'>>()
const host = useSForumAdminHost()

type TabId = 'home' | 'rail' | 'nav' | 'layout'
type GroupId = 'homeCopy' | 'railToggles' | 'railLimits' | 'railCopy' | 'nav' | 'layout'

const activeTab = ref<TabId>('home')

const fieldOrder = [
  'home.notice.zh-CN',
  'home.notice.en-US',
  'home.empty_title.zh-CN',
  'home.empty_title.en-US',
  'home.empty_description.zh-CN',
  'home.empty_description.en-US',
  'home.right_rail.enabled',
  'home.right_rail.show_hot',
  'home.right_rail.show_stats',
  'home.right_rail.show_tags',
  'home.right_rail.show_auth_card',
  'home.right_rail.hot_limit',
  'home.right_rail.tag_limit',
  'home.right_rail.welcome.zh-CN',
  'home.right_rail.welcome.en-US',
  'home.nav.show_compose',
  'home.nav.show_counts',
  'layout.show_footer',
  'layout.show_announcements'
] as const

const groupOf: Record<string, GroupId> = {
  'home.notice.zh-CN': 'homeCopy',
  'home.notice.en-US': 'homeCopy',
  'home.empty_title.zh-CN': 'homeCopy',
  'home.empty_title.en-US': 'homeCopy',
  'home.empty_description.zh-CN': 'homeCopy',
  'home.empty_description.en-US': 'homeCopy',
  'home.right_rail.enabled': 'railToggles',
  'home.right_rail.show_hot': 'railToggles',
  'home.right_rail.show_stats': 'railToggles',
  'home.right_rail.show_tags': 'railToggles',
  'home.right_rail.show_auth_card': 'railToggles',
  'home.right_rail.hot_limit': 'railLimits',
  'home.right_rail.tag_limit': 'railLimits',
  'home.right_rail.welcome.zh-CN': 'railCopy',
  'home.right_rail.welcome.en-US': 'railCopy',
  'home.nav.show_compose': 'nav',
  'home.nav.show_counts': 'nav',
  'layout.show_footer': 'layout',
  'layout.show_announcements': 'layout'
}

const tabGroups: Record<TabId, GroupId[]> = {
  home: ['homeCopy'],
  rail: ['railToggles', 'railLimits', 'railCopy'],
  nav: ['nav'],
  layout: ['layout']
}

const itemsByKey = computed(() => Object.fromEntries(props.context.items.map(item => [item.key, item])))

const tabs = computed(() => ([
  { id: 'home' as const, label: host.t('tabs.home'), icon: 'i-lucide-home' },
  { id: 'rail' as const, label: host.t('tabs.rail'), icon: 'i-lucide-panel-right' },
  { id: 'nav' as const, label: host.t('tabs.nav'), icon: 'i-lucide-panel-left' },
  { id: 'layout' as const, label: host.t('tabs.layout'), icon: 'i-lucide-layout-template' }
]))

const groupsOnTab = computed(() => {
  const order = tabGroups[activeTab.value] || []
  return order.map((id) => ({
    id,
    title: host.t(`groups.${id}`),
    keys: fieldOrder.filter(key => groupOf[key] === id && itemsByKey.value[key])
  })).filter(group => group.keys.length)
})

function labelFor(key: string) {
  return host.t(`fields.${key}.label`)
}
function descriptionFor(key: string) {
  return host.t(`fields.${key}.description`)
}
function valueOf(key: string) {
  return props.context.values[key] ?? itemsByKey.value[key]?.value ?? ''
}
function setValue(key: string, value: string | number | boolean) {
  if (typeof value === 'boolean') {
    props.context.updateValue(key, value ? 'true' : 'false')
    return
  }
  props.context.updateValue(key, String(value ?? ''))
}
function isBoolean(key: string) {
  return itemsByKey.value[key]?.type === 'boolean'
}
function isNumber(key: string) {
  return itemsByKey.value[key]?.type === 'number'
}
function recommendedLabel(key: string) {
  const item = itemsByKey.value[key]
  const raw = item?.recommendedValue || item?.default || ''
  if (!`${raw}`.trim()) {
    return host.t('emptyRecommended')
  }
  return host.t('recommended', { value: raw })
}
async function onSave() {
  await props.context.save()
  host.toast({ title: host.t('save'), description: host.t('savedHint'), kind: 'success' })
}
</script>

<template>
  <div class="space-y-4">
    <section class="rounded-lg border border-emerald-200 bg-emerald-50/80 p-4 text-sm text-emerald-950 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-100">
      <h3 class="text-base font-bold">{{ host.t('recommendedTitle') }}</h3>
      <p class="mt-1 max-w-3xl text-sm text-emerald-800 dark:text-emerald-200">
        {{ host.t('recommendedDescription') }}
      </p>
    </section>

    <UCard
      class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900"
      :ui="{ footer: 'sticky bottom-0 z-20 border-t border-slate-200 bg-white/95 p-4 backdrop-blur-sm dark:border-zinc-800 dark:bg-zinc-900/95 sm:px-6' }"
    >
      <template #header>
        <div class="space-y-4">
          <div>
            <h2 class="text-base font-bold text-slate-900 dark:text-white">{{ host.t('title') }}</h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-zinc-400">{{ host.t('intro') }}</p>
          </div>
          <div class="flex flex-wrap gap-2" role="tablist">
            <UButton
              v-for="tab in tabs"
              :key="tab.id"
              type="button"
              size="sm"
              :color="activeTab === tab.id ? 'primary' : 'neutral'"
              :variant="activeTab === tab.id ? 'solid' : 'outline'"
              :leading-icon="tab.icon"
              @click="activeTab = tab.id"
            >
              {{ tab.label }}
            </UButton>
          </div>
        </div>
      </template>

      <div class="grid max-w-3xl gap-8">
        <section v-for="group in groupsOnTab" :key="group.id" class="space-y-4">
          <div class="border-b border-slate-200 pb-2 dark:border-zinc-800">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-zinc-100">{{ group.title }}</h3>
          </div>
          <div v-for="key in group.keys" :key="key" class="grid gap-2">
            <UFormField :label="labelFor(key)" :description="descriptionFor(key)" :name="`theme-${key}`">
              <label
                v-if="isBoolean(key)"
                class="inline-flex min-h-10 items-center gap-2 rounded-md border border-slate-200 px-3 text-sm text-slate-700 dark:border-zinc-800 dark:text-zinc-200"
              >
                <input
                  type="checkbox"
                  class="size-4 accent-[var(--sf-accent)]"
                  :checked="valueOf(key) === 'true'"
                  @change="setValue(key, ($event.target as HTMLInputElement).checked)"
                >
                <span>{{ host.t('enabled') }}</span>
              </label>
              <UInput
                v-else
                :model-value="valueOf(key)"
                class="w-full"
                :type="isNumber(key) ? 'number' : 'text'"
                :placeholder="itemsByKey[key]?.placeholder || ''"
                @update:model-value="setValue(key, $event as string)"
              />
            </UFormField>
            <div class="flex flex-wrap items-center gap-2 text-xs text-slate-500 dark:text-zinc-400">
              <UBadge color="neutral" variant="outline" class="font-mono">{{ key }}</UBadge>
              <span>{{ recommendedLabel(key) }}</span>
            </div>
          </div>
        </section>
      </div>

      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <span class="text-xs text-slate-500 dark:text-zinc-400">{{ host.t('footerHint') }}</span>
          <div class="flex items-center gap-3">
            <UButton
              type="button"
              color="neutral"
              variant="outline"
              leading-icon="i-lucide-rotate-ccw"
              :loading="context.saving"
              :disabled="context.loading || context.recommendedApplied"
              @click="context.reset()"
            >
              {{ host.t('reset') }}
            </UButton>
            <UButton
              type="button"
              leading-icon="i-lucide-save"
              :loading="context.saving"
              :disabled="context.loading"
              @click="onSave"
            >
              {{ host.t('save') }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </div>
</template>
