import type { ForumComment } from '~/utils/forum/forumTaxonomy'
import { formatForumContentTime } from '~/utils/forum/forumContentTime'

export function useForumContentTime() {
  const { t, locale } = useI18n()
  const { settings } = useSiteDateTime()
  const renderedAt = useState<number>('forum-topic-rendered-at', () => Date.now())
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    renderedAt.value = Date.now()
    refreshTimer = setInterval(() => {
      renderedAt.value = Date.now()
    }, 1_000)
  })
  onBeforeUnmount(() => {
    if (refreshTimer) {
      clearInterval(refreshTimer)
    }
  })

  function format(value: string) {
    return formatForumContentTime(value, {
      settings: settings.value,
      locale: String(locale.value || 'zh-CN'),
      now: renderedAt.value
    })
  }

  function publishedTime(value: string) {
    return t('topicDetail.publishedAt', { time: format(value) })
  }

  function updatedTime(value: string) {
    return t('topicDetail.updatedAt', { time: format(value) })
  }

  function commentMeta(comment: ForumComment) {
    const published = publishedTime(comment.createdAt)
    if (comment.edited && comment.editedAt) {
      return `${published} · ${updatedTime(comment.editedAt)}`
    }
    const suffix = comment.edited ? ` · ${t('topicDetail.edited')}` : ''
    return `${published}${suffix}`
  }

  return { publishedTime, updatedTime, commentMeta }
}
