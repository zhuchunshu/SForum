<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import SFAdminBrandAssetField from '~/components/admin/settings/personalization/SFAdminBrandAssetField.vue'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const state = useAdminOptionForm(toRef(props, 'items'), map => ({
  logoUrl: map['site.logo_url']?.value || '',
  logoAttachmentId: map['site.logo_attachment_id']?.value || '',
  faviconUrl: map['site.favicon_url']?.value || '',
  faviconAttachmentId: map['site.favicon_attachment_id']?.value || '',
  appleTouchIconUrl: map['site.apple_touch_icon_url']?.value || '',
  appleTouchIconAttachmentId: map['site.apple_touch_icon_attachment_id']?.value || ''
}), value => [
  { name: 'site.logo_url', value: value.logoUrl.trim() },
  { name: 'site.logo_attachment_id', value: value.logoAttachmentId.trim() },
  { name: 'site.favicon_url', value: value.faviconUrl.trim() },
  { name: 'site.favicon_attachment_id', value: value.faviconAttachmentId.trim() },
  { name: 'site.apple_touch_icon_url', value: value.appleTouchIconUrl.trim() },
  { name: 'site.apple_touch_icon_attachment_id', value: value.appleTouchIconAttachmentId.trim() }
], () => ({ logoUrl: '', logoAttachmentId: '', faviconUrl: '', faviconAttachmentId: '', appleTouchIconUrl: '', appleTouchIconAttachmentId: '' }), items => emit('saved', items), {
  saved: t('admin.siteChrome.brand.saved'), saveFailed: t('admin.siteChrome.brand.saveFailed'), reset: t('admin.personalization.resetChanges'), restored: t('admin.siteChrome.brand.restoreEmpty')
})
</script>

<template>
  <form @submit.prevent="state.save">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 border-t p-4' }">
      <template #header><div><h3 class="text-base font-bold">{{ t('admin.siteChrome.brand.title') }}</h3><p class="mt-1 text-xs text-muted">{{ t('admin.siteChrome.brand.description') }}</p></div></template>
      <div class="grid max-w-5xl gap-5">
        <SFAdminBrandAssetField
          v-model:url="state.form.logoUrl"
          v-model:attachment-id="state.form.logoAttachmentId"
          context="logo"
          :label="t('admin.siteChrome.brand.logo')"
          :url-label="t('admin.siteChrome.brand.logoUrl')"
          :attachment-id-label="t('admin.siteChrome.brand.logoAttachmentId')"
          icon="i-lucide-image"
          preview-shape="wide"
          :disabled="state.saving.value"
        />
        <SFAdminBrandAssetField
          v-model:url="state.form.faviconUrl"
          v-model:attachment-id="state.form.faviconAttachmentId"
          context="favicon"
          :label="t('admin.siteChrome.brand.favicon')"
          :url-label="t('admin.siteChrome.brand.faviconUrl')"
          :attachment-id-label="t('admin.siteChrome.brand.faviconAttachmentId')"
          icon="i-lucide-bookmark"
          :disabled="state.saving.value"
        />
        <SFAdminBrandAssetField
          v-model:url="state.form.appleTouchIconUrl"
          v-model:attachment-id="state.form.appleTouchIconAttachmentId"
          context="apple-touch-icon"
          :label="t('admin.siteChrome.brand.appleTouchIcon')"
          :url-label="t('admin.siteChrome.brand.appleTouchUrl')"
          :attachment-id-label="t('admin.siteChrome.brand.appleTouchAttachmentId')"
          icon="i-lucide-smartphone"
          :disabled="state.saving.value"
        />
      </div>
      <template #footer><SFAdminFormFooter :saving="state.saving.value" :show-unsaved-alert="state.hasChanges.value" :submit-text="t('admin.siteChrome.brand.save')" :reset-text="t('admin.siteChrome.brand.restoreEmpty')" @reset="state.restoreRecommended" /></template>
    </UCard>
  </form>
</template>
