// 公开页 390x844 浏览器验收（M2 任务 7 / A3）。
//
// 为什么需要它：`ego_*` 载体上 `Emulation.setDeviceMetricsOverride` 被接受但不生效
// （实测 innerWidth 恒为 500），所以真正的 390 宽必须换载体。这里与
// `tests/browser-admin-390x844.mjs` 用同一套 CDP 手法（真实 Chrome/Chromium +
// `Emulation.setDeviceMetricsOverride`），区别是公开页不需要会话。
//
// 断言（对每个路径）：
//   - viewport override 真的生效（innerWidth === 390，否则 FAIL，不冒充）
//   - 可见 navbar / footer 数量符合该路径的期望（默认恰好各一；bare 期望 0）
//   - 无横向溢出（scrollWidth <= innerWidth）
//   - 无被裁切的可交互控件（button/a/input/select/textarea/[role=button] 越界）
//   - 无 console error / exception
//
// 用法：
//   node tests/browser-public-390x844.mjs
// 环境变量：
//   SFORUM_WEB_BASE_URL     默认 http://127.0.0.1:3000
//   SFORUM_PUBLIC_PATHS     逗号分隔；默认 "/loader-articles/hello,/probe-site,/probe-bare"
//   SFORUM_BARE_PATHS       逗号分隔的「期望 0 chrome」路径；默认 "/probe-bare"
//   SFORUM_L2_PATHS         逗号分隔的「期望公开 L2 组件真的挂载」路径（默认空）。
//                           命中的路径会断言 data-l2-state=mounted、目标容器有子节点、
//                           且组件渲染了自己的内容。这是组件页 390x844 的挂载证据，
//                           只有真正量到才通过——水合超时会明确 FAIL，不冒充。
//   SFORUM_QA_DIR           可选：截图目录
//   SFORUM_EVIDENCE_OUTPUT  可选：JSON 报告路径
import { spawn } from 'node:child_process'
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const web = process.env.SFORUM_WEB_BASE_URL || 'http://127.0.0.1:3000'
const paths = (process.env.SFORUM_PUBLIC_PATHS || '/loader-articles/hello,/probe-site,/probe-bare')
  .split(',').map(value => value.trim()).filter(Boolean)
const barePaths = new Set((process.env.SFORUM_BARE_PATHS || '/probe-bare')
  .split(',').map(value => value.trim()).filter(Boolean))
const l2Paths = new Set((process.env.SFORUM_L2_PATHS || '')
  .split(',').map(value => value.trim()).filter(Boolean))
const qaDir = process.env.SFORUM_QA_DIR || ''
const outputPath = process.env.SFORUM_EVIDENCE_OUTPUT || ''
// 默认就是 A3 要求的 390x844；允许覆盖以做对比诊断（例如判断某个失败是否与宽度相关）。
const WIDTH = Number(process.env.SFORUM_VIEWPORT_WIDTH || 390)
const HEIGHT = Number(process.env.SFORUM_VIEWPORT_HEIGHT || 844)
const DPR = Number(process.env.SFORUM_VIEWPORT_DPR || 3)

const failures = []
const evidence = []
let asserted = false

function fail(message, detail) {
  failures.push(detail ? `${message} :: ${detail}` : message)
}

function chromeBinary() {
  for (const candidate of ['/usr/bin/google-chrome', '/usr/bin/google-chrome-stable', '/opt/google/chrome/chrome', '/usr/bin/chromium', '/usr/bin/chromium-browser']) {
    if (existsSync(candidate)) return candidate
  }
  return ''
}

