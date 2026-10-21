import { describe, expect, test } from 'bun:test'
import {
  joinCooldownMinutes,
  nicknameCountdown,
  nicknameDraftConflict,
  nicknameRemainingMs,
  normalizeNicknameDraft,
  splitCooldownMinutes
} from '../../app/utils/settings/nicknamePolicy'
import { recommendedNicknameSettings, resolveNicknameSettings } from '../../app/composables/useWebOptions'

describe('nickname policy', () => {
  test('falls back to recommended defaults when options are missing or invalid', () => {
    expect(resolveNicknameSettings({})).toEqual(recommendedNicknameSettings)

    const dirty = resolveNicknameSettings({
      'profile.nickname.allow_user_change': 'nonsense',
      'profile.nickname.cooldown_minutes': '-5',
      'profile.nickname.min_length': '20',
      'profile.nickname.max_length': '4',
      'profile.nickname.history_public': 'on'
    })
    expect(dirty.allowUserChange).toBe(true)
    expect(dirty.cooldownMinutes).toBe(10080)
    // 上界低于下界时双双回退，不留下无人能通过的区间。
    expect(dirty.minLength).toBe(2)
    expect(dirty.maxLength).toBe(24)
    expect(dirty.historyPublic).toBe(true)
  })

  test('reads configured values', () => {
    const policy = resolveNicknameSettings({
      'profile.nickname.allow_user_change': 'disabled',
      'profile.nickname.cooldown_minutes': '90',
      'profile.nickname.min_length': '1',
      'profile.nickname.max_length': '40',
      'profile.nickname.history_public': 'disabled'
    })
    expect(policy).toEqual({
      allowUserChange: false,
      cooldownMinutes: 90,
      minLength: 1,
      maxLength: 40,
      historyPublic: false
    })
  })

  test('collapses whitespace the same way the server does', () => {
    expect(normalizeNicknameDraft('  Alice   Smith  ')).toBe('Alice Smith')
    expect(normalizeNicknameDraft('Alice\u00a0Smith')).toBe('Alice Smith')
    expect(normalizeNicknameDraft('Alice\u3000Smith')).toBe('Alice Smith')
    expect(normalizeNicknameDraft('   ')).toBe('')
  })

  test('reports draft conflicts without pretending to be authoritative', () => {
    const policy = recommendedNicknameSettings
    expect(nicknameDraftConflict('', policy)).toBeNull()
    expect(nicknameDraftConflict('   ', policy)).toBeNull()
    expect(nicknameDraftConflict('a', policy)).toBe('too_short')
    expect(nicknameDraftConflict('名'.repeat(25), policy)).toBe('too_long')
    expect(nicknameDraftConflict('Ali\nce', policy)).toBe('invalid_chars')
    expect(nicknameDraftConflict('Ali\u200bce', policy)).toBe('invalid_chars')
    expect(nicknameDraftConflict('爱丽丝', policy)).toBeNull()
    expect(nicknameDraftConflict('🐱🐱', policy)).toBeNull()
  })

  test('counts down to the minute', () => {
    const now = Date.parse('2026-10-20T00:00:00Z')
    const next = '2026-10-27T00:00:00Z'
    const remaining = nicknameRemainingMs(next, now)
    expect(remaining).toBe(7 * 24 * 60 * 60 * 1000)

    const countdown = nicknameCountdown(remaining)
    expect(countdown.days).toBe(7)
    expect(countdown.hours).toBe(0)
    expect(countdown.minutes).toBe(0)

    // 最后一分钟向上取整，不会显示成 0 分钟。
    expect(nicknameCountdown(30_000).minutes).toBe(1)
    expect(nicknameRemainingMs(null, now)).toBe(0)
    expect(nicknameRemainingMs('not-a-date', now)).toBe(0)
  })

  test('converts cooldown minutes for the admin control', () => {
    expect(splitCooldownMinutes(10080)).toEqual({ days: 7, hours: 0, minutes: 0 })
    expect(splitCooldownMinutes(90)).toEqual({ days: 0, hours: 1, minutes: 30 })
    expect(splitCooldownMinutes(0)).toEqual({ days: 0, hours: 0, minutes: 0 })
    expect(joinCooldownMinutes({ days: 1, hours: 2, minutes: 3 })).toBe(1563)
    expect(joinCooldownMinutes({ days: 0, hours: 0, minutes: 0 })).toBe(0)
  })
})
