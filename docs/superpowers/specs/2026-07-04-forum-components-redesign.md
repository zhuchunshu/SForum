# SForum UI 组件 Demo 多视觉风格扩展与丰富设计规范书

本文档规定了 SForum 前端组件 Demo 文件的重构、多风格版本扩展以及高阶组件丰富的技术规范与设计规格。

## 1. 目标与背景
现有的 UI 组件 Demo 文件 [forum-components.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components.html) 展示了基于 **Tech Boutique (精致科技蓝)** 风格的 9 类基础组件。为了拓宽设计视野，方便开发选型，本项目将扩展该 Demo，支持以下目标：
1. **多视觉流派延伸**：从单一的 Tech Boutique 风格扩展出 4 种独立的高级视觉风格，每种风格都具有该流派的典型视觉特征。
2. **高阶交互组件丰富**：将组件类别从 9 类扩展到 14 类，增加富文本编辑器、社区数据看板、勋章系统、互动投票和完整帖子页面。
3. **单文件自包含交互**：利用原生 JavaScript 驱动核心组件的交互反馈（如投票百分比动画、编辑器同步预览、模态框弹出等），实现无 Dev Server 的双击即用体验。

---

## 2. 视觉风格与 Tailwind CSS 规范

每种风格的 Demo 文件均采用独立的 HTML，通过修改 `<script>tailwind.config = ...</script>` 配置项和定制 Tailwind 类来实现高度集成的风格特征，从而保持 CSS 文件的极简与高性能。

