# SForum Clean Light Forum Demo (清爽现代浅色论坛展示 Demo)

本目录提供了一个独立、免构建工具、纯原生 HTML/CSS/JS 实现的现代论坛界面 Demo，包含论坛首页、帖子内容详情页以及多级嵌套回复评论区。

## 📁 目录结构

```
tmp/demos/clean-light-forum/
├── index.html         # 论坛首页 (现代 3 栏布局 + 帖子列表 Feed + 过滤器 + 发布新帖 Modal)
├── post.html          # 帖子详情页 (富文本正文 + 作者信息 + 交互操作栏 + 多级嵌套评论树)
├── style.css          # CSS 设计系统 (CSS Tokens + Flex/Grid 三栏响应式布局 + 微动画)
├── app.js             # 客户端交互 JS (点赞/踩计数、收藏切换、Modal 控管、动态提交新帖与嵌套回复)
└── README.md          # 本说明文档
```

## ✨ 核心特色与设计规范

1. **Clean Modern Light 视觉风格**：
   - 使用淡雅 Slate 色系底色 (`#F8FAFC`) 与纯白卡片 (`#FFFFFF`)。
   - 细致边框 (`#E2E8F0`) 与柔和自然阴影 (`0 4px 6px -1px rgba(15,23,42,0.08)`)。
   - 品牌 Emphasis 蓝 (`#2563EB`) 用于交互控件与高亮提醒。

2. **现代三栏 App 级布局**：
   - **左侧侧边栏**：导航链接 (首页/热门/关注/收藏) 与版块分类列表。
   - **中央主内容区**：首页 Feed 列表或帖子正文与评论树。
   - **右侧侧边栏**：社区公告板、热门话题榜单与社区概览统计。

3. **完备的零依赖离线交互体验**：
   - **点赞 / 取消点赞**：实时更新界面计数器与 Toast 反馈。
   - **帖子发布**：点击“发布新帖”唤起 Modal 弹窗，提交后即时在 Feed 首部插入新帖子卡片。
   - **多级嵌套评论区**：支持发布一级评论，以及点击“回复”按钮动态唤起行内输入框，插入二级缩进回复。

## 🚀 如何查看 Demo

无需运行任何 `npm` 或 `bun` 服务，直接在任意现代浏览器中双击或直接打开对应的 `.html` 文件即可：

- **打开首页**：直接在浏览器中打开 `index.html` 或点击链接 `file:///Users/inkedus/Code/SForum/tmp/demos/clean-light-forum/index.html`
- **打开内容页与评论区**：直接在浏览器中打开 `post.html` 或点击链接 `file:///Users/inkedus/Code/SForum/tmp/demos/clean-light-forum/post.html`
