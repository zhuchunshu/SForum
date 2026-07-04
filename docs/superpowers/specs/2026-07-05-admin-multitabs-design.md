# SForum 后台多标签（Multi-Tab）与双模式 UI 重构设计规范

## 1. 背景与目标
为了提升 SForum 后台管理系统的美观度与易用性，我们需要对现有的后台 UI 框架进行重构：
- **设计风格**：采用经典的 SaaS 主题（浅色模式下为暗色侧边栏 + 浅色内容区，深色模式下为整体暗色系）。
- **多标签页支持**：在顶栏下方提供类似 IDE 的多标签页（Multi-Tab）导航栏，支持动态打开、切换与 `×` 按钮关闭，并提供状态缓存机制（KeepAlive）。
- **图标规范**：禁止在后台使用任何 Emoji 字符，所有图标必须统一使用 `@iconify-json/lucide` 图标库中的 `i-lucide-*` 语义化图标。

## 2. 系统架构与技术实现

### 2.1 全局多标签状态管理 (`useAdminTabs.ts`)
我们使用 Nuxt 的全局 Composables 导出一个响应式的 `useAdminTabs` 状态，来维护打开的标签页列表：
```typescript
interface AdminTab {
  id: string       // 路由标识，例如 'admin-index' 或 'admin-roles'
  label: string    // Localized title (e.g. t('admin.nav.dashboard'))
  to: string       // 路由路径 (e.g. '/control-panel/roles')
  icon: string     // 统一以 i-lucide- 开头
  closable: boolean // 首页仪表盘固定不可关闭，其他可关闭
}
```
主要提供的 API：
- `tabs`: 当前打开的标签列表。
- `currentTabId`: 当前激活的标签 ID。
- `openTab(tab: AdminTab)`: 打开新标签（若已存在则直接激活）。
- `closeTab(id: string)`: 关闭指定标签。若关闭的是当前激活标签，则自动激活最近使用的一个标签。
- `resetTabs()`: 初始化标签页，保留默认的“仪表盘”标签。

### 2.2 多标签与页面 Keep-Alive 缓存
为了保证在切换标签页时不会丢失未提交的数据（例如正在填写的表单、表格的搜索关键字与分页），我们将采用以下方案：
- 在 `admin.vue` 布局文件中使用 `<NuxtPage>` 渲染子页面。
- 将 `<NuxtPage>` 与 Vue 3 的 `<KeepAlive>` 结合：
  ```html
  <router-view v-slot="{ Component, route }">
    <keep-alive :include="cachedTabRoutes">
      <component :is="Component" :key="route.fullPath" />
    </keep-alive>
  </router-view>
  ```
- `cachedTabRoutes` 是计算属性，其值即为 `useAdminTabs().tabs` 中所有打开路由对应的组件名称集合。当标签页被关闭时，该路由名称从 `cachedTabRoutes` 中移除，从而触发组件卸载并自动释放缓存。

### 2.3 语义化 CSS 双模式主题设计
清理 `/apps/web/app/assets/css/main.css` 中对 `:root` 全局浅色样式的强制锁定，定义一套基于 `.dark` 类响应的 CSS 语义变量：

| CSS 变量名 | 浅色模式 (Light Mode) | 深色模式 (Dark Mode) | 用途描述 |
| :--- | :--- | :--- | :--- |
| `--bg-admin-app` | `#f8fafc` (Slate 50) | `#09090b` (Zinc 950) | 主面板底层背景 |
| `--bg-admin-sidebar` | `#0f172a` (Slate 900) | `#09090b` (Zinc 950) | 侧边栏背景（浅色模式保持暗色） |
| `--bg-admin-card` | `#ffffff` | `#18181b` (Zinc 900) | 卡片/白板容器背景 |
| `--border-admin` | `#e2e8f0` (Slate 200) | `#27272a` (Zinc 800) | 细分割线与边框 |
| `--text-admin-main` | `#0f172a` (Slate 900) | `#f4f4f5` (Zinc 100) | 主要正文颜色 |
| `--text-admin-muted` | `#64748b` (Slate 500) | `#a1a1aa` (Zinc 400) | 辅助或说明性文字 |

## 3. UI 界面层改进点

### 3.1 侧边栏重构 (`admin.vue`)
- 左侧侧边栏组件统一为 `UDashboardSidebar`，通过 Nuxt UI 的 `collapsible` 支持收起。
- 导航菜单项统一移除表情符号，改用配置式 `icon` 参数，如 `i-lucide-layout-dashboard` 等。
- 底部用户卡片支持通过下拉菜单操作，并支持通过切换开关直接更改系统的颜色模式。

### 3.2 多标签页栏设计 (`admin.vue`)
- 在 Topbar (`UDashboardNavbar`) 下方绘制一条高度为 `38px` 的页签栏。
- 标签项具有圆角，激活项使用背景色与内容区融为一体（消除底边框线），文字高亮并显示 `i-lucide-x` 关闭图标。
- 关闭按钮只有在 hover 或者是激活状态下才显现，保持页面的高精致度与微动画反馈。

### 3.3 主内容区卡片美化 (`admin/index.vue`)
- 将现有概览卡片的硬编码蓝色/绿色等样式，迁移至 Nuxt UI 推荐的 soft 配色与语义化边框。
- 移除“管理模块”卡片中的表情图标，改用 Lucide 语义图标。

## 4. 验证计划
### 4.1 手动验证 (QA)
- 登录后台系统，分别在系统的深色模式和浅色模式下，检查文字对比度是否满足 AAA 级无障碍阅读标准。
- 多次点击左侧“仪表盘”、“用户组”、“站点设置”菜单，验证多标签栏是否正常打开并高亮。
- 在“站点设置”中输入测试文本，然后切换到“仪表盘”再切回“站点设置”，验证刚才输入的文本未丢失（状态成功缓存）。
- 点击“站点设置”页签上的 `×` 关闭它，然后再次打开“站点设置”，验证输入框已被重置（状态成功销毁）。

### 4.2 自动化测试
- 运行现有的 `tests/validate-admin-framework.ts` 自动化脚本，验证重构后系统后台路由的可访问性以及组件功能的基本健壮性。
