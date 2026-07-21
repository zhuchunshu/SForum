# 导航栏语言选择器样式优化设计方案

## 需求背景
当前前台导航栏的语言选择器（`SFNavbar.vue`）使用硬实边框、无语言名称文本且下拉菜单设计较为粗糙，与论坛整体 Pine Teal 现代极简设计规范（Teal 调性、去渐变、低圆角）不够协调。需要对其样式进行优化。

## 方案选型
采用 **方案 A**：无边框极简文字+地球图标。
在桌面端展示当前选中的语言名称，降低交互的认知负荷；移动端自适应隐藏文本以节省导航栏横向空间。

## 详细设计

### 1. 结构变更 (`apps/web/app/components/SFNavbar.vue`)
* 在语言选择按钮内部新增 `span` 标签用于展示当前选中的语言：
  ```html
  <span class="navbar__lang-text">{{ currentLocaleName }}</span>
  ```
* 增加计算属性 `currentLocaleName`：
  ```typescript
  const currentLocaleName = computed(() => {
    return locales.value.find(loc => loc.code === locale.value)?.name || ''
  })
  ```

### 2. 样式变更
* **选择按钮 (`.navbar__lang-btn`)**：
  * 去除默认边框和白背景：`border: none; background: transparent;`
  * 设置与导航栏链接类似的边距与高：`height: 32px; padding: 0 8px;`
  * 新增 hover 态背景和文字颜色切换：`background: #f3f4f6; color: #111827;`
* **当前语言文本 (`.navbar__lang-text`)**：
  * 字体大小设为 `13px`，粗细为 `500`，字色为 `#4b5563`。
  * 移动端自适应（`< 640px`）设置隐藏：`display: none;`
* **下拉菜单容器 (`.navbar__dropdown`)**：
  * 圆角调整为更圆润的 `10px`，投影调整为更加柔和的扩散阴影：`box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08)`。
  * 修改定位，让其在按钮下方紧凑贴合：`top: calc(100% + 4px);`。
* **下拉菜单选项 (`.navbar__dropdown-item--lang`)**：
  * 选中状态高亮颜色修改为：背景 `#f0faf9`，文字 `#0f766e`。

## 验证计划
* **功能验证**：点击语言按钮，确认下拉菜单能正确展开和关闭。选择不同语言，确认页面能正确切换，且选择按钮内的文字实时更新。
* **移动端验证**：缩窄浏览器窗口至 `640px` 以下，确认文字自动消失且仅保留地球图标。
