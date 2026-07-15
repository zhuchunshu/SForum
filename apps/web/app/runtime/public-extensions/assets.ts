import type {
  PublicFrontendAssetReference,
  PublicFrontendComponentDescriptor,
  PublicFrontendModuleV1
} from './types'
import { PublicFrontendContractError, publicFrontendExactHeaders } from './types'

type AssetEnvironment = {
  fetch: typeof fetch
  document: Document
  importModule: (url: string) => Promise<unknown>
  loadStyle: (reference: PublicFrontendAssetReference, url: string) => Promise<HTMLLinkElement>
  origin: string
  timeoutMS: number
}

export const PUBLIC_FRONTEND_ASSET_TIMEOUT_MS = 10_000

export type PublicFrontendRelease = {
  module: PublicFrontendModuleV1
  release: () => Promise<void>
}

type StyleLease = {
  refs: number
  element?: HTMLLinkElement
  ready: Promise<void>
}

const moduleCache = new Map<string, Promise<unknown>>()
const styleLeases = new Map<string, StyleLease>()

export async function loadPublicFrontendRelease(
  descriptor: PublicFrontendComponentDescriptor,
  apiBaseUrl: string,
  overrides: Partial<AssetEnvironment> = {}
): Promise<PublicFrontendRelease> {
  const environment = assetEnvironment(overrides)
  const exactHeaders = publicFrontendExactHeaders(descriptor)
  const releases: Array<() => void> = []
  try {
    for (const asset of descriptor.assets) {
      if (asset.type === 'style') {
        releases.push(await acquireStyle(asset, apiBaseUrl, exactHeaders, environment))
      } else {
        await acquireModule(asset, apiBaseUrl, exactHeaders, environment)
      }
    }
    const loaded = await acquireModule(descriptor.entry, apiBaseUrl, exactHeaders, environment)
    if (!isPublicFrontendModule(loaded)) {
      throw new PublicFrontendContractError('public component module contract mismatch')
    }
    let released = false
    return {
      module: loaded,
      release: async () => {
        if (released) return
        released = true
        for (const release of releases.reverse()) release()
      }
    }
  } catch (error) {
    for (const release of releases.reverse()) release()
    throw error
  }
}

export async function verifyPublicAssetBytes(reference: PublicFrontendAssetReference, bytes: ArrayBuffer) {
  const digest = await crypto.subtle.digest('SHA-256', bytes)
  const actual = [...new Uint8Array(digest)].map(value => value.toString(16).padStart(2, '0')).join('')
  if (actual !== reference.digest) {
    throw new PublicFrontendContractError('public asset digest mismatch')
  }
}

export function resetPublicAssetRuntimeForTest() {
  moduleCache.clear()
  for (const lease of styleLeases.values()) lease.element?.remove()
  styleLeases.clear()
}

async function acquireStyle(
  reference: PublicFrontendAssetReference,
  apiBaseUrl: string,
  exactHeaders: Record<string, string>,
  environment: AssetEnvironment
) {
  const key = assetKey(reference)
  let lease = styleLeases.get(key)
  if (!lease) {
    lease = { refs: 0, ready: Promise.resolve() }
    const owned = lease
    owned.ready = (async () => {
      await fetchVerifiedAsset(reference, apiBaseUrl, exactHeaders, environment)
      const url = sameOriginAssetURL(apiBaseUrl, reference.assetPath, environment.origin)
      const element = await environment.loadStyle(reference, url)
      element.dataset.sforumExtension = reference.extensionId
      element.dataset.sforumAsset = reference.handle
      element.dataset.sforumDigest = reference.digest
      owned.element = element
    })()
    styleLeases.set(key, owned)
  }
  lease.refs++
  try {
    await lease.ready
  } catch (error) {
    releaseStyle(key, lease)
    throw error
  }
  let released = false
  return () => {
    if (released) return
    released = true
    releaseStyle(key, lease)
  }
}

function releaseStyle(key: string, expected: StyleLease) {
  const lease = styleLeases.get(key)
  if (!lease || lease !== expected) return
  lease.refs--
  if (lease.refs > 0) return
  lease.element?.remove()
  styleLeases.delete(key)
}

async function acquireModule(
  reference: PublicFrontendAssetReference,
  apiBaseUrl: string,
  exactHeaders: Record<string, string>,
  environment: AssetEnvironment
) {
  const key = assetKey(reference)
  let loading = moduleCache.get(key)
  if (!loading) {
    loading = (async () => {
      const url = sameOriginAssetURL(apiBaseUrl, reference.assetPath, environment.origin)
      await fetchVerifiedAsset(reference, apiBaseUrl, exactHeaders, environment)
      // Native same-origin import keeps CSP at 'self'; the immutable URL binds
      // package and file digests, and Host rechecks live trust on the import GET.
      return withTimeout(
        environment.importModule(url),
        environment.timeoutMS,
        'public module import timed out'
      )
    })()
    moduleCache.set(key, loading)
    loading.catch(() => moduleCache.delete(key))
  }
  return loading
}

