#!/usr/bin/env node
// 生成扩展品牌资产：brand/icon.svg、brand/icon.png、brand/social-preview.png。
//
// 设计约束见 docs/extensions/brand-assets.md。这里只做一件事：把 spec.json 的
// 一条记录 + 包内 sforum.extension.json 的元数据，渲染成同一套模板的图。
// 生成不是 CI 步骤 —— 校验（sforum extension brand check）才在门禁里跑。

import { createRequire } from 'node:module'
import crypto from 'node:crypto'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const require = createRequire(import.meta.url)
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

const BRAND_START = '#2671ed'
const BRAND_END = '#1f6bec'
const INK = '#f4f8fd'
const INK_SOFT = '#b9c7d8'
const INK_MUTE = '#8fa0b8'
const INK_FAINT = '#5b6b85'
const CANVAS = '#0b1220'
const TILE = '#0e1728'
const HAIRLINE = 'rgba(255,255,255,0.08)'

const FONT_SANS = "'Noto Sans CJK SC','Noto Sans CJK','Noto Sans','DejaVu Sans',sans-serif"
const FONT_MONO = "'DejaVu Sans Mono','Noto Sans Mono',monospace"

const EYEBROW = {
  builtin: 'SFORUM · 内置插件',
  external: 'SFORUM · 扩展插件',
  theme: 'SFORUM · 主题',
  framework: 'SFORUM · 框架'
}

function loadSharp() {
  const candidates = [
    path.join(root, 'apps/web/node_modules/sharp'),
    ...(process.env.NODE_PATH
      ? process.env.NODE_PATH.split(path.delimiter).filter(Boolean).map((entry) => path.join(entry, 'sharp'))
      : []),
    path.join(os.homedir(), '.bun/install/global/node_modules/sharp')
  ]
  for (const candidate of candidates) {
    try {
      return require(candidate)
    } catch {
      // 继续尝试下一个来源：生成是维护者任务，不绑定单一安装位置。
    }
  }
  throw new Error(
    '找不到 sharp。生成品牌资产需要 sharp：在 apps/web 跑 bun install，或设置 NODE_PATH 指向已安装 sharp 的 node_modules。'
  )
}

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'))
}