async function launchChrome(binary) {
  const port = 9400 + Math.floor(Math.random() * 300)
  const profile = `/tmp/sforum-public-cdp-${Date.now()}`
  mkdirSync(profile, { recursive: true })
  const child = spawn(binary, [
    '--headless=new', `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`,
    '--no-first-run', '--no-default-browser-check', '--disable-gpu', '--hide-scrollbars', 'about:blank',
  ], { stdio: 'ignore' })
  for (let attempt = 0; attempt < 120; attempt += 1) {
    try {
      const response = await fetch(`http://127.0.0.1:${port}/json/version`)
      if (response.ok) return { child, port, version: await response.json() }
    } catch {
      // 等待 Chrome 就绪。
    }
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  throw new Error(`headless Chrome did not expose CDP on port ${port}`)
}

async function connect(port) {
  const response = await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: 'PUT' })
  const target = await response.json()
  const socket = new WebSocket(target.webSocketDebuggerUrl)
  await new Promise((resolve, reject) => {
    socket.addEventListener('open', resolve, { once: true })
    socket.addEventListener('error', reject, { once: true })
  })
  let nextId = 1
  const pending = new Map()
  const consoleMessages = []
  const networkLog = []
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data)
    if (message.id && pending.has(message.id)) {
      const { resolve, reject } = pending.get(message.id)
      pending.delete(message.id)
      if (message.error) reject(new Error(message.error.message))
      else resolve(message.result)
      return
    }
    if (message.method === 'Runtime.consoleAPICalled') {
      consoleMessages.push({
        type: message.params.type,
        text: (message.params.args || []).map(argument => argument.value ?? argument.description ?? '').join(' '),
      })
    }
    if (process.env.SFORUM_LOG_NETWORK && message.method === 'Network.responseReceived') {
      const url = message.params?.response?.url || ''
      if (url.includes('/extensions/runtime/') || url.includes('/_sforum/assets/')) {
        networkLog.push(`${message.params.response.status} ${url.replace(web, '').slice(0, 110)}`)
      }
    }
    if (process.env.SFORUM_LOG_NETWORK && message.method === 'Network.loadingFailed') {
      networkLog.push(`FAILED ${message.params?.errorText} ${(message.params?.requestId || '').slice(0, 12)}`)
    }
    if (message.method === 'Runtime.exceptionThrown') {
      consoleMessages.push({ type: 'exception', text: message.params.exceptionDetails?.text || 'exception' })
    }
  })
  const send = (method, params = {}) => new Promise((resolve, reject) => {
    const id = nextId++
    pending.set(id, { resolve, reject })
    socket.send(JSON.stringify({ id, method, params }))
  })
  await send('Page.enable')
  await send('Runtime.enable')
  // 可选会话：公开页不需要登录，但保留注入能力用于对比诊断
  // （例如判断某个公开挂载是否意外依赖会话）。
  if (process.env.SFORUM_SESSION_COOKIE) {
    await send('Network.enable')
    await send('Network.setCookie', {
      name: 'sforum_session', value: process.env.SFORUM_SESSION_COOKIE,
      domain: '127.0.0.1', path: '/', httpOnly: true, sameSite: 'Lax',
    })
  }
  await send('Emulation.setDeviceMetricsOverride', { width: WIDTH, height: HEIGHT, deviceScaleFactor: DPR, mobile: true })
  return { socket, send, consoleMessages, networkLog }
}

