<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui/components/DropdownMenu.vue'
import {
  adminSidebarNavigation,
  canAccessAdminPage,
  findAdminPageDefinition,
  type AdminNavigationEntry,
  requireAdminPageDefinition
} from '~/config/adminModules'
import { useAdminRoutes } from '~/composables/useAdminRoutes'
import { useAdminTabs } from '~/composables/useAdminTabs'

type SidebarNavigationItem = {
  label: string
  icon: string
  to?: string
  badge?: string
  defaultOpen?: boolean
  children?: SidebarNavigationItem[]
}

const { t } = useI18n()
const localePath = useLocalePath()
const adminRoutes = useAdminRoutes()
const { user, can } = useAuthSession()
const { request } = useApiClient()
const { siteName } = useWebOptions()

// 引入多页签状态与主题模式
const adminTabs = useAdminTabs()
const colorMode = useColorMode()

const displayName = computed(() => {
  return user.value?.displayName || user.value?.username || t('admin.shell.unknownUser')
})

const userInitial = computed(() => {
  return displayName.value.trim().slice(0, 1).toUpperCase() || 'S'
})

// 计算当前激活标签页的标题，用于面包屑展示
const activeTabLabel = computed(() => {
  const activeTab = adminTabs.tabs.value.find(tab => tab.id === adminTabs.activeTabId.value)
  return activeTab ? t(activeTab.labelKey) : t(requireAdminPageDefinition('/').labelKey)
})

const route = useRoute()

// KeepAlive 页面不会重复 mounted，路由变化时用注册表同步当前 tab。
watch(() => route.path, (newPath) => {
  const tabId = adminRoutes.routeId(newPath)
  const page = tabId ? findAdminPageDefinition(tabId) : null

  if (page) {
    adminTabs.openTab(page.id)
  }
}, { immediate: true })

const navigationItems = computed(() => {
  return adminSidebarNavigation
    .map(group => group
      .map(entry => buildNavigationItem(entry))
      .filter((item): item is SidebarNavigationItem => Boolean(item)))
    .filter(group => group.length > 0)
})

function buildNavigationItem(entry: AdminNavigationEntry): SidebarNavigationItem | null {
  if (entry.type === 'forum-home') {
    return {
      label: t(entry.labelKey),
      icon: entry.icon,
      to: localePath('/')
    }
  }

  if (entry.type === 'folder') {
    const children = entry.children
      .map(child => buildNavigationItem(child))
      .filter((item): item is SidebarNavigationItem => Boolean(item))

    if (children.length === 0) {
      return null
    }

    return {
      label: t(entry.labelKey),
      icon: entry.icon,
      defaultOpen: entry.defaultOpen,
      children
    }
  }

  const page = findAdminPageDefinition(entry.pageId)
  if (!page || !canAccessAdminPage(page, can)) {
    return null
  }

  const badgeKey = entry.badgeKey || page.badgeKey

  return {
    label: t(page.labelKey),
    icon: page.icon,
    to: adminRoutes.path(page.id),
    ...(badgeKey ? { badge: t(badgeKey) } : {})
  }
}

const sidebarNavigationUi = {
  list: 'flex flex-col gap-1',
  item: 'min-w-0',
  link: '!min-h-[42px] !gap-2 !rounded-md !px-3.5 !py-2 !text-[14.5px] !font-semibold !leading-tight',
  linkLabel: '!leading-tight',
  linkLeadingIcon: '!size-[18px]',
  linkTrailingIcon: '!size-[17px]',
  linkTrailingBadge: '!text-xs !px-2 !py-0.5',
  childList: '!mt-1 !mb-2 !ms-5 !border-s !border-dashed !border-slate-200 dark:!border-zinc-800 !ps-3',
  childItem: '!ps-0',
  childLink: '!min-h-[36px] !gap-1.5 !rounded-md !px-3 !py-1.5 !text-[13.5px] !font-medium !leading-tight',
  childLinkIcon: '!size-[15px]',
  childLinkLabel: '!leading-tight'
}

