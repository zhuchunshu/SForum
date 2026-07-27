/**
 * Host admin Login Methods 展示辅助（T8B）。
 * label/icon 只消费 Host catalog；禁止按 provider/extension id 猜供应商品牌。
 */

export type AdminIdentityProvider = {
  id: string
  kind: string
  contractVersion?: string
  priority: number
  operations: string[]
  ownerExtensionId: string
  ownerExtensionVersion?: string
  ownerPackageDigest: string
  discovered: boolean
  trusted: boolean
  enabled: boolean
  configured: boolean
  probed: boolean
  artifactBound: boolean
  runtimeInstanceId?: string
  activated: boolean
  publiclyActivated: boolean
  loginEnabled: boolean
  registrationEnabled: boolean
  linkEnabled: boolean
  revision: number
  callbackPath: string
  callbackUrl?: string
  settingsPath?: string
  safeMode: boolean
  /** Host 按 locale 解析的插件声明标签；空则 UI 回退到 owner/id。 */
  label?: string
  /** 插件声明的 Iconify 图标；空则通用钥匙图标。 */
  icon?: string
  lastProbeAt?: string
  lastProbeOk?: boolean
  lastProbeReason?: string
}

export type AdminProviderStateBadge = {
  key: 'discovered' | 'trusted' | 'enabled' | 'configured' | 'probed' | 'publiclyActivated'
  color: 'success' | 'warning' | 'error' | 'neutral' | 'info'
  on: boolean
}

/** 生命周期徽章：与 Host 聚合字段一一对应，不混用语义。 */
export function adminProviderStateBadges(item: AdminIdentityProvider): AdminProviderStateBadge[] {
  return [
    { key: 'discovered', color: 'info', on: item.discovered },
    { key: 'trusted', color: item.trusted ? 'success' : 'neutral', on: item.trusted },
    { key: 'enabled', color: item.enabled ? 'success' : 'neutral', on: item.enabled },
    { key: 'configured', color: item.configured ? 'success' : 'warning', on: item.configured },
    { key: 'probed', color: item.probed ? 'success' : 'neutral', on: item.probed },
    { key: 'publiclyActivated', color: item.publiclyActivated ? 'success' : 'neutral', on: item.publiclyActivated }
  ]
}

export function adminProviderSupportsOp(item: AdminIdentityProvider, op: 'login' | 'registration' | 'link') {
  const prefix = `${op}.`
  return (item.operations || []).some(name => name === op || name.startsWith(prefix))
}

/**
 * 展示标题：仅 Host catalog label；无 label 时回退 ownerExtensionId / id。
 * 禁止 includes('github') 等供应商 id 分支。
 */
export function adminProviderTitle(item: AdminIdentityProvider, fallbackEmpty = ''): string {
  const label = (item.label || '').trim()
  if (label) return label
  const owner = (item.ownerExtensionId || '').trim()
  if (owner) return owner
  const id = (item.id || '').trim()
  if (id) return id
  return fallbackEmpty
}

/**
 * 展示图标：仅 Host catalog icon；无 icon 时通用钥匙。
 * 禁止按 id 猜测 brand-github。
 */
export function adminProviderIcon(item: AdminIdentityProvider): string {
  const icon = (item.icon || '').trim()
  if (icon) return icon
  return 'i-lucide-key-round'
}

export function adminProviderShortDigest(digest: string): string {
  const value = (digest || '').trim()
  if (value.length <= 16) return value || '—'
  return `${value.slice(0, 8)}…${value.slice(-6)}`
}

export type AdminProbeLabelKind = 'never' | 'pending' | 'unavailable' | 'ok' | 'reason'

export function adminProbeLabelKind(item: AdminIdentityProvider): AdminProbeLabelKind {
  const reason = (item.lastProbeReason || '').trim()
  if (!reason) return 'never'
  if (reason === 'probe_pending') return 'pending'
  if (reason === 'probe_unavailable') return 'unavailable'
  if (item.probed) return 'ok'
  return 'reason'
}

/** 探针 Toast/文案：真实 ok 用成功；否则展示 redacted reason（非 brand 猜测）。 */
export function adminProbeFeedback(result: { ok?: boolean, reason?: string, status?: string } | null | undefined): {
  success: boolean
  reason: string
} {
  const ok = Boolean(result?.ok)
  const reason = String(result?.reason || result?.status || '').trim()
  return { success: ok, reason }
}
