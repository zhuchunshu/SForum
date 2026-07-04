<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui/components/DropdownMenu.vue'
import { useAdminRoutes } from '~/composables/useAdminRoutes'
import { useAdminTabs } from '~/composables/useAdminTabs'

const { t } = useI18n()
const localePath = useLocalePath()
const adminRoutes = useAdminRoutes()
const { user } = useAuthSession()
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
  return activeTab ? t(activeTab.labelKey) : t('admin.nav.dashboard')
})

// 监听路由变化，同步更新 activeTabId，解决 KeepAlive 缓存组件切换时 active 状态不更新的问题
const route = useRoute()
const resolveTabIdFromPath = (path: string) => {
  const adminPrefix = adminRoutes.prefix
  let cleanPath = path
  const localeMatch = cleanPath.match(/^\/([a-zA-Z]{2}(-[a-zA-Z]{2})?)\//)
  if (localeMatch) {
    cleanPath = cleanPath.substring(localeMatch[0].length - 1)
  }
  if (cleanPath.startsWith(adminPrefix)) {
    const childPath = cleanPath.substring(adminPrefix.length)
    return childPath === '' ? '/' : childPath
  }
  return null
}

watch(() => route.path, (newPath) => {
  const tabId = resolveTabIdFromPath(newPath)
  if (tabId) {
    adminTabs.activeTabId.value = tabId
  }
}, { immediate: true })

// 无 Emoji，严格使用 i-lucide-
// 支持多级折叠嵌套菜单
const navigationItems = computed(() => [
  [
    {
      label: t('admin.nav.dashboard'),
      icon: 'i-lucide-layout-dashboard',
      to: adminRoutes.path('/')
    },
    {
      label: t('admin.nav.userPermission'),
      icon: 'i-lucide-user-cog',
      defaultOpen: true,
      children: [
        {
          label: t('admin.nav.userManagement'),
          icon: 'i-lucide-contact',
          to: adminRoutes.path('/users')
        },
        {
          label: t('admin.nav.userGroups'),
          icon: 'i-lucide-users',
          to: adminRoutes.path('/roles'),
          badge: t('admin.nav.rolesBadge')
        },
        {
          label: t('admin.nav.permissionManagement'),
          icon: 'i-lucide-shield-check',
          to: adminRoutes.path('/permissions')
        }
      ]
    },
    {
      label: '系统配置',
      icon: 'i-lucide-settings-2',
      defaultOpen: true,
      children: [
        {
          label: t('admin.nav.settings'),
          icon: 'i-lucide-sliders',
          to: adminRoutes.path('/settings')
        }
      ]
    }
  ],
  [
    {
      label: t('admin.nav.forumHome'),
      icon: 'i-lucide-house',
      to: localePath('/')
    }
  ]
])

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
      label: colorMode.value === 'dark' ? '切换至浅色模式' : '切换至深色模式',
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
      :min-size="13"
      :max-size="22"
      class="border-r border-slate-200 dark:border-zinc-800 bg-[var(--bg-admin-sidebar)] text-slate-600 dark:text-zinc-400"
    >
      <template #header="{ collapsed }">
        <NuxtLink
          :to="adminRoutes.path('/')"
          class="flex h-12 min-w-0 items-center gap-3 rounded-md px-2 text-slate-900 dark:text-white hover:bg-slate-100 dark:hover:bg-zinc-800"
          :aria-label="siteName"
        >
          <span class="grid size-8 shrink-0 place-items-center rounded-md bg-teal-600 text-white">
            <UIcon name="i-lucide-message-square-text" class="size-4" />
          </span>
          <span v-if="!collapsed" class="min-w-0">
            <span class="block truncate text-sm font-semibold text-slate-900 dark:text-white">
              {{ siteName }}
            </span>
            <span class="block truncate text-xs text-slate-500 dark:text-zinc-400">
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
            class="justify-start px-2 text-slate-600 dark:text-zinc-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-zinc-800"
            @click="() => { colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark' }"
          >
            <UIcon :name="colorMode.value === 'dark' ? 'i-lucide-sun' : 'i-lucide-moon'" class="size-4" />
            <span class="text-sm font-medium">
              {{ colorMode.value === 'dark' ? '浅色模式' : '深色模式' }}
            </span>
          </UButton>

          <UDropdownMenu :items="userMenuItems" :content="{ side: 'top', align: 'start' }">
            <UButton
              color="neutral"
              variant="ghost"
              block
              class="justify-start px-2 text-slate-600 dark:text-zinc-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-zinc-800"
              :class="{ 'justify-center': collapsed }"
            >
              <UAvatar :text="userInitial" size="sm" />
              <span v-if="!collapsed" class="min-w-0 flex-1 text-left">
                <span class="block truncate text-sm font-medium text-slate-900 dark:text-white">
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

    <UDashboardPanel class="flex flex-col min-w-0 flex-1 bg-slate-50 dark:bg-zinc-950 text-slate-900 dark:text-zinc-100">
      <!-- 1. 置顶全局 Topbar -->
      <div class="flex items-center justify-between h-[54px] px-6 bg-white dark:bg-zinc-900 border-b border-slate-200 dark:border-zinc-800 flex-shrink-0 z-20">
        <div class="flex items-center gap-2">
          <span class="text-sm font-semibold text-slate-900 dark:text-zinc-100">SForum 控制台</span>
          <span class="text-xs text-slate-400 dark:text-zinc-500">/</span>
          <span class="text-xs text-slate-500 dark:text-zinc-400">{{ activeTabLabel }}</span>
        </div>
        <div class="flex items-center gap-4 text-xs">
          <span class="inline-flex items-center gap-1.5 text-slate-500 dark:text-zinc-400">
            <span class="size-2 rounded-full bg-teal-600 dark:bg-teal-400"></span>
            管理员: <strong class="text-slate-800 dark:text-zinc-200">{{ user?.username }}</strong>
          </span>
        </div>
      </div>

      <!-- 2. 多页签页签栏 (Taller tab bar: 44px) -->
      <div class="flex items-end h-[44px] px-3 gap-1.5 bg-white dark:bg-zinc-900 border-b border-slate-200 dark:border-zinc-800 overflow-x-auto flex-shrink-0 select-none no-scrollbar z-15">
        <div
          v-for="tab in adminTabs.tabs.value"
          :key="tab.id"
          class="group inline-flex items-center gap-1.5 h-[36px] px-4 border border-b-0 border-slate-200 dark:border-zinc-800 mb-[-1px] rounded-t-lg cursor-pointer transition-colors text-xs font-semibold relative z-10"
          :class="adminTabs.activeTabId.value === tab.id 
            ? 'bg-slate-50 dark:bg-zinc-950 text-slate-900 dark:text-zinc-100 border-slate-200 dark:border-zinc-800' 
            : 'bg-transparent text-slate-500 dark:text-zinc-400 border-transparent hover:text-slate-900 dark:hover:text-zinc-100'"
          @click="navigateTo(tab.to)"
        >
          <UIcon :name="tab.icon" class="size-4" />
          <span>{{ t(tab.labelKey) }}</span>
          
          <span
            v-if="tab.closable"
            class="inline-flex items-center justify-center size-4 rounded-full text-slate-500 dark:text-zinc-400 hover:bg-red-500/20 hover:text-red-500 transition-colors"
            @click.stop="adminTabs.closeTab(tab.id)"
          >
            <UIcon name="i-lucide-x" class="size-2.5" />
          </span>
        </div>
      </div>

      <!-- 3. 内容区滚动面板 -->
      <div class="flex-1 overflow-y-auto flex flex-col p-4 sm:p-6 bg-slate-50 dark:bg-zinc-950">
        <slot />
      </div>
    </UDashboardPanel>
  </UDashboardGroup>
</template>
