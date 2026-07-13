const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8')
}

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function section(source, start, end) {
  const startIndex = source.indexOf(start)
  const endIndex = source.indexOf(end, startIndex + start.length)
  assert(startIndex >= 0 && endIndex > startIndex, `missing contract section ${start}`)
  return source.slice(startIndex, endIndex)
}

const schemas = read('contracts/openapi/schemas/extensions.yaml')
const versionSchema = section(schemas, 'ExtensionVersion:\n', 'Extension:\n')
for (const field of ['version', 'manifest', 'packageDigest', 'adminFrontendDigest', 'packagePath', 'installedAt']) {
  assert(versionSchema.includes(`  - ${field}\n`), `ExtensionVersion must require ${field}`)
  assert(versionSchema.includes(`    ${field}:\n`), `ExtensionVersion must expose ${field}`)
}
assert(!versionSchema.includes('    id:\n'), 'ExtensionVersion must not expose its database id')

const extensionSchema = section(schemas, '\nExtension:\n', '\nExtensionInstallResult:\n')
assert(extensionSchema.includes('    stagedVersion:\n      "$ref": "#/ExtensionVersion"'), 'Extension must expose stagedVersion through ExtensionVersion')

const installResultSchema = section(schemas, '\nExtensionInstallResult:\n', '\nExtensionMigrationRecord:\n')
assert(installResultSchema.includes('    activationPending:\n'), 'InstallResult must expose activationPending')
assert(installResultSchema.includes('Static package\n        upload and staging preserve grants'), 'trustRevoked must preserve active grants during static staging')
assert(installResultSchema.includes('Static package upload and staging preserve active status'), 'requiredReEnable must preserve active status during static staging')

const paths = read('contracts/openapi/paths/extensions.yaml')
assert(paths.includes('stores an immutable upgrade candidate without changing the active package'), 'upload contract must describe inert candidate staging')
assert(paths.includes('InstallResult (extension plus activationPending metadata)'), 'upload response must describe activationPending')

const types = read('apps/web/app/utils/adminExtensions.ts')
assert(types.includes('export type AdminExtensionVersion = {'), 'web types must define AdminExtensionVersion')
assert(types.includes('stagedVersion?: AdminExtensionVersion'), 'AdminExtension must expose stagedVersion')

const manager = read('apps/web/app/composables/useAdminExtensionsManager.ts')
assert(manager.includes('activationPending?: boolean'), 'upload result type must expose activationPending')
assert(manager.includes("t('admin.extensions.upgradeStagedHint')"), 'staged upload toast must explain pending activation')

const overview = read('apps/web/app/pages/admin/extensions/index.vue')
assert(overview.includes("t('admin.extensions.stagedVersionBadge'"), 'extension list must identify a staged candidate')
assert(overview.includes("t('admin.extensions.stagedVersion')"), 'extension details must identify the staged version')

for (const localeName of ['zh-CN', 'en-US']) {
  const locale = JSON.parse(read(`apps/web/i18n/locales/${localeName}.json`))
  const messages = locale.admin?.extensions
  assert(messages?.upgradeStaged && messages?.upgradeStagedHint, `${localeName} must describe the staged upload result`)
  assert(messages?.stagedVersion && messages?.stagedVersionBadge, `${localeName} must label the staged version`)
}

console.log('Staged extension management contract validation passed.')
