# 后台多标签与双模式 UI 重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 SForum 后台管理系统框架，引入经典的深色侧边栏 SaaS 双模式（Light/Dark）自适应视觉风格，并实现支持无限滚动、动态增删、防丢失缓存（Keep-Alive）的多标签页页签导航系统。

**Architecture:** 
- 使用全局 Composable `useAdminTabs` 追踪与持久化打开的标签页列表及当前激活项。
- 各管理页面在 `setup` 阶段调用 `openTab` 自动挂载自己。
- 通过重定义 `main.css` 的 CSS 语义变量支持双主题切换，并将 `app.vue` 中的 `<NuxtPage>` 绑定 `:keepalive` 实现按标签打开状态动态精准缓存，标签关闭状态随之销毁。

**Tech Stack:** Nuxt 4, Vue 3, Nuxt UI v4, Tailwind CSS v4, Lucide Icon, `@nuxtjs/color-mode`

---

### Task 1: 创建多标签状态管理器 Composable

**Files:**
- Create: `apps/web/app/composables/useAdminTabs.ts`

- [ ] **Step 1: 编写 `useAdminTabs` Composable 逻辑**

创建 `apps/web/app/composables/useAdminTabs.ts`，写入如下代码：
```typescript
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
      activeTabId.value = fallbackTab.id
      navigateTo(fallbackTab.to)
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
```

- [ ] **Step 2: 运行类型检查以验证没有错误**

Run: `bun run --cwd apps/web typecheck`
Expected: PASS

- [ ] **Step 3: 提交修改**

```bash
git add apps/web/app/composables/useAdminTabs.ts
git commit -m "feat: add useAdminTabs composable for multi-tab management"
```

---

### Task 2: 重构全局 CSS 主题变量以实现双模式

**Files:**
- Modify: `apps/web/app/assets/css/main.css`

- [ ] **Step 1: 重写 `main.css` 引入语义化变量与系统模式自适应**

修改 `apps/web/app/assets/css/main.css` 的开头段落：
```css
@import "tailwindcss";
@import "@nuxt/ui";

/* Tailwind CSS v4 Rebuild Trigger Comment - Scanned new demo routes */

@theme {
  --color-primary-50: var(--color-zinc-50);
  --color-primary-100: var(--color-zinc-100);
  --color-primary-200: var(--color-zinc-200);
  --color-primary-300: var(--color-zinc-300);
  --color-primary-400: var(--color-zinc-400);
  --color-primary-500: var(--color-zinc-500);
  --color-primary-600: var(--color-zinc-600);
  --color-primary-700: var(--color-zinc-700);
  --color-primary-800: var(--color-zinc-800);
  --color-primary-900: var(--color-zinc-900);
  --color-primary-950: var(--color-zinc-950);
}

:root {
  font-family: Inter, "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif;
  
  /* 语义化基础变量：浅色模式 */
  --bg-app: #fcfcfd;
  --text-main: #09090b;
  --text-muted: #71717a;
  --border-default: #e4e4e7;

  /* 后台定制：浅色模式 */
  --bg-admin-sidebar: #0f172a; /* Slate 900 */
  --bg-admin-sidebar-hover: rgba(255, 255, 255, 0.06);
  --text-admin-sidebar: #94a3b8;
  --text-admin-sidebar-active: #2dd4bf;
  
  --bg-admin-app: #f8fafc;
  --bg-admin-card: #ffffff;
  --border-admin: #e2e8f0;
  --text-admin-main: #0f172a;
}

.dark {
  /* 语义化基础变量：深色模式 */
  --bg-app: #09090b;
  --text-main: #f4f4f5;
  --text-muted: #a1a1aa;
  --border-default: #27272a;

  /* 后台定制：深色模式 */
  --bg-admin-sidebar: #09090b; /* Zinc 950 */
  --bg-admin-sidebar-hover: rgba(255, 255, 255, 0.03);
  --text-admin-sidebar: #71717a;
  --text-admin-sidebar-active: #2dd4bf;
  
  --bg-admin-app: #09090b;
  --bg-admin-card: #18181b;
  --border-admin: #27272a;
  --text-admin-main: #f4f4f5;
}

body {
  margin: 0;
  background-color: var(--bg-app);
  color: var(--text-main);
  transition: background-color 0.2s, color 0.2s;
}

/* 覆盖后台特定框架的默认边框颜色 */
.border-default {
  border-color: var(--border-admin);
}
.bg-default\/95 {
  background-color: var(--bg-admin-sidebar);
}
```

