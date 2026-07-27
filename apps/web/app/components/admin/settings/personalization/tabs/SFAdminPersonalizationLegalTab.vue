<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import SFAdminFormFooter from '~/components/admin/SFAdminFormFooter.vue'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const state = useAdminOptionForm(toRef(props, 'items'), map => ({
  termsZh: map['legal.terms.body.zh-CN']?.value || '',
  termsEn: map['legal.terms.body.en-US']?.value || '',
  privacyZh: map['legal.privacy.body.zh-CN']?.value || '',
  privacyEn: map['legal.privacy.body.en-US']?.value || '',
  guidelinesZh: map['legal.guidelines.body.zh-CN']?.value || '',
  guidelinesEn: map['legal.guidelines.body.en-US']?.value || ''
}), value => [
  { name: 'legal.terms.body.zh-CN', value: value.termsZh },
  { name: 'legal.terms.body.en-US', value: value.termsEn },
  { name: 'legal.privacy.body.zh-CN', value: value.privacyZh },
  { name: 'legal.privacy.body.en-US', value: value.privacyEn },
  { name: 'legal.guidelines.body.zh-CN', value: value.guidelinesZh },
  { name: 'legal.guidelines.body.en-US', value: value.guidelinesEn }
], () => ({ termsZh: '', termsEn: '', privacyZh: '', privacyEn: '', guidelinesZh: '', guidelinesEn: '' }), items => emit('saved', items), {
  saved: t('admin.siteChrome.legal.saved'), saveFailed: t('admin.siteChrome.legal.saveFailed'), reset: t('admin.personalization.resetChanges'), restored: t('admin.personalization.restoreRecommended')
})
</script>

<template>
  <form @submit.prevent="state.save">
    <UCard class="border-slate-200 bg-white dark:border-zinc-800 dark:bg-zinc-900" :ui="{ footer: 'sticky bottom-0 z-20 bg-white/95 dark:bg-zinc-900/95 border-t p-4' }">
      <template #header><div><h3 class="text-base font-bold">{{ t('admin.siteChrome.legal.title') }}</h3><p class="mt-1 text-xs text-muted">{{ t('admin.siteChrome.legal.description') }}</p></div></template>
      <div class="grid max-w-5xl gap-5">
        <div class="grid gap-3 lg:grid-cols-2"><UFormField :label="t('admin.siteChrome.legal.termsZh')"><UTextarea v-model="state.form.termsZh" :rows="8" class="w-full" /></UFormField><UFormField :label="t('admin.siteChrome.legal.termsEn')"><UTextarea v-model="state.form.termsEn" :rows="8" class="w-full" /></UFormField></div>
        <div class="grid gap-3 lg:grid-cols-2"><UFormField :label="t('admin.siteChrome.legal.privacyZh')"><UTextarea v-model="state.form.privacyZh" :rows="8" class="w-full" /></UFormField><UFormField :label="t('admin.siteChrome.legal.privacyEn')"><UTextarea v-model="state.form.privacyEn" :rows="8" class="w-full" /></UFormField></div>
        <div class="grid gap-3 lg:grid-cols-2"><UFormField :label="t('admin.siteChrome.legal.guidelinesZh')"><UTextarea v-model="state.form.guidelinesZh" :rows="8" class="w-full" /></UFormField><UFormField :label="t('admin.siteChrome.legal.guidelinesEn')"><UTextarea v-model="state.form.guidelinesEn" :rows="8" class="w-full" /></UFormField></div>
      </div>
      <template #footer><SFAdminFormFooter :saving="state.saving.value" :show-unsaved-alert="state.hasChanges.value" :submit-text="t('admin.siteChrome.legal.save')" @reset="state.resetChanges" /></template>
    </UCard>
  </form>
</template>
