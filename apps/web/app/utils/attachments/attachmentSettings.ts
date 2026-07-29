// Core 驱动 id 或 plugin:<extensionId>（E6.1）。
export type AttachmentProvider = string

export type AttachmentStorageCandidate = {
  value: string
  kind: 'core' | 'plugin'
  label: string
  extensionId?: string
  settingsPath?: string
  available: boolean
}

export type AttachmentSettings = {
  providerSlot?: string
  drivers?: string[]
  candidates?: AttachmentStorageCandidate[]
  provider: AttachmentProvider
  uploadEnabled: boolean
  pathTemplate: string
  publicBaseUrl: string
  maxFileSizeMb: number
  allowedExtensions: string[]
  allowedMimeTypes: string[]
  defaultVisibility: 'public' | 'private'
  cleanupOrphanAfterDays: number
  local: { root: string, publicPrefix: string }
}

const defaultCoreCandidates: AttachmentStorageCandidate[] = [
  { value: 'local', kind: 'core', label: 'Local filesystem', available: true }
]

export function createDefaultAttachmentSettings(): AttachmentSettings {
  return {
    providerSlot: 'attachment.storage.provider',
    drivers: ['local'],
    candidates: defaultCoreCandidates.map(item => ({ ...item })),
    provider: 'local',
    uploadEnabled: true,
    pathTemplate: '{yyyy}/{mm}/{dd}/{public_id}{ext}',
    publicBaseUrl: '',
    maxFileSizeMb: 20,
    allowedExtensions: ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.pdf', '.txt', '.zip'],
    allowedMimeTypes: ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'application/pdf', 'text/plain', 'application/zip'],
    defaultVisibility: 'public',
    cleanupOrphanAfterDays: 30,
    local: { root: 'storage/app/attachments', publicPrefix: '' }
  }
}

export function isPluginAttachmentProvider(provider: string) {
  return provider.trim().startsWith('plugin:')
}

export function resetAttachmentSettingsToRecommended(current?: AttachmentSettings): AttachmentSettings {
  const defaults = createDefaultAttachmentSettings()
  if (!current) {
    return defaults
  }

  return defaults
}

export function isRecommendedAttachmentSettings(settings: AttachmentSettings) {
  const defaults = createDefaultAttachmentSettings()
  return settings.provider === defaults.provider
    && settings.uploadEnabled === defaults.uploadEnabled
    && settings.pathTemplate === defaults.pathTemplate
    && settings.publicBaseUrl === defaults.publicBaseUrl
    && settings.maxFileSizeMb === defaults.maxFileSizeMb
    && settings.defaultVisibility === defaults.defaultVisibility
    && settings.cleanupOrphanAfterDays === defaults.cleanupOrphanAfterDays
    && settings.local.root === defaults.local.root
    && settings.local.publicPrefix === defaults.local.publicPrefix
    && sameList(settings.allowedExtensions, defaults.allowedExtensions)
    && sameList(settings.allowedMimeTypes, defaults.allowedMimeTypes)
}

export function splitAttachmentSettingList(value: string) {
  return value.split(',').map(item => item.trim()).filter(Boolean)
}

function sameList(left: string[], right: string[]) {
  if (left.length !== right.length) {
    return false
  }
  return left.every((item, index) => item === right[index])
}
