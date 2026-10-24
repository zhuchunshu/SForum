# 插件仓库标准

一个 SForum 插件有两种载体：仓内扩展包（`extensions/builtin`、`extensions/optional`）与独立仓库（作为外部扩展源根）。两者共用同一套包契约，差别只在仓库级文件。本文回答两件事：**一个插件必须有什么**、**每样东西放在哪里**。

包契约本身（Manifest V3 字段、信任、Host API）见[插件编写指南](../../extensions/authoring-guide.md)；品牌资产的尺寸与校验见[品牌资产](../../extensions/brand-assets.md)。

## 1. 独立仓库骨架

```text
sforum-<name>/
├── README.md                  运营者向：解决什么问题、怎么装、怎么配
├── AGENTS.md                  AI/协作者向：目录布局、开发闭环、验证标准、宿主约束
├── NOTES.md                   设计取舍与实测结论（为什么这样做）
├── CHANGELOG.md               版本变更
├── LICENSE
├── plugins/<id>/              扩展包，全仓只允许一份
│   ├── sforum.extension.json
│   ├── manifest/{settings.json,langs/}
│   ├── backend/               有后端时必需
│   ├── brand/                 必需，见第 4 节
│   └── schemas/ frontend/ templates/ openapi/   按需
├── scripts/prepare-package.sh 构建 + 刷新摘要 + 全部门禁
└── .github/workflows/{ci,release}.yml
```

| 文件 | 必需 | 作用 |
| --- | --- | --- |
| `README.md` | 是 | 运营者视角。首屏是社交卡，随后一句话说明它解决什么问题、安装步骤、可调边界 |
| `AGENTS.md` | 是 | 协作者与 AI 的第一入口。必须写清扩展 id、包路径、开发闭环、验证标准、宿主约束速查 |
| `NOTES.md` | 建议 | 取舍与实测数字（超时、截断、限流等）。宿主约束写在这里，不要只留在对话里 |
| `CHANGELOG.md` | 是 | 每个发布版本的变更；破坏性变更要单独标注 |
| `LICENSE` | 是 | 开源许可证全文 |
| `plugins/<id>/` | 是 | 扩展包目录；同一扩展 id 只能出现在一个源 |
| `scripts/prepare-package.sh` | 是 | 一条命令跑完 build → digest → validate → brand check → test |
| `.github/workflows/` | 是 | CI 与发布；两者都从 SForum 检出调用 `go run ./cmd/sforum` |

## 2. 包内骨架

| 路径 | 必需 | 说明 |
| --- | --- | --- |
| `sforum.extension.json` | 是 | 显式 `manifestVersion: 3`，`packageFiles` 摘要与文件一致 |
| `manifest/settings.json` + `manifest/langs/` | 有设置时 | 通过 `includes` 引用，不内联进主清单 |
| `backend/` | 有后端时 | 静态构建（`CGO_ENABLED=0`），入口与 `packageFiles` 的 `executable` 对齐 |
| `brand/icon.svg`、`brand/icon.png`、`brand/social-preview.png` | 是 | 见第 4 节 |
| `schemas/` | 有响应/请求契约时 | 与 `openapi` 声明同源，不重复定义 |
| `frontend/` | 有预构建前端时 | 只放构建产物；源码构建由 `extension build` 负责 |
| `templates/` | 主题或 Page Registry 页面 | 路径必须与 `theme.json`、`templates[]`、`packageFiles[]` 三处一致 |
| `README.md` | 是 | 仓内包也要有：它会在 GitHub 上被单独打开 |

## 3. 必须有、且必须成对的东西

- 清单声明与 `packageFiles` 摘要：改了文件就必须重跑 `extension digest --write`，否则包会被拒绝启用。
- 权限键与推荐角色映射：插件只声明，授予由宿主负责。
- 设置项的推荐默认值与一键恢复：面向第一次部署的运营者。
- 品牌资产与 README 首屏：见第 4 节。

## 4. 品牌资产

每个随产品分发的包都带 `brand/`：

| 文件 | 尺寸 | 上限 |
| --- | --- | --- |
| `brand/icon.svg` | `viewBox="0 0 512 512"`，自包含 | 40 KiB |
| `brand/icon.png` | 512×512 PNG | 120 KiB |
| `brand/social-preview.png` | 1280×640 PNG（中文主卡） | 600 KiB |

生成与校验：

```bash
# 参数速查
node scripts/brand/generate.mjs --help
# 预览（不写仓库）+ 联排样张
node scripts/brand/generate.mjs --draft /tmp/sforum-brand --sheet /tmp/sforum-brand
# 只重生成一个包（按包目录名后缀匹配）
node scripts/brand/generate.mjs --only sforum-smtp
# 写入
node scripts/brand/generate.mjs
# 校验（独立仓库同样调用宿主 CLI）
cd <SForum>/apps/api && go run ./cmd/sforum extension brand check <包路径>
```

`brand/` 不进 `packageFiles`（品牌不是运行时制品），但它属于包树，会改变包摘要 —— 这是正常制品变更，不要为了消掉漂移去改版本号。完整规则、模板参数与 GitHub 手动上传步骤见[品牌资产](../../extensions/brand-assets.md)。

## 5. 门禁

| 位置 | 跑什么 |
| --- | --- |
| 插件仓本地 | `scripts/prepare-package.sh`：build → `extension digest --write` → `extension validate` → `extension brand check` → `extension test` |
| 插件仓 CI | 同一串命令，SForum 钉在 `SFORUM_REF` 检出；发布工作流额外 `extension package --exclude-source` |
| SForum 仓库内 | `go test ./...`（含品牌资产契约）、`node tests/validate-docs.mjs`、`node tests/validate-architecture-boundaries.mjs`、`./scripts/test.sh --fast` |

顺序不是随意的：`brand check` 放在 `validate` 之后、`test` 之前，让"图没做"在契约检查阶段就失败，而不是等到发布才发现。宿主 CLI 是唯一入口 —— 不要在插件仓里另写一份校验脚本。

## 6. 记录与知识库

- **SForum 仓内**：`knowledge/` 是项目记忆。改到信任、摘要、权限、数据面、公共 UI 契约时，必须写 `knowledge/decisions/`；功能面变化写 `knowledge/modules/`；一次工作结束时留一份 hot handoff 在 `knowledge/sessions/`，并把 `knowledge/index.md` 指向它。
- **插件仓**：不复制 SForum 的知识库结构。宿主约束与实测结论写 `NOTES.md`，协作者须知写 `AGENTS.md`，版本变化写 `CHANGELOG.md`。跨仓的架构决定留在 SForum 的决策记录里，插件仓只链接过去。

## 7. 新增插件清单

1. 用 `make:plugin` 生成骨架，确定扩展 id（`sforum.*`）与类型。
2. 写 `sforum.extension.json`：`manifestVersion: 3`、能力、设置、事件、路由、权限按需声明。
3. 写设置项时同时给出推荐默认值与恢复路径。
4. 在 `scripts/brand/spec.json` 增加一条，生成并校验 `brand/` 三个文件。
5. README 首屏放社交卡，补安装、配置、可调边界。
6. `AGENTS.md` 写清开发闭环与验证标准；`NOTES.md` 记取舍与实测数字。
7. 跑通 `prepare-package.sh`，再在后台走「安装 → 信任 → 启用」。
8. 改到契约面时，回 SForum 补决策记录。
