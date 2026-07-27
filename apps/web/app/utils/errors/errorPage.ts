export type ErrorPageContent = {
  statusCode: number
  pageId: string
  titleKey: string
  descriptionKey: string
  icon: string
  showRetry: boolean
}

export function normalizeErrorStatus(statusCode: unknown) {
  const parsed = Number(statusCode)
  if (!Number.isFinite(parsed)) {
    return 500
  }

  const status = Math.trunc(parsed)
  return status >= 400 && status <= 599 ? status : 500
}

export function resolveErrorPageContent(statusCode: unknown): ErrorPageContent {
  const status = normalizeErrorStatus(statusCode)

  switch (status) {
    case 403:
      return content(status, 'system.forbidden', 'forbidden', 'i-lucide-shield-alert')
    case 404:
      return content(status, 'system.not_found', 'notFound', 'i-lucide-file-question')
    case 429:
      return content(status, 'system.rate_limited', 'rateLimited', 'i-lucide-timer-reset', true)
    case 500:
      return content(status, 'system.server_error', 'server', 'i-lucide-server', true)
    case 502:
      return content(status, 'system.server_error', 'badGateway', 'i-lucide-unplug', true)
    case 503:
      return content(status, 'system.server_error', 'serviceUnavailable', 'i-lucide-refresh-cw', true)
    case 504:
      return content(status, 'system.server_error', 'gatewayTimeout', 'i-lucide-clock-alert', true)
    default:
      return content(status, 'core', 'generic', 'i-lucide-circle-alert')
  }
}

export function systemErrorPageIdForStatus(statusCode: unknown) {
  return resolveErrorPageContent(statusCode).pageId
}

export function isThemeableSystemErrorStatus(statusCode: unknown) {
  return systemErrorPageIdForStatus(statusCode) !== 'core'
}

function content(statusCode: number, pageId: string, key: string, icon: string, showRetry = false): ErrorPageContent {
  return {
    statusCode,
    pageId,
    titleKey: `errors.page.${key}.title`,
    descriptionKey: `errors.page.${key}.description`,
    icon,
    showRetry
  }
}
