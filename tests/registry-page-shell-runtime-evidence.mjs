// Route A 第 6 层的**运行时**证据脚本：公开 Page Registry 页的外壳组合路径。
//
// 为什么是运行时脚本而不是组件单测：`bun test` 没有 SFC 编译步骤，仓库里所有组件
// 测试都是读源码字符串，无法证明「渲染后恰好一个 chrome」；而这一条契约本身要求的
// 就是渲染层证据。本脚本对真实 SSR 输出与真实 resolve 接口做断言。
//
// 用法：
//   node tests/registry-page-shell-runtime-evidence.mjs
// 环境变量：
//   SFORUM_WEB_BASE_URL  默认 http://127.0.0.1:3000
//   SFORUM_API_BASE_URL  默认 http://127.0.0.1:8080
//   SFORUM_EVIDENCE_REQUIRE_LIVE=1  服务不可达时失败而不是跳过（CI 用）
import { writeFile } from 'node:fs/promises'

const web = process.env.SFORUM_WEB_BASE_URL || 'http://127.0.0.1:3000'
const api = process.env.SFORUM_API_BASE_URL || 'http://127.0.0.1:8080'
const requireLive = process.env.SFORUM_EVIDENCE_REQUIRE_LIVE === '1'
const outputPath = process.env.SFORUM_EVIDENCE_OUTPUT || ''

const failures = []
const evidence = []
// SKIP 路径不得打印「全部通过」。
let asserted = false

function fail(message, detail) {
  failures.push(detail ? `${message} :: ${detail}` : message)
}

function count(haystack, needle) {
  return haystack.split(needle).length - 1
}

async function fetchText(url) {
  try {
    const response = await fetch(url, { redirect: 'manual' })
    return { status: response.status, body: await response.text() }
  } catch (error) {
    return { status: 0, body: '', error: String(error?.message || error) }
  }
}

async function fetchJSON(url) {
  const result = await fetchText(url)
  if (result.status !== 200) return { ...result, data: null }
  try {
    return { ...result, data: JSON.parse(result.body)?.data ?? null }
  } catch (error) {
    return { ...result, data: null, error: String(error?.message || error) }
  }
}

// 页面契约：期望的外壳来源与可见 chrome 数量。
const pages = [
  { path: '/probe-site', layout: 'site', shellSource: 'theme', navbars: 1, footers: 1 },
  { path: '/probe-default', layout: 'site', shellSource: 'theme', navbars: 1, footers: 1 },
  { path: '/probe-bare', layout: 'bare', shellSource: 'none', navbars: 0, footers: 0 }
]

async function collect() {
  for (const page of pages) {
    const ssr = await fetchText(`${web}${page.path}`)
    if (ssr.status !== 200) {
      fail(`${page.path}: SSR did not render`, `status=${ssr.status} ${ssr.error || ''}`)
      continue
    }
    // data-registry-shell 承载 layout（site|bare），data-registry-shell-source 承载
    // 外壳来源（theme|host|none）。两者语义不同，不能互相代替。
    const shellSource = /data-registry-shell-source="([a-z]+)"/.exec(ssr.body)?.[1] ?? ''
    const shellLayout = /data-registry-shell="([a-z]+)"/.exec(ssr.body)?.[1] ?? ''
    // 恰好一个 chrome 拥有者：按 <nav> 元素的 class token 精确分类，不用子串计数
    // （`sf-public-mobile-bottom-nav__link` 这类子元素会把子串计数撑大）。
    const navTags = [...ssr.body.matchAll(/<nav\b[^>]*>/g)].map(match => match[0])
    const hasClass = (tag, token) => new RegExp(`class="[^"]*\\b${token}\\b[^"]*"`).test(tag)
    const topbar = navTags.filter(tag => hasClass(tag, 'navbar__desktop-nav')).length
    const mobileNav = navTags.filter(tag => hasClass(tag, 'sf-public-mobile-bottom-nav')).length
    const otherNavs = navTags.length - topbar - mobileNav
    const footers = count(ssr.body, '<footer class="sf-footer"')
    const navbarOwners = (topbar > 0 || mobileNav > 0) ? 1 : 0

    if (shellSource !== page.shellSource) {
      fail(`${page.path}: shell source`, `want=${page.shellSource} got=${shellSource || '(absent)'}`)
    }
    if (shellLayout !== page.layout) {
      fail(`${page.path}: shell layout attribute`, `want=${page.layout} got=${shellLayout || '(absent)'}`)
    }
    if (otherNavs !== 0) {
      fail(`${page.path}: unexpected nav elements`, `${otherNavs} nav(s) are neither the desktop nor the mobile navbar`)
    }
    if (page.navbars === 1 && (topbar !== 1 || mobileNav !== 1)) {
      fail(`${page.path}: navbar responsive pair`, `want one desktop + one mobile nav, got topbar=${topbar} mobile=${mobileNav}`)
    }
    if (navbarOwners !== page.navbars) {
      fail(`${page.path}: navbar owners`, `want=${page.navbars} got=${navbarOwners} (topbar=${topbar} mobile=${mobileNav})`)
    }
    if (topbar > 1 || mobileNav > 1) {
      fail(`${page.path}: duplicate navbar elements`, `topbar=${topbar} mobile=${mobileNav}`)
    }
    if (footers !== page.footers) {
      fail(`${page.path}: footers`, `want=${page.footers} got=${footers}`)
    }

    const resolved = await fetchJSON(`${api}/api/v1/pages/resolve-path?path=${encodeURIComponent(page.path)}`)
    const data = resolved.data || {}
    if (resolved.status !== 200 || !data.layout) {
      fail(`${page.path}: resolve-path`, `status=${resolved.status} ${resolved.error || ''}`)
      continue
    }
    if (data.layout !== page.layout) {
      fail(`${page.path}: resolved layout`, `want=${page.layout} got=${data.layout}`)
    }
    if (!data.selectedProvider || !data.selectedPackageDigest) {
      fail(`${page.path}: selected identity missing`, JSON.stringify({ provider: data.selectedProvider, digest: data.selectedPackageDigest }))
    }
    if (page.layout === 'bare') {
      if (data.shellHtml) fail(`${page.path}: bare must not carry a theme shell`, `shellHtml length=${data.shellHtml.length}`)
      if (data.shellProvider) fail(`${page.path}: bare must not name a shell provider`, String(data.shellProvider))
    } else {
      if (!data.shellHtml) fail(`${page.path}: site must carry the theme shell html`, 'shellHtml empty')
      if (!data.shellProvider || !data.shellPackageDigest) {
        fail(`${page.path}: site shell identity missing`, JSON.stringify({ provider: data.shellProvider, digest: data.shellPackageDigest }))
      }
    }

    evidence.push({
      path: page.path,
      shellSource,
      navbarOwners,
      topbarElements: topbar,
      mobileNavElements: mobileNav,
      footers,
      layout: data.layout,
      selectedProvider: data.selectedProvider,
      selectedPackageDigest: (data.selectedPackageDigest || '').slice(0, 16),
      shellProvider: data.shellProvider || null,
      shellPackageDigest: (data.shellPackageDigest || '').slice(0, 16) || null,
      templatePath: data.templatePath || null
    })
  }
}

