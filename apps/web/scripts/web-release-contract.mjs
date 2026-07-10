import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

const SCHEMA_VERSION = 1

export function readDesiredRelease({ releaseRoot, legacyRoot = releaseRoot, fallback = {} }) {
  const currentPath = findCurrentFile(releaseRoot, legacyRoot)
  if (!currentPath) return fallbackSelection(fallback)

  let current
  try {
    current = JSON.parse(fs.readFileSync(currentPath, 'utf8'))
  } catch {
    return fallbackSelection(fallback)
  }
  if (current?.schemaVersion === SCHEMA_VERSION) {
    return {
      kind: 'release',
      schemaVersion: SCHEMA_VERSION,
      releaseId: positiveInteger(current.releaseId),
      compositionHash: requiredString(current.compositionHash, 'compositionHash'),
      artifactPath: resolveReleasePath(releaseRoot, current.artifactPath),
      artifactDigest: requiredString(current.artifactDigest, 'artifactDigest'),
      serverEntry: resolveReleasePath(releaseRoot, current.serverEntry),
      themeId: requiredString(current.themeId, 'themeId'),
      themeVersion: requiredString(current.themeVersion, 'themeVersion'),
      reloadMode: current.reloadMode === 'force' ? 'force' : 'prompt',
      requestedAt: typeof current.requestedAt === 'string' ? current.requestedAt : ''
    }
  }
  if (current?.mode === 'default') return fallbackSelection(fallback)
  const legacyServer = typeof current?.server === 'string' ? current.server.trim() : ''
  const legacyLayer = typeof current?.layerPath === 'string' ? current.layerPath.trim() : ''
  if (legacyServer || legacyLayer) {
    return {
      kind: 'legacy',
      serverEntry: legacyServer ? resolveReleasePath(legacyRoot, legacyServer) : '',
      themeLayer: resolveOptionalPath(legacyRoot, legacyLayer),
      themeId: typeof current.extensionId === 'string' ? current.extensionId : '',
      reloadMode: 'prompt'
    }
  }
  return fallbackSelection(fallback)
}

export async function verifyReleaseArtifact(selection) {
  if (selection.kind !== 'release') return selection
  const verified = readReleaseManifest(selection)
  const { manifestPath, ...manifest } = verified

  const relativeServer = path.relative(selection.artifactPath, selection.serverEntry)
  if (relativeServer === '..' || relativeServer.startsWith(`..${path.sep}`) || path.isAbsolute(relativeServer)) {
    throw new Error('web release server entry escapes artifact')
  }
  const server = await fs.promises.stat(selection.serverEntry)
  if (!server.isFile()) throw new Error('web release server entry is not a regular file')
  const digest = await digestArtifactTree(selection.artifactPath)
  if (digest !== selection.artifactDigest) throw new Error('web release artifact digest does not match')
  return { ...selection, ...manifest, manifestPath }
}

export function readReleaseManifest(selection) {
  if (selection.kind !== 'release') return selection
  const manifestPath = path.join(path.dirname(selection.artifactPath), 'release.json')
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
  for (const field of ['releaseId', 'compositionHash', 'artifactPath', 'artifactDigest', 'serverEntry', 'themeId', 'themeVersion', 'reloadMode']) {
    if (normalizedField(field, manifest[field]) !== normalizedField(field, selection[field])) {
      throw new Error(`web release manifest ${field} does not match desired pointer`)
    }
  }
  return {
    ...selection,
    themeLayer: resolveOptionalPath(path.dirname(manifestPath), manifest.themeLayer),
    devInput: resolveOptionalPath(path.dirname(manifestPath), manifest.devInput),
    registryRoot: resolveOptionalPath(path.dirname(manifestPath), manifest.registryRoot),
    manifestPath
  }
}

