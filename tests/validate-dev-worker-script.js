const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const workerScriptPath = path.join(root, 'scripts/worker-dev.sh')
const readmePath = path.join(root, 'README.md')

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

assert(fs.existsSync(workerScriptPath), 'scripts/worker-dev.sh must exist for local background jobs')

const workerScript = fs.readFileSync(workerScriptPath, 'utf8')
assert(workerScript.includes('.air.worker.toml'), 'worker-dev.sh must run the worker Air config')
assert(workerScript.includes('cmd/worker') || workerScript.includes('sforum-worker'), 'worker-dev.sh should clearly target the worker process')
assert(workerScript.includes('THEME_WEB_ROOT'), 'worker-dev.sh must set or preserve THEME_WEB_ROOT for local theme builds')

const readme = fs.readFileSync(readmePath, 'utf8')
assert(readme.includes('EMBED_WORKER_IN_API'), 'README must document the embedded worker switch')
assert(readme.includes('./scripts/worker-dev.sh'), 'README must document the optional local worker command')
assert(readme.includes('split-process development') || readme.includes('拆分进程开发'), 'README should explain that worker-dev is for split-process development')

console.log('Development worker script validation passed.')