// 外壳身份必须与活动主题制品一致：两个身份端点不得各说各话。
async function assertShellIdentityAgrees() {
  const skin = await fetchJSON(`${api}/api/v1/site/active-theme/skin`)
  if (skin.status !== 200 || !skin.data?.packageDigest) {
    fail('active theme skin', `status=${skin.status} ${skin.error || ''}`)
    return
  }
  const siteEvidence = evidence.filter(entry => entry.shellPackageDigest)
  if (siteEvidence.length === 0) {
    fail('shell identity agreement', 'no site page reported a shell digest')
    return
  }
  for (const entry of siteEvidence) {
    if (!skin.data.packageDigest.startsWith(entry.shellPackageDigest)) {
      fail(
        `${entry.path}: shell digest disagrees with /site/active-theme/skin`,
        `resolve=${entry.shellPackageDigest} skin=${skin.data.packageDigest.slice(0, 16)}`
      )
    }
    if (entry.shellProvider !== skin.data.extensionId) {
      fail(
        `${entry.path}: shell provider disagrees with /site/active-theme/skin`,
        `resolve=${entry.shellProvider} skin=${skin.data.extensionId}`
      )
    }
  }
  evidence.push({
    activeTheme: skin.data.extensionId,
    activeThemeVersion: skin.data.version,
    activeThemeDigest: String(skin.data.packageDigest).slice(0, 16)
  })
}

async function main() {
  const probe = await fetchText(`${web}/probe-site`)
  if (probe.status !== 200) {
    const message = `registry page shell evidence needs the running dev servers (web=${web} api=${api})`
    if (requireLive) {
      fail(message, probe.error || `status=${probe.status}`)
    } else {
      console.log(`SKIP ${message}`)
      return
    }
  } else {
    asserted = true
    await collect()
    await assertShellIdentityAgrees()
  }

  const report = { web, api, evidence, failures }
  if (outputPath) await writeFile(outputPath, `${JSON.stringify(report, null, 2)}\n`, 'utf8')

  console.log(`registry page shell runtime evidence (web=${web} api=${api})`)
  for (const entry of evidence) {
    if (entry.activeTheme) {
      console.log(`  active theme: ${entry.activeTheme}@${entry.activeThemeVersion} digest=${entry.activeThemeDigest}`)
      continue
    }
    console.log(
      `  ${entry.path}: shell=${entry.shellSource} navbarOwners=${entry.navbarOwners} footers=${entry.footers} ` +
        `layout=${entry.layout} provider=${entry.selectedProvider} shell=${entry.shellProvider || '(none)'}`
    )
  }
  if (failures.length > 0) {
    for (const failure of failures) console.error(`  FAIL ${failure}`)
    process.exitCode = 1
    return
  }
  if (!asserted) {
    console.log('  skipped: no assertion ran')
    return
  }
  console.log('  all Route A composition assertions passed')
}

await main()
