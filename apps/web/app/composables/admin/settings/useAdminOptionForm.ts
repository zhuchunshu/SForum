import type { Ref, UnwrapNestedRefs } from 'vue'
import type { AdminWebOption, WebOption } from '~/composables/useWebOptions'
import { adminOptionMap, useAdminOptionTab } from '~/composables/admin/settings/useAdminOptionTab'
import { useSettingsSection } from '~/composables/settings/useSettingsSection'

export function useAdminOptionForm<T extends Record<string, unknown>>(
  items: Ref<AdminWebOption[]>,
  readForm: (map: Record<string, AdminWebOption>) => T,
  buildPayload: (form: UnwrapNestedRefs<T>) => WebOption[],
  recommended: () => T,
  emitSaved: (items: AdminWebOption[]) => void,
  messages: {
    saved: string
    saveFailed: string
    reset: string
    restored: string
  }
) {
  const toast = useToast()
  const section = useSettingsSection()
  const { saveOptions } = useAdminOptionTab(emitSaved)
  const map = computed(() => adminOptionMap(items.value))
  const initial = computed(() => readForm(map.value))
  const form = reactive(clone(initial.value)) as UnwrapNestedRefs<T>
  const hasChanges = computed(() => JSON.stringify(form) !== JSON.stringify(initial.value))

  watch(items, resetFromItems, { immediate: true, deep: true })

  function resetFromItems() {
    Object.assign(form, clone(initial.value))
  }

  async function save() {
    await section.runSave({
      successTitle: messages.saved,
      failureTitle: messages.saveFailed,
      save: () => saveOptions(buildPayload(form))
    })
  }

  function resetChanges() {
    resetFromItems()
    toast.add({ color: 'neutral', icon: 'i-lucide-undo-2', title: messages.reset, duration: 10000 })
  }

  function restoreRecommended() {
    section.runRestore({
      title: messages.restored,
      apply: () => Object.assign(form, clone(recommended()))
    })
  }

  return {
    form,
    hasChanges,
    saving: section.saving,
    save,
    resetChanges,
    restoreRecommended
  }
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
