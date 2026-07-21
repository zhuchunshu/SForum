# 首页帖子信息流 UI 重新设计文档

重构 SForum 首页帖子信息流行组件 `SFFeedRow` 的布局结构与视觉外观，落实「左头像 + 右标题/元数据 + 底部交互药丸栏」的无摘要紧凑融合方案。

## 背景与目的

目前的帖子列表行（`SFFeedRow`）采用的是偏复古和机械的经典卡片式等分布局：左侧是“赞同”分栏，中间是标题、摘要与小字元数据，右侧是回复与浏览数纵向堆叠。这一设计不仅占用了大量纵向高度，而且在视觉上缺乏呼吸感，被用户反馈不够美观。

在通过可视化助手进行的 2 轮迭代脑暴后，用户最终选定了 **无摘要紧凑融合方案**。该方案去除了冗余的帖子内容摘要，提高信息密度，聚焦标题本身以引导点击，并通过左头像、标题、右置交互按钮和底部元数据的组合，展现更加现代、高档的论坛美学。

## 详细设计

### 1. `SFFeedRow.vue` 组件重构

我们将修改 [apps/web/app/components/SFFeedRow.vue](file:///Users/inkedus/Code/SForum/apps/web/app/components/SFFeedRow.vue)：
- **属性（Props）更新**：
  - 增加 `avatar` (string) 属性或直接复用 `author` 属性来传入作者的头像标识符（`SFAvatar` 可以自动通过作者名字生成首字母头像）。我们这里直接复用 `author` 属性用于在左侧渲染 `<SFAvatar :name="author" size="sm" />`。
- **模板结构更新**：
  ```html
  <article class="sf-feed-row">
    <!-- 1. 左侧圆形头像 -->
    <div class="sf-feed-row__avatar-wrapper">
      <SFAvatar :name="author" size="sm" />
    </div>
    
    <!-- 2. 右侧核心内容区 -->
    <div class="sf-feed-row__content">
      <!-- 2.1 标题 + 右置交互区 -->
      <div class="sf-feed-row__header">
        <h3 class="sf-feed-row__title">
          {{ title }}
        </h3>
        <div class="sf-feed-row__actions">
          <!-- 赞同/踩赞药丸 -->
          <div class="sf-feed-row__vote">
            <button class="sf-feed-row__vote-btn">▲</button>
            <span class="sf-feed-row__vote-val">{{ score }}</span>
            <button class="sf-feed-row__vote-btn">▼</button>
          </div>
          <!-- 回复数 -->
          <div class="sf-feed-row__action-tag">
            💬 {{ replies }}
          </div>
        </div>
      </div>
      
      <!-- 2.2 底部元数据行 -->
      <div class="sf-feed-row__meta-row">
        <!-- 标签集合 -->
        <span v-if="badges.length" class="sf-feed-row__badges">
          <SFBadge
            v-for="badge in badges"
            :key="badge.label"
            :variant="badge.variant || 'neutral'"
          >
            {{ badge.label }}
          </SFBadge>
        </span>
        <!-- 发布人 & 时间 -->
        <span class="sf-feed-row__author">{{ author }}</span>
        <span class="sf-feed-row__time">• {{ meta }}</span>
        <!-- 浏览量右对齐 -->
        <span class="sf-feed-row__views">👁️ {{ views }} 浏览</span>
      </div>
    </div>
  </article>
  ```

### 2. CSS 样式重构

修改组件库样式表 [apps/web/app/assets/css/sforum-components.css](file:///Users/inkedus/Code/SForum/apps/web/app/assets/css/sforum-components.css#L776-L838)：
- 废弃原有 `sf-feed-row` 类下的网格布局，应用全新 Flex 布局样式：
  - `.sf-feed-row`：`display: flex; gap: 14px; padding: 10px 16px; align-items: center;`
  - `.sf-feed-row__avatar-wrapper`：包含左侧 `SFAvatar` 的容器。
  - `.sf-feed-row__content`：`flex-grow: 1; min-width: 0;`。
  - `.sf-feed-row__header`：`display: flex; justify-content: space-between; align-items: center;`。
  - `.sf-feed-row__title`：修改字号为 `15px`，`font-weight: 700`。
  - `.sf-feed-row__actions`：`display: flex; align-items: center; gap: 10px;`。
  - `.sf-feed-row__vote`：药丸形踩赞容器，`display: inline-flex; align-items: center; background: var(--sf-muted); border-radius: 20px; padding: 1px;`。
  - `.sf-feed-row__vote-btn`：小踩赞按钮，去除背景和边框，`padding: 2px 6px; font-size: 9px; cursor: pointer; border-radius: 10px;`，hover 时背景为 `var(--sf-border-light)`。
  - `.sf-feed-row__vote-val`：数值字号 `10px`，粗体。
  - `.sf-feed-row__action-tag`：评论气泡，`display: inline-flex; align-items: center; gap: 4px; font-size: 11px; color: var(--sf-fg-secondary); background: var(--sf-muted); padding: 3px 8px; border-radius: 20px; font-weight: 500;`。
  - `.sf-feed-row__meta-row`：`display: flex; align-items: center; gap: 12px; font-size: 12px; color: var(--sf-fg-tertiary); margin-top: 4px;`。
  - `.sf-feed-row__views`：`margin-left: auto;`。

### 3. 首页调用更新

更新 [apps/web/app/pages/index.vue](file:///Users/inkedus/Code/SForum/apps/web/app/pages/index.vue)：
- 移除 `<SFFeedRow>` 的 `:excerpt="thread.excerpt"` 属性传递（顺应无摘要版设计）。
- 外层包裹帖子列表的 `SFCard` 的 `p-5` padding 调整为 `p-0`，将 padding 职责完全交给 `SFFeedRow` 组件自身，消除双重边距，使得帖子行能够拥有通栏分割线效果（更符合专业论坛规范）。
- 新增 `divide-y divide-slate-100` 或类似的分割线样式在包裹层，使多条帖子整齐排开。

### 4. 静态测试校验脚本更新

更新 [tests/validate-homepage.js](file:///Users/inkedus/Code/SForum/tests/validate-homepage.js)：
- 原本在 Task 1 优化的类名校验将保持原样，无需修改。
- 确认静态测试能顺利通过。

## 验证计划

- **自动化测试**：运行 `node tests/validate-homepage.js` 及 `node tests/validate-sf-components.js`。
- **构建测试**：运行 `bun run -C apps/web typecheck`，确保改动后 Vue 组件无类型错误。