- [ ] **Step 2: 提交 CSS 修改**

```bash
git add apps/web/app/assets/css/main.css
git commit -m "style: define dynamic CSS variables for dual mode support"
```

---

### Task 3: 重构后台布局页面 `admin.vue`

**Files:**
- Modify: `apps/web/app/layouts/admin.vue`

- [ ] **Step 1: 重构 `admin.vue` 模板与页签渲染**

修改 `apps/web/app/layouts/admin.vue` 引入页签栏，并改用 Lucide 图标做侧边栏，同时在底部加入主题切换按钮：
```html
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
```

- [ ] **Step 2: 提交布局修改**

```bash
git add apps/web/app/layouts/admin.vue
git commit -m "feat: implement multi-tab navigation bar and theme toggle in admin layout"
```

---

### Task 4: 重构后台管理页面注册多标签与清除 Emoji

**Files:**
- Modify: `apps/web/app/pages/admin/index.vue`
- Modify: `apps/web/app/pages/admin/roles.vue`
- Modify: `apps/web/app/pages/admin/settings/index.vue`

- [ ] **Step 1: 重构 `admin/index.vue`**

修改 `apps/web/app/pages/admin/index.vue` 以定义明确的组件名，在生命周期中注册页签，并移除 UI 元素中的 Emoji：
```html
<script setup lang="ts">
import { useAdminRoutes } from '~/composables/useAdminRoutes'
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

// 明确声明组件名用于 KeepAlive 匹配
defineOptions({
  name: 'AdminIndex'
})

const { t } = useI18n()
const adminRoutes = useAdminRoutes()
const adminTabs = useAdminTabs()

// 挂载当前页签
onMounted(() => {
  adminTabs.openTab('/', 'admin.nav.dashboard', 'i-lucide-layout-dashboard', 'AdminIndex')
})

const overviewCards = computed(() => [
  {
    label: t('admin.home.cards.access.label'),
    value: t('admin.home.cards.access.value'),
    icon: 'i-lucide-shield-check',
    tone: 'text-teal-600 dark:text-teal-400'
  },
  {
    label: t('admin.home.cards.prefix.label'),
    value: adminRoutes.prefix,
    icon: 'i-lucide-route',
    tone: 'text-blue-600 dark:text-blue-400'
  },
  {
    label: t('admin.home.cards.stack.label'),
    value: t('admin.home.cards.stack.value'),
    icon: 'i-lucide-box',
    tone: 'text-green-600 dark:text-green-400'
  }
])

const nextSections = computed(() => [
  {
    title: t('admin.home.next.roles.title'),
    description: t('admin.home.next.roles.description'),
    icon: 'i-lucide-users-round',
    to: adminRoutes.path('/roles')
  },
  {
    title: t('admin.home.next.audit.title'),
    description: t('admin.home.next.audit.description'),
    icon: 'i-lucide-scroll-text'
  },
  {
    title: t('admin.home.next.settings.title'),
    description: t('admin.home.next.settings.description'),
    icon: 'i-lucide-settings-2',
    to: adminRoutes.path('/settings')
  }
])

useSeoMeta({
  title: t('admin.home.metaTitle')
})
</script>

<template>
  <UDashboardNavbar :title="t('admin.home.title')" icon="i-lucide-layout-dashboard" class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
    <template #right>
      <UButton
        :to="adminRoutes.path('/roles')"
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-shield-check"
        class="border-[var(--border-admin)]"
      >
        {{ t('admin.home.rolesLink') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-app)] text-[var(--text-admin-muted)]">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-[var(--text-admin-muted)]">
        <UIcon name="i-lucide-lock-keyhole" class="size-4" />
        <span class="truncate">{{ t('admin.home.intro') }}</span>
      </div>
    </template>
    <template #right>
      <UBadge color="neutral" variant="soft" class="border border-[var(--border-admin)]">
        {{ adminRoutes.prefix }}
      </UBadge>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-6 p-4 sm:p-6 bg-[var(--bg-admin-app)]">
    <div class="grid gap-4 lg:grid-cols-3">
      <UCard v-for="card in overviewCards" :key="card.label" class="border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <p class="text-sm text-[var(--text-admin-muted)]">
              {{ card.label }}
            </p>
            <p class="mt-2 truncate text-xl font-semibold text-[var(--text-admin-main)]">
              {{ card.value }}
            </p>
          </div>
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-[var(--bg-admin-app)] border border-[var(--border-admin)]">
            <UIcon :name="card.icon" class="size-5" :class="card.tone" />
          </span>
        </div>
      </UCard>
    </div>

    <UCard class="border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-[var(--text-admin-main)]">
              {{ t('admin.home.nextTitle') }}
            </h2>
            <p class="mt-1 text-sm text-[var(--text-admin-muted)]">
              {{ t('admin.home.nextIntro') }}
            </p>
          </div>
          <UIcon name="i-lucide-list-checks" class="size-5 text-[var(--text-admin-muted)]" />
        </div>
      </template>

      <div class="divide-y divide-[var(--border-admin)]">
        <component
          :is="section.to ? 'NuxtLink' : 'div'"
          v-for="section in nextSections"
          :key="section.title"
          :to="section.to"
          class="flex items-center gap-4 py-4 first:pt-0 last:pb-0"
        >
          <span class="grid size-10 shrink-0 place-items-center rounded-md bg-[var(--bg-admin-app)] border border-[var(--border-admin)]">
            <UIcon :name="section.icon" class="size-5 text-[var(--text-admin-muted)]" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block font-medium text-[var(--text-admin-main)]">
              {{ section.title }}
            </span>
            <span class="mt-1 block text-sm text-[var(--text-admin-muted)]">
              {{ section.description }}
            </span>
          </span>
          <UIcon
            v-if="section.to"
            name="i-lucide-arrow-right"
            class="size-4 shrink-0 text-[var(--text-admin-muted)]"
          />
        </component>
      </div>
    </UCard>
  </div>
</template>
```

