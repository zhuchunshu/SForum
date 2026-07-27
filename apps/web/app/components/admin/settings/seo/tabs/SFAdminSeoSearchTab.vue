<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'
import SFSEOSearchAppearance from './SFSEOSearchAppearance.vue'

const props = defineProps<{ items: AdminWebOption[], productSiteName: string, siteUrl: string }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm(
  toRef(props, 'items'),
  map => ({
    inheritSiteName: normalizeEnabledOption(map['seo.site.inherit_site_name']?.value, true),
    seoSiteName: map['seo.site.name']?.value || '',
    homeTitle: map['seo.home.title']?.value || '',
    homeDescription: map['seo.home.description']?.value || map['seo.meta_description']?.value || '',
    homeKeywords: map['seo.home.keywords']?.value || map['seo.meta_keywords']?.value || '',
    homeOGImageUrl: map['seo.home.og_image_url']?.value || map['seo.og_image_url']?.value || ''
  }),
  value => [
    { name: 'seo.site.inherit_site_name', value: enabledOptionValue(value.inheritSiteName) },
    { name: 'seo.site.name', value: value.seoSiteName },
    { name: 'seo.home.title', value: value.homeTitle },
    { name: 'seo.home.description', value: value.homeDescription },
    { name: 'seo.home.keywords', value: value.homeKeywords },
    { name: 'seo.home.og_image_url', value: value.homeOGImageUrl }
  ],
  () => ({ inheritSiteName: true, seoSiteName: '', homeTitle: '', homeDescription: '', homeKeywords: '', homeOGImageUrl: '' }),
  items => emit('saved', items),
  { saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored') }
)
</script>

<template>
  <SFAdminSeoTabFrame tab="search" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <SFSEOSearchAppearance
      v-model:inherit-site-name="form.form.inheritSiteName"
      v-model:seo-site-name="form.form.seoSiteName"
      v-model:home-title="form.form.homeTitle"
      v-model:home-description="form.form.homeDescription"
      v-model:home-keywords="form.form.homeKeywords"
      v-model:home-o-g-image-url="form.form.homeOGImageUrl"
      :product-site-name="productSiteName"
      :site-url="siteUrl"
      @restore="form.restoreRecommended"
    />
  </SFAdminSeoTabFrame>
</template>
