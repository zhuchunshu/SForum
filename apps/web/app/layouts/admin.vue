<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui/components/DropdownMenu.vue'
import { useAdminRoutes } from '~/composables/useAdminRoutes'

const { t } = useI18n()
const localePath = useLocalePath()
const adminRoutes = useAdminRoutes()
const { user } = useAuthSession()
const apiBaseUrl = useRuntimeConfig().public.apiBaseUrl as string

const displayName = computed(() => {
  return user.value?.displayName || user.value?.username || t('admin.shell.unknownUser')
})

const userInitial = computed(() => {
  return displayName.value.trim().slice(0, 1).toUpperCase() || 'S'
})

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
      label: t('admin.shell.signOut'),
      icon: 'i-lucide-log-out',
      onSelect: () => {
        void signOut()
      }
    }
  ]
])

async function signOut() {
  await $fetch(`${apiBaseUrl}/auth/logout`, {
    method: 'POST',
    credentials: 'include'
  }).catch(() => null)

  user.value = null
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
      class="border-r border-default bg-default/95"
    >
      <template #header="{ collapsed }">
        <NuxtLink
          :to="adminRoutes.path('/')"
          class="flex h-12 min-w-0 items-center gap-3 rounded-md px-2 text-highlighted hover:bg-elevated"
          :aria-label="t('admin.shell.brand')"
        >
          <span class="grid size-8 shrink-0 place-items-center rounded-md bg-primary text-inverted">
            <UIcon name="i-lucide-message-square-text" class="size-4" />
          </span>
          <span v-if="!collapsed" class="min-w-0">
            <span class="block truncate text-sm font-semibold">
              {{ t('admin.shell.brand') }}
            </span>
            <span class="block truncate text-xs text-muted">
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
          class="-mx-2"
        />
      </template>

      <template #footer="{ collapsed }">
        <UDropdownMenu :items="userMenuItems" :content="{ side: 'top', align: 'start' }">
          <UButton
            color="neutral"
            variant="ghost"
            block
            class="justify-start px-2"
            :class="{ 'justify-center': collapsed }"
          >
            <UAvatar :text="userInitial" size="sm" />
            <span v-if="!collapsed" class="min-w-0 flex-1 text-left">
              <span class="block truncate text-sm font-medium">
                {{ displayName }}
              </span>
              <span class="block truncate text-xs text-muted">
                {{ user?.roleKeys?.join(', ') || t('admin.shell.member') }}
              </span>
            </span>
            <UIcon v-if="!collapsed" name="i-lucide-chevrons-up-down" class="size-4 text-muted" />
          </UButton>
        </UDropdownMenu>
      </template>
    </UDashboardSidebar>

    <UDashboardPanel>
      <slot />
    </UDashboardPanel>
  </UDashboardGroup>
</template>