| 风格名称 | 文件路径 | 主色调 & 辅助色 | 圆角规范 (`border-radius`) | 投影规范 (`box-shadow`) | 核心排版风格 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Tech Boutique**<br>(基准科技) | [forum-components.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components.html) | 科技蓝 (`#0052FF`) / 靛蓝<br>背景: `#FAFAFA`<br>前景色: `#0F172A` | `rounded-2xl` (16px)<br>核心卡片大倒角 | 柔和双层阴影<br>`shadow-sm`<br>`hover:shadow-md` | Inter, 现代系统默认无衬线字体 |
| **Swiss Modernism**<br>(瑞士极简) | [forum-components-swiss.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components-swiss.html) | 极简黑 (`#000000`) / 灰<br>背景: `#FFFFFF`<br>前景色: `#000000` | `rounded-none` (0px)<br>绝对直角网格 | 无阴影<br>仅通过 `border-px` 进行区块分割 | Inter, 粗体高对比排版，数学间距 |
| **Cyber Glassmorphism**<br>(暗黑毛玻璃) | [forum-components-glass.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components-glass.html) | 霓虹青 (`#00F0FF`) / 紫<br>背景: `#0B0F19`<br>前景色: `#F8FAFC` | `rounded-xl` (12px)<br>精细的中倒角 | 霓虹发光投影<br>`shadow-neon` | System-ui, 带有 Aurora 渐变光晕背景 |
| **Neobrutalism**<br>(新粗野主义) | [forum-components-neobrutalism.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components-neobrutalism.html) | 复古黄 (`#FFDE47`) / 绿<br>背景: `#F4F2EC` (纸张黄)<br>前景色: `#000000` | `rounded-md` (6px) 或直角<br>粗边框搭配 | 硬朗平行实心阴影<br>`shadow-[4px_4px_0px_#000]` | 粗笔画、无衬线粗体排版 |
| **Aurora Soft Light**<br>(极光微光) | [forum-components-aurora.html](file:///Users/inkedus/Code/SForum/apps/web/app/assets/demos/forum-components-aurora.html) | 桃花粉 (`#FF7B9B`) / 薰衣草紫<br>背景: `#FAF9FB`<br>前景色: `#2D2244` | `rounded-3xl` (24px)<br>极致圆润大倒角 | 超轻柔羽化漫反射阴影<br>`shadow-xl shadow-purple-100` | 温暖亲和无衬线排版 |

---

## 3. 组件列表及交互规格 (14 类)

每个风格 Demo 文件均必须完整包含以下 14 类组件：

### 1. 导航类 (Navigation)
* **顶栏变体**：标准顶栏 / 带搜索居中及键盘快捷键提示顶栏。
* **侧边栏**：图标型紧凑侧栏 / 文字+图标宽侧栏。
* **面包屑 & 分页器**：多态面包屑 / 带“上一页/下一页”及“加载更多”的响应式分页栏。

### 2. 头像与资料卡 (Avatars)
* **状态点叠加 (Status Dot)**：在线、离开、离线三种带环形遮罩的状态头像。
* **层叠头像组 (Stacked Group)**：带数量叠加标识（如 `+8`）的社交头像流。
* **Hover 名片悬浮窗 (Popover)**：鼠标悬浮头像时，淡入显示用户详细介绍、关注数、粉丝数及“关注”按钮的卡片。
* **横幅背景迷你用户栏 (Banner Profile)**：顶部带渐变横幅的卡片式用户介绍。
* **几何缩写头像 (Geometric Initials)**：利用 `clip-path` 或不同形状（正方形、六边形、圆角矩形）绘制的拼音简写头像。

### 3. 列表与信息流 (Feed Rows)
* **极简单行表格行**：HN / V2EX 风格的高密度列表，带标签、回复数和时间。
* **图表复合卡片行**：Twitter / Reddit 风格，支持标题、摘要、图片占位、点赞/分享快捷条。
* **Q&A 问答盒子行**：StackOverflow 风格，突出投票数和回答采纳状态。
* **社交无标题正文行**：Chat / 微言风格，直击内容。
* **时间轴进程轨迹行 (Timeline)**：带贯穿线与节点标记的进程时间轴。

### 4. 表单插件与选择器 (Form Addons)
* **下拉操作菜单 (Dropdown)**：无 JS 纯 CSS 触发的下拉多项操作菜单（📌收藏、🔗复制链接、🚩举报、🗑删除）。
* **二分卡片单选器 (Segmented Card)**：公开 / 私密大面积卡片式选择状态。
* **交互式星级评分器 (Star Rating)**：可点击交互的 5 星评分条。
* **虚线框拖拽上传器 (Uploader)**：拖拽上传虚线卡片，支持悬浮高亮状态。
* **自动补全关联框 (Autocomplete)**：带分类标签的检索自动补全下拉框。

### 5. 讨论与评论嵌套 (Comments)
* **扁平时间轴流**：带楼层标记（如 `#1`、`#2`）的线性评论。
* **左缩进树状嵌套回复**：Reddit / 贴吧风格的层级嵌套回复，带左侧引导线。
* **折叠子回复气泡抽屉**：支持点击展开更多回复的微型抽屉。
* **双栏侧滑评论区**：主内容与评论区分离的双栏版式。
* **引用盖楼链**：传统论坛特有的“引用上楼”多层嵌套楼中楼。

### 6. 检索与数据过滤 (Search & Filters)
* **顶置过滤排序条**：热门 / 最新 / 净化等一键过滤栏。
* **话题标签云**：带有字号权重的话题标签聚落。
* **侧边复合过滤面板**：带复选框、单选按钮和筛选按钮的侧边栏。
* **键盘命令搜索台 (Command Palette)**：浮动在页面中央的全局搜索弹窗。
* **高级过滤下拉框**：点击展开的时间与排序复合过滤器。

### 7. 个人页头区与设置 (Profile Head)
* **极简居中名片**：头像居中、简介居中的个人展示牌。
* **背景横幅交叠头**：Banner 图与头像交叠的经典个人页主头部。
* **双列多功能设置面板**：左侧为菜单，右侧为表单字段的设置区。
* **数据看板级头部**：带个人数字汇总（帖子数、获赞数、采纳率）的数据化头部。
* **侧栏微型资料卡**：窄版侧边栏用户资料卡。

### 8. 交互组件 (Interactions)
* **确认对话框 (Modal Confirm)**：遮罩居中的确认删除警告模态弹窗。
* **轻量抽屉 (Sheet)**：从底部或右侧滑出的发帖面板。
* **点赞/收藏交互按钮**：支持点击触发微动画和数字递增的按钮组。

### 9. 排行榜与通知 (Lists)
* **用户排行榜**：本周贡献榜的前三名展示卡。
* **通知列表**：包含未读、已读状态点的消息卡片流。
* **话题表格视图**：紧凑型的表格数据视图。

### 10. [NEW] 互动 Markdown 编辑器 (Interactive Markdown Editor)
* **功能**：左侧为文本编辑域（Textarea），顶部为格式化快捷工具栏，右侧为实时预览区域（可展示加粗、链接、代码块）。
* **交互**：原生 JS 驱动预览区更新；点击工具栏可在光标处插入 Markdown 语法字符。

### 11. [NEW] 社区数据看板与微型图表 (Analytics & Mini Charts)
* **功能**：三个统计指标卡（今日在线、本周发帖、用户保留率）。
* **交互**：每个指标卡底部嵌入一个精美的 SVG 路径绘制折线图或柱状图，支持 hover 时数据显示提示。

### 12. [NEW] 勋章成就与等级系统 (Achievement Badge Cards)
* **功能**：用户的成就与成长系统。包含成长经验条（如 "Lv.5 -> Lv.6 (85%)"）、成就勋章墙。
* **交互**：悬浮于勋章上时，通过 `transform: translateY(-4px)` 和微缩放提供良好的物理感知。

### 13. [NEW] 可交互投票卡片 (Polls & Voting Widgets)
* **功能**：社区投票小组件。包含投票题目、选项列表和总票数。
* **交互**：点击未投票的选项后，切换至已投票状态；所有选项的百分比条以平滑动画展开，票数累加 1。

### 14. [NEW] 完整帖子详情页排版 (Thread Detail Page Layout)
* **功能**：一个完整的主帖页微缩排版，包含左侧主帖内容与底部评论区，右侧为作者信息和热门话题侧栏，使组件在实际场景中融合呈现。

---

## 4. 验证与测试方案

### 4.1 自动化测试规范
由于是纯静态 HTML 页面，使用简单的 shell 脚本进行以下自动化校验：
1. **链接与资源检查**：确保所有引用的外部 CDN 链接（Tailwind CSS, Google Fonts）均为 HTTPS 加密链接且有效。
2. **HTML 标签闭合校验**：通过静态解析确认所有 div 与 section 标签完整闭合，避免页面排版塌陷。
3. **Tailwind 类可用性**：确保没有使用未配置的非法自定义颜色或字体类。

### 4.2 手动功能验证 (交互性验证)
开发完成后，必须逐一验证以下原生 JavaScript 交互行为：
* 双击在浏览器中直接打开 HTML，检查是否能正常显示全部样式。
* **Markdown 编辑器**：在输入框打字，右侧预览区是否即时更新。
* **可交互投票卡片**：点击选项，百分比进度条是否有平滑的拉伸过渡（CSS `transition`）。
* **Cmd+K 命令面板**：点击顶栏搜索框或按快捷键是否能呼出/关闭命令弹窗。
* **确认模态框**：点击排行榜或列表中的“删除”操作，模态框是否遮罩居中显示；点击取消是否能隐去。
* **响应式检查**：将浏览器拉宽/拉窄，确保在 375px (手机) 到 1200px+ (桌面) 的所有宽度下，左右无异常滚动条，组件栅格自动折行适配。
