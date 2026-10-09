import type { AvatarView } from '~/composables/profile/useProfileApi'
import type { SForumMentionSuggestionItem } from '~/utils/editor/mentions'

type ComposerReferenceOption = {
  id: number
  label: string
  secondaryLabel?: string
  avatar?: AvatarView
}

type ComposerReferenceOptionList = {
  items: ComposerReferenceOption[]
  hasMore?: boolean
}

/** 提及候选一次请求的最大条数（后端 ReferenceSelectorMaxLimit=20 内）。 */
const MENTION_SUGGESTION_LIMIT = 8

/**
 * 论坛编辑器 @ 候选：复用 composer 引用选择器用户查询（kind=user），
 * 登录后可用；失败/未登录一律回落为空列表，交互层只做静默降级。
 * 防抖与“至少 1 个字符”由编辑器 suggestion 插件负责，这里只管单次请求与字段映射。
 */
export function useMentionSuggestions() {
  const { request } = useApiClient()

  async function fetchMentionSuggestions(query: string): Promise<SForumMentionSuggestionItem[]> {
    const trimmed = query.trim()
    if (!trimmed) {
      return []
    }
    const params = new URLSearchParams({
      kind: 'user',
      query: trimmed.slice(0, 64),
      limit: `${MENTION_SUGGESTION_LIMIT}`
    })
    const result = await request<ComposerReferenceOptionList>(`/composer/references?${params}`)
    return (result.items || [])
      .map((option): SForumMentionSuggestionItem | null => {
        // secondaryLabel 形如 "@username"；username 是提及写入正文的唯一权威标识。
        const username = (option.secondaryLabel || '').replace(/^@/, '').trim()
        if (!username) {
          return null
        }
        const item: SForumMentionSuggestionItem = {
          id: option.id,
          username,
          displayName: option.label || username
        }
        if (option.avatar?.url) {
          item.avatarUrl = option.avatar.url
          item.avatarAlt = option.avatar.alt
        }
        return item
      })
      .filter((item): item is SForumMentionSuggestionItem => item !== null)
  }

  return { fetchMentionSuggestions }
}
