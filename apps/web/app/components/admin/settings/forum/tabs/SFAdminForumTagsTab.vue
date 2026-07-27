<script setup lang="ts">
import { usePermissions } from '~/composables/identity/usePermissions'
import type { ForumSettings, ForumTagCreationMode } from '~/utils/forum/forumTaxonomy'
import { useAdminForumSettingsTab } from '~/composables/admin/settings/useAdminForumSettingsTab'
import SFAdminForumTabFrame from '../SFAdminForumTabFrame.vue'

const props = defineProps<{ settings: ForumSettings }>()
const emit = defineEmits<{ saved: [settings: ForumSettings] }>()
const { t } = useI18n()
const { can } = usePermissions()
const canManage = computed(() => can('tag.manage'))
const keys = ['tagCreationMode', 'tagPublicPages', 'tagMinPerTopic', 'tagMaxPerTopic'] as const
const tab = useAdminForumSettingsTab(toRef(props, 'settings'), keys, () => canManage.value, settings => emit('saved', settings))
const modes = computed<Array<{ value: ForumTagCreationMode, label: string, description: string }>>(() => ['controlled', 'review', 'open'].map(value => ({
  value: value as ForumTagCreationMode,
  label: t(`admin.forum.settings.modes.${value}.label`),
  description: t(`admin.forum.settings.modes.${value}.description`)
})))
</script>

<template>
  <SFAdminForumTabFrame tab="tags" :dirty="tab.hasChanges.value" :validation-key="tab.validationKey.value" :saving="tab.saving.value" :restoring="tab.restoring.value" :can-save="tab.canSave.value" @save="tab.saveCurrent" @reset="tab.resetChanges" @restore="tab.restoreCurrent">
    <section class="space-y-3">
      <h3 class="text-sm font-semibold">{{ t('admin.forum.settings.tagCreationMode') }}</h3>
      <div class="grid gap-3 md:grid-cols-3">
        <button v-for="mode in modes" :key="mode.value" type="button" :disabled="!canManage" class="rounded-md border border-slate-200 p-3 text-left disabled:opacity-60 dark:border-zinc-700" @click="tab.form.tagCreationMode = mode.value"><strong class="block text-sm">{{ mode.label }}</strong><span class="mt-1 block text-xs text-muted">{{ mode.description }}</span></button>
      </div>
    </section>
    <section class="grid gap-4 border-t border-slate-200 pt-5 dark:border-zinc-800 md:grid-cols-2">
      <label class="flex items-start gap-3 rounded-md border border-slate-200 p-3 text-sm dark:border-zinc-800"><input v-model="tab.form.tagPublicPages" type="checkbox" :disabled="!canManage" class="mt-1 size-4"><span><strong class="block">{{ t('admin.forum.settings.publicTagPages') }}</strong><span class="text-xs text-muted">{{ t('admin.forum.settings.publicTagPagesHelp') }}</span></span></label>
      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField :label="t('admin.forum.settings.minTagsPerTopic')" name="min-tags"><UInputNumber v-model="tab.form.tagMinPerTopic" :min="0" :max="10" :disabled="!canManage" class="w-full" /></UFormField>
        <UFormField :label="t('admin.forum.settings.maxTagsPerTopic')" name="max-tags"><UInputNumber v-model="tab.form.tagMaxPerTopic" :min="0" :max="10" :disabled="!canManage" class="w-full" /></UFormField>
      </div>
    </section>
  </SFAdminForumTabFrame>
</template>
