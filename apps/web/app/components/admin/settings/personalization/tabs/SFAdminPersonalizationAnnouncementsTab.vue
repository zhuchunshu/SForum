<script setup lang="ts">
import { apiErrorMessage } from '~/composables/useApiClient'
import { useSiteChromeApi, type SiteAnnouncement, type SiteAnnouncementStyle } from '~/composables/admin/useSiteChromeApi'

const { t, locale } = useI18n()
const toast = useToast()
const api = useSiteChromeApi()
const loading = ref(false)
const items = ref<SiteAnnouncement[]>([])
const draft = reactive({ titleZhCN: '', titleEnUS: '', bodyZhCN: '', bodyEnUS: '', style: 'info' as SiteAnnouncementStyle, href: '', dismissible: true, position: 0, enabled: true })
const styles = computed(() => (['info', 'success', 'warning', 'danger'] as SiteAnnouncementStyle[]).map(value => ({ value, label: t(`admin.siteChrome.announcements.styles.${value}`) })))
defineExpose({ refresh: load, loading })
onMounted(load)

async function load() {
  loading.value = true
  try { items.value = sort(await api.listAdminAnnouncements()) } catch (error) { failure(error) } finally { loading.value = false }
}
async function add() {
  try {
    const created = await api.createAnnouncement({ ...draft, titleZhCN: draft.titleZhCN.trim(), titleEnUS: draft.titleEnUS.trim(), bodyZhCN: draft.bodyZhCN.trim(), bodyEnUS: draft.bodyEnUS.trim(), href: draft.href.trim() })
    items.value = sort([...items.value, created])
    Object.assign(draft, { titleZhCN: '', titleEnUS: '', bodyZhCN: '', bodyEnUS: '', href: '' })
    success(t('admin.siteChrome.announcements.created'))
  } catch (error) { failure(error) }
}
async function toggle(item: SiteAnnouncement) {
  try { const updated = await api.updateAnnouncement(item.id, { enabled: !item.enabled }); items.value = sort(items.value.map(row => row.id === updated.id ? updated : row)); success(t('admin.siteChrome.announcements.updated')) } catch (error) { failure(error) }
}
async function remove(item: SiteAnnouncement) {
  try { await api.deleteAnnouncement(item.id); items.value = items.value.filter(row => row.id !== item.id); success(t('admin.siteChrome.announcements.deleted')) } catch (error) { failure(error) }
}
function sort(rows: SiteAnnouncement[]) { return [...rows].sort((a, b) => a.position - b.position || a.id - b.id) }
function english() { return String(locale.value).toLowerCase().startsWith('en') }
function title(item: SiteAnnouncement) { return english() ? item.titleEnUS || item.titleZhCN : item.titleZhCN || item.titleEnUS }
function body(item: SiteAnnouncement) { return english() ? item.bodyEnUS || item.bodyZhCN : item.bodyZhCN || item.bodyEnUS }
function success(title: string) { toast.add({ color: 'success', icon: 'i-lucide-check', title, duration: 10000 }) }
function failure(error: unknown) { toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: apiErrorMessage(error) || t('admin.siteChrome.announcements.saveFailed') }) }
</script>

<template>
  <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
    <template #header><div><h3 class="text-base font-bold">{{ t('admin.siteChrome.announcements.title') }}</h3><p class="mt-1 text-xs text-muted">{{ t('admin.siteChrome.announcements.description') }}</p></div></template>
    <div class="mb-4 grid gap-3 rounded-md border border-slate-200 bg-slate-50 p-3 dark:border-zinc-800 dark:bg-zinc-950/60 md:grid-cols-2">
      <UInput v-model="draft.titleZhCN" :placeholder="t('admin.siteChrome.announcements.titleZh')" /><UInput v-model="draft.titleEnUS" :placeholder="t('admin.siteChrome.announcements.titleEn')" />
      <UTextarea v-model="draft.bodyZhCN" :rows="3" :placeholder="t('admin.siteChrome.announcements.bodyZh')" /><UTextarea v-model="draft.bodyEnUS" :rows="3" :placeholder="t('admin.siteChrome.announcements.bodyEn')" />
      <USelect v-model="draft.style" :items="styles" value-key="value" label-key="label" /><UInput v-model="draft.href" icon="i-lucide-link" />
      <div class="flex gap-3 md:col-span-2"><UInput v-model.number="draft.position" type="number" class="w-36" /><UButton leading-icon="i-lucide-plus" @click="add">{{ t('admin.siteChrome.announcements.add') }}</UButton></div>
    </div>
    <div v-if="loading" class="py-8 text-center text-sm text-muted">{{ t('admin.common.loading') }}</div>
    <div v-else-if="items.length === 0" class="py-8 text-center text-sm text-muted">{{ t('admin.siteChrome.announcements.empty') }}</div>
    <ul v-else class="divide-y divide-slate-200 dark:divide-zinc-800">
      <li v-for="item in items" :key="item.id" class="flex justify-between gap-3 py-3">
        <div><strong>{{ title(item) }}</strong><p class="mt-1 text-sm text-muted">{{ body(item) }}</p></div>
        <div class="flex shrink-0 gap-2"><UButton size="sm" color="neutral" variant="outline" @click="toggle(item)">{{ item.enabled ? t('admin.siteChrome.disable') : t('admin.siteChrome.enable') }}</UButton><UButton size="sm" color="error" variant="soft" icon="i-lucide-trash-2" :aria-label="t('admin.siteChrome.delete')" @click="remove(item)" /></div>
      </li>
    </ul>
  </UCard>
</template>