- [ ] **Step 2: 重构 `admin/roles.vue`**

修改 `apps/web/app/pages/admin/roles.vue` 以定义明确的组件名，在生命周期中注册页签，并移除 UI 元素中的 Emoji：
```html
<script setup lang="ts">
import type { ApiEnvelope } from '~/composables/useApiClient'
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminRoles'
})

const { t } = useI18n()
const { apiBaseUrl, apiHeaders } = useApiClient()
const search = ref('')
const adminTabs = useAdminTabs()

onMounted(() => {
  adminTabs.openTab('/roles', 'admin.nav.roles', 'i-lucide-shield-check', 'AdminRoles')
})

type Role = {
  id: number
  key: string
  alias: string
  description: string
  isSystem: boolean
  isDefault: boolean
  isDeletable: boolean
  isEnabled: boolean
}

const { data: rolesEnvelope, pending, error, refresh } = await useFetch<ApiEnvelope<Role[]>>(`${apiBaseUrl}/roles`, {
  credentials: 'include',
  headers: apiHeaders(),
  default: () => ({
    code: 200,
    message: 'OK',
    data: []
  })
})

const roles = computed(() => rolesEnvelope.value?.data ?? [])

const columns = computed(() => [
  {
    accessorKey: 'key',
    header: t('admin.roles.key')
  },
  {
    accessorKey: 'alias',
    header: t('admin.roles.alias')
  },
  {
    accessorKey: 'description',
    header: t('admin.roles.description')
  },
  {
    id: 'status',
    header: t('admin.roles.status')
  }
])

const filteredRoles = computed(() => {
  const keyword = search.value.trim().toLowerCase()

  if (!keyword) {
    return roles.value
  }

  return roles.value.filter((role) => {
    return [role.key, role.alias, role.description]
      .some((value) => value.toLowerCase().includes(keyword))
  })
})

useSeoMeta({
  title: t('admin.roles.metaTitle')
})
</script>

<template>
  <UDashboardNavbar :title="t('admin.roles.title')" icon="i-lucide-shield-check" class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pending"
        class="border-[var(--border-admin)]"
        @click="refresh()"
      >
        {{ t('admin.roles.refresh') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-app)] text-[var(--text-admin-muted)]">
    <template #left>
      <UInput
        v-model="search"
        icon="i-lucide-search"
        :placeholder="t('admin.roles.searchPlaceholder')"
        class="w-72 max-w-full"
      />
    </template>
    <template #right>
      <UBadge color="neutral" variant="soft" class="border border-[var(--border-admin)]">
        {{ t('admin.roles.count', { count: filteredRoles.length }) }}
      </UBadge>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-4 p-4 sm:p-6 bg-[var(--bg-admin-app)]">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.roles.loadFailed')"
    />

    <UCard v-else class="border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
      <UTable
        :data="filteredRoles"
        :columns="columns"
        :loading="pending"
        :empty="t('admin.roles.empty')"
        :caption="t('admin.roles.caption')"
        sticky
        class="max-h-[calc(100vh-13rem)]"
      >
        <template #key-cell="{ row }">
          <code class="rounded bg-[var(--bg-admin-app)] border border-[var(--border-admin)] px-2 py-1 text-xs font-medium text-[var(--text-admin-main)]">
            {{ row.original.key }}
          </code>
        </template>

        <template #alias-cell="{ row }">
          <span class="font-medium text-[var(--text-admin-main)]">
            {{ row.original.alias }}
          </span>
        </template>

        <template #description-cell="{ row }">
          <span class="text-[var(--text-admin-muted)]">
            {{ row.original.description || t('admin.roles.noDescription') }}
          </span>
        </template>

        <template #status-cell="{ row }">
          <div class="flex flex-wrap gap-1.5">
            <UBadge v-if="row.original.isSystem" color="info" variant="soft">
              {{ t('admin.roles.system') }}
            </UBadge>
            <UBadge v-if="row.original.isDefault" color="success" variant="soft">
              {{ t('admin.roles.default') }}
            </UBadge>
            <UBadge v-if="row.original.isDeletable" color="neutral" variant="outline">
              {{ t('admin.roles.custom') }}
            </UBadge>
          </div>
        </template>
      </UTable>
    </UCard>
  </div>
</template>
```