function escapeXML(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

// 粗略字宽估算：只用于在生成时警告溢出，不参与排版。
function estimateWidth(text, size) {
  let width = 0
  for (const char of String(text)) {
    width += /[\u3000-\u9fff\uff00-\uffef]/.test(char) ? size : size * 0.55
  }
  return width
}

function gradientDefs() {
  return `
    <linearGradient id="brand" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="${BRAND_START}"/>
      <stop offset="1" stop-color="${BRAND_END}"/>
    </linearGradient>
    <linearGradient id="wash" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0.4" stop-color="${BRAND_START}" stop-opacity="0"/>
      <stop offset="1" stop-color="${BRAND_START}" stop-opacity="0.14"/>
    </linearGradient>
    <linearGradient id="tile" x1="0" y1="0" x2="0.35" y2="1">
      <stop offset="0" stop-color="#101b2d"/>
      <stop offset="1" stop-color="${CANVAS}"/>
    </linearGradient>
    <radialGradient id="glow" cx="0.5" cy="0.5" r="0.5">
      <stop offset="0" stop-color="${BRAND_START}" stop-opacity="0.17"/>
      <stop offset="1" stop-color="${BRAND_START}" stop-opacity="0"/>
    </radialGradient>`
}

function glyphMarkup(glyphBody, translateX, translateY, boxSize) {
  const scale = boxSize / 24
  return `<g transform="translate(${translateX} ${translateY}) scale(${scale})">${glyphBody}</g>`
}

function iconSVG(glyphBody) {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <defs>${gradientDefs()}</defs>
  <rect width="512" height="512" rx="112" fill="url(#tile)"/>
  <rect width="512" height="512" rx="112" fill="url(#glow)"/>
  <rect x="0.5" y="0.5" width="511" height="511" rx="111.5" fill="none" stroke="${HAIRLINE}" stroke-width="1"/>
  ${glyphMarkup(glyphBody, 128, 128, 256)}
</svg>
`
}

function logoMarkup(file, x, y, size) {
  const raw = fs.readFileSync(file, 'utf8')
  const inner = raw
    .replace(/^<\?xml[^>]*\?>\s*/i, '')
    .replace(/^<svg[^>]*>/, '')
    .replace(/<\/svg>\s*$/i, '')
  return `<svg x="${x}" y="${y}" width="${size}" height="${size}" viewBox="0 0 310 310">${inner}</svg>`
}

function cardSVG({ eyebrow, title, name, id, version, tagline, slug, glyphBody, logo }) {
  const metaParts = id && version
    ? [
        `<tspan fill="${INK_FAINT}">${escapeXML(id)}</tspan>`,
        `<tspan fill="#475569">   ·   </tspan>`,
        `<tspan fill="#4d7fd6">v${escapeXML(version)}</tspan>`
      ].join('')
    : ''

  const art = `<circle cx="1016" cy="320" r="260" fill="url(#glow)"/>
  <rect x="856" y="160" width="320" height="320" rx="72" fill="url(#tile)"/>
  <rect x="856.5" y="160.5" width="319" height="319" rx="71.5" fill="none" stroke="${HAIRLINE}" stroke-width="1"/>
  ${logo ? logoMarkup(logo, 901, 205, 230) : glyphMarkup(glyphBody, 916, 220, 200)}`

  return `<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="640" viewBox="0 0 1280 640">
  <defs>${gradientDefs()}</defs>
  <rect width="1280" height="640" fill="${CANVAS}"/>
  <rect width="1280" height="640" fill="url(#wash)"/>
  <rect x="24.5" y="24.5" width="1231" height="591" rx="20" fill="none" stroke="rgba(255,255,255,0.07)" stroke-width="1"/>
  <line x1="792" y1="150" x2="792" y2="500" stroke="rgba(255,255,255,0.06)" stroke-width="1"/>

  <text x="88" y="176" font-family="${FONT_SANS}" font-size="14" fill="#6b7d94"><tspan letter-spacing="3">SFORUM</tspan><tspan dx="7" letter-spacing="0.5">· ${escapeXML(eyebrow.split('·')[1]?.trim() ?? eyebrow)}</tspan></text>
  <text x="88" y="282" font-family="${FONT_SANS}" font-size="64" font-weight="700" fill="${INK}">${escapeXML(title)}</text>
  <text x="88" y="326" font-family="${FONT_SANS}" font-size="20" fill="${INK_MUTE}">${escapeXML(name)}</text>
  <text x="88" y="396" font-family="${FONT_SANS}" font-size="26" fill="${INK_SOFT}">${escapeXML(tagline)}</text>
  ${metaParts ? `<text x="88" y="446" font-family="${FONT_MONO}" font-size="16">${metaParts}</text>` : ''}

  ${art}

  <g transform="translate(88 556)">${logoMarkup(path.join(root, 'apps/web/public/brand/sforum-logo.svg'), 0, 0, 28)}</g>
  <text x="128" y="577" font-family="${FONT_SANS}" font-size="19" font-weight="600" fill="#dce5f2">SForum</text>
  <text x="1192" y="577" text-anchor="end" font-family="${FONT_MONO}" font-size="14" fill="#46586f">${escapeXML(slug)}</text>
</svg>
`
}

async function writeAsset(sharp, svg, file, { width, height }) {
  const buffer = await sharp(Buffer.from(svg)).png({ compressionLevel: 9 }).toBuffer()
  fs.writeFileSync(file, buffer)
  const meta = await sharp(buffer).metadata()
  return {
    file,
    bytes: buffer.length,
    width: meta.width,
    height: meta.height,
    digest: crypto.createHash('sha256').update(buffer).digest('hex').slice(0, 12),
    expected: { width, height }
  }
}

function loadGlyphs() {
  const icons = readJSON(path.join(root, 'apps/web/node_modules/@iconify-json/tabler/icons.json'))
  return (name) => {
    const icon = icons.icons[name] ?? (icons.aliases?.[name] ? icons.icons[icons.aliases[name].parent] : null)
    if (!icon) throw new Error(`Tabler 图标不存在：${name}`)
    return icon.body.replaceAll('currentColor', 'url(#brand)')
  }
}

// 返回 null 表示这个包在当前机器上没有检出（spec 覆盖同级仓库，例如
// ../sforum-fetch-url）。全量生成时跳过，显式 --only 时才报错。
function resolvePackage(entry) {
  const dir = path.isAbsolute(entry.package) ? entry.package : path.join(root, entry.package)
  const manifestPath = path.join(dir, 'sforum.extension.json')
  if (!fs.existsSync(manifestPath)) return null
  return { dir, manifest: readJSON(manifestPath) }
}

const USAGE = `用法：
  node scripts/brand/generate.mjs                     写入所有包的 brand/ 与主仓社交卡
  node scripts/brand/generate.mjs --draft <目录>       只写到临时目录，不碰仓库
  node scripts/brand/generate.mjs --sheet <目录>       额外输出联排样张（可配合 --draft）
  node scripts/brand/generate.mjs --only <包名后缀>     只重生成一个包，例如 sforum-smtp
`

// 取值型参数缺值时给出用法，而不是让 path.resolve(undefined) 抛栈。
function flagValue(args, name) {
  const index = args.indexOf(name)
  if (index < 0) return null
  const value = args[index + 1]
  if (!value || value.startsWith('--')) {
    console.error(`[brand] ${name} 需要一个值\n\n${USAGE}`)
    process.exit(1)
  }
  return path.resolve(value)
}

async function main() {
  const args = process.argv.slice(2)
  if (args.includes('--help') || args.includes('-h')) {
    console.log(USAGE)
    return
  }
  const draftDir = flagValue(args, '--draft')
  const sheetDir = flagValue(args, '--sheet')
  const onlyIndex = args.indexOf('--only')
  const only = onlyIndex >= 0 ? args[onlyIndex + 1] : null

  const sharp = loadSharp()
  const glyphFor = loadGlyphs()
  const spec = readJSON(path.join(root, 'scripts/brand/spec.json'))

  const rendered = []
  for (const entry of spec.extensions) {
    if (only && !entry.package.endsWith(only)) continue
    const resolved = resolvePackage(entry)
    if (!resolved) {
      if (only) throw new Error(`找不到扩展清单：${entry.package}`)
      console.log(`skip ${entry.package}（当前机器未检出）`)
      continue
    }
    const { dir, manifest } = resolved
    const glyphBody = glyphFor(entry.glyph)
    const outDir = draftDir ? path.join(draftDir, manifest.id) : path.join(dir, 'brand')
    fs.mkdirSync(outDir, { recursive: true })

    const iconFile = path.join(outDir, 'icon.png')
    const cardFile = path.join(outDir, 'social-preview.png')
    const iconSvg = iconSVG(glyphBody)
    const card = cardSVG({
      eyebrow: EYEBROW[entry.kind] ?? EYEBROW.builtin,
      title: entry.title,
      name: manifest.name,
      id: manifest.id,
      version: manifest.version,
      tagline: entry.tagline,
      slug: entry.slug,
      glyphBody
    })

    if (!draftDir) fs.writeFileSync(path.join(outDir, 'icon.svg'), iconSvg)

    const icon = await writeAsset(sharp, iconSvg, iconFile, { width: 512, height: 512 })
    const social = await writeAsset(sharp, card, cardFile, { width: 1280, height: 640 })

    for (const asset of [icon, social]) {
      if (asset.width !== asset.expected.width || asset.height !== asset.expected.height) {
        throw new Error(`${asset.file} 尺寸 ${asset.width}x${asset.height}，期望 ${asset.expected.width}x${asset.expected.height}`)
      }
    }
    if (estimateWidth(entry.title, 56) > 660) {
      console.warn(`[warn] ${manifest.id}: 标题可能溢出（估算 ${Math.round(estimateWidth(entry.title, 56))}px）`)
    }
    if (estimateWidth(entry.tagline, 22) > 700) {
      console.warn(`[warn] ${manifest.id}: 一句话可能溢出（估算 ${Math.round(estimateWidth(entry.tagline, 22))}px）`)
    }

    rendered.push({ entry, manifest, icon, social, card })
  }

  for (const repo of spec.repos) {
    const outFile = draftDir
      ? path.join(draftDir, 'repo', path.basename(repo.output))
      : path.join(root, repo.output)
    fs.mkdirSync(path.dirname(outFile), { recursive: true })
    const card = cardSVG({
      eyebrow: EYEBROW[repo.kind] ?? EYEBROW.framework,
      title: repo.title,
      name: 'Plugin-first open-source forum framework',
      id: null,
      version: null,
      tagline: repo.tagline,
      slug: repo.slug,
      logo: path.join(root, 'apps/web/public/brand/sforum-logo.svg')
    })
    const social = await writeAsset(sharp, card, outFile, { width: 1280, height: 640 })
    rendered.push({ entry: repo, manifest: { id: 'sforum', name: repo.title }, social })
  }

  for (const item of rendered) {
    const label = item.manifest.id.padEnd(26)
    const parts = []
    if (item.icon) parts.push(`icon ${item.icon.width}px ${(item.icon.bytes / 1024).toFixed(1)}KiB`)
    parts.push(`card ${item.social.width}x${item.social.height} ${(item.social.bytes / 1024).toFixed(1)}KiB`)
    console.log(`${label} ${parts.join('  ')}`)
  }

  if (sheetDir) await buildSheets(sharp, rendered, sheetDir)
}

async function buildSheets(sharp, rendered, outDir) {
  fs.mkdirSync(outDir, { recursive: true })
  const cards = rendered.filter((item) => item.social)
  const icons = rendered.filter((item) => item.icon)

  const cellW = 420
  const cellH = 210
  const gap = 18
  const pad = 28
  const cols = 3
  const rows = Math.ceil(cards.length / cols)
  const width = cols * cellW + (cols - 1) * gap + pad * 2
  const height = rows * cellH + (rows - 1) * gap + pad * 2
  const cardCells = cards
    .map((item, index) => {
      const x = pad + (index % cols) * (cellW + gap)
      const y = pad + Math.floor(index / cols) * (cellH + gap)
      const data = fs.readFileSync(item.social.file).toString('base64')
      return `<image x="${x}" y="${y}" width="${cellW}" height="${cellH}" href="data:image/png;base64,${data}"/>`
    })
    .join('\n')
  const cardSheet = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
  <rect width="${width}" height="${height}" fill="#05080f"/>
  ${cardCells}
</svg>`
  const cardFile = path.join(outDir, 'sheet-cards.png')
  fs.writeFileSync(cardFile, await sharp(Buffer.from(cardSheet)).png().toBuffer())

  const iconCell = 128
  const labelH = 26
  const iconCols = 6
  const iconRows = Math.ceil(icons.length / iconCols)
  const iw = iconCols * iconCell + (iconCols - 1) * gap + pad * 2
  const ih = iconRows * (iconCell + labelH) + (iconRows - 1) * gap + pad * 2
  const iconCells = icons
    .map((item, index) => {
      const x = pad + (index % iconCols) * (iconCell + gap)
      const y = pad + Math.floor(index / iconCols) * (iconCell + labelH + gap)
      const data = fs.readFileSync(item.icon.file).toString('base64')
      return `<image x="${x}" y="${y}" width="${iconCell}" height="${iconCell}" href="data:image/png;base64,${data}"/>
  <text x="${x + iconCell / 2}" y="${y + iconCell + 17}" text-anchor="middle" font-family="${FONT_MONO}" font-size="11" fill="#64748b">${escapeXML(item.manifest.id)}</text>`
    })
    .join('\n')
  const iconSheet = `<svg xmlns="http://www.w3.org/2000/svg" width="${iw}" height="${ih}" viewBox="0 0 ${iw} ${ih}">
  <rect width="${iw}" height="${ih}" fill="#05080f"/>
  ${iconCells}
</svg>`
  const iconFile = path.join(outDir, 'sheet-icons.png')
  fs.writeFileSync(iconFile, await sharp(Buffer.from(iconSheet)).png().toBuffer())

  console.log(`sheet: ${cardFile}`)
  console.log(`sheet: ${iconFile}`)
}

main().catch((error) => {
  console.error(`[brand] ${error.message}`)
  process.exit(1)
})
