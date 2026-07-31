export const SYSTEM_UPDATE_PROMPT_COOLDOWN_MS = 6 * 60 * 60 * 1000

type SystemUpdatePromptRecord = {
  shownAt: number
}

export function isSystemUpdatePromptSuppressed(raw: string | null, now = Date.now()) {
  if (!raw) return false

  try {
    const record = JSON.parse(raw) as Partial<SystemUpdatePromptRecord>
    if (typeof record.shownAt !== 'number' || !Number.isFinite(record.shownAt)) return false
    const elapsed = now - record.shownAt
    return elapsed >= 0 && elapsed < SYSTEM_UPDATE_PROMPT_COOLDOWN_MS
  } catch {
    return false
  }
}

export function systemUpdatePromptRecord(now = Date.now()) {
  return JSON.stringify({ shownAt: now } satisfies SystemUpdatePromptRecord)
}
