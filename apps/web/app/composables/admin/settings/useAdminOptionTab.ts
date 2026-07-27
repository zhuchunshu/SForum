import type { AdminWebOption, WebOption } from '~/composables/useWebOptions'

export function adminOptionMap(items: AdminWebOption[]) {
  return Object.fromEntries(items.map(item => [item.name, item])) as Record<string, AdminWebOption>
}

export function useAdminOptionTab(
  emitSaved: (items: AdminWebOption[]) => void
) {
  const { saveMany } = useWebOptions()

  async function saveOptions(items: WebOption[]) {
    const updated = await saveMany(items)
    emitSaved(updated)
  }

  return { saveOptions }
}
