export type ErrorPageContent = {
  statusCode: number
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
      return content(status, 'forbidden', 'i-lucide-shield-alert')
    case 404:
      return content(status, 'notFound', 'i-lucide-file-question')
    case 500:
      return content(status, 'server', 'i-lucide-server', true)
    case 503:
      return content(status, 'serviceUnavailable', 'i-lucide-refresh-cw', true)
    default:
      return content(status, 'generic', 'i-lucide-circle-alert')
  }
}

function content(statusCode: number, key: string, icon: string, showRetry = false): ErrorPageContent {
  return {
    statusCode,
    titleKey: `errors.page.${key}.title`,
    descriptionKey: `errors.page.${key}.description`,
    icon,
    showRetry
  }
}
