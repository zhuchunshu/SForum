// Guards the runtime timezone invariant: the shipped API/worker image has no
// /usr/share/zoneinfo, so non-UTC zones only resolve when the binary embeds
// time/tzdata (or the image installs tzdata). Losing this breaks saving
// "Site Settings -> timezone" and rejects plugin schedule timezones with
// HTTP 422 options.invalid.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const apiDir = path.join(root, 'apps/api')
const dockerfilePath = path.join(apiDir, 'Dockerfile')
const timezonePkgDir = path.join(apiDir, 'app/Support/Timezone')
const timezoneSourcePath = path.join(timezonePkgDir, 'timezone.go')

function fail(message) {
  console.error(`[validate-runtime-tzdata] ${message}`)
  process.exit(1)
}

function walk(dir, onFile) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(full, onFile)
      continue
    }
    if (entry.isFile()) onFile(full)
  }
}

if (!fs.existsSync(timezoneSourcePath)) {
  fail('apps/api/app/Support/Timezone/timezone.go is missing; host timezone resolution must stay centralized')
}

const dockerfile = fs.readFileSync(dockerfilePath, 'utf8')
const apiStageStart = dockerfile.search(/^FROM\s+\S+\s+AS\s+api\s*$/m)
if (apiStageStart < 0) {
  fail('apps/api/Dockerfile must declare a final "AS api" stage')
}
const apiStage = dockerfile.slice(apiStageStart)
const imageInstallsTzdata = /apk add[^\n]*\btzdata\b/.test(apiStage)

const timezoneSource = fs.readFileSync(timezoneSourcePath, 'utf8')
const embedsTzdata = /^\s*_\s+"time\/tzdata"\s*$/m.test(timezoneSource)

if (!imageInstallsTzdata && !embedsTzdata) {
  fail('runtime image installs no tzdata and app/Support/Timezone does not import _ "time/tzdata"; non-UTC timezones will fail to save in containers')
}

const rawLoadLocation = []
walk(apiDir, (file) => {
  if (!file.endsWith('.go')) return
  if (file.startsWith(timezonePkgDir + path.sep)) return
  const source = fs.readFileSync(file, 'utf8')
  if (/time\.LoadLocation\(/.test(source)) {
    rawLoadLocation.push(path.relative(root, file))
  }
})
if (rawLoadLocation.length > 0) {
  fail(`use timezone.Load instead of time.LoadLocation so the embedded database is linked: ${rawLoadLocation.join(', ')}`)
}

console.log('Runtime timezone validation passed.')
