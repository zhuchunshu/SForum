import { spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const releaseRoot = process.env.SFORUM_THEME_RELEASE_ROOT || '/var/lib/sforum/theme-releases'
const fallbackOutput = process.env.SFORUM_FALLBACK_OUTPUT || path.resolve(process.cwd(), '.output')
const currentFile = path.join(releaseRoot, 'current.json')

let child = null
let activeServer = ''
let restartTimer = null

function readCurrentServer() {
  try {
    const raw = fs.readFileSync(currentFile, 'utf8')
    const current = JSON.parse(raw)
    if (typeof current.server === 'string' && current.server.trim()) {
      return current.server
    }
  } catch (error) {
    if (error.code !== 'ENOENT') {
      console.error('[sforum-web-runtime] invalid current release:', error.message)
    }
  }
  return path.join(fallbackOutput, 'server/index.mjs')
}

function stopChild() {
  if (!child) {
    return
  }
  child.kill('SIGTERM')
  child = null
}

function startCurrent() {
  const server = readCurrentServer()
  if (server === activeServer && child) {
    return
  }
  if (!fs.existsSync(server)) {
    console.error(`[sforum-web-runtime] selected server does not exist: ${server}`)
    return
  }
  stopChild()
  activeServer = server
  child = spawn(process.execPath, [server], {
    stdio: 'inherit',
    env: {
      ...process.env,
      HOST: process.env.HOST || '0.0.0.0',
      PORT: process.env.PORT || '3000'
    }
  })
  child.on('exit', (code, signal) => {
    console.error(`[sforum-web-runtime] child exited code=${code ?? ''} signal=${signal ?? ''}`)
    child = null
    setTimeout(startCurrent, 1000)
  })
}

function scheduleRestart() {
  clearTimeout(restartTimer)
  restartTimer = setTimeout(startCurrent, 250)
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

startCurrent()
