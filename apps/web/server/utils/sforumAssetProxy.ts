const PUBLIC_ASSET_PREFIX = '/_sforum/assets/'
const PRIVATE_ASSET_PREFIX = '/_sforum/private-assets/'
const DIGEST_PATTERN = /^[0-9a-f]{64}$/

export function buildPublicAssetTarget(apiBaseURL: string, pathname: string) {
  const segments = assetSegments(pathname, PUBLIC_ASSET_PREFIX)
  const [owner, extensionId, packageDigest, ...relative] = segments
  if (!extensionId || !DIGEST_PATTERN.test(packageDigest || '') || relative.length === 0) {
    throw new Error('invalid public asset path')
  }

  if (owner === 'themes') {
    const target = apiTarget(apiBaseURL, ['site', 'theme-assets', extensionId, ...relative])
    target.searchParams.set('v', packageDigest!)
    return target
  }
  if (owner === 'extensions') {
    return apiTarget(apiBaseURL, ['extensions', 'runtime', extensionId, 'packages', packageDigest!, ...relative])
  }
  throw new Error('unknown public asset owner')
}

export function buildPrivateAssetTarget(apiBaseURL: string, pathname: string) {
  const segments = assetSegments(pathname, PRIVATE_ASSET_PREFIX)
  const [owner, extensionId, packageDigest, asset, ...extra] = segments
  if (owner !== 'extensions' || !extensionId || !DIGEST_PATTERN.test(packageDigest || '')
    || (asset !== 'entry' && asset !== 'style') || extra.length !== 0) {
    throw new Error('invalid private asset path')
  }
  return apiTarget(apiBaseURL, ['admin', 'extensions', extensionId, 'frontend', 'assets', packageDigest!, asset])
}

function assetSegments(pathname: string, prefix: string) {
  if (!pathname.startsWith(prefix) || pathname.includes('\\')) {
    throw new Error('invalid asset namespace')
  }
  return pathname.slice(prefix.length).split('/').map((segment) => {
    let decoded: string
    try {
      decoded = decodeURIComponent(segment)
    } catch {
      throw new Error('invalid asset path encoding')
    }
    if (!decoded || decoded === '.' || decoded === '..' || decoded.includes('/') || decoded.includes('\\')
      || /[\u0000-\u001f\u007f]/.test(decoded)) {
      throw new Error('invalid asset path segment')
    }
    return decoded
  })
}

function apiTarget(apiBaseURL: string, segments: string[]) {
  const target = new URL(apiBaseURL)
  if ((target.protocol !== 'http:' && target.protocol !== 'https:') || target.username || target.password
    || target.search || target.hash) {
    throw new Error('NUXT_API_INTERNAL_BASE_URL must be an HTTP(S) URL without credentials, query, or fragment')
  }
  const basePath = target.pathname.replace(/\/$/, '')
  target.pathname = `${basePath}/${segments.map(segment => encodeURIComponent(segment)).join('/')}`
  return target
}
