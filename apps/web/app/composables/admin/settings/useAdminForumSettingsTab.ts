import type { Ref } from 'vue'
import type { AdminForumSettingKey } from '~/utils/admin/adminForum'
import type { ForumSettings } from '~/utils/forum/forumTaxonomy'
import {
  createAdminForumApi,
  createDefaultForumSettings,
  forumSettingsPartialPayload,
  forumSettingsValidationError,
  normalizeForumSettings
} from '~/utils/admin/adminForum'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

export function useAdminForumSettingsTab(
  settings: Ref<ForumSettings>,
  keys: readonly AdminForumSettingKey[],
  canWrite: (key: AdminForumSettingKey) => boolean,
  emitSaved: (settings: ForumSettings) => void
) {
  const { t } = useI18n()
  const toast = useToast()
  const { request } = useApiClient()
  const forumApi = createAdminForumApi(request)
  const saveSection = useSettingsSection()
  const restoreSection = useSettingsSection()
  const form = reactive(createDefaultForumSettings())
  const allowedKeys = computed(() => keys.filter(canWrite))
  const initial = computed(() => forumSettingsPartialPayload(normalizeForumSettings(settings.value), keys))
  const current = computed(() => forumSettingsPartialPayload(form, keys))
  const hasChanges = computed(() => JSON.stringify(current.value) !== JSON.stringify(initial.value))
  const validationKey = computed(() => forumSettingsValidationError(normalizeForumSettings({
    ...settings.value,
    ...current.value
  })))
  const canSave = computed(() => allowedKeys.value.length > 0)

  watch(settings, resetFromSettings, { immediate: true, deep: true })

  function resetFromSettings() {
    Object.assign(form, normalizeForumSettings(settings.value))
  }

  async function saveCurrent() {
    if (!canSave.value || validationKey.value) return
    await saveSection.runSave({
      successTitle: t('admin.forum.settings.saved'),
      failureTitle: t('admin.forum.settings.saveFailed'),
      save: async () => {
        const updated = await forumApi.updateSettings(forumSettingsPartialPayload(form, allowedKeys.value))
        Object.assign(form, updated)
        emitSaved(updated)
      }
    })
  }

  async function restoreCurrent() {
    if (!canSave.value) return
    const recommended = createDefaultForumSettings()
    await restoreSection.runSave({
      successTitle: t('admin.forum.settings.restored'),
      failureTitle: t('admin.forum.settings.restoreFailed'),
      save: async () => {
        const updated = await forumApi.updateSettings(forumSettingsPartialPayload(recommended, allowedKeys.value))
        Object.assign(form, updated)
        emitSaved(updated)
      }
    })
  }

  function resetChanges() {
    Object.assign(form, initial.value)
    toast.add({
      color: 'neutral',
      icon: 'i-lucide-undo-2',
      title: t('admin.forum.settings.discarded'),
      duration: 10000
    })
  }

  return {
    form,
    hasChanges,
    validationKey,
    canSave,
    saving: saveSection.saving,
    restoring: restoreSection.saving,
    saveCurrent,
    restoreCurrent,
    resetChanges
  }
}
