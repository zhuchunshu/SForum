# 后台分类与标签图标颜色设计

## 背景与目标

SForum 的论坛 taxonomy 已经支持后台管理分类、分类分组、标签和标签策略。后台现在只能编辑文本、状态、排序等结构字段，缺少分类和标签的视觉标识配置。

本设计目标是在后台给分类和标签增加可配置的图标与图标颜色，并在后台列表中直接预览。首版只服务后台管理体验，不改变前台公开页面、主题摘要、标签摘要或搜索文档形态。

## 已确认范围

### 首版包含

- 后台分类 `categories` 支持设置 `icon` 与 `iconColor`。
- 后台标签 `tags` 支持设置 `icon` 与 `iconColor`。
- 后台创建、编辑、列表接口读写并返回这两个字段。
- 后台分类与标签表单复用现有 `SFIconPicker` 选择图标。
- 后台表单提供颜色输入，保存十六进制颜色值。
- 后台列表在名称旁显示图标和颜色预览。
- OpenAPI contract、前端类型、后端模型、迁移和测试同步更新。

### 首版不包含

- 不给 `category_groups` 增加图标或颜色字段。
- 不改变公共 `GET /categories`、`GET /category-groups`、`GET /tags` 的前台页面展示语义。字段可随同后台读取返回，但前台主题层不消费、不展示。
- 不改变 `TopicTagSummary`、`TopicSummary`、搜索索引文档或话题列表行。
- 不新增权限 key；继续使用现有 `category.manage` 与 `tag.manage`。
- 不做图标资源上传；图标仍来自 Nuxt Icon / Iconify 本地集合。

## 数据模型

在数据库中直接给稳定业务实体加字段，而不是使用 JSON metadata 或 `web_options`：

- `categories.icon TEXT NOT NULL DEFAULT ''`
- `categories.icon_color TEXT NOT NULL DEFAULT ''`
- `tags.icon TEXT NOT NULL DEFAULT ''`
- `tags.icon_color TEXT NOT NULL DEFAULT ''`

直接列字段的好处是契约明确、迁移简单、后台列表不需要额外配置查询。未来如果前台要展示分类/标签视觉标识，可以直接复用同一模型字段。

字段语义：

- `icon` 保存 Nuxt Icon 名称，例如 `i-tabler-message-circle` 或 `i-lucide-tag`。
- `iconColor` 保存十六进制颜色，例如 `#0f766e`。
- 空字符串表示未配置。后台预览空值时使用默认图标和主题色，不向数据库写入推断值。

## 后端设计

### 模型与规范化

在 `forum.Category` 和 `forum.Tag` 中增加 `Icon`、`IconColor` JSON 字段。创建和更新输入同步增加 `Icon`、`IconColor`。

规范化规则：

- `icon`：trim 后转小写；为空允许；非空必须是简单 Nuxt Icon 形式，匹配 `^i-[a-z0-9]+-[a-z0-9][a-z0-9-]*$`。
- `iconColor`：trim 后转小写；为空允许；非空必须匹配 `^#[0-9a-f]{6}$`。
- 分类字段非法时返回现有 `forum.topic_invalid`。
- 标签字段非法时返回现有 `forum.tag_invalid`。

不校验图标是否真实存在于本地集合。原因是现有 `SFIconPicker` 已支持自定义 Nuxt Icon 名称，后端只负责保存安全、可渲染的名称形态，避免把前端图标目录细节耦合进 Go API。

### Store 与查询

Postgres store 的分类和标签查询、插入、更新都包含新增字段：

- `ListCategories`
- `ListCategoryGroups`
- `ListTags`
- `CreateCategory`
- `UpdateCategory`
- `CreateTag`
- `UpdateTag`

缓存 store 不需要改变缓存策略；分类/标签写入后仍走已有 store 写路径，读缓存的 generation 机制继续负责失效。

### 控制器与权限

Admin request struct 增加：

- 分类 create/update: `icon`, `iconColor`
- 标签 create/update: `icon`, `iconColor`

权限不变：

- 分类创建/更新仍要求 `category.manage`。
- 标签创建/更新仍要求 `tag.manage`。

本设计不新增 unsafe route，也不降低 API policy 的权威性。

## OpenAPI 设计

更新 `contracts/openapi/schemas/forum.yaml`：

