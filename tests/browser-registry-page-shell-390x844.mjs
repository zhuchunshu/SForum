// 真正的 390x844 浏览器验收（M2 任务 7 / A3、M10 行 8）。
//
// 为什么需要这个脚本：`ego_*` 侧栏载体的 viewport 被钳在 500px，
// `Emulation.setDeviceMetricsOverride` 被接受但不生效；要拿到真正的 390x844
// 必须用真实 Chrome 走 CDP。上一轮的手工 pass 只留下了截图与表格，没有可复现的
// 工件——本脚本把它固化成可重复命令。
//
// 用法：
//   node tests/browser-registry-page-shell-390x844.mjs
// 环境变量：
//   SFORUM_WEB_BASE_URL     默认 http://127.0.0.1:3000
//   SFORUM_CHROME           默认自动探测 google-chrome / chromium / msedge
//   SFORUM_EVIDENCE_OUTPUT  可选：把 JSON 报告写到该路径
//   SFORUM_QA_DIR           可选：截图输出目录（默认不截图）
//   SFORUM_EVIDENCE_REQUIRE_LIVE=1  服务或浏览器不可用时失败而不是跳过
import { spawn } from 'node:child_process'
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const web = process.env.SFORUM_WEB_BASE_URL || 'http://127.0.0.1:3000'
const requireLive = process.env.SFORUM_EVIDENCE_REQUIRE_LIVE === '1'
const outputPath = process.env.SFORUM_EVIDENCE_OUTPUT || ''
const qaDir = process.env.SFORUM_QA_DIR || ''
const WIDTH = 390
const HEIGHT = 844
const DPR = 3
// 行 8 的 JS 禁用子项：`SFORUM_DISABLE_JS=1` 时通过 CDP 关闭脚本执行，断言 chrome
// 与正文仍由 SSR 提供（不依赖 hydration）。
const disableJS = process.env.SFORUM_DISABLE_JS === '1'

const failures = []
const evidence = []
// 断言是否真的跑过：SKIP 路径不得打印「全部通过」，否则门禁会出现假阳性。
let asserted = false
let chromeProcess = null

function fail(message, detail) {
  failures.push(detail ? `${message} :: ${detail}` : message)
}

function chromeBinary() {
  if (process.env.SFORUM_CHROME) return process.env.SFORUM_CHROME
  for (const candidate of [
    '/usr/bin/google-chrome',
    '/usr/bin/google-chrome-stable',
    '/opt/google/chrome/chrome',
    '/usr/bin/chromium',
    '/usr/bin/microsoft-edge'
  ]) {
    if (existsSync(candidate)) return candidate
  }
  return ''
}

async function launchChrome(binary) {
  const port = 9222 + Math.floor(Math.random() * 500)
  const profile = `/tmp/sforum-cdp-${Date.now()}`
  mkdirSync(profile, { recursive: true })
  const child = spawn(binary, [
    '--headless=new',
    `--remote-debugging-port=${port}`,
    `--user-data-dir=${profile}`,
    '--no-first-run',
    '--no-default-browser-check',
    '--disable-gpu',
    '--hide-scrollbars',
    'about:blank'
  ], { stdio: 'ignore' })
  chromeProcess = child
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      const response = await fetch(`http://127.0.0.1:${port}/json/version`)
      if (response.ok) return { child, port, version: await response.json() }
    } catch {
      // Chrome 还没起来。
    }
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  throw new Error(`headless Chrome did not expose CDP on port ${port}`)
}

