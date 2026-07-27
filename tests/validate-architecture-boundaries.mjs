import {
  existsSync,
  readFileSync,
  readdirSync
} from 'node:fs'
import { extname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(fileURLToPath(new URL('..', import.meta.url)))
const baselinePath = resolve(root, 'tests/architecture-boundaries-baseline.json')
const baseline = JSON.parse(readFileSync(baselinePath, 'utf8'))
const failures = []

function repoPath(path) {
  return resolve(root, path)
}

function normalizePath(path) {
  return path.replaceAll('\\', '/')
}

function read(path) {
  return readFileSync(repoPath(path), 'utf8')
}

function lineCount(content) {
  if (!content) return 0
  const normalized = content.replaceAll('\r\n', '\n')
  return normalized.split('\n').length - (normalized.endsWith('\n') ? 1 : 0)
}

function walkFiles(directory) {
  const absolute = repoPath(directory)
  if (!existsSync(absolute)) return []

  const files = []
  for (const entry of readdirSync(absolute, { withFileTypes: true })) {
    const child = join(absolute, entry.name)
    if (entry.isDirectory()) {
      files.push(...walkFiles(normalizePath(relative(root, child))))
    } else if (entry.isFile()) {
      files.push(normalizePath(relative(root, child)))
    }
  }
  return files
}

function directFiles(directory) {
  const absolute = repoPath(directory)
  if (!existsSync(absolute)) return []
  return readdirSync(absolute, { withFileTypes: true })
    .filter(entry => entry.isFile())
    .map(entry => normalizePath(join(directory, entry.name)))
}

function isGenerated(path, content) {
  return path.includes('/gen/') ||
    path.includes('/database/sqlc/') ||
    path.endsWith('.pb.go') ||
    content.slice(0, 512).includes('Code generated')
}

function shouldScanProductionSource(path) {
  if (path.startsWith('apps/api/')) {
    return path.endsWith('.go') &&
      !path.endsWith('_test.go') &&
      !path.includes('/testdata/')
  }
  if (path.startsWith('apps/web/app/')) {
    return path.endsWith('.vue') || path.endsWith('.ts')
  }
  return false
}

const productionFiles = [
  ...walkFiles('apps/api'),
  ...walkFiles('apps/web/app')
].filter(shouldScanProductionSource)

let reviewSizeFileCount = 0
for (const path of productionFiles) {
  const content = read(path)
  if (isGenerated(path, content)) continue

  const lines = lineCount(content)
  const legacyCap = baseline.legacyLargeFiles[path]
  if (lines > 500) reviewSizeFileCount += 1

  if (legacyCap !== undefined) {
    if (lines > legacyCap) {
      failures.push(
        `${path} grew from its legacy cap ${legacyCap} to ${lines} lines; split responsibilities instead`
      )
    } else if (lines < legacyCap) {
      failures.push(
        `${path} shrank from ${legacyCap} to ${lines} lines; lower or remove its legacy baseline in the same change`
      )
    }
    continue
  }

  if (lines > 1000) {
    failures.push(
      `${path} has ${lines} lines; new handwritten production files must stay at or below 1000`
    )
  }
}

for (const rule of baseline.flatDirectoryCaps) {
  const matching = directFiles(rule.path).filter((path) => {
    if (!rule.extensions.includes(extname(path))) return false
    return !(rule.excludeSuffixes || []).some(suffix => path.endsWith(suffix))
  })
  if (matching.length > rule.maxFiles) {
    failures.push(
      `${rule.path} has ${matching.length} direct files (cap ${rule.maxFiles}); place new product code in a domain subdirectory/package`
    )
  } else if (matching.length < rule.maxFiles) {
    failures.push(
      `${rule.path} dropped to ${matching.length} direct files; lower its flat-directory baseline from ${rule.maxFiles}`
    )
  }
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

for (const rule of baseline.receiverMethodCaps) {
  const receiver = escapeRegExp(rule.type)
  const methodPattern = new RegExp(
    `^func\\s+\\(\\s*[A-Za-z_][A-Za-z0-9_]*\\s+\\*?${receiver}\\s*\\)`,
    'gm'
  )
  let methods = 0
  for (const path of directFiles(rule.path)) {
    if (!path.endsWith('.go') || path.endsWith('_test.go')) continue
    methods += read(path).match(methodPattern)?.length || 0
  }
  if (methods > rule.maxMethods) {
    failures.push(
      `${rule.path} ${rule.type} has ${methods} receiver methods (cap ${rule.maxMethods}); add a focused collaborator instead`
    )
  } else if (methods < rule.maxMethods) {
    failures.push(
      `${rule.path} ${rule.type} dropped to ${methods} receiver methods; lower its baseline from ${rule.maxMethods}`
    )
  }
}

function literalTabBranches(content) {
  const values = new Set()
  const pattern = /\b(?:activeTab|activeView|section)\s*===\s*['"]([A-Za-z0-9_-]+)['"]/g
  for (const match of content.matchAll(pattern)) values.add(match[1])
  return values
}

for (const [path, rule] of Object.entries(baseline.legacyInlineAdminTabPages)) {
  if (!existsSync(repoPath(path))) continue
  const content = read(path)
  const lines = lineCount(content)
  const branches = literalTabBranches(content).size
  if (lines > rule.maxLines) {
    failures.push(
      `${path} is a legacy inline-tab page and grew from ${rule.maxLines} to ${lines} lines; extract a tab component`
    )
  } else if (lines < rule.maxLines) {
    failures.push(
      `${path} shrank from ${rule.maxLines} to ${lines} lines; lower or remove its inline-tab baseline`
    )
  }
  if (branches > rule.maxLiteralBranches) {
    failures.push(
      `${path} added an inline tab branch (${branches}, cap ${rule.maxLiteralBranches}); fixed Core tabs require separate files`
    )
  } else if (branches < rule.maxLiteralBranches) {
    failures.push(
      `${path} reduced inline tab branches to ${branches}; lower or remove its branch baseline from ${rule.maxLiteralBranches}`
    )
  }
}

const legacyInlineTabPages = new Set(Object.keys(baseline.legacyInlineAdminTabPages))
for (const path of walkFiles('apps/web/app/pages/admin').filter(path => path.endsWith('.vue'))) {
  if (legacyInlineTabPages.has(path)) continue
  const content = read(path)
  if (!content.includes('role="tablist"')) continue

  const branches = literalTabBranches(content)
  const importsTabComponents = /from\s+['"][^'"]+\/tabs\/[^'"]+\.vue['"]/.test(content)
  if (branches.size >= 2 && !importsTabComponents) {
    failures.push(
      `${path} implements ${branches.size} fixed tab branches inline; move substantial tabs under a domain /tabs/ component directory`
    )
  }
}

if (failures.length > 0) {
  console.error('Architecture boundary validation failed:')
  for (const failure of failures) console.error(`- ${failure}`)
  process.exit(1)
}

console.log(
  `Architecture boundary validation passed (${productionFiles.length} production files scanned, ${reviewSizeFileCount} files remain above the 500-line review threshold).`
)