export async function digestArtifactTree(root) {
  const rootStat = await fs.promises.lstat(root)
  if (!rootStat.isDirectory() || rootStat.isSymbolicLink()) throw new Error('artifact root must be a regular directory')
  const files = []
  await collectFiles(root, root, files)
  files.sort((left, right) => Buffer.compare(Buffer.from(left.relative), Buffer.from(right.relative)))
  const hash = crypto.createHash('sha256')
  for (const file of files) {
    const stat = await fs.promises.lstat(file.absolute)
    if (!stat.isFile() || stat.isSymbolicLink()) throw new Error(`artifact contains non-regular file: ${file.relative}`)
    hash.update(file.relative)
    hash.update('\0')
    hash.update((stat.mode & 0o777).toString(8))
    hash.update('\0')
    hash.update(String(stat.size))
    hash.update('\0')
    hash.update(await fs.promises.readFile(file.absolute))
  }
  return hash.digest('hex')
}

export async function writeActiveAcknowledgement(releaseRoot, active) {
  await writeJSONAtomic(path.join(releaseRoot, 'active.json'), {
    ...active,
    switchedAt: active.switchedAt || new Date().toISOString()
  })
}

export async function writeFailureAcknowledgement(releaseRoot, failure) {
  const releaseId = positiveInteger(failure.releaseId)
  await writeJSONAtomic(path.join(releaseRoot, 'failures', `${releaseId}.json`), {
    releaseId,
    reason: requiredString(failure.reason, 'reason'),
    message: String(failure.message || '').replace(/\s+/g, ' ').trim().slice(0, 2000),
    failedAt: failure.failedAt || new Date().toISOString()
  })
}

export function watchableReleaseFile(filename) {
  return filename === 'current.json' || filename === 'current.json.tmp'
}

async function collectFiles(root, current, files) {
  const entries = await fs.promises.readdir(current, { withFileTypes: true })
  for (const entry of entries) {
    const absolute = path.join(current, entry.name)
    const relative = path.relative(root, absolute).split(path.sep).join('/')
    if (entry.isSymbolicLink()) throw new Error(`artifact contains symbolic link: ${relative}`)
    if (entry.isDirectory()) await collectFiles(root, absolute, files)
    else if (entry.isFile()) files.push({ absolute, relative })
    else throw new Error(`artifact contains non-regular file: ${relative}`)
  }
}

async function writeJSONAtomic(target, value) {
  await fs.promises.mkdir(path.dirname(target), { recursive: true })
  const temporary = `${target}.tmp`
  const handle = await fs.promises.open(temporary, 'w', 0o644)
  try {
    await handle.writeFile(`${JSON.stringify(value, null, 2)}\n`)
    await handle.sync()
  } finally {
    await handle.close()
  }
  await fs.promises.rename(temporary, target)
  const directory = await fs.promises.open(path.dirname(target), 'r')
  try { await directory.sync() } finally { await directory.close() }
}

function findCurrentFile(releaseRoot, legacyRoot) {
  for (const candidate of [...new Set([releaseRoot, legacyRoot])]) {
    const current = path.join(candidate, 'current.json')
    if (fs.existsSync(current)) return current
  }
  return ''
}

function fallbackSelection(fallback) {
  return {
    kind: 'fallback',
    serverEntry: requiredString(fallback.serverEntry, 'fallback serverEntry'),
    themeLayer: typeof fallback.themeLayer === 'string' ? fallback.themeLayer : '',
    themeId: typeof fallback.themeId === 'string' ? fallback.themeId : 'sforum.default-theme',
    reloadMode: 'prompt'
  }
}

function resolveReleasePath(root, value) {
  const clean = requiredString(value, 'release path')
  return path.isAbsolute(clean) ? path.normalize(clean) : path.resolve(root, clean)
}

function resolveOptionalPath(root, value) {
  if (typeof value !== 'string' || !value.trim()) return ''
  return path.isAbsolute(value) ? path.normalize(value) : path.resolve(root, value)
}

function normalizedField(field, value) {
  if (field === 'artifactPath' || field === 'serverEntry') return path.resolve(String(value || ''))
  return String(value ?? '')
}

function requiredString(value, name) {
  if (typeof value !== 'string' || !value.trim()) throw new Error(`web release ${name} is required`)
  return value.trim()
}

function positiveInteger(value) {
  const number = Number(value)
  if (!Number.isSafeInteger(number) || number <= 0) throw new Error('web release id must be a positive integer')
  return number
}
