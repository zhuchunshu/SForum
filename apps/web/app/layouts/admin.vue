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

// 引入多页签状态
const adminTabs = useAdminTabs()
const colorMode = useColorMode()

const displayName = computed(() => {
  return user.value?.displayName || user.value?.username || t('admin.shell.unknownUser')
})

const userInitial = computed(() => {
  return displayName.value.trim().slice(0, 1).toUpperCase() || 'S'
})

// 无 Emoji，严格使用 i-lucide-
const navigationItems = computed(() => [
  [
    {
      label: t('admin.nav.dashboard'),
      icon: 'i-lucide-layout-dashboard',
      to: adminRoutes.path('/')
    },
    {
      label: t('admin.nav.roles'),
      icon: 'i-lucide-shield-check',
      to: adminRoutes.path('/roles'),
      badge: t('admin.nav.rolesBadge')
    },
    {
      label: t('admin.nav.settings'),
      icon: 'i-lucide-settings-2',
      to: adminRoutes.path('/settings')
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
  adminTabs.resetTabs() // 清理打开的页签
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
      class="border-r border-default bg-default/95 text-slate-400!"
    >
      <template #header="{ collapsed }">
        <NuxtLink
          :to="adminRoutes.path('/')"
          class="flex h-12 min-w-0 items-center gap-3 rounded-md px-2 text-white hover:bg-slate-800"
          :aria-label="siteName"
        >
          <span class="grid size-8 shrink-0 place-items-center rounded-md bg-teal-600 text-white">
            <UIcon name="i-lucide-message-square-text" class="size-4" />
          </span>
          <span v-if="!collapsed" class="min-w-0">
            <span class="block truncate text-sm font-semibold text-white">
              {{ siteName }}
            </span>
            <span class="block truncate text-xs text-slate-400">
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
          color="neutral"
          orientation="vertical"
          class="-mx-2 text-slate-400!"
        />
      </template>

      <template #footer="{ collapsed }">
        <div class="flex flex-col gap-2 w-full">
          <!-- 桌面端快捷切换主题按钮，方便在侧边栏直接点击 -->
          <UButton
            v-if="!collapsed"
            color="neutral"
            variant="ghost"
            block
            class="justify-start px-2 text-slate-400 hover:text-white"
            @click="colorMode.preference = colorMode.value === 'dark' ? 'light' : 'dark'"
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
              class="justify-start px-2 text-slate-400 hover:text-white"
              :class="{ 'justify-center': collapsed }"
            >
              <UAvatar :text="userInitial" size="sm" />
              <span v-if="!collapsed" class="min-w-0 flex-1 text-left">
                <span class="block truncate text-sm font-medium text-white">
                  {{ displayName }}
                </span>
                <span class="block truncate text-xs text-slate-400">
                  {{ user?.roleKeys?.join(', ') || t('admin.shell.member') }}
                </span>
              </span>
              <UIcon v-if="!collapsed" name="i-lucide-chevrons-up-down" class="size-4 text-slate-500" />
            </UButton>
          </UDropdownMenu>
        </div>
      </template>
    </UDashboardSidebar>

    <UDashboardPanel class="flex flex-col min-w-0 flex-1 bg-[var(--bg-admin-app)] text-[var(--text-admin-main)]">
      <!-- 多页签页签栏 -->
      <div class="flex items-end h-[38px] px-3 gap-1 bg-[var(--bg-admin-card)] border-b border-[var(--border-admin)] overflow-x-auto flex-shrink-0 select-none no-scrollbar">
        <div
          v-for="tab in adminTabs.tabs.value"
          :key="tab.id"
          class="group inline-flex items-center gap-1.5 h-[30px] px-2.5 border border-[var(--border-admin)] border-bottom-none rounded-t-md cursor-pointer transition-colors text-xs font-medium"
          :class="adminTabs.activeTabId.value === tab.id 
            ? 'bg-[var(--bg-admin-app)] text-[var(--text-admin-main)] border-b-[var(--bg-admin-app)] z-10' 
            : 'bg-transparent text-[var(--text-admin-muted)] border-transparent hover:text-[var(--text-admin-main)]'"
          @click="navigateTo(tab.to)"
        >
          <UIcon :name="tab.icon" class="size-3.5" />
          <span>{{ t(tab.labelKey) }}</span>
          
          <span
            v-if="tab.closable"
            class="inline-flex items-center justify-center size-3.5 rounded-full text-[var(--text-admin-muted)] hover:bg-red-500/20 hover:text-red-500 transition-colors"
            @click.stop="adminTabs.closeTab(tab.id)"
          >
            <UIcon name="i-lucide-x" class="size-2.5" />
          </span>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto flex flex-col">
        <slot />
      </div>
    </UDashboardPanel>
  </UDashboardGroup>
</template>
