// plain nuxt 入口：原始 nuxt dev + Web Release 激活确认旁路。
// 不注入主题 layer、不因 current.json 重启；仅写 active.json 让插件启停能完成。
import { spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { resolvePlainReleaseRoot, startPlainReleaseAck } from './dev-plain-release-ack.mjs'

const here = path.dirname(fileURLToPath(import.meta.url))
const webRoot = path.resolve(here, '..')
const bunPath = process.env.SFORUM_BUN_PATH || 'bun'
const releaseRoot = resolvePlainReleaseRoot({ cwd: webRoot })

const ack = startPlainReleaseAck({ releaseRoot })

const child = spawn(
  bunPath,
  ['x', 'nuxt', 'dev', '--host', '0.0.0.0', '--dotenv', '../../.env'],
  {
    cwd: webRoot,
    stdio: 'inherit',
    env: process.env,
  },
)

let exiting = false

function shutdown(code = 0) {
  if (exiting) return
  exiting = true
  ack.close()
  if (child.exitCode === null && child.signalCode === null) {
    child.kill('SIGTERM')
  }
  // 给 nuxt 一点时间退出；超时再强杀。
  const force = setTimeout(() => {
    if (child.exitCode === null && child.signalCode === null) {
      child.kill('SIGKILL')
    }
    process.exit(code)
  }, 5000)
  force.unref?.()
  child.once('exit', (exitCode, signal) => {
    clearTimeout(force)
    process.exit(exitCode ?? (signal ? 1 : code))
  })
}

process.on('SIGINT', () => shutdown(0))
process.on('SIGTERM', () => shutdown(0))

child.on('error', (err) => {
  console.error('[sforum-dev-plain] failed to start nuxt:', err.message)
  ack.close()
  process.exit(1)
})

child.on('exit', (code, signal) => {
  if (exiting) return
  ack.close()
  process.exit(code ?? (signal ? 1 : 0))
})

console.log(`[sforum-dev-plain] web release ack watching ${releaseRoot}`)
console.log('[sforum-dev-plain] theme layers will NOT auto-switch; use `bun run dev` for full theme supervisor')
