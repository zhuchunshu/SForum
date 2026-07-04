# 首页布局宽度与比例优化设计文档

设计并实施 SForum 首页布局的宽度优化，扩宽整体容器并优化栏宽比例，提高主帖子流的阅读空间和视觉呼吸感。

## 背景与目的

SForum 首页目前采用的经典三栏式容器最大宽度为 `max-w-6xl (1152px)`。在这种限制下，中间的主帖子列表流在桌面端只有 `576px`，左右侧栏各占 `288px`。这导致中间的帖子列表内容展示非常拥挤，特别是帖子标题、摘要和徽章的排列比较紧凑，影响了阅读体验。
本设计通过将最外层容器放宽至 `max-w-[1376px]`，并使用更精确的非等分栏宽划分（左栏 `270px`，右栏 `290px`，中栏 `720px`），使得中间主内容区域的宽度提升达 25%，从而显著提升阅读体验。

## 详细设计

### 1. 最外层容器宽度优化

将 [apps/web/app/pages/index.vue](file:///Users/inkedus/Code/SForum/apps/web/app/pages/index.vue) 模板中的 `<div class="max-w-6xl mx-auto px-4 sm:px-6">` 更改为 `<div class="max-w-[1376px] mx-auto px-4 sm:px-6">`。

### 2. 三栏网格重新划分

更换原有的 `grid-cols-12` 12列等分布局，采用 Tailwind 任意列宽属性 `grid-cols-[...]`。
具体网格定义如下：
```html
<div class="grid grid-cols-1 md:grid-cols-[1fr_290px] lg:grid-cols-[270px_1fr_290px] gap-6">
```

- **左侧栏 (Aside - Left)**：
  - 类名修改：由 `hidden lg:block lg:col-span-3 space-y-6` 变更为 `hidden lg:block space-y-6`。
  - 宽度在桌面端固定为 `270px`。
- **中间主内容区 (Section - Middle)**：
  - 类名修改：由 `col-span-12 md:col-span-9 lg:col-span-6 space-y-4` 变更为 `space-y-4`（在 Grid 下自动填充 `1fr`）。
  - 在最大容器宽度下，扣除边距后宽度为 `720px`。
- **右侧栏 (Aside - Right)**：
  - 类名修改：由 `hidden md:block md:col-span-3 space-y-6` 变更为 `hidden md:block space-y-6`。
  - 宽度在平板和桌面端均固定为 `290px`。

### 3. 测试与验证设计

修改验证首页结构的测试脚本 [tests/validate-homepage.js](file:///Users/inkedus/Code/SForum/tests/validate-homepage.js)：
- 移除对 `'grid-cols-12'`、`'col-span-12'` 和 `'lg:col-span-6'` 的过时检查。
- 新增对 `'max-w-[1376px]'`、`'md:grid-cols-[1fr_290px]'` 和 `'lg:grid-cols-[270px_1fr_290px]'` 的校验，确保布局逻辑符合本设计。

## 验证计划

- **静态测试**：运行 `node tests/validate-homepage.js`，验证结构、组件及SEO元数据是否正常。
- **项目构建与类型检查**：运行项目原有的构建验证命令，确保项目无编译错误。
- **视觉及响应式验证**：通过浏览器可视化小助手或本地开发服务预览，验证在不同分辨率下（Mobile/Tablet/Desktop）的排版表现。
