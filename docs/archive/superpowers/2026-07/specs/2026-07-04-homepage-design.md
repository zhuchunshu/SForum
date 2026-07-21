# SForum 论坛首页 UI 设计规范与实现规范

本设计文档定义了 SForum 论坛首页（`apps/web/app/pages/index.vue`）的布局结构、UI 组件编排、交互行为和多语言（i18n）映射关系。首个首页版本将采用松石绿（Pine Teal Clean）的视觉主题与经典三栏式门户（3-Column Portal）布局。

## 视觉主题与色彩配置

* **基础背景色**: `#F7F8FA` (Slate-50 级别浅灰底色)
* **卡片背景色**: `#FFFFFF` (纯白，薄边框，微投影)
* **主色调 (Accent)**: `#0F766E` (Pine Teal 经典松石绿)
* **文本色**: 主色 `#111827` (Slate-900)，次要色 `#475569` (Slate-600)
* **边框色**: `#E2E8F0` (Slate-200)

## 页面布局架构

整个首页嵌套在 Nuxt 默认布局中，核心容器结构如下：

```html
<main class="max-w-6xl mx-auto px-4 sm:px-6 py-8">
  <div class="grid grid-cols-12 gap-6">
    <!-- 1. 左栏 (Col-span 3): 导航与板块 -->
    <aside class="hidden lg:block lg:col-span-3 space-y-6">
      <!-- 导航列表卡片 -->
      <!-- 讨论板块分类卡片 -->
    </aside>

    <!-- 2. 中栏 (Col-span 6 / 12): 信息流与帖子列表 -->
    <section class="col-span-12 lg:col-span-6 space-y-4">
      <!-- 帖子筛选 Tabs 与 搜索框 -->
      <!-- 帖子流列表 (循环 SFFeedRow) -->
      <!-- 分页组件 -->
    </section>

    <!-- 3. 右栏 (Col-span 3): 小工具栏 -->
    <aside class="hidden md:block md:col-span-3 space-y-6">
      <!-- 用户状态/登录卡片 -->
      <!-- 签到模块 -->
      <!-- 热门讨论卡片 -->
      <!-- 站点数据统计卡片 -->
    </aside>
  </div>
</main>
```

### 自适应排版响应式规则
1. **桌面分辨率 (>= 1024px)**: 完整显示左、中、右三栏布局。
2. **平板分辨率 (768px - 1023px)**: 隐藏左栏，右栏移到最右，中栏自动扩宽至占满 9 栏（`col-span-9`）。
3. **移动分辨率 (< 768px)**: 隐藏左栏与右栏，中栏独立占满全宽（`col-span-12`）。分类和侧栏小工具的入口折叠进导航栏。

---

## 编排组件库细节

首页将复用及整合以下现有的 `SF` 前端组件：

| 组件名称 | 复用位置与用途 |
| :--- | :--- |
| `SFNavbar.vue` | 页面顶栏（在 Layout 中全局生效，非 `index.vue` 直接编写，但与首页呼应） |
| `SFCard.vue` | 侧边栏所有卡片框、中栏帖子项的统一外容器 |
| `SFSearch.vue` | 中栏帖子流顶部的即时搜索过滤框 |
| `SFTabs.vue` | 最新、热门、精华等帖子排序规则选择选项卡 |
| `SFFeedRow.vue` | 核心信息流单行，展示标题、分类徽标、浏览回复数、最后回复时间 |
| `SFAvatar.vue` | 帖子发布者头像、右侧边栏当前登录用户头像 |
| `SFBadge.vue` | 热门板块前的徽标、帖子置顶/加精等状态标签 |
| `SFButton.vue` | 每日签到按钮、发表主题按钮、未登录下的登录/注册按钮 |
| `SFPage.vue` | (未启用) |
| `SFPagination.vue`| 帖子列表底部的分页翻页组件 |
| `SFSkeleton.vue` | 加载过程中占位的骨架屏多行效果 |
| `SFEmptyState.vue`| 搜索无结果或无帖子时的友好提示插画 |

---

## 交互行为与状态机

### 1. 帖子过滤与切换
* 触发 `SFTabs` 的切换事件时，更新内部状态 `currentTab`（可选值：`latest` / `hot` / `featured` / `following`）。
* 主流区域应响应式触发异步数据重载，并在加载时显示 `SFSkeleton` 骨架行（默认渲染 5 行）。

### 2. 即时搜索与过滤
* 搜索框绑定值 `searchQuery`。
* 过滤帖子列表，当无符合条件的帖子时展示 `SFEmptyState` 组件。

### 3. 每日签到
* 用户点击“签到”按钮后，状态变为“已签到”（状态持久化于 Vue 组件 State，并在真实应用中对接 API）。
* 签到按钮伴有微妙的缩放变形过渡动效。

