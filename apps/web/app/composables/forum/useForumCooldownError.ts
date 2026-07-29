import {
  apiErrorReason,
  apiErrorRetryAfterSeconds,
  apiErrorRetryAt
} from '~/composables/useApiClient'

type ForumCooldownKind = 'topic' | 'comment'

export function useForumCooldownError(kind: ForumCooldownKind) {
  const { t } = useI18n()
  const retryAtMs = ref(0)
  const clockMs = ref(Date.now())
  let timer: ReturnType<typeof setInterval> | undefined

  const reason = kind === 'topic' ? 'forum.topic_cooldown' : 'forum.comment_cooldown'
  const messageKey = kind === 'topic' ? 'composer.topicCooldown' : 'topicDetail.commentCooldown'
  const retryAfterSeconds = computed(() => Math.max(0, Math.ceil((retryAtMs.value - clockMs.value) / 1000)))
  const active = computed(() => retryAfterSeconds.value > 0)
  const message = computed(() => active.value
    ? t(messageKey, { seconds: retryAfterSeconds.value })
    : '')

  function stopTimer() {
    if (timer) {
      clearInterval(timer)
      timer = undefined
    }
  }

  function tick() {
    clockMs.value = Date.now()
    if (retryAtMs.value <= clockMs.value) {
      retryAtMs.value = 0
      stopTimer()
    }
  }

  function capture(error: unknown) {
    if (apiErrorReason(error) !== reason) {
      return false
    }

    const now = Date.now()
    const absoluteRetryAt = Date.parse(apiErrorRetryAt(error))
    const retrySeconds = apiErrorRetryAfterSeconds(error)
    retryAtMs.value = Number.isFinite(absoluteRetryAt) && absoluteRetryAt > now
      ? absoluteRetryAt
      : now + retrySeconds * 1000
    clockMs.value = now

    if (retryAtMs.value <= now) {
      return false
    }
    stopTimer()
    timer = setInterval(tick, 1000)
    return true
  }

  function clear() {
    retryAtMs.value = 0
    stopTimer()
  }

  onBeforeUnmount(stopTimer)

  return { active, message, retryAfterSeconds, capture, clear }
}
