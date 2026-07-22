export const ACTIVE_THEME_CACHE_TTL_MS = 60_000
export const ACTIVE_THEME_SKIN_STORAGE_KEY = 'sforum-active-theme-css'

const SKIN_CACHE_SCHEMA = 'sforum.active-theme-skin@1'
const SETTINGS_CACHE_SCHEMA = 'sforum.active-theme-settings@1'
const THEME_ASSET_PREFIX = '/_sforum/assets/themes/'
const MAX_SKIN_LINKS = 16

export type ActiveThemeIdentity = {
  extensionId: string
  version?: string
  packageDigest: string
  nodeRevision?: number
}

export type ActiveThemeSkinResponse = {
  extensionId?: string
  version?: string
  packageDigest?: string
  css?: string[]
  tokens?: string
  nodeRevision?: number
}

export type ActiveThemeSkinCacheRecord = {
  schema: typeof SKIN_CACHE_SCHEMA
  createdAt: number
  identity: ActiveThemeIdentity
  links: string[]
}

export type ActiveThemeSettingsResponse = {
  themeId?: string
  version?: string
  packageDigest?: string
  nodeRevision?: number
  settings?: Record<string, string>
}

export type ActiveThemeSettingsCacheRecord<T extends ActiveThemeSettingsResponse = ActiveThemeSettingsResponse> = {
  schema: typeof SETTINGS_CACHE_SCHEMA
  createdAt: number
  identity: ActiveThemeIdentity
  data: T
}

type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

export function normalizeActiveThemeIdentity(
  value: {
    extensionId?: unknown
    themeId?: unknown
    version?: unknown
    packageDigest?: unknown
    nodeRevision?: unknown
  } | null | undefined,
  options: { requireRevision?: boolean } = {}
): ActiveThemeIdentity | null {
  const extensionId = safeToken(value?.extensionId ?? value?.themeId)
  const packageDigest = safeDigest(value?.packageDigest)
  if (!extensionId || !packageDigest) {
    return null
  }
  const nodeRevision = safeRevision(value?.nodeRevision)
  if (options.requireRevision && nodeRevision === undefined) {
    return null
  }
  const version = safeVersion(value?.version)
  return {
    extensionId,
    packageDigest,
    ...(version ? { version } : {}),
    ...(nodeRevision !== undefined ? { nodeRevision } : {})
  }
}

export function mergeActiveThemeIdentity(
  current: ActiveThemeIdentity | null,
  next: ActiveThemeIdentity | null
): ActiveThemeIdentity | null {
  if (!next) {
    return null
  }
  if (
    current
    && current.extensionId === next.extensionId
    && current.packageDigest === next.packageDigest
    && next.nodeRevision === undefined
    && current.nodeRevision !== undefined
  ) {
    return { ...next, nodeRevision: current.nodeRevision }
  }
  return next
}

export function sameActiveThemeIdentity(
  left: ActiveThemeIdentity | null | undefined,
  right: ActiveThemeIdentity | null | undefined,
  options: { requireRevision?: boolean } = {}
) {
  if (!left || !right) {
    return false
  }
  if (left.extensionId !== right.extensionId || left.packageDigest !== right.packageDigest) {
    return false
  }
  if (!options.requireRevision) {
    return true
  }
  return left.nodeRevision !== undefined
    && right.nodeRevision !== undefined
    && left.nodeRevision === right.nodeRevision
}

export function normalizeActiveThemeSkinPayload(
  skin: ActiveThemeSkinResponse | null | undefined,
  now = Date.now()
) {
  const identity = normalizeActiveThemeIdentity(skin)
  const links = identity
    ? skinLinks(skin).filter(href => isThemeAssetHrefForIdentity(href, identity))
    : []
  const cacheIdentity = normalizeActiveThemeIdentity(skin, { requireRevision: true })
  const record = cacheIdentity && links.length
    ? {
        schema: SKIN_CACHE_SCHEMA,
        createdAt: now,
        identity: cacheIdentity,
        links
      } satisfies ActiveThemeSkinCacheRecord
    : null
  return { identity, links, record }
}

export function readStoredSkinRecord(
  storage: StorageLike | null | undefined,
  currentIdentity: ActiveThemeIdentity | null | undefined,
  now = Date.now()
): ActiveThemeSkinCacheRecord | null {
  if (!storage || !currentIdentity) {
    return null
  }
  try {
    return parseStoredSkinRecord(storage.getItem(ACTIVE_THEME_SKIN_STORAGE_KEY), currentIdentity, now)
  } catch {
    return null
  }
}

export function writeStoredSkinRecord(
  storage: StorageLike | null | undefined,
  record: ActiveThemeSkinCacheRecord
) {
  if (!storage) {
    return
  }
  try {
    storage.setItem(ACTIVE_THEME_SKIN_STORAGE_KEY, JSON.stringify(record))
  } catch {
    // 隐私模式/配额满时忽略；内存态 last-good 仍可用。
  }
}

