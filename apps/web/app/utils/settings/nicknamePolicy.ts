import type { NicknameSettings } from '~/composables/useWebOptions'

/**
 * 昵称草稿的本地预检。
 *
 * 这里只做「格式与长度」这一类纯函数判断：保留词、撞用户名、冷却都必须由
 * 服务端裁决（前端拿不到保留词列表，也不该拿）。前端校验的职责是让用户在
 * 提交前看到明显错误，而不是替服务端做决定。
 */
export type NicknameDraftConflict = 'invalid_chars' | 'too_short' | 'too_long' | null

// 与服务端一致：Cc（控制符）与 Cf（零宽/方向标记）都拒绝。
const forbiddenNicknameChars = /[\p{Cc}\p{Cf}]/u

/** 折叠空白：连续空白（含全角空格与 NBSP）归一成半角单空格。 */
export function normalizeNicknameDraft(value: string): string {
  return value
    .trim()
    .split(/\s+/u)
    .filter(Boolean)
    .join(' ')
}

export function nicknameDraftConflict(value: string, policy: NicknameSettings): NicknameDraftConflict {
  if (forbiddenNicknameChars.test(value)) {
    return 'invalid_chars'
  }
  const normalized = normalizeNicknameDraft(value)
  // 空值表示「显示用户名」，不参与长度校验。
  if (!normalized) {
    return null
  }
  const length = Array.from(normalized).length
  if (length < policy.minLength) {
    return 'too_short'
  }
  if (length > policy.maxLength) {
    return 'too_long'
  }
  return null
}

export type NicknameCountdown = {
  totalMinutes: number
  days: number
  hours: number
  minutes: number
}

/** 距离下次可改名的剩余毫秒；解析失败或已过期都返回 0。 */
export function nicknameRemainingMs(nextChangeAt: string | null | undefined, nowMs: number): number {
  const target = Date.parse(nextChangeAt || '')
  if (!Number.isFinite(target)) {
    return 0
  }
  return Math.max(0, target - nowMs)
}

/** 精确到分钟：向上取整，避免最后一分钟显示成 0。 */
export function nicknameCountdown(remainingMs: number): NicknameCountdown {
  const totalMinutes = Math.max(0, Math.ceil(remainingMs / 60000))
  return {
    totalMinutes,
    days: Math.floor(totalMinutes / 1440),
    hours: Math.floor((totalMinutes % 1440) / 60),
    minutes: totalMinutes % 60
  }
}

/** 后台策略控件：分钟 ↔ 天/时/分。 */
export function splitCooldownMinutes(totalMinutes: number): { days: number, hours: number, minutes: number } {
  const total = Math.max(0, Math.trunc(totalMinutes))
  return {
    days: Math.floor(total / 1440),
    hours: Math.floor((total % 1440) / 60),
    minutes: total % 60
  }
}

export function joinCooldownMinutes(parts: { days: number, hours: number, minutes: number }): number {
  const days = Math.max(0, Math.trunc(parts.days || 0))
  const hours = Math.max(0, Math.trunc(parts.hours || 0))
  const minutes = Math.max(0, Math.trunc(parts.minutes || 0))
  return days * 1440 + hours * 60 + minutes
}
