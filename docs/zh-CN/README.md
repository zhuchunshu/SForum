# SForum 文档（简体中文）

[English](../en-US/README.md) · [文档总入口](../README.md)

SForum 是可维护、插件优先的开源论坛框架：核心做宿主与契约，垂直能力与厂商逻辑走扩展。

## 按角色阅读

| 你是… | 从这里开始 |
| --- | --- |
| 第一次跑起来 | [快速开始](./getting-started.md) |
| 站长 / 运营 | [使用说明](./usage/README.md) |
| 贡献者 / 二次开发 | [开发指南](./development/README.md) |
| 运维上线 | [生产部署](./deployment.md) |
| 插件 / 主题作者 | [扩展概览](./usage/extensions.md) → [技术参考](../extensions/authoring-guide.md) |
| 了解产品与架构 | [产品说明](./product.md) · [架构](./architecture.md) · [路线图](./roadmap.md) |

## 目录

### 使用

- [使用说明总览](./usage/README.md)
- [首次注册与超级管理员](./usage/first-login.md)
- [管理后台](./usage/admin.md)
- [论坛日常](./usage/forum.md)
- [通知](./usage/notifications.md)
- [搜索](./usage/search.md)
- [扩展与主题（运营侧）](./usage/extensions.md)

### 开发

- [开发总览](./development/README.md)
- [环境搭建](./development/setup.md)
- [日常工作流](./development/workflow.md)
- [开发者 CLI](./development/cli.md)
- [测试与质量门禁](./development/testing.md)
- [仓库地图](./development/repository.md)

### 其他

- [生产部署](./deployment.md)
- [架构](./architecture.md)
- [产品说明](./product.md)
- [路线图](./roadmap.md)

## 技术参考（语言中立 / 路径固定）

以下路径被 CI 与生成器引用，请勿随意移动：

- [插件编写指南](../extensions/authoring-guide.md)
- [Host API v2](../extensions/host-api-v2.md)
- [运行时主题](../extensions/runtime-themes.md)
- [生成目录 catalogs](../extensions/catalogs/)
- [V3 平台文档](../extensions/v3/)

## 与 knowledge 的分工

| 位置 | 用途 |
| --- | --- |
| `docs/zh-CN` / `docs/en-US` | 给人读的使用与开发手册 |
| `docs/extensions` | 扩展契约与生成目录 |
| `knowledge/` | 决策、模块状态、会话交接（偏内部记忆） |
