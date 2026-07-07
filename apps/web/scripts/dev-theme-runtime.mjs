// 本地开发的主题感知 supervisor：包裹 `bun run dev:plain`（原始 nuxt dev）。
//
// 工作方式：
// - 启动时读 theme-releases/current.json，决定初始的 SFORUM_THEME_LAYER。
//   mode==='default' 或无文件：不设环境变量，nuxt 用默认主题 layer。
//   layerPath 非空：把它作为 SFORUM_THEME_LAYER 注入，nuxt 把上传主题作为优先 layer。
// - fs.watch 监听 releaseRoot，current.json 变化后 250ms 防抖重启 dev:plain。
// - 只管自己 spawn 的子进程，不碰端口 3000 上的其它进程。
//
// 与生产 runtime.mjs 的区别：生产切换已构建好的 Nitro server 产物；
// 开发没有产物，只能重启 nuxt dev 让它重新应用新的 layer。
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || path.resolve(process.cwd(), '../../storage/theme-releases')
const currentFile = path.join(releaseRoot, 'current.json')
const bunPath = process.env.SFORUM_BUN_PATH || 'bun'

let child = null
let activeLayer = null
let restartTimer = null

// 读取当前应使用的 layer 路径。返回 null 表示用默认主题。
// layerPath 可能是绝对路径（新版 builder）或相对路径（历史/手写），
// 相对路径按仓库根（apps/web 的上两级）解析。
function readLayerPath() {
  let raw
  try {
    raw = fs.readFileSync(currentFile, 'utf8')
  } catch (error) {
    if (error.code !== 'ENOENT') {
      console.error('[sforum-dev-runtime] invalid current release:', error.message)
    }
    return null
  }
  let current
  try {
    current = JSON.parse(raw)
  } catch (error) {
    console.error('[sforum-dev-runtime] invalid current release:', error.message)
    return null
  }
  if (current.mode === 'default') {
    return null
  }
  const layerPath = typeof current.layerPath === 'string' ? current.layerPath.trim() : ''
  if (!layerPath) {
    return null
  }
  if (path.isAbsolute(layerPath)) {
    return layerPath
  }
  // 仓库根 = apps/web 的父目录的父目录。
  return path.resolve(process.cwd(), '../../', layerPath)
}

function stopChild() {
  if (!child) {
    return null
  }
  const pid = child.pid
  child = null
  if (!pid) {
    return null
  }
  try {
    // 用进程组信号（负 PID）杀掉 bun 以及它派生的 nuxt dev 子进程。
    // 仅 kill bun 会让真正占用端口的 node 进程变成孤儿继续服务，
    // 导致重启后新进程抢不到端口、前台停留在旧主题。
    process.kill(-pid, 'SIGTERM')
  } catch (error) {
    if (error.code !== 'ESRCH') {
      console.error('[sforum-dev-runtime] failed to signal child group:', error.message)
    }
  }
  // 返回旧进程 pid，供调用方等待端口释放。
  return pid
}

// 等待旧 dev 进程组退出并释放端口。bun→node 两层进程 SIGTERM 后
// 端口真正释放有延迟，直接 spawn 会让新进程抢不到端口。
function waitForChildExit(pid, done) {
  if (!pid) {
    done()
    return
  }
  const deadline = Date.now() + 5000
  const tick = () => {
    // 进程组里只要还有进程存活，kill(-pid,0) 就不抛 ESRCH。
    try {
      process.kill(-pid, 0)
      if (Date.now() > deadline) {
        // 超时则升级为 SIGKILL，避免无限等待。
        try {
          process.kill(-pid, 'SIGKILL')
        } catch (_) {
          // 已经退出
        }
        setTimeout(done, 200)
        return
      }
      setTimeout(tick, 100)
    } catch (error) {
      if (error.code === 'ESRCH') {
        // 进程组已退出，再给端口一点释放时间。
        setTimeout(done, 200)
        return
      }
      done()
    }
  }
  tick()
}

function startDev(reason = 'startup') {
  const nextLayer = readLayerPath()
  if (nextLayer === activeLayer && child) {
    return
  }
  if (nextLayer && !fs.existsSync(nextLayer)) {
    // layer 目录不存在时不要盲目重启，保留当前 dev 继续可用。
    console.error(`[sforum-dev-runtime] theme layer does not exist: ${nextLayer}`)
    if (child) {
      return
    }
  }
  const previousPid = stopChild()
  const spawnChild = () => {
    activeLayer = nextLayer
    const env = { ...process.env }
    if (nextLayer && fs.existsSync(nextLayer)) {
      env.SFORUM_THEME_LAYER = nextLayer
      console.log(`[sforum-dev-runtime] (${reason}) starting Nuxt dev with theme layer: ${nextLayer}`)
    } else {
      delete env.SFORUM_THEME_LAYER
      console.log(`[sforum-dev-runtime] (${reason}) starting Nuxt dev with default theme`)
    }
    child = spawn(bunPath, ['run', 'dev:plain'], {
      stdio: 'inherit',
      env,
      // detached 让子进程成为独立进程组组长，便于用 -pid 杀掉整个进程组。
      detached: true
    })
    child.on('exit', (code, signal) => {
      console.error(`[sforum-dev-runtime] nuxt dev exited code=${code ?? ''} signal=${signal ?? ''}`)
      child = null
      // 非主动重启时，1s 后尝试拉起，避免 dev 崩溃后整个 supervisor 退出。
      if (code !== 0 && code !== null) {
        setTimeout(() => startDev('nuxt-dev-restart'), 1000)
      }
    })
  }
  // 等旧进程组退出、端口释放后再 spawn 新的 dev，避免端口冲突。
  waitForChildExit(previousPid, spawnChild)
}

function scheduleRestart() {
  clearTimeout(restartTimer)
  restartTimer = setTimeout(() => startDev('current.json changed'), 250)
}

fs.mkdirSync(releaseRoot, { recursive: true })
fs.watch(releaseRoot, scheduleRestart)
process.on('SIGTERM', () => {
  stopChild()
  process.exit(0)
})
process.on('SIGINT', () => {
  stopChild()
  process.exit(0)
})

startDev('startup')
