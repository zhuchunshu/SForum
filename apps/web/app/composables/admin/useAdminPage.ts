import { onMounted } from 'vue'
import { requireAdminPageDefinition } from '~/config/adminModules'
import { useAdminTabs } from '~/composables/admin/useAdminTabs'

export const useAdminPage = (id: string) => {
  const page = requireAdminPageDefinition(id)
  const adminTabs = useAdminTabs()

  onMounted(() => {
    adminTabs.openTab(page.id)
  })

  return page
}
