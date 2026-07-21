/**
 * 列表呈现用稳定哈希与色板。
 * 主题通过 CSS 变量换肤；宿主组件用 Tailwind 消费 token，不再双写 BEM。
 */

/** demo fullwidth-3col .a1–.a8 */
export const FORUM_AVATAR_TONE_CLASSES = [
  'bg-[#5c7cfa]',
  'bg-[#20c997]',
  'bg-[#fab005]',
  'bg-[#cc5de8]',
  'bg-[#339af0]',
  'bg-[#ff6b6b]',
  'bg-[#845ef7]',
  'bg-[#51cf66]'
] as const

/** hybrid demo variant B .topic-category.tone-* */
export const FORUM_CATEGORY_CHIP_TONE_CLASSES = [
  'bg-[#eaf2ff] text-[#1467d9]',
  'bg-[#edf8ef] text-[#29934d]',
  'bg-[#fff0f0] text-[#dc4b4b]',
  'bg-[#fff4e8] text-[#e1761d]'
] as const

export function hashStableTone(seed: string, modulo: number): number {
  let h = 0
  const source = seed || 'x'
  for (let i = 0; i < source.length; i += 1) {
    h = (h * 31 + source.charCodeAt(i)) >>> 0
  }
  return modulo > 0 ? h % modulo : 0
}

export function forumAvatarToneClass(seed: string): string {
  const index = hashStableTone(seed, FORUM_AVATAR_TONE_CLASSES.length)
  return FORUM_AVATAR_TONE_CLASSES[index] || FORUM_AVATAR_TONE_CLASSES[0]
}

export function forumCategoryChipToneClass(seed: string): string {
  const index = hashStableTone(seed, FORUM_CATEGORY_CHIP_TONE_CLASSES.length)
  return FORUM_CATEGORY_CHIP_TONE_CLASSES[index] || FORUM_CATEGORY_CHIP_TONE_CLASSES[0]
}

/** 扩展列表徽章 tone → Tailwind */
export const FORUM_LIST_BADGE_TONE_CLASSES: Record<string, string> = {
  neutral: 'bg-slate-100 text-slate-600 dark:bg-slate-500/20 dark:text-slate-300',
  info: 'bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
  success: 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300',
  warning: 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
  danger: 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
}

export function forumListBadgeToneClass(tone?: string): string {
  const key = tone && tone in FORUM_LIST_BADGE_TONE_CLASSES ? tone : 'neutral'
  return FORUM_LIST_BADGE_TONE_CLASSES[key] ?? FORUM_LIST_BADGE_TONE_CLASSES.neutral ?? 'bg-slate-100 text-slate-600'
}
