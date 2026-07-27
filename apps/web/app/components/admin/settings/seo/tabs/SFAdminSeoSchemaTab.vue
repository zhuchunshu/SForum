<script setup lang="ts">
import type { AdminWebOption } from '~/composables/useWebOptions'
import { enabledOptionValue, normalizeEnabledOption } from '~/composables/useWebOptions'
import { useAdminOptionForm } from '~/composables/admin/settings/useAdminOptionForm'
import SFAdminSeoTabFrame from '../SFAdminSeoTabFrame.vue'

const props = defineProps<{ items: AdminWebOption[] }>()
const emit = defineEmits<{ saved: [items: AdminWebOption[]] }>()
const { t } = useI18n()
const form = useAdminOptionForm(toRef(props, 'items'), map => ({
  enabled: normalizeEnabledOption(map['seo.schema_org.enabled']?.value, true),
  searchAction: normalizeEnabledOption(map['seo.schema_org.search_action_enabled']?.value, true),
  discussion: normalizeEnabledOption(map['seo.schema_org.discussion_enabled']?.value, true),
  logoUrl: map['seo.schema_org.organization_logo_url']?.value || ''
}), value => [
  { name: 'seo.schema_org.enabled', value: enabledOptionValue(value.enabled) },
  { name: 'seo.schema_org.search_action_enabled', value: enabledOptionValue(value.searchAction) },
  { name: 'seo.schema_org.discussion_enabled', value: enabledOptionValue(value.discussion) },
  { name: 'seo.schema_org.organization_logo_url', value: value.logoUrl }
], () => ({ enabled: true, searchAction: true, discussion: true, logoUrl: '' }), items => emit('saved', items), {
  saved: t('admin.seo.saved'), saveFailed: t('admin.seo.saveFailed'), reset: t('admin.seo.resetChanges'), restored: t('admin.seo.recommendedRestored')
})
</script>

<template>
  <SFAdminSeoTabFrame tab="schema" :dirty="form.hasChanges.value" :saving="form.saving.value" @save="form.save" @reset="form.resetChanges">
    <div class="grid max-w-5xl gap-5">
      <div class="flex justify-end"><UButton type="button" color="neutral" variant="outline" icon="i-lucide-rotate-ccw" @click="form.restoreRecommended">{{ t('admin.seo.restoreRecommended') }}</UButton></div>
      <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.enabled" type="checkbox" class="size-4"><strong>{{ t('admin.seo.schemaEnabled') }}</strong></label>
      <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.searchAction" type="checkbox" class="size-4"><strong>{{ t('admin.seo.schemaSearchAction') }}</strong></label>
      <label class="flex gap-3 rounded-md border border-slate-200 p-4 dark:border-zinc-800"><input v-model="form.form.discussion" type="checkbox" class="size-4"><strong>{{ t('admin.seo.schemaDiscussion') }}</strong></label>
      <UFormField :label="t('admin.seo.organizationLogoUrl')" name="organization-logo"><UInput v-model="form.form.logoUrl" type="url" class="w-full" /></UFormField>
    </div>
  </SFAdminSeoTabFrame>
</template>
