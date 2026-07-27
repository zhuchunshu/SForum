<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'

const props = defineProps<{ items: AdminWebOption[], siteUrl: string }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm(toRef(props, 'items'), map => ({
  enabled: normalizeEnabledOption(map['seo.sitemap.enabled']?.value, true),
  staticPages: normalizeEnabledOption(map['seo.sitemap.include_static_pages']?.value, true),
  forumContent: normalizeEnabledOption(map['seo.sitemap.include_forum_content']?.value, true)
}), value => [
  { name: 'seo.sitemap.enabled', value: enabledOptionValue(value.enabled) },
  { name: 'seo.sitemap.include_static_pages', value: enabledOptionValue(value.staticPages) },
  { name: 'seo.sitemap.include_forum_content', value: enabledOptionValue(value.forumContent) }
], () => ({ enabled: true, staticPages: true, forumContent: true }), items => emit('saved', items), {
  saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored')
})
</script>

<template>
  <SFAdminSeoTabFrame tab="sitemap" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <div class="grid max-w-5xl gap-5">
      <div class="flex justify-end"><UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="form.restoreRecommended">{{ t('admin.seo.restoreRecommended') }}</UButton></div>
      <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.enabled" type="checkbox" class="size-4"><strong>{{ t('admin.seo.sitemapEnabled') }}</strong></label>
      <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.staticPages" type="checkbox" class="size-4"><strong>{{ t('admin.seo.sitemapStaticPages') }}</strong></label>
      <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.forumContent" type="checkbox" class="size-4"><strong>{{ t('admin.seo.sitemapForumContent') }}</strong></label>
      <div class="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60"><strong>{{ t('admin.seo.sitemapUrl') }}</strong><code class="mt-1 block text-xs">{{ siteUrl.replace(/\/+$/, '') }}/sitemap.xml</code></div>
    </div>
  </SFAdminSeoTabFrame>
</template>