// 极简 CDP 客户端：够用即可（一个 target、顺序发命令）。
async function openTarget(port, url) {
  const response = await fetch(`http://127.0.0.1:${port}/json/new?${encodeURIComponent(url)}`, { method: 'PUT' })
  if (!response.ok) throw new Error(`cannot open target: ${response.status}`)
  const target = await response.json()
  const socket = new WebSocket(target.webSocketDebuggerUrl)
  await new Promise((resolve, reject) => {
    socket.addEventListener('open', resolve, { once: true })
    socket.addEventListener('error', reject, { once: true })
  })
  let nextId = 1
  const pending = new Map()
  const consoleMessages = []
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data)
    if (message.id && pending.has(message.id)) {
      const { resolve, reject } = pending.get(message.id)
      pending.delete(message.id)
      if (message.error) reject(new Error(`${message.error.message} (${JSON.stringify(message.error.data ?? '')})`))
      else resolve(message.result)
      return
    }
    if (message.method === 'Runtime.consoleAPICalled') {
      consoleMessages.push({
        type: message.params.type,
        text: (message.params.args || []).map(argument => argument.value ?? argument.description ?? '').join(' ')
      })
    }
    if (message.method === 'Runtime.exceptionThrown') {
      consoleMessages.push({
        type: 'exception',
        text: message.params.exceptionDetails?.exception?.description || message.params.exceptionDetails?.text || 'exception'
      })
    }
  })
  const send = (method, params = {}) => new Promise((resolve, reject) => {
    const id = nextId++
    pending.set(id, { resolve, reject })
    socket.send(JSON.stringify({ id, method, params }))
  })
  await send('Page.enable')
  await send('Runtime.enable')
  await send('Emulation.setDeviceMetricsOverride', {
    width: WIDTH, height: HEIGHT, deviceScaleFactor: DPR, mobile: true
  })
  if (disableJS) {
    await send('Emulation.setScriptExecutionDisabled', { value: true })
  }
  await send('Page.navigate', { url })
  // 等 load 事件；超时也继续，用下面的 DOM 断言兜底。
  await new Promise(resolve => {
    const timer = setTimeout(resolve, 15000)
    const onMessage = event => {
      const message = JSON.parse(event.data)
      if (message.method === 'Page.loadEventFired') {
        clearTimeout(timer)
        socket.removeEventListener('message', onMessage)
        resolve()
      }
    }
    socket.addEventListener('message', onMessage)
  })
  return { socket, send, consoleMessages, targetId: target.id }
}

async function evaluate(client, expression) {
  const result = await client.send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  if (result.exceptionDetails) {
    throw new Error(result.exceptionDetails.exception?.description || 'evaluate failed')
  }
  return result.result.value
}

const measure = `(() => {
  const visible = element => {
    const style = getComputedStyle(element)
    const box = element.getBoundingClientRect()
    return style.display !== 'none' && style.visibility !== 'hidden' && box.width > 0 && box.height > 0
  }
  const navs = [...document.querySelectorAll('nav')]
  const hasClass = (element, token) => element.classList.contains(token)
  const shell = document.querySelector('[data-registry-shell]')
  const clipped = [...document.querySelectorAll('button, a, input, select, textarea')]
    .filter(visible)
    .filter(element => { const box = element.getBoundingClientRect(); return box.right > innerWidth + 1 || box.left < -1 })
    .length
  return {
    innerWidth, innerHeight, dpr: devicePixelRatio, screenWidth: screen.width,
    visibleNavbars: navs.filter(visible).length,
    desktopNavs: navs.filter(element => hasClass(element, 'navbar__desktop-nav')).length,
    mobileNavs: navs.filter(element => hasClass(element, 'sf-public-mobile-bottom-nav')).length,
    visibleFooters: [...document.querySelectorAll('footer')].filter(visible).length,
    shellSource: shell ? shell.getAttribute('data-registry-shell-source') : null,
    shellLayout: shell ? shell.getAttribute('data-registry-shell') : null,
    horizontalOverflow: document.documentElement.scrollWidth > innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
    clippedControls: clipped,
    bodyText: document.body.innerText.slice(0, 60)
  }
})()`

const pages = [
  { path: '/probe-site', layout: 'site', shellSource: 'theme', navbars: 1, footers: 1 },
  { path: '/probe-default', layout: 'site', shellSource: 'theme', navbars: 1, footers: 1 },
  { path: '/probe-bare', layout: 'bare', shellSource: 'none', navbars: 0, footers: 0 }
]