### 4. 未登录引导状态
* 当 `useAuthSession` 检查到用户未登录时，右栏的“个人卡片”转为展示“欢迎加入 SForum”的插页卡片，提供快速的“登录”与“注册”按钮；“我关注的”帖子 tab 显示置灰或登录提示。

---

## 多语言 (i18n) 变量定义

需更新 `apps/web/i18n/locales/` 下的 `zh-CN.json` 和 `en-US.json` 资源文件。

### 简体中文 (zh-CN)
```json
"home": {
  "metaTitle": "SForum 社区首页",
  "metaDescription": "SForum 是一个可维护、高性能的极简社区论坛。",
  "searchPlaceholder": "搜索帖子标题、内容...",
  "filter": {
    "latest": "最新",
    "hot": "热门",
    "featured": "精华",
    "following": "关注"
  },
  "sidebar": {
    "navTitle": "导航",
    "navHome": "论坛首页",
    "navCategories": "板块分类",
    "navTags": "热门标签",
    "navMembers": "活跃成员",
    "sections": "讨论板块",
    "secTech": "技术分享",
    "secCreative": "创意设计",
    "secLife": "日常生活",
    "secNotice": "官方公告",
    "userCard": "个人中心",
    "userPosts": "帖子",
    "userLikes": "获赞",
    "welcomeTitle": "欢迎来到 SForum",
    "welcomeDesc": "这里是一个认真讨论事情的地方。立即加入以和数千名创作者一同分享经验。",
    "loginBtn": "登录账号",
    "registerBtn": "注册新账号",
    "checkIn": "每日签到",
    "checkedIn": "已连续签到 {days} 天",
    "hotThreads": "热门讨论",
    "forumStats": "全站数据统计",
    "statThreads": "主题总数",
    "statReplies": "累计回复",
    "statMembers": "注册会员",
    "statOnline": "当前在线"
  }
}
```

### 英文 (en-US)
```json
"home": {
  "metaTitle": "SForum Community",
  "metaDescription": "SForum is a maintainable, high-performance minimalist community forum.",
  "searchPlaceholder": "Search threads, content...",
  "filter": {
    "latest": "Latest",
    "hot": "Hot",
    "featured": "Featured",
    "following": "Following"
  },
  "sidebar": {
    "navTitle": "Navigation",
    "navHome": "Home",
    "navCategories": "Categories",
    "navTags": "Hot Tags",
    "navMembers": "Members",
    "sections": "Discussion Boards",
    "secTech": "Tech Talk",
    "secCreative": "Creative Design",
    "secLife": "Daily Life",
    "secNotice": "Announcements",
    "userCard": "Profile",
    "userPosts": "Posts",
    "userLikes": "Likes",
    "welcomeTitle": "Welcome to SForum",
    "welcomeDesc": "This is a place to discuss things seriously. Join today and start sharing with thousands of creators.",
    "loginBtn": "Log In",
    "registerBtn": "Register Account",
    "checkIn": "Check In",
    "checkedIn": "Checked in {days} d",
    "hotThreads": "Hot Discussions",
    "forumStats": "Community Stats",
    "statThreads": "Total Threads",
    "statReplies": "Total Replies",
    "statMembers": "Registered Members",
    "statOnline": "Online Active"
  }
}
```

---

## 首页数据契约 (Data Contracts)

我们将在 `index.vue` 中定义 Mock 数据，并预留 `useAsyncData` 的骨架调用：

```typescript
interface Thread {
  id: number
  title: string
  category: string
  categoryKey: string
  author: {
    username: string
    displayName: string
    avatarChar: string
  }
  repliesCount: number
  viewsCount: number
  timeAgo: string
  isPinned?: boolean
  isFeatured?: boolean
}

interface HotTopic {
  id: number
  title: string
  repliesCount: number
}

interface Category {
  name: string
  key: string
  count: number
}
```

---

## SEO 元数据配置

```typescript
useSeoMeta({
  title: () => t('home.metaTitle'),
  description: () => t('home.metaDescription'),
  ogTitle: () => t('home.metaTitle'),
  ogDescription: () => t('home.metaDescription'),
  ogType: 'website'
})
```

---

## 验证与验收准则

1. **多端响应式正常**：在手机、平板、桌面三种屏幕尺寸下，各栏的可见性与排版完全符合响应式规则，无横向滚动条溢出。
2. **多语言切换流畅**：点击中英文切换链接时，首页面板所有的板块名称、侧栏标签、按钮动作均能完美渲染对应的多语言文案。
3. **交互正确**：
   * 帖子分类（Tabs）可正常点击并高亮当前项。
   * 搜索框输入时能正确筛选中栏的帖子标题（匹配 Mock 数据）。
   * 签到按钮点击后正确变更为“已签到”样式，并显示动效。
4. **编译与质量校验**：
   * 运行 `bun run typecheck` 无 TypeScript 错误。
   * 运行 `bun run build` 正常生成生产包，无异常。