const measure = `(() => {
  const visible = element => {
    const style = getComputedStyle(element)
    const box = element.getBoundingClientRect()
    return style.display !== 'none' && style.visibility !== 'hidden' && box.width > 0 && box.height > 0
  }
  const chrome = document.querySelector('[data-registry-shell]')
  const clipped = [...document.querySelectorAll('button, a, input, select, textarea, [role="button"]')]
    .filter(visible)
    .filter(element => { const box = element.getBoundingClientRect(); return box.right > innerWidth + 1 || box.left < -1 })
  return {
    url: location.pathname,
    innerWidth, innerHeight, dpr: devicePixelRatio,
    horizontalOverflow: document.documentElement.scrollWidth > innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
    visibleHeaders: [...document.querySelectorAll('header')].filter(visible).length,
    visibleFooters: [...document.querySelectorAll('footer')].filter(visible).length,
    clippedControls: clipped.length,
    clippedSample: clipped.slice(0, 3).map(element => element.tagName + '.' + (element.className || '').toString().slice(0, 40)),
    shell: chrome ? chrome.getAttribute('data-registry-shell') : null,
    shellSource: chrome ? chrome.getAttribute('data-registry-shell-source') : null,
    bodyText: document.body.innerText.replace(/\\s+/g, ' ').slice(0, 160),
    // 组件页（view: component）的公开 L2 挂载标记：正文出口内挂载的组件会渲染自己的
    // 文案与信任提示。没有它就无法区分「页面打开了」和「组件真的挂上了」。
    trustNoticeRendered: document.body.innerText.includes('完全信任'),
    // 诊断用：内容是否在 DOM 里但被隐藏（移动端布局），还是根本没渲染。
    htmlHasPluginText: document.body.innerHTML.includes('page component reference body') ||
      document.body.innerHTML.includes('Loaded from the plugin runtime'),
    mainDisplay: (() => { const m = document.querySelector('main'); return m ? getComputedStyle(m).display + '/' + m.getBoundingClientRect().width + 'x' + m.getBoundingClientRect().height : null })(),
    mainTextLength: (document.querySelector('main') || document.body).innerText.length,
    // 正文出口 / 岛的存在性（判断移动端是否根本没渲染正文）。
    islandCount: document.querySelectorAll('[data-sforum-island]').length,
    islandIds: Array.from(document.querySelectorAll('[data-sforum-island]')).map(e => e.getAttribute('data-sforum-island')).slice(0, 8),
    bodyOutletPresent: !!document.querySelector('[data-registry-body]') ||
      !!document.querySelector('[data-plugin-page-body]') ||
      !!document.querySelector('[data-schema-view]'),
    componentMarker: (document.body.innerText.match(/page component reference body[^\\n]*/) || [null])[0],
    // 公开 L2 的真实挂载状态：只有 mounted + 目标容器里有子节点，才说明包内组件真的挂上了
    // （而不是「页面打开了、正文出口空着」）。view: component 页与 region widget 共用它。
    l2State: (() => { const w = document.querySelector('.sf-extension-widget'); return w ? w.getAttribute('data-l2-state') : null })(),
    l2TargetChildren: (() => { const t = document.querySelector('.sf-extension-widget [data-l2-target]'); return t ? t.children.length : -1 })(),
    hydrated: !!document.querySelector('#__nuxt')?.__vue_app__,
  }
})()`

