# 2026-07-12 Session Handoff

## Changed

- Admin 扩展管理新增 **应用商城** 菜单与框架页：
  - 路由：`/admin/extensions/store`（page id `/extensions/store`）
  - 注册：`apps/web/app/config/adminModules.ts`
  - 页面：`apps/web/app/pages/admin/extensions/store.vue`
  - 文案：`zh-CN` / `en-US` 下 `admin.nav.extensionStore` 与
    `admin.extensions.store.*`
- 版式对齐 demo `01C`：粘性筛选条（搜索 + 排序 + 分类 Chip）+ 卡片网格 +
  底部「即将上线」提示；安装按钮禁用，可跳转插件/主题列表。
- 占位目录为前端静态数据，本地可筛选/排序，无后端 API。

## Decisions

- 选型 **01C 粘性筛选货架**，不实现商城安装/支付/远程目录。
- 权限复用 `extension.manage`，不新增权限键。

## Next

- 远程应用目录 API、签名校验、一键安装/更新流水线。
- 若占位 demo 卡片不需要，可改为纯空状态 + Coming Soon。

## Open Questions

- 商城目录是否走官方中心，还是站点可配置镜像源。
