import {
  apiErrorReason,
  apiErrorRetryAfterSeconds,
  apiErrorRetryAt
} from '~/composables/useApiClient'

type MailResendReason = 'auth.password_reset_rate_limited' | 'auth.email_verification_rate_limited'
type MailResendResponse = { retryAfterSeconds?: number, retryAt?: string }

export function useMailResendCooldown(reason: MailResendReason) {
  const retryAtMs = ref(0)
  const clockMs = ref(Date.now())
  let timer: ReturnType<typeof setInterval> | undefined

  const remainingSeconds = computed(() => Math.max(0, Math.ceil((retryAtMs.value - clockMs.value) / 1000)))
  const active = computed(() => remainingSeconds.value > 0)

  function stopTimer() {
    if (timer) clearInterval(timer)
    timer = undefined
  }

  function tick() {
    clockMs.value = Date.now()
    if (retryAtMs.value <= clockMs.value) {
      retryAtMs.value = 0
      stopTimer()
    }
  }

  function start(response: MailResendResponse) {
    const now = Date.now()
    const absoluteRetryAt = Date.parse(response.retryAt || '')
    const retryAfterSeconds = Number(response.retryAfterSeconds)
    retryAtMs.value = Number.isFinite(absoluteRetryAt) && absoluteRetryAt > now
      ? absoluteRetryAt
      : now + (Number.isFinite(retryAfterSeconds) && retryAfterSeconds > 0 ? Math.ceil(retryAfterSeconds) * 1000 : 0)
    clockMs.value = now
    stopTimer()
    if (retryAtMs.value > now) timer = setInterval(tick, 1000)
  }

  function capture(error: unknown) {
    if (apiErrorReason(error) !== reason) return false
    start({
      retryAt: apiErrorRetryAt(error),
      retryAfterSeconds: apiErrorRetryAfterSeconds(error)
    })
    return active.value
  }

  function clear() {
    retryAtMs.value = 0
    clockMs.value = Date.now()
    stopTimer()
  }

  onBeforeUnmount(stopTimer)

  return { active, remainingSeconds, start, capture, clear }
}