export function clearStoredSkinRecord(storage: StorageLike | null | undefined) {
  if (!storage) {
    return
  }
  try {
    storage.removeItem(ACTIVE_THEME_SKIN_STORAGE_KEY)
  } catch {
    // ignore
  }
}

export function parseStoredSkinRecord(
  raw: string | null | undefined,
  currentIdentity: ActiveThemeIdentity,
  now = Date.now()
): ActiveThemeSkinCacheRecord | null {
  const parsed = parseObject(raw)
  if (!parsed || parsed.schema !== SKIN_CACHE_SCHEMA || !freshEnough(parsed.createdAt, now)) {
    return null
  }
  const identity = normalizeActiveThemeIdentity(parsed.identity as Record<string, unknown>, { requireRevision: true })
  if (!sameActiveThemeIdentity(identity, currentIdentity, { requireRevision: true })) {
    return null
  }
  const links = Array.isArray(parsed.links) ? parsed.links : []
  if (
    !links.length
    || links.length > MAX_SKIN_LINKS
    || links.some(href => typeof href !== 'string' || !isThemeAssetHrefForIdentity(href, identity!))
  ) {
    return null
  }
  return {
    schema: SKIN_CACHE_SCHEMA,
    createdAt: Number(parsed.createdAt),
    identity: identity!,
    links: [...links]
  }
}

export function canUseActiveThemeSkinRecord(
  record: ActiveThemeSkinCacheRecord | null | undefined,
  currentIdentity: ActiveThemeIdentity | null | undefined,
  now = Date.now()
) {
  if (
    !record
    || !currentIdentity
    || record.schema !== SKIN_CACHE_SCHEMA
    || !freshEnough(record.createdAt, now)
    || !sameActiveThemeIdentity(record.identity, currentIdentity, { requireRevision: true })
  ) {
    return false
  }
  return record.links.length > 0
    && record.links.length <= MAX_SKIN_LINKS
    && record.links.every(href => isThemeAssetHrefForIdentity(href, record.identity))
}

export function makeActiveThemeSettingsRecord<T extends ActiveThemeSettingsResponse>(
  data: T,
  now = Date.now()
): ActiveThemeSettingsCacheRecord<T> | null {
  const identity = normalizeActiveThemeIdentity(data)
  if (!identity) {
    return null
  }
  return {
    schema: SETTINGS_CACHE_SCHEMA,
    createdAt: now,
    identity,
    data
  }
}

export function canUseActiveThemeSettingsRecord(
  record: ActiveThemeSettingsCacheRecord | null | undefined,
  currentIdentity: ActiveThemeIdentity | null | undefined,
  now = Date.now()
) {
  const requireRevision = Boolean(record?.identity.nodeRevision !== undefined || currentIdentity?.nodeRevision !== undefined)
  return Boolean(
    record
    && currentIdentity
    && record.schema === SETTINGS_CACHE_SCHEMA
    && freshEnough(record.createdAt, now)
    && sameActiveThemeIdentity(record.identity, currentIdentity, { requireRevision })
  )
}

function skinLinks(skin: ActiveThemeSkinResponse | null | undefined) {
  const hrefs = [...(Array.isArray(skin?.css) ? skin.css : [])]
  if (typeof skin?.tokens === 'string' && skin.tokens.trim()) {
    hrefs.unshift(skin.tokens)
  }
  return hrefs.map(href => href.trim()).filter(Boolean).slice(0, MAX_SKIN_LINKS)
}

function isThemeAssetHrefForIdentity(href: string, identity: ActiveThemeIdentity) {
  if (!href || /^[a-z][a-z0-9+.-]*:/i.test(href) || href.includes('\\')) {
    return false
  }
  try {
    const url = new URL(href, 'http://sforum.local')
    if (url.origin !== 'http://sforum.local' || url.hash) {
      return false
    }
    const prefix = `${THEME_ASSET_PREFIX}${encodeURIComponent(identity.extensionId)}/${encodeURIComponent(identity.packageDigest)}/`
    return url.pathname.startsWith(prefix) && !url.search
  } catch {
    return false
  }
}

function parseObject(raw: string | null | undefined): Record<string, unknown> | null {
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed as Record<string, unknown>
      : null
  } catch {
    return null
  }
}

function freshEnough(createdAt: unknown, now: number) {
  return typeof createdAt === 'number'
    && Number.isFinite(createdAt)
    && createdAt <= now
    && now - createdAt <= ACTIVE_THEME_CACHE_TTL_MS
}

function safeToken(value: unknown) {
  const raw = typeof value === 'string' ? value.trim() : ''
  return /^[a-zA-Z0-9_.-]{1,128}$/.test(raw) ? raw : ''
}

function safeVersion(value: unknown) {
  const raw = typeof value === 'string' ? value.trim() : ''
  return /^[a-zA-Z0-9_.:+-]{1,128}$/.test(raw) ? raw : ''
}

function safeDigest(value: unknown) {
  const raw = typeof value === 'string' ? value.trim() : ''
  return /^[a-zA-Z0-9_.:-]{1,128}$/.test(raw) ? raw : ''
}

function safeRevision(value: unknown) {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0
    ? value
    : undefined
}