async function fetchVerifiedAsset(
  reference: PublicFrontendAssetReference,
  apiBaseUrl: string,
  exactHeaders: Record<string, string>,
  environment: AssetEnvironment
) {
  const controller = new AbortController()
  const timeout = globalThis.setTimeout(() => controller.abort(), environment.timeoutMS)
  try {
    const response = await environment.fetch(sameOriginAssetURL(apiBaseUrl, reference.assetPath, environment.origin), {
      credentials: 'same-origin',
      cache: 'force-cache',
      signal: controller.signal,
      headers: {
        Accept: reference.type === 'style' ? 'text/css' : 'application/javascript',
        ...exactHeaders
      }
    })
    if (!response.ok) {
      throw new PublicFrontendContractError(`public asset request failed (${response.status})`)
    }
    const contentType = response.headers.get('content-type')?.toLowerCase() || ''
    const expectedType = reference.type === 'style' ? 'text/css' : 'application/javascript'
    if (!contentType.startsWith(expectedType)
      || response.headers.get('x-sforum-asset-digest') !== reference.digest
      || response.headers.get('x-sforum-asset-integrity') !== reference.integrity) {
      throw new PublicFrontendContractError('public asset response identity mismatch')
    }
    const body = await response.arrayBuffer()
    await verifyPublicAssetBytes(reference, body)
    return body
  } catch (error) {
    if (error instanceof PublicFrontendContractError) throw error
    throw new PublicFrontendContractError(
      controller.signal.aborted ? 'public asset request timed out' : 'public asset request failed'
    )
  } finally {
    globalThis.clearTimeout(timeout)
  }
}

export function sameOriginAssetURL(apiBaseUrl: string, assetPath: string, origin: string) {
  let browserOrigin: URL
  let api: URL
  try {
    browserOrigin = new URL(origin)
    api = new URL(apiBaseUrl || '/', `${browserOrigin.origin}/`)
  } catch {
    throw new PublicFrontendContractError('public asset origin is invalid')
  }
  if (api.origin !== browserOrigin.origin || api.search || api.hash || !assetPath.startsWith('/extensions/runtime/')) {
    throw new PublicFrontendContractError('public assets must use the Host same-origin runtime')
  }
  const pathname = `${api.pathname.replace(/\/$/, '')}${assetPath}`
  const url = new URL(pathname, `${browserOrigin.origin}/`)
  if (url.origin !== browserOrigin.origin) {
    throw new PublicFrontendContractError('public asset escaped the Host origin')
  }
  return url.href
}

function assetKey(reference: PublicFrontendAssetReference) {
  return `${reference.impactDigest}:${reference.handle}:${reference.digest}`
}

function isPublicFrontendModule(input: unknown): input is PublicFrontendModuleV1 {
  if (typeof input !== 'object' || input === null) return false
  const module = input as Partial<PublicFrontendModuleV1>
  return module.apiVersion === 1 && typeof module.mount === 'function'
    && (module.unmount === undefined || typeof module.unmount === 'function')
}

function assetEnvironment(overrides: Partial<AssetEnvironment>): AssetEnvironment {
  const runtimeDocument = overrides.document ?? document
  const timeoutMS = overrides.timeoutMS ?? PUBLIC_FRONTEND_ASSET_TIMEOUT_MS
  if (!Number.isFinite(timeoutMS) || timeoutMS <= 0) {
    throw new PublicFrontendContractError('public asset timeout is invalid')
  }
  return {
    fetch: overrides.fetch ?? globalThis.fetch.bind(globalThis),
    document: runtimeDocument,
    importModule: overrides.importModule ?? (url => import(/* @vite-ignore */ url)),
    loadStyle: overrides.loadStyle ?? ((reference, url) => loadStylesheet(runtimeDocument, reference, url, timeoutMS)),
    origin: overrides.origin ?? globalThis.location.origin,
    timeoutMS
  }
}

function loadStylesheet(
  runtimeDocument: Document,
  reference: PublicFrontendAssetReference,
  url: string,
  timeoutMS: number
): Promise<HTMLLinkElement> {
  return new Promise((resolve, reject) => {
    const element = runtimeDocument.createElement('link')
    element.rel = 'stylesheet'
    element.href = url
    element.integrity = reference.integrity
    element.crossOrigin = 'anonymous'
    let settled = false
    const timeout = globalThis.setTimeout(() => finish(false, 'public stylesheet load timed out'), timeoutMS)
    const finish = (loaded: boolean, reason = 'public stylesheet load failed') => {
      if (settled) return
      settled = true
      globalThis.clearTimeout(timeout)
      if (loaded) {
        resolve(element)
        return
      }
      element.remove()
      reject(new PublicFrontendContractError(reason))
    }
    element.addEventListener('load', () => finish(true), { once: true })
    element.addEventListener('error', () => finish(false), { once: true })
    runtimeDocument.head.append(element)
  })
}

function withTimeout<T>(promise: Promise<T>, timeoutMS: number, message: string): Promise<T> {
  return new Promise((resolve, reject) => {
    const timeout = globalThis.setTimeout(() => reject(new PublicFrontendContractError(message)), timeoutMS)
    promise.then(
      value => {
        globalThis.clearTimeout(timeout)
        resolve(value)
      },
      error => {
        globalThis.clearTimeout(timeout)
        reject(error)
      }
    )
  })
}
