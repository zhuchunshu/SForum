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
  aliyunOss: { endpoint: string, bucket: string, region: string, accessKeyId: string, accessKeySecret?: string, accessKeySecretSet: boolean }
  tencentCos: { region: string, bucket: string, secretId: string, secretKey?: string, secretKeySet: boolean, cdnDomain: string }
  ftp: { host: string, port: number, username: string, password?: string, passwordSet: boolean, rootPath: string, passive: boolean, explicitTls: boolean, publicBaseUrl: string }
  sftp: { host: string, port: number, username: string, password?: string, passwordSet: boolean, privateKey?: string, privateKeySet: boolean, passphrase?: string, passphraseSet: boolean, rootPath: string, hostKeyFingerprint: string, publicBaseUrl: string }
}

const defaultCoreCandidates: AttachmentStorageCandidate[] = [
  { value: 'local', kind: 'core', label: 'Local filesystem', available: true },
  { value: 'aliyun_oss', kind: 'core', label: 'Aliyun OSS', available: true },
  { value: 'tencent_cos', kind: 'core', label: 'Tencent COS', available: true },
  { value: 'ftp', kind: 'core', label: 'FTP', available: true },
  { value: 'sftp', kind: 'core', label: 'SFTP', available: true }
]

export function createDefaultAttachmentSettings(): AttachmentSettings {
  return {
    providerSlot: 'attachment.storage.provider',
    drivers: ['local', 'aliyun_oss', 'tencent_cos', 'ftp', 'sftp'],
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
    local: { root: 'storage/app/attachments', publicPrefix: '' },
    aliyunOss: { endpoint: '', bucket: '', region: '', accessKeyId: '', accessKeySecret: '', accessKeySecretSet: false },
    tencentCos: { region: '', bucket: '', secretId: '', secretKey: '', secretKeySet: false, cdnDomain: '' },
    ftp: { host: '', port: 21, username: '', password: '', passwordSet: false, rootPath: '/', passive: true, explicitTls: false, publicBaseUrl: '' },
    sftp: { host: '', port: 22, username: '', password: '', passwordSet: false, privateKey: '', privateKeySet: false, passphrase: '', passphraseSet: false, rootPath: '/', hostKeyFingerprint: '', publicBaseUrl: '' }
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

  // 恢复推荐配置不主动清空已保存密钥；密钥输入留空时后端会保留原值。
  defaults.aliyunOss.accessKeySecretSet = current.aliyunOss.accessKeySecretSet
  defaults.tencentCos.secretKeySet = current.tencentCos.secretKeySet
  defaults.ftp.passwordSet = current.ftp.passwordSet
  defaults.sftp.passwordSet = current.sftp.passwordSet
  defaults.sftp.privateKeySet = current.sftp.privateKeySet
  defaults.sftp.passphraseSet = current.sftp.passphraseSet
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
