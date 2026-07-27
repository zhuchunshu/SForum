<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { isLocalSiteUrl, normalizeEnabledOption } from '~/composables/useWebOptions'
import { adminOptionMap } from '~/composables/admin/settings/useAdminOptionTab'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'

const props = defineProps<{ items: AdminWebOption[], siteUrl: string }>()
const { t } = useI18n()
const map = computed(() => adminOptionMap(props.items))
const indexing = computed(() => normalizeEnabledOption(map.value['seo.allow_indexing']?.value, true) && !isLocalSiteUrl(props.siteUrl))
const sitemap = computed(() => normalizeEnabledOption(map.value['seo.sitemap.enabled']?.value, true))
const schema = computed(() => normalizeEnabledOption(map.value['seo.schema_org.enabled']?.value, true))
const base = computed(() => props.siteUrl.replace(/\/+$/, '') || 'http://127.0.0.1:3000')
const items = computed(() => [
  { label: t('admin.seo.overview.indexing'), value: indexing.value ? t('admin.seo.statusEnabled') : t('admin.seo.statusDisabled'), icon: indexing.value ? 'i-lucide-check-circle-2' : 'i-lucide-circle-slash' },
  { label: t('admin.seo.overview.canonical'), value: `${base.value}/`, icon: 'i-lucide-link' },
  { label: t('admin.seo.overview.sitemap'), value: sitemap.value ? `${base.value}/sitemap.xml` : t('admin.seo.statusDisabled'), icon: 'i-lucide-map' },
  { label: t('admin.seo.overview.schema'), value: schema.value ? t('admin.seo.statusEnabled') : t('admin.seo.statusDisabled'), icon: 'i-lucide-braces' }
])
</script>

<template>
  <SFAdminSeoTabFrame tab="overview" readonly>
    <div class="grid gap-5">
      <UAlert v-if="isLocalSiteUrl(siteUrl)" color="warning" variant="soft" icon="i-lucide-shield-alert" :title="t('admin.seo.localProtectionTitle')" :description="t('admin.seo.localProtectionDescription')" />
      <div class="grid gap-3 md:grid-cols-2">
        <div v-for="item in items" :key="item.label" class="rounded-md border border-slate-200 bg-slate-50 p-4 dark:border-zinc-800 dark:bg-zinc-950/60">
          <div class="flex items-center gap-2 text-xs font-semibold text-muted"><UIcon :name="item.icon" class="size-4" />{{ item.label }}</div>
          <div class="mt-2 break-all text-sm font-medium">{{ item.value }}</div>
        </div>
      </div>
    </div>
  </SFAdminSeoTabFrame>
</template>
