# 2026-07-05 Admin Multi-Tabs and Layout Rules

## Status

Accepted.

## Context

SForum has refactored the admin dashboard into a premium multi-tab, dual-theme layout. To ensure that future development of new admin screens remains consistent, maintainable, and visual-standard compliant, we need a set of clear structural rules and conventions documented in the knowledge base.

## Decision

When developing new pages or sections in the SForum Admin Control Panel (`apps/web/app/pages/admin`), developers must follow these five design and technical rules:

### 1. Page Component Structure (页面结构规范)
- **Do not render** `<UDashboardNavbar>` in individual admin page components. The topbar is globally managed by the layout `admin.vue`.
- **Render a local page header** section instead, e.g., `<div class="mb-4"><h2 class="text-xl font-bold flex items-center gap-2">...</h2></div>`.
- **Use `<UDashboardToolbar>`** directly below the local header to wrap page actions (e.g., search inputs, refresh buttons, and action links in the `#right` slot). This unifies page density and visual layout.

### 2. Multi-Tab Registration (多页签注册)
- Every admin page must register itself as a tab in its `onMounted` hook.
- Use `useAdminTabs().openTab(routePath, labelTranslationKey, iconString, componentName)`:
  ```typescript
  const adminTabs = useAdminTabs()
  onMounted(() => {
    adminTabs.openTab('/roles', 'admin.nav.roles', 'i-lucide-shield-check', 'AdminRoles')
  })
  ```

### 3. Keep-Alive Component Caching (状态缓存与销毁)
- All admin pages must declare their exact component name using `defineOptions` matching the `componentName` string passed to `openTab`:
  ```typescript
  defineOptions({
    name: 'AdminRoles'
  })
  ```
- This enables precise state caching when tabs are open, and automatically purges page memory and input states when the tab is closed by the user.
- **Keep-Alive Navigation Synchronization**: Because pages are cached by `<KeepAlive>`, switching between already open tabs does not re-trigger the page's `onMounted` hook. To ensure the active tab indicator stays synchronized, the global `admin.vue` layout watches `route.path` and parses the current location into a valid tab ID using `resolveTabIdFromPath` to update `adminTabs.activeTabId.value` reactively.

### 4. Sidebar Theme Adaptivity and Nested Menus (侧边栏及多级菜单)
- The sidebar background and text colors must adapt to the color theme (white background in light mode, dark zinc-950 in dark mode). Avoid inline dark overrides (e.g., `text-slate-400!`).
- Sidebar menu configurations (`navigationItems`) support nested lists. Use the `children` array and `defaultOpen: true` to trigger automatic accordion folding:
  ```typescript
  {
    label: 'System Config',
    icon: 'i-lucide-settings-2',
    defaultOpen: true,
    children: [
      { label: 'Site Settings', icon: 'i-lucide-sliders', to: adminRoutes.path('/settings') }
    ]
  }
  ```
- Any menu font-size or item padding overrides must be done via the `#sforum-admin-sidebar` CSS selector in `main.css`.

### 5. Strict No-Emoji Policy (严禁 Emoji 图标)
- All sidebar options, tabs, page headers, tables, buttons, and alert states **must not use emojis**.
- Use Lucide icons (`i-lucide-*`) or project-approved icon bundles.

### 6. Standardized Form Actions & Sticky Footer (统一表单操作与吸底保存条)
- **吸底布局规范**：所有后台配置页面、长表单编辑页面，若含有“保存”、“重置/取消”等操作，**严禁**直接平铺在卡片底部。
- **统一使用 `SFAdminFormFooter`**：必须在 `<form>` 的底部引入 `<SFAdminFormFooter>` 组件。它会自动粘性吸附在内容视口底部，确保用户无需滚动页面即可执行保存。
- **状态联动规范**：
  - 保存按钮必须绑定 `:loading="saving"`，防止重复提交。
  - 表单的 Input/Select 在提交中应当伴随禁用，或通过 `disabled` 属性统一传递给 Footer。
  - 按钮图标严格使用 `i-lucide-*`，遵循无 emoji 规范。

## Consequences

- New admin features can be quickly created by copying a template page that uses the layout and registering the tab in `onMounted`.
- Page states (e.g. form fields) are preserved across tabs naturally.
- The UI maintains a clean, modern SaaS hierarchy with a global topbar and tab navigation.
