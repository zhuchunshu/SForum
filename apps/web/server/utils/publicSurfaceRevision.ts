type ApiEnvelope<T> = {
  data: T
}

type WebOption = {
  name: string
  value: string
}

/** Nitro SWR varies 请求头：与 nuxt.config `/t/**` cache.varies 对齐。 */
export const PUBLIC_SURFACE_REVISION_HEADER = 'x-sforum-public-surface-revision'

const OPTION_NAME = 'site.public_surface_revision'
const FALLBACK_REVISION = '1'
const CACHE_TTL_MS = 1000

let cachedRevision = FALLBACK_REVISION
let cachedAt = 0
let inflight: Promise<string> | null = null

/**
 * 读取宿主公开前端贡献面 revision（至少为 1）。
 * 短进程内缓存，避免每条 /t/** 都打 API；bump 后最多约 1s 内仍用旧键。
 */
export async function loadPublicSurfaceRevision(): Promise<string> {
  const now = Date.now()
  if (now - cachedAt < CACHE_TTL_MS && cachedRevision) {
    return cachedRevision
  }
  if (inflight) {
    return inflight
  }

  inflight = fetchPublicSurfaceRevision()
    .then((revision) => {
      cachedRevision = revision
      cachedAt = Date.now()
      return revision
    })
    .catch(() => {
      // API 短暂不可用时沿用上次成功值，避免缓存键抖动。
      return cachedRevision || FALLBACK_REVISION
    })
    .finally(() => {
      inflight = null
    })

  return inflight
}

async function fetchPublicSurfaceRevision(): Promise<string> {
  const apiBaseUrl = (process.env.NUXT_API_INTERNAL_BASE_URL || 'http://api:8080/api/v1').replace(/\/+$/, '')
  try {
    const envelope = await $fetch<ApiEnvelope<WebOption>>(`${apiBaseUrl}/web-options/${encodeURIComponent(OPTION_NAME)}`, {
      timeout: 800
    })
    const raw = String(envelope?.data?.value ?? '').trim()
    const parsed = Number.parseInt(raw, 10)
    if (Number.isFinite(parsed) && parsed >= 1) {
      return String(parsed)
    }
  } catch {
    // 回落：尝试整表 web-options（兼容仅 list 可用的环境）。
    try {
      const envelope = await $fetch<ApiEnvelope<WebOption[]>>(`${apiBaseUrl}/web-options`, {
        timeout: 800
      })
      const item = (envelope?.data || []).find(option => option.name === OPTION_NAME)
      const raw = String(item?.value ?? '').trim()
      const parsed = Number.parseInt(raw, 10)
      if (Number.isFinite(parsed) && parsed >= 1) {
        return String(parsed)
      }
    } catch {
      // keep fallback
    }
  }
  return FALLBACK_REVISION
}
