<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { normalizeSEOVerificationToken } from '~/composables/useWebOptions'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm(toRef(props, 'items'), map => ({
  google: map['seo.google_verification']?.value || '',
  bing: map['seo.bing_verification']?.value || '',
  baidu: map['seo.baidu_verification']?.value || '',
  yandex: map['seo.yandex_verification']?.value || ''
}), value => [
  { name: 'seo.google_verification', value: normalizeSEOVerificationToken(value.google) },
  { name: 'seo.bing_verification', value: normalizeSEOVerificationToken(value.bing) },
  { name: 'seo.baidu_verification', value: normalizeSEOVerificationToken(value.baidu) },
  { name: 'seo.yandex_verification', value: normalizeSEOVerificationToken(value.yandex) }
], () => ({ google: '', bing: '', baidu: '', yandex: '' }), items => emit('saved', items), {
  saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored')
})
</script>

<template>
  <SFAdminSeoTabFrame tab="verification" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <div class="grid max-w-5xl gap-5">
      <div class="flex justify-end"><UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="form.restoreRecommended">{{ t('admin.seo.restoreRecommended') }}</UButton></div>
      <UFormField :label="t('admin.seo.googleVerification')" name="google"><UInput v-model="form.form.google" maxlength="120" class="w-full" /></UFormField>
      <UFormField :label="t('admin.seo.bingVerification')" name="bing"><UInput v-model="form.form.bing" maxlength="120" class="w-full" /></UFormField>
      <UFormField :label="t('admin.seo.baiduVerification')" name="baidu"><UInput v-model="form.form.baidu" maxlength="120" class="w-full" /></UFormField>
      <UFormField :label="t('admin.seo.yandexVerification')" name="yandex"><UInput v-model="form.form.yandex" maxlength="120" class="w-full" /></UFormField>
    </div>
  </SFAdminSeoTabFrame>
</template>
