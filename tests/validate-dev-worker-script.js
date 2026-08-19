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

assert(fs.existsSync(workerScriptPath), 'scripts/worker-dev.sh must preserve a clear compatibility error')

const workerScript = fs.readFileSync(workerScriptPath, 'utf8')
assert(workerScript.includes('Standalone Worker is no longer supported'), 'worker-dev.sh must reject the retired standalone topology')
assert(workerScript.includes('./scripts/api-dev.sh'), 'worker-dev.sh must direct operators to the API-owned worker')
assert(workerScript.includes('exit 1'), 'worker-dev.sh must fail instead of starting a duplicate consumer')

const readme = fs.readFileSync(readmePath, 'utf8')
assert(!readme.includes('EMBED_WORKER_IN_API'), 'README must not document a retired worker ownership switch')
assert(!readme.includes('split-worker'), 'README must not document a retired standalone Worker profile')
assert(readme.includes('API process always embeds'), 'README must document API ownership of background jobs')

console.log('Development worker script validation passed.')