- [ ] **Step 3: 重构 `admin/settings/index.vue`**

修改 `apps/web/app/pages/admin/settings/index.vue` 以定义明确的组件名，在生命周期中注册页签，并移除 UI 元素中的 Emoji：
```html
<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminSettings'
})

const { t } = useI18n()
const toast = useToast()
const { options, fetchEnvelope, save } = useWebOptions()
const adminTabs = useAdminTabs()

onMounted(() => {
  adminTabs.openTab('/settings', 'admin.nav.settings', 'i-lucide-settings-2', 'AdminSettings')
})

const siteName = ref(options.value['site.name'] || 'SForum')
const saving = ref(false)

const { pending, error, refresh } = await useAsyncData('admin-web-options', async () => {
  const envelope = await fetchEnvelope()
  options.value = {
    ...options.value,
    ...Object.fromEntries(envelope.data.map((item) => [item.name, item.value]))
  }
  siteName.value = options.value['site.name'] || 'SForum'
  return envelope.data
})

useSeoMeta({
  title: t('admin.settings.metaTitle')
})

async function submit() {
  saving.value = true
  try {
    await save('site.name', siteName.value)
    toast.add({
      color: 'success',
      icon: 'i-lucide-check',
      title: t('admin.settings.saved')
    })
  } catch (error) {
    toast.add({
      color: 'error',
      icon: 'i-lucide-triangle-alert',
      title: apiErrorMessage(error) || t('admin.settings.saveFailed')
    })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UDashboardNavbar :title="t('admin.settings.title')" icon="i-lucide-settings-2" class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
    <template #right>
      <UButton
        color="neutral"
        variant="outline"
        leading-icon="i-lucide-refresh-cw"
        :loading="pending"
        class="border-[var(--border-admin)]"
        @click="refresh()"
      >
        {{ t('admin.settings.refresh') }}
      </UButton>
    </template>
  </UDashboardNavbar>

  <UDashboardToolbar class="border-b border-[var(--border-admin)] bg-[var(--bg-admin-app)] text-[var(--text-admin-muted)]">
    <template #left>
      <div class="flex min-w-0 items-center gap-2 text-sm text-[var(--text-admin-muted)]">
        <UIcon name="i-lucide-database" class="size-4" />
        <span class="truncate">{{ t('admin.settings.intro') }}</span>
      </div>
    </template>
  </UDashboardToolbar>

  <div class="flex flex-1 flex-col gap-4 p-4 sm:p-6 bg-[var(--bg-admin-app)]">
    <UAlert
      v-if="error"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('admin.settings.loadFailed')"
    />

    <UCard class="border-[var(--border-admin)] bg-[var(--bg-admin-card)] text-[var(--text-admin-main)]">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-[var(--text-admin-main)]">
              {{ t('admin.settings.basic.title') }}
            </h2>
            <p class="mt-1 text-sm text-[var(--text-admin-muted)]">
              {{ t('admin.settings.basic.description') }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft" class="border border-[var(--border-admin)]">
            site.name
          </UBadge>
        </div>
      </template>

      <form class="grid max-w-2xl gap-4" @submit.prevent="submit">
        <UFormField :label="t('admin.settings.siteName')" name="site-name">
          <UInput
            v-model="siteName"
            icon="i-lucide-message-square-text"
            :placeholder="t('admin.settings.siteNamePlaceholder')"
            maxlength="80"
            required
          />
        </UFormField>

        <div class="flex justify-end">
          <UButton
            type="submit"
            leading-icon="i-lucide-save"
            :loading="saving"
          >
            {{ t('admin.settings.save') }}
          </UButton>
        </div>
      </form>
    </UCard>
  </div>
</template>
```