- `Category` 增加 required 字段 `icon`、`iconColor`。
- `Tag` 增加 required 字段 `icon`、`iconColor`。
- `CreateCategoryRequest` / `UpdateCategoryRequest` 增加可选 `icon`、`iconColor`。
- `CreateTagRequest` / `UpdateTagRequest` 增加可选 `icon`、`iconColor`。

字段 schema：

- `icon`: string，说明为 Nuxt Icon name such as `i-tabler-message-circle`。
- `iconColor`: string，说明为 hex color such as `#0f766e`。

路径文件不需要新增 endpoint，只需保持既有 schema 引用。

## 前端后台设计

### 类型与 API helper

更新 `apps/web/app/utils/forumTaxonomy.ts`：

- `ForumCategory` 增加 `icon: string`、`iconColor: string`。
- `ForumTag` 增加 `icon: string`、`iconColor: string`。

更新 `apps/web/app/utils/adminForum.ts`：

- `AdminForumCategoryPayload` 增加 `icon`、`iconColor`。
- `AdminForumTagPayload` 增加 `icon`、`iconColor`。
- `createCategoryPayload` 与 `createTagPayload` 默认返回空字符串。
- payload helper trim 输入并保留空值。

### 分类后台页

在 `apps/web/app/pages/admin/forum/categories.vue` 的分类表单中增加“图标”和“图标颜色”区域：

- `LazySFIconPicker v-model="categoryForm.icon"`。
- 颜色输入使用原生 `input type="color"` 搭配文本输入或清空按钮，避免颜色值不可清空。
- 空颜色预览使用 CSS 变量 `var(--sf-accent)`，但保存 payload 仍为空字符串。

分类列表行在分类名称前显示图标预览：

- 有 `category.icon` 时用该图标，否则使用 `i-lucide-folder-open`。
- 有 `category.iconColor` 时使用该颜色，否则使用主题 accent。
- 图标和名称作为一个紧凑组合，保留 slug、可见性、统计信息和编辑按钮。

分类分组表单和分组列表不增加图标字段。

### 标签后台页

在 `apps/web/app/pages/admin/forum/tags.vue` 的标签表单中增加同样的图标与颜色输入。

标签列表行在 `#标签名` 前显示图标预览：

- 有 `tag.icon` 时用该图标，否则使用 `i-lucide-tag`。
- 有 `tag.iconColor` 时使用该颜色，否则使用主题 accent。
- 状态 tab、审核、停用和编辑动作不变。

### 交互与反馈

保存成功继续使用现有 success Toast，颜色遵守 SForum appearance tokens。字段验证错误继续由 API 返回的阻塞错误提示，不用成功或错误 Toast 替代表单必读信息。

## 国际化

新增或复用后台文案：

- 图标
- 图标颜色
- 清空颜色
- 默认主题色

中英文 locale 都需要补齐。UI 不使用 emoji 作为图标或状态标记。

## 测试与验证

### 后端测试

优先补服务层或 store 层测试：

- 分类创建时保存规范化后的 `icon` 和 `iconColor`。
- 分类更新可修改或清空 `icon` 和 `iconColor`。
- 标签创建/更新同样覆盖。
- 非法 icon 或非法颜色分别返回现有 invalid 错误。

### 前端测试

更新 `apps/web/tests/adminForum.test.ts`：

- 默认分类 payload 包含空 `icon`、`iconColor`。
- 默认标签 payload 包含空 `icon`、`iconColor`。
- 从已有分类/标签创建 payload 时保留 icon/color。

必要时更新 taxonomy 类型相关测试，保证新增字段不会破坏既有 helper。

### Contract 与命令

编辑 OpenAPI 后运行：

- `ruby scripts/validate-openapi-refs.rb`

实现完成后至少运行：

- `cd apps/api && go test -count=1 ./app/Models/Forum ./app/Http/Controllers/Forum`
- `cd apps/web && bun test tests/adminForum.test.ts tests/forumTaxonomy.test.ts`
- `cd apps/web && bun run typecheck`

若本轮改动触及公共 API 读取路径，可追加相关 controller 测试或全量 `./scripts/test.sh`。

## 后续计划入口

用户确认本 spec 后，进入 implementation plan。计划应拆为：

1. 后端迁移、模型、规范化与 store 读写。
2. 后端控制器、OpenAPI 与 Go 测试。
3. 前端类型、payload helper 与 Bun 测试。
4. 后台分类/标签表单和列表预览 UI。
5. i18n、typecheck、OpenAPI 校验和最终验证。
