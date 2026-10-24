# Brand asset generator

生成每个扩展包 `brand/` 下的图标与社交卡。契约（尺寸、上限、校验规则）见
[`docs/extensions/brand-assets.md`](../../docs/extensions/brand-assets.md)。

- `spec.json` — 唯一真相：中文名、一句话、Tabler 字形、种类、仓库 slug。
  `id` / `name` / `version` 从各包 `sforum.extension.json` 读取，不在 spec 里重复。
  覆盖同级仓库的条目写成相对路径（如 `../sforum-fetch-url/plugins/…`）：没有检出
  该仓库时全量生成会打印 `skip` 并跳过，`--only` 指定它才报错。
- `generate.mjs` — 合成 SVG 并用 sharp 光栅化。输出确定：同输入两次运行字节一致。

## 用法

```bash
node scripts/brand/generate.mjs --help          # 参数速查

# 预览到临时目录（不碰仓库），并输出联排样张
node scripts/brand/generate.mjs --draft /tmp/sforum-brand --sheet /tmp/sforum-brand

# 只重生成一个包（按包目录名后缀匹配）
node scripts/brand/generate.mjs --only sforum-smtp

# 写入所有包 + 主仓社交卡
node scripts/brand/generate.mjs
```

`--draft` 的产物布局：`<目录>/<扩展 id>/{icon.png,social-preview.png}`，主仓卡在
`<目录>/repo/social-preview.png`；`--sheet` 另外写 `sheet-cards.png` 与
`sheet-icons.png`。

## 依赖

生成需要 `sharp`，按顺序解析：`apps/web/node_modules/sharp` → `NODE_PATH` →
`~/.bun/install/global/node_modules/sharp`。三者都没有时，先在 `apps/web` 跑
`bun install`（记得设置代理）。

**生成不是 CI 步骤。** 流水线只校验已提交的 PNG（`sforum extension brand check`），
因此构建环境不需要原生图像依赖。

## 改设计

模板参数集中在 `generate.mjs` 顶部（品牌色、字色、字体、眉标文案）。改动后必须
重跑一次全量生成，并目视复核联排样张：字重、字距、溢出、以及 128px 缩略下的可辨性。
