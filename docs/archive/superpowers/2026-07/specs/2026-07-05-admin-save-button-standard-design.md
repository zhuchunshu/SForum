# 后台保存按钮与表单操作规范设计 (Admin Save Button Standard Design)

为了提升 SForum 管理后台的交互质量与一致性，本设计规范明确了后台表单保存按钮的视觉样式、交互联动、吸底悬浮定位以及相关开发规范。

## 1. 目标与背景

当前后台站点设置等表单页面，保存按钮平铺于卡片底部。在内容较长或屏幕高度有限时，用户必须向下滚动到底部才能找到保存按钮，体验不够直观。
本方案通过实现一个**悬浮粘性底栏 (Sticky Bottom Bar)** 组件及配套规范，确保保存/重置等表单操作始终在视口中可见，无需额外滚动。

---

## 2. 详细设计

### 2.1 统一操作组件 `SFAdminFormFooter.vue`

新建通用后台表单操作组件 `apps/web/app/components/SFAdminFormFooter.vue`。

#### 组件 API (Props)
- `saving` (类型: `boolean`, 默认值: `false`): 控制保存按钮的 loading 加载状态，且加载时重置按钮禁用。
- `disabled` (类型: `boolean`, 默认值: `false`): 禁用所有操作。
- `submitText` (类型: `string`, 可选): 自定义保存文案，默认使用 `admin.form.save`（“保存”）。
- `resetText` (类型: `string`, 可选): 自定义重置文案，默认使用 `admin.form.reset`（“重置”）。
- `submitIcon` (类型: `string`, 默认值: `'i-lucide-save'`): 保存按钮图标。
- `resetIcon` (类型: `string`, 默认值: `'i-lucide-rotate-ccw'`): 重置按钮图标。
- `showUnsavedAlert` (类型: `boolean`, 默认值: `false`): 若为 `true`，在左侧显示“检测到有未保存的更改”的警告图标与呼吸提示。

#### 组件插槽 (Slots)
- `#left`: 用于覆盖左侧提示文案或放置额外状态（如表单校验失败详情、自动保存时间戳等）。
- `#actions`: 用于完全替换右侧按钮组（仅在特定非保存表单场景下使用）。

#### 视觉设计与样式
- **悬浮粘性定位**: `sticky bottom-0 z-20 mt-8`，自动吸附在滚动区域底端。
- **毛玻璃效果**: 
  - 背景：`bg-white/90 dark:bg-zinc-900/90 backdrop-blur-sm`
  - 边框：`border border-slate-200 dark:border-zinc-800`
  - 阴影：`shadow-lg`
  - 圆角：`rounded-xl`
  - 内边距：`px-6 py-4`
- **动画**: 警示图标与文字呼吸渐变效果。

---

### 2.2 语言包 (i18n) 补充

在两套语言包的 `"admin"` 节点下新增 `"form"` 分组：

- `apps/web/i18n/locales/zh-CN.json`
  ```json
  "form": {
    "save": "保存",
    "saving": "保存中...",
    "reset": "重置",
    "unsavedChanges": "检测到有未保存的更改"
  }
  ```
- `apps/web/i18n/locales/en-US.json`
  ```json
  "form": {
    "save": "Save",
    "saving": "Saving...",
    "reset": "Reset",
    "unsavedChanges": "Unsaved changes detected"
  }
  ```

---

### 2.3 后台决策与开发规范扩展

更新 [2026-07-05-admin-multitabs-and-layout-rules.md](file:///Users/inkedus/Code/SForum/knowledge/decisions/2026-07-05-admin-multitabs-and-layout-rules.md)。
增加第六条：**6. Standardized Form Actions & Sticky Footer (统一表单操作与吸底保存条)**
- 强制要求所有包含配置、表单项的编辑页面，其保存操作必须使用 `SFAdminFormFooter`，并放置于 `<form>` 的末尾。
- 严禁手写平铺的保存按钮。
- 所有按钮图标严格使用 `i-lucide-*`，遵循无 emoji 规范。

---

## 3. 验证方案

### 3.1 页面验证
修改 `apps/web/app/pages/admin/settings/index.vue`（站点设置），将底部的旧“保存”按钮替换为新组件 `<SFAdminFormFooter>`：
1. 验证在屏幕较矮或内容较长时，保存栏能够正确吸底，不遮挡表单最后一项，保留充足 padding。
2. 验证点击“重置”能够恢复原站点名称。
3. 验证点击“保存”时显示加载动画，按钮不可重复点击，并且修改成功后弹出 toast 提示。
4. 验证在修改输入框内容后，左侧可以动态显示“检测到有未保存的更改”。

### 3.2 自动化测试验证
由于修改了保存按钮，若有任何 E2E 测试或组件单元测试，需确保通过。
运行测试脚本验证后台路由与设置界面运行正常。