- [ ] **Step 4: 提交页面修改**

```bash
git add apps/web/app/pages/admin/index.vue apps/web/app/pages/admin/roles.vue apps/web/app/pages/admin/settings/index.vue
git commit -m "feat: integrate page components with useAdminTabs and set explicit vue names for cache matching"
```

---

### Task 5: 全局集成 Keep-Alive 精准缓存

**Files:**
- Modify: `apps/web/app/app.vue`

- [ ] **Step 1: 修改 `app.vue` 将 `<NuxtPage>` 绑定缓存参数**

修改 `apps/web/app/app.vue`：
```html
<script setup lang="ts">
import { useAdminTabs } from '~/composables/useAdminTabs'

const localeHead = useLocaleHead({
  dir: true,
  lang: true,
  seo: true
})
const { siteName, refresh } = useWebOptions()
const startupOptionsTimeout = import.meta.dev ? 800 : 2000

// 引入页签缓存控制列表
const { cachedTabNames } = useAdminTabs()

await useAsyncData('web-options', async () => {
  // 开发热重载时 API 可能还在编译，首屏先使用本地默认站点配置。
  await refresh({ timeout: startupOptionsTimeout }).catch(() => null)
  return true
})

useHead(() => ({
  htmlAttrs: localeHead.value.htmlAttrs,
  link: localeHead.value.link,
  meta: localeHead.value.meta,
  titleTemplate: (title) => title ? `${title} - ${siteName.value}` : siteName.value
}))
</script>

<template>
  <UApp>
    <NuxtLayout>
      <NuxtPage :keepalive="{ include: cachedTabNames }" />
    </NuxtLayout>
  </UApp>
</template>
```

- [ ] **Step 2: 提交集成缓存的修改**

```bash
git add apps/web/app/app.vue
git commit -m "feat: bind NuxtPage keepalive to active tabs list for precise layout state caching"
```

---

### Task 6: 验证运行与测试

**Files:**
- Test: `tests/validate-admin-framework.ts`

- [ ] **Step 1: 运行现有的自动化测试验证重构的完整性**

Run: `bun run tests/validate-admin-framework.ts`
Expected: PASS ("Admin framework validation passed.")

- [ ] **Step 2: 运行类型检查以确保全新代码完全合规**

Run: `bun run --cwd apps/web typecheck`
Expected: PASS

- [ ] **Step 3: 构建后台以验证静态包生成完全无误**

Run: `bun run --cwd apps/web build`
Expected: Success without build errors