const userMenuItems = computed<DropdownMenuItem[][]>(() => [
  [
    {
      label: displayName.value,
      avatar: {
        text: userInitial.value
      },
      type: 'label'
    }
  ],
  [
    {
      label: t('admin.shell.visitForum'),
      icon: 'i-lucide-house',
      to: localePath('/')
    },
    {
      label: themeToggleLabel.value,
      icon: colorMode.value === 'dark' ? 'i-lucide-sun' : 'i-lucide-moon',
      onSelect: () => {
        colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark'
      }
    },
    {
      label: t('admin.shell.signOut'),
      icon: 'i-lucide-log-out',
      onSelect: () => {
        void signOut()
      }
    }
  ]
])

const themeToggleLabel = computed(() => {
  return colorMode.value === 'dark' ? t('admin.shell.lightMode') : t('admin.shell.darkMode')
})

async function signOut() {
  await request<null>('/auth/logout', {
    method: 'POST'
  }).catch(() => null)

  user.value = null
  adminTabs.resetTabs()
  await navigateTo(localePath('/login'))
}
</script>

<template>
  <UDashboardGroup storage-key="sforum-admin">
    <UDashboardSidebar
      id="sforum-admin-sidebar"
      collapsible
      resizable
      :default-size="16"
      :min-size="14"
      :max-size="22"
      class="sforum-admin-sidebar border-r border-[var(--border-admin)] bg-[var(--bg-admin-sidebar)] text-[var(--text-admin-sidebar)]"
    >
      <template #header="{ collapsed }">
        <NuxtLink
          :to="adminRoutes.path('/')"
          class="flex h-[50px] min-w-0 items-center gap-2.5 rounded-md px-2 text-[var(--text-admin-main)] hover:bg-[var(--bg-admin-sidebar-hover)]"
          :aria-label="siteName"
        >
          <span class="grid size-[30px] shrink-0 place-items-center rounded-md bg-[var(--sf-accent)] text-[var(--sf-accent-contrast)]">
            <UIcon name="i-lucide-message-square-text" class="size-[17px]" />
          </span>
          <span v-if="!collapsed" class="min-w-0">
            <span class="block truncate text-[14.5px] font-bold text-[var(--text-admin-main)]">
              {{ siteName }}
            </span>
            <span class="block truncate text-xs font-medium text-slate-500 dark:text-zinc-400">
              {{ t('admin.shell.section') }}
            </span>
          </span>
        </NuxtLink>
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu
          :items="navigationItems"
          :collapsed="collapsed"
          tooltip
          highlight
          color="primary"
          orientation="vertical"
          class="-mx-2"
          :ui="sidebarNavigationUi"
        />
      </template>

      <template #footer="{ collapsed }">
        <div class="flex flex-col gap-2 w-full">
          <!-- 桌面端快捷切换主题按钮 -->
          <UButton
            v-if="!collapsed"
            color="neutral"
            variant="ghost"
            block
            class="justify-start px-2 py-2 text-[var(--text-admin-sidebar)] hover:bg-[var(--bg-admin-sidebar-hover)] hover:text-[var(--text-admin-main)]"
            @click="() => { colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark' }"
          >
            <UIcon :name="colorMode.value === 'dark' ? 'i-lucide-sun' : 'i-lucide-moon'" class="size-4" />
            <span class="text-sm font-semibold">
              {{ themeToggleLabel }}
            </span>
          </UButton>

          <UDropdownMenu :items="userMenuItems" :content="{ side: 'top', align: 'start' }">
            <UButton
              color="neutral"
              variant="ghost"
              block
              class="justify-start px-2 py-3.5 text-[var(--text-admin-sidebar)] hover:bg-[var(--bg-admin-sidebar-hover)] hover:text-[var(--text-admin-main)]"
              :class="{ 'justify-center': collapsed }"
            >
              <UAvatar :text="userInitial" size="lg" class="shadow-sm border border-slate-100 dark:border-zinc-800" />
              <span v-if="!collapsed" class="min-w-0 flex-1 text-left ml-2.5">
                <span class="block truncate text-base font-bold text-slate-900 dark:text-white">
                  {{ displayName }}
                </span>
                <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">
                  {{ user?.roleKeys?.join(', ') || t('admin.shell.member') }}
                </span>
              </span>
              <UIcon v-if="!collapsed" name="i-lucide-chevrons-up-down" class="size-4 text-slate-400 dark:text-zinc-500" />
            </UButton>
          </UDropdownMenu>
        </div>
      </template>
    </UDashboardSidebar>

    <UDashboardPanel class="flex flex-col min-w-0 flex-1 bg-[var(--bg-admin-app)] text-[var(--text-admin-main)]">
      <!-- 1. 置顶全局 Topbar -->
      <div class="flex items-center justify-between h-[68px] sm:h-[76px] px-4 sm:px-8 bg-[var(--bg-admin-card)] border-b border-[var(--border-admin)] flex-shrink-0 z-20 transition-all">
        <div class="flex min-w-0 items-center gap-2 sm:gap-3">
          <span class="shrink-0 text-base sm:text-lg font-bold text-slate-900 dark:text-zinc-100 tracking-wide">
            {{ t('admin.shell.controlPanel', { siteName }) }}
          </span>
          <span class="shrink-0 text-sm text-slate-300 dark:text-zinc-600">/</span>
          <span class="truncate text-sm sm:text-base font-semibold text-slate-600 dark:text-zinc-300">{{ activeTabLabel }}</span>
        </div>
        <div class="hidden sm:flex items-center gap-4 text-sm">
          <span class="inline-flex items-center gap-2.5 text-slate-600 dark:text-zinc-300 bg-[var(--sf-accent-soft)] px-4 py-2.5 rounded-full border border-[var(--sf-accent-soft-border)]">
            <span class="size-2.5 rounded-full bg-[var(--sf-accent)] dark:bg-[var(--sf-accent-dark)] animate-pulse"></span>
            {{ t('admin.shell.administratorLabel') }}:
            <strong class="text-slate-800 dark:text-zinc-200 font-semibold">{{ user?.username }}</strong>
          </span>
        </div>
      </div>

      <!-- 2. 多页签页签栏 (Taller tab bar: 52px) -->
      <div class="flex items-end h-[52px] px-3 gap-1.5 bg-[var(--bg-admin-card)] border-b border-[var(--border-admin)] overflow-x-auto flex-shrink-0 select-none no-scrollbar z-15">
        <div
          v-for="tab in adminTabs.tabs.value"
          :key="tab.id"
          class="group inline-flex items-center gap-2 h-[44px] px-5 border border-b-0 border-[var(--border-admin)] mb-[-1px] rounded-t-lg cursor-pointer transition-colors text-sm font-semibold relative z-10"
          :class="adminTabs.activeTabId.value === tab.id 
            ? 'bg-[var(--bg-admin-app)] text-[var(--sf-accent)] border-[var(--border-admin)]' 
            : 'bg-transparent text-slate-500 dark:text-zinc-400 border-transparent hover:text-[var(--text-admin-main)]'"
          @click="navigateTo(tab.to)"
        >
          <UIcon :name="tab.icon" class="size-4.5" />
          <span>{{ t(tab.labelKey) }}</span>
          
          <span
            v-if="tab.closable"
            class="inline-flex items-center justify-center size-4.5 rounded-full text-slate-500 dark:text-zinc-400 hover:bg-red-500/20 hover:text-red-500 transition-colors"
            @click.stop="adminTabs.closeTab(tab.id)"
          >
            <UIcon name="i-lucide-x" class="size-3" />
          </span>
        </div>
      </div>

      <!-- 3. 内容区滚动面板 -->
      <div class="flex-1 overflow-y-auto flex flex-col p-4 sm:p-6 bg-[var(--bg-admin-app)]">
        <slot />
      </div>
    </UDashboardPanel>
  </UDashboardGroup>
</template>
