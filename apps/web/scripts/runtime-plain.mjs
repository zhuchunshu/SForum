// 生产入口：直接启动 Nitro，不再根据主题 release 指针切换主题层。
// 公开主题通过 Page Registry + L0 CSS 注入；Web Release 仅服务可信管理端插件前端。
import { spawn } from 'node:child_process'
import path from 'node:path'
import fs from 'node:fs'

const port = process.env.PORT || '3000'
const host = process.env.HOST || '0.0.0.0'
const serverEntry = process.env.SFORUM_NITRO_SERVER
  || path.resolve(process.cwd(), '.output/server/index.mjs')

if (!fs.existsSync(serverEntry)) {
  console.error('[sforum-web] Nitro server entry missing:', serverEntry)
  process.exit(1)
}

console.log(`[sforum-web] starting Nitro ${serverEntry} on ${host}:${port}`)
const child = spawn(process.execPath, [serverEntry], {
  stdio: 'inherit',
  env: {
    ...process.env,
    PORT: String(port),
    HOST: host,
    NITRO_PORT: String(port),
    NITRO_HOST: host,
  },
})

function shutdown(signal) {
  if (!child.killed) child.kill(signal)
}
process.on('SIGTERM', () => shutdown('SIGTERM'))
process.on('SIGINT', () => shutdown('SIGINT'))
child.on('exit', (code, signal) => {
  if (signal) process.kill(process.pid, signal)
  process.exit(code ?? 1)
})
