import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

export const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
export const npmRegistry = 'https://registry.npmjs.org/'

const sdkDescriptors = [
  {
    name: '@sforum/admin-sdk',
    directory: 'apps/web/packages/admin-sdk',
    requiredFiles: ['package/LICENSE', 'package/README.md', 'package/package.json', 'package/src/index.ts']
  },
  {
    name: '@sforum/plugin-ui',
    directory: 'apps/web/packages/plugin-ui',
    requiredFiles: [
      'package/LICENSE',
      'package/README.md',
      'package/package.json',
      'package/src/index.ts',
      'package/src/style.css',
      'package/src/components/layout/SPluginPage.vue'
    ]
  }
]

function fail(message) {
  throw new Error(`web SDK validation: ${message}`)
}

function parseVersion(version, packageName) {
  const match = /^(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?$/.exec(version)
  if (!match) fail(`${packageName} has an invalid semantic version: ${version}`)
  return { major: Number(match[1]), minor: Number(match[2]), patch: Number(match[3]) }
}

export function readAndValidateSDKPackages() {
  const packages = sdkDescriptors.map((descriptor) => {
    const root = join(repositoryRoot, descriptor.directory)
    const manifest = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8'))
    if (manifest.name !== descriptor.name) fail(`${descriptor.directory} must be named ${descriptor.name}`)
    if (manifest.private === true) fail(`${descriptor.name} must not be private`)
    if (manifest.license !== 'MIT') fail(`${descriptor.name} must declare the repository MIT license`)
    if (manifest.publishConfig?.access !== 'public') fail(`${descriptor.name} must publish with public access`)
    if (manifest.publishConfig?.registry !== npmRegistry) fail(`${descriptor.name} must publish only to ${npmRegistry}`)
    if (!Array.isArray(manifest.files) || !manifest.files.includes('src')) fail(`${descriptor.name} must publish src`)
    for (const lifecycle of ['prepack', 'prepare', 'postpack']) {
      if (manifest.scripts?.[lifecycle]) fail(`${descriptor.name} must not run ${lifecycle} while creating release tarballs`)
    }
    for (const filename of ['LICENSE', 'README.md']) {
      if (!existsSync(join(root, filename))) fail(`${descriptor.name} is missing ${filename}`)
    }
    return { ...descriptor, root, manifest, semver: parseVersion(manifest.version, descriptor.name) }
  })

  const adminSDK = packages.find((entry) => entry.name === '@sforum/admin-sdk')
  const pluginUI = packages.find((entry) => entry.name === '@sforum/plugin-ui')
  const adminSource = readFileSync(join(adminSDK.root, 'src/index.ts'), 'utf8')
  const bridgeMatch = /ADMIN_MICRO_FRONTEND_API_VERSION\s*=\s*(\d+)/.exec(adminSource)
  if (!bridgeMatch) fail('@sforum/admin-sdk does not export ADMIN_MICRO_FRONTEND_API_VERSION')
  const bridgeVersion = Number(bridgeMatch[1])
  if (adminSDK.semver.major !== bridgeVersion) {
    fail(`@sforum/admin-sdk major ${adminSDK.semver.major} must match bridge API ${bridgeVersion}`)
  }
  if (pluginUI.semver.major !== bridgeVersion) {
    fail(`@sforum/plugin-ui major ${pluginUI.semver.major} must match bridge API ${bridgeVersion}`)
  }

  const scaffold = readFileSync(join(repositoryRoot, 'apps/api/cmd/sforum/generator_vue_admin.go'), 'utf8')
  for (const entry of packages) {
    const dependency = `"${entry.name}": "^${entry.manifest.version}"`
    if (!scaffold.includes(dependency)) fail(`Vue scaffold must depend on ${dependency}`)
  }
  return packages
}

function listFiles(root, prefix = '') {
  const files = []
  for (const name of readdirSync(root).sort()) {
    const absolute = join(root, name)
    const path = prefix ? `${prefix}/${name}` : name
    if (statSync(absolute).isDirectory()) files.push(...listFiles(absolute, path))
    else files.push(path)
  }
  return files
}

export function packSDKPackages(outputDirectory) {
  const packages = readAndValidateSDKPackages()
  mkdirSync(outputDirectory, { recursive: true })
  const cache = join(outputDirectory, '.npm-cache')
  mkdirSync(cache, { recursive: true })
  const packed = packages.map((entry) => {
    const stdout = execFileSync('npm', ['pack', entry.root, '--json', '--ignore-scripts', '--pack-destination', outputDirectory], {
      cwd: repositoryRoot,
      encoding: 'utf8',
      env: { ...process.env, npm_config_cache: cache }
    })
    const result = JSON.parse(stdout)
    if (!Array.isArray(result) || result.length !== 1) fail(`npm pack returned an unexpected result for ${entry.name}`)
    const metadata = result[0]
    if (metadata.name !== entry.name || metadata.version !== entry.manifest.version) {
      fail(`npm pack identity mismatch for ${entry.name}`)
    }
    const archive = join(outputDirectory, metadata.filename)
    if (!existsSync(archive)) fail(`npm pack did not create ${metadata.filename}`)
    const integrity = `sha512-${createHash('sha512').update(readFileSync(archive)).digest('base64')}`
    if (metadata.integrity !== integrity) fail(`npm pack integrity mismatch for ${entry.name}`)
    const filePaths = new Set(metadata.files.map((file) => `package/${file.path}`))
    for (const required of entry.requiredFiles) {
      if (!filePaths.has(required)) fail(`${entry.name} tarball is missing ${required}`)
    }
    for (const path of filePaths) {
      if (path.includes('/node_modules/')) fail(`${entry.name} tarball contains node_modules content`)
    }
    return { name: entry.name, version: entry.manifest.version, filename: metadata.filename, integrity }
  })
  const manifestPath = join(outputDirectory, 'web-sdks.json')
  writeFileSync(manifestPath, `${JSON.stringify({ registry: npmRegistry, packages: packed }, null, 2)}\n`)
  return { manifestPath, packages: packed }
}

export function verifyPackedSDKConsumer(packedRoot, packedPackages) {
  const viteCLI = join(repositoryRoot, 'apps/web/node_modules/vite/bin/vite.js')
  const vuePlugin = join(repositoryRoot, 'apps/web/node_modules/@vitejs/plugin-vue/dist/index.mjs')
  const vueRuntime = join(repositoryRoot, 'apps/web/node_modules/vue/dist/vue.runtime.esm-bundler.js')
  for (const required of [viteCLI, vuePlugin, vueRuntime]) {
    if (!existsSync(required)) fail(`offline consumer build dependency is missing: ${relative(repositoryRoot, required)}`)
  }

  const fixture = mkdtempSync(join(tmpdir(), 'sforum-web-sdk-consumer-'))
  try {
    const aliases = []
    for (const sdk of packedPackages) {
      const descriptor = sdkDescriptors.find((entry) => entry.name === sdk.name)
      const extractRoot = join(fixture, sdk.name.replace('@sforum/', ''))
      mkdirSync(extractRoot, { recursive: true })
      execFileSync('tar', ['-xzf', join(packedRoot, sdk.filename), '-C', extractRoot])
      const packageRoot = join(extractRoot, 'package')
      aliases.push({ find: `${sdk.name}/style.css`, replacement: join(packageRoot, 'src/style.css') })
      aliases.push({ find: sdk.name, replacement: join(packageRoot, 'src/index.ts') })
    }
    aliases.push({ find: 'vue', replacement: vueRuntime })

    const entry = join(fixture, 'entry.ts')
    writeFileSync(entry, `
import { createApp, h } from 'vue'
import { ADMIN_MICRO_FRONTEND_API_VERSION } from '@sforum/admin-sdk'
import {
  SPluginAlert, SPluginButton, SPluginEmptyState, SPluginField, SPluginInput,
  SPluginPage, SPluginSection, SPluginSelect, SPluginTable
} from '@sforum/plugin-ui'
import '@sforum/plugin-ui/style.css'

const components = [SPluginAlert, SPluginButton, SPluginEmptyState, SPluginField,
  SPluginInput, SPluginPage, SPluginSection, SPluginSelect, SPluginTable]
export const apiVersion = ADMIN_MICRO_FRONTEND_API_VERSION
export function mount(target: HTMLElement) {
  const app = createApp({ render: () => h(components[5], null, () => String(components.length)) })
  app.mount(target)
  return () => app.unmount()
}
`)
    const config = join(fixture, 'vite.config.mjs')
    writeFileSync(config, `
import vue from ${JSON.stringify(pathToFileURL(vuePlugin).href)}
export default {
  plugins: [vue()],
  resolve: { alias: ${JSON.stringify(aliases)}, dedupe: ['vue'] },
  build: {
    outDir: ${JSON.stringify(join(fixture, 'dist'))},
    emptyOutDir: true,
    cssCodeSplit: false,
    lib: { entry: ${JSON.stringify(entry)}, formats: ['es'], fileName: () => 'consumer.mjs', cssFileName: 'consumer' },
    rollupOptions: { preserveEntrySignatures: 'exports-only' }
  }
}
`)
    execFileSync(process.execPath, [viteCLI, 'build', '--config', config], {
      cwd: fixture,
      stdio: 'inherit',
      env: { ...process.env, NODE_ENV: 'production' }
    })
    const outputs = listFiles(join(fixture, 'dist'))
    const esm = outputs.find((file) => file.endsWith('.mjs'))
    if (!esm || !outputs.some((file) => file.endsWith('.css'))) {
      fail(`offline consumer build did not emit ESM and CSS: ${outputs.join(', ')}`)
    }
    if (statSync(join(fixture, 'dist', esm)).size < 10_000) {
      fail(`offline consumer ESM is unexpectedly empty: ${esm}`)
    }
  } finally {
    rmSync(fixture, { recursive: true, force: true })
  }
}

export function readPackedManifest(manifestPath) {
  const root = dirname(resolve(manifestPath))
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'))
  if (manifest.registry !== npmRegistry || !Array.isArray(manifest.packages) || manifest.packages.length !== sdkDescriptors.length) {
    fail('packed SDK manifest has an unexpected registry or package count')
  }
  const expected = readAndValidateSDKPackages()
  const seen = new Set()
  for (const sdk of manifest.packages) {
    if (seen.has(sdk.name)) fail(`packed SDK manifest repeats ${sdk.name}`)
    seen.add(sdk.name)
    const local = expected.find((entry) => entry.name === sdk.name)
    if (!local || sdk.version !== local.manifest.version || !/^sha512-[A-Za-z0-9+/]+=*$/.test(sdk.integrity)) {
      fail(`packed SDK manifest has an invalid package entry for ${sdk.name}`)
    }
    const archive = resolve(root, sdk.filename)
    if (dirname(archive) !== root || !existsSync(archive)) fail(`packed SDK archive path is invalid: ${sdk.filename}`)
    const integrity = `sha512-${createHash('sha512').update(readFileSync(archive)).digest('base64')}`
    if (integrity !== sdk.integrity) fail(`packed SDK archive changed after packing: ${sdk.filename}`)
  }
  for (const sdk of expected) {
    if (!seen.has(sdk.name)) fail(`packed SDK manifest is missing ${sdk.name}`)
  }
  return { root, manifest }
}
