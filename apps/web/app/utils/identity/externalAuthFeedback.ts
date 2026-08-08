/**
 * Host OAuth 回调通过安全本地 302 携带最小化 `ext_auth` reason。
 * 浏览器只渲染稳定 reason 的本地化文案，绝不回显 code/token/state/subject。
 */

export type ExternalAuthFeedbackKind = 'success' | 'info' | 'error'

export type ExternalAuthFeedback = {
  reason: string
  kind: ExternalAuthFeedbackKind
  /** i18n key under auth.external.reasons.* or auth.external.success.* */
  messageKey: string
  /** 成功态优先 Toast；错误/提示在登录注册壳用 SFAlert 更稳。 */
  preferToast: boolean
}

export type ExternalAuthFeedbackSurface = 'auth' | 'global'

const SUCCESS_REASONS = new Set([
  'auth.external_login_ok',
  'auth.external_link_ok'
])

const INFO_REASONS = new Set([
  'auth.provider_cancelled',
  'auth.external_identity_unlinked'
])

/** 已知稳定 reason → 文案 key（缺省走 generic）。 */
const REASON_MESSAGE_KEYS: Record<string, string> = {
  'auth.external_login_ok': 'auth.external.success.loginOk',
  'auth.external_link_ok': 'auth.external.success.linkOk',
  'auth.provider_cancelled': 'auth.external.reasons.providerCancelled',
  'auth.provider_not_enabled': 'auth.external.reasons.providerNotEnabled',
  'auth.provider_not_found': 'auth.external.reasons.providerNotFound',
  'auth.provider_unavailable': 'auth.external.reasons.providerUnavailable',
  'auth.provider_callback_expired': 'auth.external.reasons.callbackExpired',
  'auth.provider_callback_invalid': 'auth.external.reasons.callbackInvalid',
  'auth.provider_callback_replayed': 'auth.external.reasons.callbackReplayed',
  'auth.external_identity_unlinked': 'auth.external.reasons.identityUnlinked',
  'auth.external_subject_conflict': 'auth.external.reasons.subjectConflict',
  'auth.external_link_conflict': 'auth.external.reasons.linkConflict',
  'auth.external_registration_ticket_invalid': 'auth.external.reasons.ticketInvalid',
  'auth.external_registration_ticket_expired': 'auth.external.reasons.ticketExpired',
  'auth.external_bootstrap_required': 'auth.external.reasons.bootstrapRequired',
  'auth.registration_disabled': 'auth.external.reasons.registrationDisabled',
  'auth.required': 'auth.external.reasons.authRequired',
  'auth.recent_auth_required': 'auth.external.reasons.recentAuthRequired',
  'auth.last_login_method_required': 'auth.external.reasons.lastLoginMethodRequired',
  // start 超限返回 JSON rate_limit.exceeded；callback 超限映射为 provider_unavailable。
  'auth.provider_rate_limited': 'auth.external.reasons.providerUnavailable'
}

export function normalizeExtAuthReason(raw: unknown): string {
  if (typeof raw !== 'string') {
    return ''
  }
  const reason = raw.trim()
  // 仅接受 Host 风格的稳定 reason，拒绝任意 query 注入长串。
  if (!reason || reason.length > 120 || !/^auth\.[a-z0-9_.]+$/i.test(reason)) {
    return ''
  }
  return reason
}

export function resolveExternalAuthFeedback(raw: unknown): ExternalAuthFeedback | null {
  const reason = normalizeExtAuthReason(raw)
  if (!reason) {
    return null
  }

  const kind: ExternalAuthFeedbackKind = SUCCESS_REASONS.has(reason)
    ? 'success'
    : INFO_REASONS.has(reason)
      ? 'info'
      : 'error'

  return {
    reason,
    kind,
    messageKey: REASON_MESSAGE_KEYS[reason] || 'auth.external.reasons.generic',
    preferToast: kind === 'success'
  }
}

export function externalAuthFeedbackDelivery(
  item: ExternalAuthFeedback,
  surface: ExternalAuthFeedbackSurface
): 'alert' | 'toast' {
  return item.kind === 'success' || (surface === 'auth' && item.preferToast) ? 'toast' : 'alert'
}

export function externalAuthFeedbackToastDuration(item: ExternalAuthFeedback): number {
  return item.kind === 'error' ? 0 : 10000
}

export function externalAuthFeedbackUsesInlineSurface(path: string): boolean {
  const normalized = path.length > 1 ? path.replace(/\/+$/, '') : path
  return normalized === '/login' || normalized === '/register'
}

/** 从 query 中去掉 ext_auth，保留其余参数（ticket/redirect 等）。 */
export function stripExtAuthQuery(
  query: Record<string, unknown>
): Record<string, unknown> {
  const next: Record<string, unknown> = { ...query }
  delete next.ext_auth
  return next
}

export function isExternalAuthSuccessReason(reason: string): boolean {
  return SUCCESS_REASONS.has(reason)
}
