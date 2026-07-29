export function normalizeSiteDomain(value: string | undefined) {
  return (value || '')
    .trim()
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
}

export function siteDomainFromUrl(value: string | undefined) {
  try {
    return normalizeSiteDomain(new URL(value || '').host)
  } catch {
    return ''
  }
}
