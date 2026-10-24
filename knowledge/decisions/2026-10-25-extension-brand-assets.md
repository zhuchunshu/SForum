# Decision: Extension Brand Assets (icon + social preview)

## Status

Accepted

## Date

2026-10-25

## Context

12 个扩展（8 个内置插件、默认主题、3 个独立仓插件）此前没有任何品牌资产：仓库里
只有 `apps/web/public/brand/sforum-logo.svg` 与 `docs/assets/screenshots/*`。
每个插件的 GitHub 仓库页首屏是一段英文散文，社交卡是 GitHub 自动截的标题，
看不出它们属于同一个产品线。

三个事实决定了方案边界：

1. **今天没有运行时消费者。** Manifest V3 没有 `icon` / `socialPreview` 字段；
   `assets[]` 是前端运行时资源（handle/type/path/digest），不是品牌图；
   `packageFiles.kind` 没有品牌类别；管理页插件列表也没有图标位。真实消费者只有
   README、GitHub 仓库 Social preview（无公开 API，只能网页手传，1280×640 / <1 MB），
   以及草稿态商城计划里写明的「每产品图标 + OG 图」。
2. **包树摘要覆盖全部文件。** `DigestTree` 对包内除 `.git`/`node_modules`/`vendor`
   外的每个文件取哈希；`ValidatePackageFiles` 只校验已声明文件，所以额外文件合法
   （包内 `README.md` 就是先例）。
3. **内置插件有自动更新，但跳过同版本摘要漂移。** 品牌文件一旦进包，8 个内置包会
   出现「同版本、不同摘要」的候选，需要一次显式升级，或随 patch 版本一起晋升。

## Decision

- **位置**：品牌资产进包内 `brand/`，固定文件名、不加 id 前缀：
  `brand/icon.svg`、`brand/icon.png`（512×512）、`brand/social-preview.png`
  （1280×640，中文主卡）。可选 `brand/social-preview.en-US.png` 为增量文件。
  主仓社交卡放 `docs/assets/brand/social-preview.png`。
- **不进 `packageFiles`**：品牌不是运行时制品；不改清单 schema、不加 kind。
- **视觉**：单一模板、单一强调色。品牌蓝渐变 `#2671ed → #1f6bec` + 深底
  `#0b1220` + Tabler 描边字形（与 `sforum-logo.svg` 同一描边语言）。不做每插件
  配色；字形承担辨识，配色承担家族感。
- **生成**：`scripts/brand/spec.json` 是文案与字形的唯一真相，`generate.mjs` 用
  sharp 光栅化，输出确定（同输入两次运行字节一致）。生成不是 CI 步骤。
- **门禁**：新增 `sforum extension brand check <pkg>` / `--root <collection>`，
  校验存在性、字节上限、PNG 魔数与 IHDR 尺寸、SVG viewBox 与自包含性；仓内由
  `apps/api/cmd/sforum` 的 Go 测试扫 `extensions/builtin/**` 与
  `extensions/optional/**` 兜底。外部三仓在 `prepare-package.sh` 里调用同一 CLI。
- **文档**：契约在 `docs/extensions/brand-assets.md`；插件仓库标准（骨架、必须有
  什么、放在哪、门禁、记录约定）在 `docs/{zh-CN,en-US}/development/plugin-standard.md`。

## Consequences

- 12 个包的 artifact 摘要会变一次。内置 8 个要么在后台逐一点「升级」，要么随
  下一个 patch 版本由内置自动更新接管。**不要为了消掉漂移去改版本号。**
- 品牌图随发布 zip 分发（`--exclude-source` 不排除 PNG/SVG），每个包多约 120 KiB；
  `.github/` 会被排除，所以不要把品牌资产放那里。
- 生成需要 sharp（`apps/web/node_modules` → `NODE_PATH` → bun 全局），但流水线只
  校验已提交 PNG，构建镜像不引入原生图像依赖。
- GitHub 仓库 Social preview 没有公开 API，四个仓库各需手动上传一次；文件即上传源。
- 本轮**不加** manifest `icon` / `socialPreview` 字段，也不改管理页。等出现真实
  消费者（管理页插件卡、商城详情页）时再单独立项，届时复用 `brand/` 里的文件，
  不需要重新设计。
- `extensions/dev/`（gitignore 脚手架）与 `extensions/fixtures/`（测试输入）豁免，
  它们不是随产品分发的包。
