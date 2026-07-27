import { computed } from 'vue'
import {
  ADMIN_DASHBOARD_PAGE_ID,
  type AdminPageDefinition,
  requireAdminPageDefinition
} from '~/config/adminModules'
import { useAdminRoutes } from '~/composables/admin/useAdminRoutes'

export interface AdminTab {
  id: string          // 相对路径，例如 '/', '/roles', '/settings'
  labelKey?: string   // 翻译键名，例如 'admin.nav.dashboard'
  label?: string      // 动态扩展页等无需翻译的标题
  to: string          // 路由路径
  icon: string        // 统一使用 i-lucide- 图标
  closable: boolean
  componentName: string // 对应的 Vue 组件 name，用于 KeepAlive :include
}

export const useAdminTabs = () => {
  const adminRoutes = useAdminRoutes()
  const dashboardPage = requireAdminPageDefinition(ADMIN_DASHBOARD_PAGE_ID)

  const tabs = useState<AdminTab[]>('admin-tabs', () => [
    createAdminTab(dashboardPage, adminRoutes.path)
  ])

  const activeTabId = useState<string>('admin-active-tab-id', () => '/')

  const cachedTabNames = computed(() => {
    return tabs.value.map(tab => tab.componentName)
  })

  const openTab = (id: string) => {
    const page = requireAdminPageDefinition(id)
    const existing = tabs.value.find(tab => tab.id === page.id)
    if (!existing) {
      tabs.value.push(createAdminTab(page, adminRoutes.path))
    }
    activeTabId.value = page.id
    return page
  }

  const openCustomTab = (tab: AdminTab) => {
    const existing = tabs.value.find(item => item.id === tab.id)
    if (!existing) {
      tabs.value.push(tab)
    } else {
      Object.assign(existing, tab)
    }
    activeTabId.value = tab.id
  }

  const activateTab = (id: string) => {
    const existing = tabs.value.find(tab => tab.id === id)
    if (!existing) {
      return false
    }

    activeTabId.value = existing.id
    return true
  }

  const closeTab = (id: string) => {
    if (id === '/') return

    const index = tabs.value.findIndex(tab => tab.id === id)
    if (index === -1) return

    tabs.value.splice(index, 1)

    if (activeTabId.value === id) {
      const fallbackTab = tabs.value[tabs.value.length - 1] || tabs.value[0]
      if (fallbackTab) {
        activeTabId.value = fallbackTab.id
        navigateTo(fallbackTab.to)
      }
    }
  }

  const resetTabs = () => {
    tabs.value = [
      createAdminTab(dashboardPage, adminRoutes.path)
    ]
    activeTabId.value = dashboardPage.id
  }

  return {
    tabs,
    activeTabId,
    cachedTabNames,
    openTab,
    openCustomTab,
    activateTab,
    closeTab,
    resetTabs
  }
}

function createAdminTab(page: AdminPageDefinition, path: (childPath?: string) => string): AdminTab {
  return {
    id: page.id,
    labelKey: page.labelKey,
    to: path(page.id),
    icon: page.icon,
    closable: page.closable ?? page.id !== ADMIN_DASHBOARD_PAGE_ID,
    componentName: page.componentName
  }
}