async function main() {
  const binary = chromeBinary()
  if (!binary) {
    if (requireLive) fail('no Chrome/Chromium binary found for the 390x844 pass')
    else console.log('SKIP no Chrome/Chromium binary found for the 390x844 pass')
    return finish()
  }
  const probe = await fetch(`${web}/probe-site`, { redirect: 'manual' }).catch(() => null)
  if (!probe || probe.status !== 200) {
    if (requireLive) fail('the 390x844 pass needs the running web dev server', `web=${web}`)
    else console.log(`SKIP the 390x844 pass needs the running web dev server (web=${web})`)
    return finish()
  }

  asserted = true
  const { child, port, version } = await launchChrome(binary)
  evidence.push({ browser: version.Browser, viewport: `${WIDTH}x${HEIGHT} dpr ${DPR}`, javaScript: disableJS ? 'disabled' : 'enabled' })
  for (const page of pages) {
    const client = await openTarget(port, `${web}${page.path}`)
    try {
      const result = await evaluate(client, measure)
      const consoleErrors = client.consoleMessages.filter(message => message.type === 'error' || message.type === 'exception')
      evidence.push({ path: page.path, ...result, consoleErrors })
      if (result.innerWidth !== WIDTH || result.innerHeight !== HEIGHT) {
        fail(`${page.path}: viewport override did not apply`, `innerWidth=${result.innerWidth} innerHeight=${result.innerHeight}`)
      }
      if (result.visibleNavbars !== page.navbars) {
        fail(`${page.path}: visible navbars`, `want=${page.navbars} got=${result.visibleNavbars}`)
      }
      if (result.visibleFooters !== page.footers) {
        fail(`${page.path}: visible footers`, `want=${page.footers} got=${result.visibleFooters}`)
      }
      if (result.shellSource !== page.shellSource || result.shellLayout !== page.layout) {
        fail(`${page.path}: shell`, `want=${page.layout}/${page.shellSource} got=${result.shellLayout}/${result.shellSource}`)
      }
      if (result.horizontalOverflow) {
        fail(`${page.path}: horizontal overflow`, `scrollWidth=${result.scrollWidth} innerWidth=${result.innerWidth}`)
      }
      if (result.clippedControls !== 0) {
        fail(`${page.path}: clipped controls`, `${result.clippedControls} visible control(s) extend past the viewport`)
      }
      if (consoleErrors.length > 0) {
        fail(`${page.path}: console errors`, JSON.stringify(consoleErrors.slice(0, 3)))
      }
      if (disableJS) {
        // JS 关闭时不能依赖客户端渲染：外壳标记与正文必须在首屏 DOM 里就已存在。
        if (result.shellSource !== page.shellSource) {
          fail(`${page.path}: SSR shell marker missing without JavaScript`, `shellSource=${result.shellSource}`)
        }
        if (!String(result.bodyText || '').trim()) {
          fail(`${page.path}: SSR body text missing without JavaScript`)
        }
      }
      if (qaDir) {
        mkdirSync(qaDir, { recursive: true })
        const shot = await client.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
        const file = join(qaDir, `390x844${page.path.replaceAll('/', '-')}.png`)
        writeFileSync(file, Buffer.from(shot.data, 'base64'))
        evidence[evidence.length - 1].screenshot = file
      }
    } finally {
      client.socket.close()
      await fetch(`http://127.0.0.1:${port}/json/close/${client.targetId}`).catch(() => {})
    }
  }
  child.kill('SIGKILL')
  finish()
}

function finish() {
  if (outputPath) writeFileSync(outputPath, `${JSON.stringify({ web, failures, evidence }, null, 2)}\n`, 'utf8')
  console.log(`registry page shell 390x844 browser evidence (web=${web})`)
  for (const entry of evidence) {
    if (entry.browser) {
      console.log(`  browser: ${entry.browser} viewport=${entry.viewport}`)
      continue
    }
    console.log(
      `  ${entry.path}: ${entry.innerWidth}x${entry.innerHeight} dpr=${entry.dpr} navbars=${entry.visibleNavbars} ` +
        `footers=${entry.visibleFooters} shell=${entry.shellLayout}/${entry.shellSource} overflow=${entry.horizontalOverflow} ` +
        `clipped=${entry.clippedControls} consoleErrors=${entry.consoleErrors.length}`
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
  console.log('  all 390x844 acceptance assertions passed')
}

try {
  await main()
} catch (error) {
  fail('harness error', String(error?.message || error))
  finish()
} finally {
  if (chromeProcess) chromeProcess.kill('SIGKILL')
}
