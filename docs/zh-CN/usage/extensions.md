# 扩展与主题（运营侧）

[← 使用说明](./README.md)

## 插件 vs 主题

| 类型 | 作用 | 典型操作 |
| --- | --- | --- |
| **插件** | 业务能力：邮件、存储、搜索、内容策略、后台入口等 | 安装 → 信任（如需）→ 启用 → 配置 |
| **主题** | 外观与公共页呈现（Page Registry L0/L1，可选 L2） | 安装 → 激活（同步，不重建 Nuxt） |

## 安全模型（务必理解）

1. **上传 ZIP 只做惰性校验与存储**，不会立刻执行包内代码。  
2. **首次启用可执行逻辑**前，需要超级管理员基于**精确制品摘要**的信任确认。  
3. 高风险能力（替换路由、原始数据库等）必须在清单中声明，并出现在信任披露中。  
4. **Safe Mode** 与恢复 CLI 由宿主拥有，插件不可覆盖。  

## 设置 UI

扩展设置通常三级：

| 级别 | 方式 | 是否需要站长编译前端 |
| --- | --- | --- |
| 普通字段 | Host Schema 渲染 | 否 |
| 探测/动作 | Schema + Settings Actions | 否 |
| 复杂后台组件 | 作者预构建 + 摘要绑定动态加载 | 否（运营侧） |

## 内置与可选包位置

| 目录 | 含义 |
| --- | --- |
| `extensions/builtin/` | 随产品启动同步的内置插件/主题（受保护） |
| `extensions/optional/` | 仓库内可选包，需安装后才进入运营流程 |
| `extensions/dev/` | 本地实验（默认不进后台列表） |
| 运行时 `EXTENSION_ROOT` | 上传安装后的包存储 |
| `EXTERNAL_EXTENSION_ROOTS` | 逗号分隔的独立源码集合；每个根目录含 `plugins/`、`themes/` |

外部源码集合会在 API/worker 启动时被静态校验并复制为不可变快照。首次发现
只进入“已安装”状态；内容变化只产生待审核版本。扫描不会自动启用、继承信任、
切换 provider 或删除已安装插件。Docker 部署必须填写容器内路径，并将宿主目录
只读挂载到 API 和独立 worker 容器。

## 主题激活

- 激活 = 切换 Page Registry 绑定 + L0 皮肤等
- **不需要**重建站点前端、不需要重启 Nitro 作为常规路径
- 当前主题也会提供 Host 选择的系统错误页（403、404、429、5xx）的 L1
  呈现。HTTP 状态、缓存、SEO、重试行为与紧急回退仍由 Host 拥有；插件和公开
  L2 组件不能替换这些 `system.*` 页面。
- 详情：[运行时主题](../../extensions/runtime-themes.md)

## 开发者文档

编写插件请读：

- [开发者 CLI](../development/cli.md)（`make:plugin`、digest、打包、seed）  
- [插件编写指南](../../extensions/authoring-guide.md)  
- [场景速查](../../extensions/scenario-map.md)  
- [Host API v2](../../extensions/host-api-v2.md)  
- [V3 平台](../../extensions/v3/README.md)  