async function main() {
  const binary = chromeBinary()
  if (!binary) {
    console.log('SKIP no Chrome/Chromium binary found')
    return finish()
  }
  asserted = true
  const { child, port, version } = await launchChrome(binary)
  const client = await connect(port)
  if (process.env.SFORUM_LOG_NETWORK) await client.send('Network.enable')
  evidence.push({ browser: version.Browser, viewport: `${WIDTH}x${HEIGHT} dpr ${DPR}` })
  try {
    for (const path of paths) {
      const expectChrome = !barePaths.has(path)
      client.consoleMessages.length = 0
      await client.send('Page.navigate', { url: `${web}${path}` })
      await new Promise(resolve => {
        const timer = setTimeout(resolve, 20000)
        const onMessage = event => {
          const message = JSON.parse(event.data)
          if (message.method === 'Page.loadEventFired') {
            clearTimeout(timer)
            client.socket.removeEventListener('message', onMessage)
            resolve()
          }
        }
        client.socket.addEventListener('message', onMessage)
      })
      // 先滚到底再测量：公开 L2 组件按可见性惰性挂载（视口外不 mount），不滚动就会量到
      // 「页面开了但组件没挂上」。滚动同时也让「无被裁切控件 / 无横向溢出」的断言覆盖全页。
      await client.send('Runtime.evaluate', {
        expression: 'window.scrollTo(0, document.documentElement.scrollHeight)',
        returnByValue: true,
      })
      // 客户端水合/挂载等待。**这段曾经是错的，且造成过假阴性。** 原先的判据是
      // 「body.innerText.length 连续两次不变」，但 SSR 正文在水合之前就已经稳定，
      // 于是判据会在 ~2s 就成立；而 Nuxt dev 模式下 entry bundle 加载 + 公开 L2 组件
      // 挂载实测要到 ~5.4s（390 宽、同一页、同一载体的真实测量）。后果是组件页被量成
      // 「组件没挂载、连描述符请求都没发出」，从而被误判为「headless 上下文不挂载」。
      // 现在分两步：(1) 等 Nuxt 真的水合（Vue 挂载后会在容器上放 __vue_app__）；
      // (2) 再等「正文长度 + L2 状态 + 挂载子节点数」的内容签名稳定。
      const settleMs = Number(process.env.SFORUM_SETTLE_MS || 20000)
      const hydrationBudgetMs = Math.floor(settleMs * 0.75)
      const settleStart = Date.now()
      const signatureExpr = `(() => {
        const w = document.querySelector('.sf-extension-widget')
        const t = w && w.querySelector('[data-l2-target]')
        return [
          document.body ? document.body.innerText.length : 0,
          w ? w.getAttribute('data-l2-state') : '-',
          t ? t.children.length : -1,
          document.querySelector('#__nuxt')?.__vue_app__ ? 1 : 0,
        ].join('|')
      })()`
      let signature = ''
      let stablePolls = 0
      let hydrated = false
      let hydratedMs = null
      let hydrationTimedOut = false
      // 期望公开 L2 挂载的路径：稳定判据必须等到该组件进入终态（mounted/fallback/quarantined），
      // 否则「水合刚完成、widget 还是 idle」也会被判成稳定，把挂载前的状态当成结论。
      const expectL2 = l2Paths.has(path)
      for (let elapsed = 0; elapsed < settleMs; elapsed += 250) {
        await new Promise(resolve => setTimeout(resolve, 250))
        const probe = await client.send('Runtime.evaluate', { expression: signatureExpr, returnByValue: true })
        const next = String(probe.result.value)
        hydrated = next.endsWith('|1')
        if (hydrated && hydratedMs === null) hydratedMs = Date.now() - settleStart
        const l2Field = next.split('|')[1]
        const l2Settled = !expectL2 || (l2Field !== '-' && l2Field !== 'idle' && l2Field !== 'loading')
        // 水合完成之前不承认「稳定」：SSR 正文此刻本来就稳定，它不代表挂载完成。
        stablePolls = hydrated && l2Settled && next === signature ? stablePolls + 1 : 0
        signature = next
        if (stablePolls >= 2) break
        if (!hydrated && Date.now() - settleStart > hydrationBudgetMs) {
          // 客户端脚本没起来：如实记录并继续测量（此时量到的是 SSR 状态，不是挂载结果）。
          hydrationTimedOut = true
          break
        }
      }
      const settleMsSpent = Date.now() - settleStart
      const result = await client.send('Runtime.evaluate', { expression: measure, returnByValue: true, awaitPromise: true })
      const measurement = result.result.value
      const consoleErrors = client.consoleMessages.filter(message => message.type === 'error' || message.type === 'exception')
      const entry = {
        ...measurement,
        expectChrome,
        consoleErrors,
        expectL2: l2Paths.has(path),
        settleMsSpent,
        hydratedMs,
        hydrationTimedOut,
      }
      if (process.env.SFORUM_LOG_NETWORK) entry.network = [...client.networkLog]
      if (qaDir) {
        mkdirSync(qaDir, { recursive: true })
        const shot = await client.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false })
        const file = join(qaDir, `2026-10-16-public-390x844${path.replaceAll('/', '-')}.png`)
        writeFileSync(file, Buffer.from(shot.data, 'base64'))
        entry.screenshot = file
      }
      evidence.push(entry)

      if (measurement.innerWidth !== WIDTH) fail(`${path}: viewport override did not apply`, `innerWidth=${measurement.innerWidth}`)
      if (measurement.horizontalOverflow) fail(`${path}: horizontal overflow`, `scrollWidth=${measurement.scrollWidth}`)
      if (measurement.clippedControls !== 0) fail(`${path}: clipped controls`, JSON.stringify(measurement.clippedSample))
      if (consoleErrors.length > 0) fail(`${path}: console errors`, JSON.stringify(consoleErrors.slice(0, 3)))
      if (expectChrome) {
        if (measurement.visibleHeaders !== 1) fail(`${path}: expected exactly one visible header`, `got ${measurement.visibleHeaders}`)
        if (measurement.visibleFooters !== 1) fail(`${path}: expected exactly one visible footer`, `got ${measurement.visibleFooters}`)
      } else {
        if (measurement.visibleHeaders !== 0 || measurement.visibleFooters !== 0) {
          fail(`${path}: bare page must have no chrome`, `headers=${measurement.visibleHeaders} footers=${measurement.visibleFooters}`)
        }
      }
      if (l2Paths.has(path)) {
        // 公开 L2 挂载证据。水合没发生就明确失败（那说明这次测量根本没有量到客户端结果），
        // 而不是把「没量到」写成「通过了」。
        if (hydrationTimedOut || !measurement.hydrated) {
          fail(`${path}: expected a mounted public L2 component but the client never hydrated`,
            `hydrated=${measurement.hydrated} hydratedMs=${hydratedMs} settleMsSpent=${settleMsSpent}`)
        } else if (measurement.l2State !== 'mounted') {
          fail(`${path}: expected public L2 state "mounted"`, `l2State=${measurement.l2State}`)
        } else if (!(measurement.l2TargetChildren > 0)) {
          fail(`${path}: public L2 reported mounted but rendered no child node`, `l2TargetChildren=${measurement.l2TargetChildren}`)
        } else if (!measurement.htmlHasPluginText && !measurement.componentMarker) {
          fail(`${path}: public L2 mounted but its own content is absent`,
            `htmlHasPluginText=${measurement.htmlHasPluginText} componentMarker=${JSON.stringify(measurement.componentMarker)}`)
        }
      }
    }
  } finally {
    client.socket.close()
    child.kill('SIGKILL')
  }
  finish()
}

