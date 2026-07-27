<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { enabledOptionValue, isLocalSiteUrl, normalizeEnabledOption, parseSEORobotsPathList } from '~/composables/useWebOptions'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'

const props = defineProps<{ items: AdminWebOption[], siteUrl: string }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm(
  toRef(props, 'items'),
  map => ({
    allowIndexing: normalizeEnabledOption(map['seo.allow_indexing']?.value, true),
    extraAllow: map['seo.robots.extra_allow']?.value || '',
    extraDisallow: map['seo.robots.extra_disallow']?.value || '',
    blockAiBots: normalizeEnabledOption(map['seo.robots.block_ai_bots']?.value),
    blockNonSeoBots: normalizeEnabledOption(map['seo.robots.block_non_seo_bots']?.value)
  }),
  value => [
    { name: 'seo.allow_indexing', value: enabledOptionValue(value.allowIndexing) },
    { name: 'seo.robots.extra_allow', value: value.extraAllow },
    { name: 'seo.robots.extra_disallow', value: value.extraDisallow },
    { name: 'seo.robots.block_ai_bots', value: enabledOptionValue(value.blockAiBots) },
    { name: 'seo.robots.block_non_seo_bots', value: enabledOptionValue(value.blockNonSeoBots) }
  ],
  () => ({ allowIndexing: true, extraAllow: '', extraDisallow: '', blockAiBots: false, blockNonSeoBots: false }),
  items => emit('saved', items),
  { saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored') }
)
const preview = computed(() => {
  if (!form.form.allowIndexing || isLocalSiteUrl(props.siteUrl)) return 'User-agent: *\nDisallow: /'
  const lines = ['User-agent: *', 'Disallow: /api/', 'Disallow: /login', 'Disallow: /register', 'Disallow: /control-panel/']
  parseSEORobotsPathList(form.form.extraAllow).forEach(path => lines.push(`Allow: ${path}`))
  parseSEORobotsPathList(form.form.extraDisallow).forEach(path => lines.push(`Disallow: ${path}`))
  lines.push(`Sitemap: ${props.siteUrl.replace(/\/+$/, '')}/sitemap.xml`)
  return lines.join('\n')
})
</script>

<template>
  <SFAdminSeoTabFrame tab="robots" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <div class="grid max-w-5xl gap-5">
      <div class="flex justify-end"><UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="form.restoreRecommended">{{ t('admin.seo.restoreRecommended') }}</UButton></div>
      <label class="flex items-start gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.allowIndexing" type="checkbox" class="mt-1 size-4"><span><strong class="block text-sm">{{ t('admin.seo.allowIndexing') }}</strong><span class="text-xs text-muted">{{ t('admin.seo.allowIndexingDescription') }}</span></span></label>
      <div class="grid gap-4 md:grid-cols-2">
        <UFormField :label="t('admin.seo.extraAllow')" name="extra-allow"><UTextarea v-model="form.form.extraAllow" :rows="6" class="w-full font-mono text-xs" /></UFormField>
        <UFormField :label="t('admin.seo.extraDisallow')" name="extra-disallow"><UTextarea v-model="form.form.extraDisallow" :rows="6" class="w-full font-mono text-xs" /></UFormField>
      </div>
      <div class="grid gap-3 md:grid-cols-2">
        <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.blockAiBots" type="checkbox" class="size-4"><span class="text-sm">{{ t('admin.seo.blockAiBots') }}</span></label>
        <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.blockNonSeoBots" type="checkbox" class="size-4"><span class="text-sm">{{ t('admin.seo.blockNonSeoBots') }}</span></label>
      </div>
      <pre class="overflow-x-auto rounded-md border border-slate-200 bg-slate-50 p-4 text-xs dark:border-zinc-800 dark:bg-zinc-950">{{ preview }}</pre>
    </div>
  </SFAdminSeoTabFrame>
</template>
