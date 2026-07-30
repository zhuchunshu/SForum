export type UploadPermissionMode = 'inherit' | 'allow' | 'deny'

export type PermissionOverrides = {
  allow: string[]
  deny: string[]
}

export function uploadPermissionMode(overrides: PermissionOverrides): UploadPermissionMode {
  if (overrides.deny.includes('attachment.upload')) return 'deny'
  if (overrides.allow.includes('attachment.upload')) return 'allow'
  return 'inherit'
}

export function replaceUploadPermissionOverride(
  overrides: PermissionOverrides,
  mode: UploadPermissionMode
): PermissionOverrides {
  const allow = overrides.allow.filter(key => key !== 'attachment.upload')
  const deny = overrides.deny.filter(key => key !== 'attachment.upload')
  if (mode === 'allow') allow.push('attachment.upload')
  if (mode === 'deny') deny.push('attachment.upload')
  return { allow, deny }
}

export function uploadLimitMaxMB(siteBytes: number, transportBytes: number) {
  return Math.max(1, Math.floor(Math.min(siteBytes, transportBytes) / (1024 * 1024)))
}

export function formatUploadLimit(bytes: number) {
  const mb = bytes / (1024 * 1024)
  return `${Number.isInteger(mb) ? mb : mb.toFixed(1)} MB`
}