function finish() {
  if (outputPath) writeFileSync(outputPath, `${JSON.stringify({ web, paths, failures, evidence }, null, 2)}\n`, 'utf8')
  console.log(`public 390x844 browser evidence (web=${web})`)
  for (const entry of evidence) {
    if (entry.browser) {
      console.log(`  browser: ${entry.browser} viewport=${entry.viewport}`)
      continue
    }
    console.log(
      `  ${entry.url}: ${entry.innerWidth}x${entry.innerHeight} dpr=${entry.dpr} shell=${entry.shell}/${entry.shellSource} ` +
        `headers=${entry.visibleHeaders} footers=${entry.visibleFooters} overflow=${entry.horizontalOverflow} ` +
        `clipped=${entry.clippedControls} consoleErrors=${entry.consoleErrors.length} ` +
        `hydrated=${entry.hydrated}@${entry.hydratedMs}ms settle=${entry.settleMsSpent}ms ` +
        `l2=${entry.l2State}/kids=${entry.l2TargetChildren} trustNotice=${entry.trustNoticeRendered} ` +
        `marker=${JSON.stringify(entry.componentMarker)} ` +
        `mainTextLen=${entry.mainTextLength} islands=${entry.islandCount} outlet=${entry.bodyOutletPresent} ` +
        `text=${JSON.stringify(entry.bodyText.slice(0, 60))}`,
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
  console.log('  all public 390x844 acceptance assertions passed')
}

try {
  await main()
} catch (error) {
  fail('harness error', String(error?.message || error))
  finish()
}
