# 2026-07-05 侧边栏卡片字体与排版优化设计

## 背景与目标
在目前的论坛首页布局中，两侧侧边栏卡片内的部分文本、标签和数字字号过小（例如使用 `text-[9px]` 或 `text-[10px]`），导致在高分辨率屏幕上阅读和识别较为费力。
本设计的目标是在保持卡片紧凑、精致的设计感的前提下，适当增大字号，调整排版间距，使其在各种屏幕尺寸下都具备良好的可读性。

## 设计方案

修改将集中在 [index.vue](file:///Users/inkedus/Code/SForum/apps/web/app/pages/index.vue) 文件中。

### 1. 左侧边栏 (Left Sidebar)
* **导航卡片**
  * 将小标题“导航”字号保持 `text-xs` (12px)，颜色修改为更柔和的 `text-slate-400`，字距设为 `tracking-widest`。
  * 导航项链接增加垂直内间距，由 `py-2` 调整为 `py-2.5`。
  * 未激活链接文本颜色由 `text-slate-600` 升级为 `text-slate-700`，并使用 `font-medium` 以增强可读性。
* **讨论板块分类卡片**
  * 分类项链接文字由 `text-sm` 升级为 `text-[14px] text-slate-700 font-medium`，内间距由 `py-1.5` 调整为 `py-2`。

### 2. 右侧边栏 (Right Sidebar)
* **用户中心卡片**
  * “帖子”和“获赞”的统计数值由 `text-sm` (14px) 提升为 `text-base font-bold` (16px)。
  * 对应的标签文本由极小的 `text-[10px]` 提升为 `text-xs` (12px)，颜色调整为较轻的 `text-slate-400`。
* **每日签到卡片**
  * 卡片标题“每日签到”由 `text-xs` 提升至 `text-sm`。
  * 签到描述字号由极小的 `text-[10px]` 提升至 `text-xs`。
* **热门讨论卡片**
  * 排名索引图标的尺寸从 `w-4 h-4 text-[9px]` 调整为 `w-[18px] h-[18px] text-[10px] px-1`。
  * 帖子链接字号从 `text-xs` (12px) 提升到 `text-sm` (14px)；回复数由极小的 `text-[9px]` 提升到 `text-xs` (12px) 并改为中灰色。
* **全站数据统计卡片**
  * 条目文本和数值的整体字号由 `text-xs` 统一提升到 `text-sm` (14px)；其中条目名称使用 `text-slate-500 font-normal`，条目数值使用 `font-semibold font-mono text-slate-800`。

## 验证计划
### 运行开发服务器验证
- 启动 `apps/web` 服务的开发模式，检查页面在 100% 缩放下的实际显示效果，确保排版未出现错行或布局崩塌。
- 确认两侧边栏的视觉比例与中间的帖子列表流保持协调。
