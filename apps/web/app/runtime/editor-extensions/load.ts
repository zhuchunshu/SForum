import {
  EditorL2ContractError,
  isEditorL2Module,
  type EditorCatalogModule,
  type EditorL2BridgeV1,
  type EditorL2ModuleV1
} from './types'

export const EDITOR_L2_LOAD_TIMEOUT_MS = 10_000

type LoadEnvironment = {
  fetch: typeof fetch
  importModule: (url: string) => Promise<unknown>
  origin: string
  timeoutMS: number
  subtle: SubtleCrypto
}

export type LoadedEditorL2Module = {
  module: EditorL2ModuleV1
  bridge: EditorL2BridgeV1
  extensions: unknown[]
}

/**
 * Digest-verify then same-origin import a prebuilt editor L2 module.
 * Failures throw EditorL2ContractError so SFEditor can quarantine and keep core.
 */
export async function loadTrustedEditorL2Module(
  catalogModule: EditorCatalogModule,
  apiBaseUrl: string,
  overrides: Partial<LoadEnvironment> = {}
): Promise<LoadedEditorL2Module> {
  const environment = loadEnvironment(overrides)
  if (!catalogModule.assetPath.includes(catalogModule.packageDigest)) {
    throw new EditorL2ContractError('editor asset path is not package-digest bound')
  }
  const url = sameOriginURL(apiBaseUrl, catalogModule.assetPath, environment.origin)
  const bytes = await fetchVerifiedBytes(url, catalogModule.l2Digest, environment)
  // Re-verify before import so a poisoned cache cannot skip the Host digest.
  await verifyDigest(bytes, catalogModule.l2Digest, environment.subtle)
  const loaded = await withTimeout(
    environment.importModule(url),
    environment.timeoutMS,
    'editor module import timed out'
  )
  if (!isEditorL2Module(loaded)) {
    throw new EditorL2ContractError('editor L2 module contract mismatch')
  }
  const bridge: EditorL2BridgeV1 = {
    apiVersion: 1,
    extensionId: catalogModule.extensionId,
    extensionVersion: catalogModule.extensionVersion,
    packageDigest: catalogModule.packageDigest,
    modulePath: catalogModule.l2Module,
    moduleDigest: catalogModule.l2Digest
  }
  const extensions = await withTimeout(
    Promise.resolve(loaded.createExtensions(bridge)),
    environment.timeoutMS,
    'editor createExtensions timed out'
  )
  if (!Array.isArray(extensions)) {
    throw new EditorL2ContractError('editor createExtensions must return an array')
  }
  return { module: loaded, bridge, extensions }
}

export async function verifyDigest(bytes: ArrayBuffer, expected: string, subtle: SubtleCrypto) {
  const digest = await subtle.digest('SHA-256', bytes)
  const actual = [...new Uint8Array(digest)].map(value => value.toString(16).padStart(2, '0')).join('')
  if (actual !== expected) {
    throw new EditorL2ContractError('editor module digest mismatch')
  }
}

async function fetchVerifiedBytes(url: string, expectedDigest: string, environment: LoadEnvironment) {
  const response = await withTimeout(
    environment.fetch(url, { credentials: 'same-origin' }),
    environment.timeoutMS,
    'editor module fetch timed out'
  )
  if (!response.ok) {
    throw new EditorL2ContractError(`editor module fetch failed with ${response.status}`)
  }
  const bytes = await response.arrayBuffer()
  await verifyDigest(bytes, expectedDigest, environment.subtle)
  return bytes
}

function sameOriginURL(apiBaseUrl: string, assetPath: string, origin: string) {
  const base = apiBaseUrl.replace(/\/$/, '')
  const path = assetPath.startsWith('/') ? assetPath : `/${assetPath}`
  const absolute = path.startsWith('http') ? path : `${base}${path}`
  const parsed = new URL(absolute, origin || 'http://localhost')
  if (origin && parsed.origin !== new URL(origin).origin) {
    throw new EditorL2ContractError('editor module must load from same origin')
  }
  return parsed.toString()
}

function loadEnvironment(overrides: Partial<LoadEnvironment>): LoadEnvironment {
  return {
    fetch: overrides.fetch || globalThis.fetch.bind(globalThis),
    importModule: overrides.importModule || (url => import(/* @vite-ignore */ url)),
    origin: overrides.origin || (typeof location !== 'undefined' ? location.origin : 'http://localhost'),
    timeoutMS: overrides.timeoutMS || EDITOR_L2_LOAD_TIMEOUT_MS,
    subtle: overrides.subtle || globalThis.crypto.subtle
  }
}

function withTimeout<T>(promise: Promise<T>, timeoutMS: number, message: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const timer = globalThis.setTimeout(() => reject(new EditorL2ContractError(message)), timeoutMS)
    promise.then(
      value => {
        globalThis.clearTimeout(timer)
        resolve(value)
      },
      error => {
        globalThis.clearTimeout(timer)
        reject(error)
      }
    )
  })
}
