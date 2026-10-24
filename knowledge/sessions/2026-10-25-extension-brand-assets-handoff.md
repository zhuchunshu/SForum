# 2026-10-25 Session Handoff — 扩展品牌资产（图标 / 社交卡）

## Changed

- 新增 `scripts/brand/`：`spec.json`（12 个扩展 + 主仓的中文名、一句话、Tabler 字形、
  仓库 slug）、`generate.mjs`（SVG 合成 + sharp 光栅化，输出确定）、`README.md`。
- 12 个扩展包新增 `brand/icon.svg`、`brand/icon.png`（512×512）、
  `brand/social-preview.png`（1280×640）：8 个内置插件、`sforum-default` 主题、
  `sforum-fetch-url` / `sforum-oauth2-provider` / `sforum-web-search`。
- 主仓新增 `docs/assets/brand/social-preview.png`，`README.md` 在简介后插入首屏社交卡。
- 12 个包 `README.md` 标题下插入社交卡（外部仓 `sforum-oauth2-provider` 原本没有包
  README，已补一份包级说明）。
- 新增 `sforum extension brand check`（`apps/api/cmd/sforum/extension_brand.go`，
  255 行）+ 表驱动测试 + 仓库级契约测试；`validate.go` 注册子命令；
  `tests/validate-docs.mjs` 增加 `'extension_brand.go': ['extension','brand']`。
- 文档：`docs/extensions/brand-assets.md`（契约、模板参数、错误码、GitHub 手传）、
  `docs/{zh-CN,en-US}/development/plugin-standard.md`（插件仓库标准）、
  `docs/extensions/authoring-guide.md`（放置表 + 清单第 16 条）、
  `extensions/README.md`（包内布局）、两处 `development/README.md` 章节表、
  两处 `development/cli.md`（`extension brand check`）。
- 外部三仓：`AGENTS.md` 增「品牌资产（必须）」节、`scripts/prepare-package.sh`
  在 validate 之后加 brand check、仓库 README 首屏加卡。

## Decisions

- 品牌资产进包内 `brand/`、固定文件名、不进 `packageFiles`；包树摘要会变，属正常
  制品变更，不为消漂移改版本号。见 `../decisions/2026-10-25-extension-brand-assets.md`。
- 视觉单一模板单色：品牌蓝渐变 + 深底 + Tabler 描边字形；中文主卡。
- 门禁是宿主 CLI + 仓内 Go 测试，不另写插件仓私有校验脚本。
- 本轮不加 manifest `icon`/`socialPreview`，不改管理页插件列表：今天没有运行时消费者。

## Next

- 4 个仓库的 GitHub **Social preview 需手动上传**（无公开 API）：
  `zhuchunshu/SForum` 用 `docs/assets/brand/social-preview.png`，三个插件仓用各自
  `plugins/<id>/brand/social-preview.png`。
- 重启 API 让 `SyncBuiltins` 重新 stage 8 个内置包：会出现一次同版本摘要漂移候选，
  逐一点「升级」，或随下个 patch 版本由内置自动更新接管。
- 外部三仓 CI 尚未加 brand 步骤：它们的 workflow 把 SForum 钉在 `SFORUM_REF`
  （当前 `v3.0.13`，没有 `extension brand`）。SForum 打下个 tag 后提升该 ref，
  再在 `ci.yml` / `release.yml` 加一步 `extension brand check`。
- 如需英文卡：加 `brand/social-preview.en-US.png`（契约允许增量文件，校验忽略额外文件）。

## Open Questions

- 管理页插件列表是否要显示 `brand/icon.png`？需要时再立小项：清单加可选
  `icon` 字段（或直接读包内 `brand/`）+ 列表卡片 + 测试，不需要重新设计图。
- 商城（`sforum-app-store` 草稿）发布时从仓库读 `brand/` 还是让开发者上传？
  草稿计划 D6 写的是后台上传 + 内容寻址；若改为读仓库，`brand/` 已是稳定路径。
