import { computed } from 'vue'
import { useAdminRoutes } from '~/composables/useAdminRoutes'

export interface AdminTab {
  id: string          // 相对路径，例如 '/', '/roles', '/settings'
  labelKey: string    // 翻译键名，例如 'admin.nav.dashboard'
  to: string          // 路由路径
  icon: string        // 统一使用 i-lucide- 图标
  closable: boolean
  componentName: string // 对应的 Vue 组件 name，用于 KeepAlive :include
}

export const useAdminTabs = () => {
  const adminRoutes = useAdminRoutes()

  const tabs = useState<AdminTab[]>('admin-tabs', () => [
    {
      id: '/',
      labelKey: 'admin.nav.dashboard',
      to: adminRoutes.path('/'),
      icon: 'i-lucide-layout-dashboard',
      closable: false,
      componentName: 'AdminIndex'
    }
  ])

  const activeTabId = useState<string>('admin-active-tab-id', () => '/')

  const cachedTabNames = computed(() => {
    return tabs.value.map(tab => tab.componentName)
  })

  const openTab = (id: string, labelKey: string, icon: string, componentName: string) => {
    const existing = tabs.value.find(tab => tab.id === id)
    if (!existing) {
      tabs.value.push({
        id,
        labelKey,
        to: adminRoutes.path(id),
        icon,
        closable: id !== '/',
        componentName
      })
    }
    activeTabId.value = id
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
      {
        id: '/',
        labelKey: 'admin.nav.dashboard',
        to: adminRoutes.path('/'),
        icon: 'i-lucide-layout-dashboard',
        closable: false,
        componentName: 'AdminIndex'
      }
    ]
    activeTabId.value = '/'
  }

  return {
    tabs,
    activeTabId,
    cachedTabNames,
    openTab,
    closeTab,
    resetTabs
  }
}
