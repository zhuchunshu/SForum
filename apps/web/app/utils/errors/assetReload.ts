/**
 * 静态资源 / 动态 import 失败识别与刷新冷却的纯函数。
 *
 * 背景：站点发布新版本后，浏览器持有的旧页面继续请求已被替换的
 * /_nuxt/** chunk，产生 404 / 网络层失败（ERR_NETWORK_CHANGED 等）。
 * 这类失败与「后端 API 断线」表现相似但根因不同，应走「新版本已发布，
 * 正在刷新」轻提示 + 自动刷新，而不是全局 API 断线模态。
 */

export const ASSET_RELOAD_STORAGE_KEY = 'sforum:asset-reload-at'
export const ASSET_RELOAD_COOLDOWN_MS = 10000

// eslint-disable-next-line prettier/prettier -- 单行正则：字面量内嵌换行会被当成待匹配字符
const DYNAMIC_IMPORT_FAILURE_PATTERN = /(failed\s+to\s+fetch\s+dynamically\s+imported\s+module|importing\s+a\s+module\s+script\s+failed|error\s+loading\s+dynamically\s+imported\s+module|err_network_changed)/i

export function isNuxtAssetUrl(url: unknown) {
  return typeof url === 'string' && url.includes('/_nuxt/')
}

export function isDynamicImportFailureMessage(message: unknown) {
  return typeof message === 'string' && DYNAMIC_IMPORT_FAILURE_PATTERN.test(message)
}

/** 沿 message + cause 链收集文本，覆盖 Error 包一层 Error 的包装形态。 */
export function extractErrorChainMessage(reason: unknown, depth = 3) {
  const parts: string[] = []
  let current: unknown = reason
  for (let hop = 0; hop <= depth && current; hop += 1) {
    if (current instanceof Error) {
      parts.push(current.message)
    } else if (current && typeof current === 'object' && 'message' in current) {
      const message = (current as { message?: unknown }).message
      if (typeof message === 'string') {
        parts.push(message)
      }
    }
    current = (current as { cause?: unknown } | null)?.cause
  }
  return parts.join(' ')
}

export function readAssetReloadTimestamp(storage: Pick<Storage, 'getItem'>) {
  const raw = storage.getItem(ASSET_RELOAD_STORAGE_KEY)
  if (!raw) {
    return null
  }
  const value = Number(raw)
  return Number.isFinite(value) && value > 0 ? value : null
}

export function writeAssetReloadTimestamp(storage: Pick<Storage, 'setItem'>, timestamp: number) {
  storage.setItem(ASSET_RELOAD_STORAGE_KEY, String(timestamp))
}

export function isWithinAssetReloadCooldown(lastAt: number | null, now: number) {
  return lastAt !== null && now - lastAt < ASSET_RELOAD_COOLDOWN_MS
}
