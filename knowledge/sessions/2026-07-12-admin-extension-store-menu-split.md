# 2026-07-12 Session Handoff

## Changed

- 后台侧栏将 **应用商城** 从「扩展管理」中拆出，成为独立一级菜单：
  - 子菜单：**主题**（`/extensions/store/themes`）、**插件**（`/extensions/store/plugins`）
  - 注册：`apps/web/app/config/adminModules.ts`
  - 共享货架：`apps/web/app/components/admin/SFAdminExtensionStoreShelf.vue`
  - 旧入口 `/extensions/store` 重定向到插件商城
- 文案：`admin.nav.extensionStoreThemes` / `extensionStorePlugins` 及
  `admin.extensions.store.themes*` / `plugins*`
- 回归：`tests/validate-admin-framework.ts` 校验一级商城文件夹顺序

## Decisions

- 商城与本地「扩展管理」（安装包生命周期）分离，避免菜单混杂。
- 主题商城权限 `extension.theme.manage|view`；插件商城
  `extension.plugin.manage|view`；不新增权限键。

## Next

- 远程目录 API、签名校验、一键安装/更新流水线。

## Open Questions

- 无。
