<script setup lang="ts">
import { usePermissions } from '~/composables/identity/usePermissions'
import type { ForumSettings } from '~/utils/forum/forumTaxonomy'
import { useAdminForumSettingsTab } from '~/composables/admin/settings/useAdminForumSettingsTab'
import SFAdminForumTabFrame from '../SFAdminForumTabFrame.vue'

const props = defineProps<{ settings: ForumSettings }>()
const emit = defineEmits<{ saved: [settings: ForumSettings] }>()
const { t } = useI18n()
const { can } = usePermissions()
const canManage = computed(() => can('forum.settings.manage') || can('settings.manage'))
const tab = useAdminForumSettingsTab(toRef(props, 'settings'), ['excerptRuneLimit'], () => canManage.value, settings => emit('saved', settings))
</script>

<template>
  <SFAdminForumTabFrame tab="reading" :dirty="tab.hasChanges.value" :validation-key="tab.validationKey.value" :saving="tab.saving.value" :restoring="tab.restoring.value" :can-save="tab.canSave.value" @save="tab.saveCurrent" @reset="tab.resetChanges" @restore="tab.restoreCurrent">
    <UFormField :label="t('admin.forum.settings.excerptRuneLimit')" name="excerpt-limit"><UInputNumber v-model="tab.form.excerptRuneLimit" :min="40" :max="500" :disabled="!canManage" class="w-full" /></UFormField>
    <section class="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-950/60"><strong>{{ t('admin.forum.settings.readingNoteTitle') }}</strong><p class="mt-1 text-xs text-muted">{{ t('admin.forum.settings.readingNoteBody') }}</p></section>
  </SFAdminForumTabFrame>
</template>
