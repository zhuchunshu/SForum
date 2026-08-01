export type NotificationRevisionState = { value: number }
export type NotificationRevisionRefresh = () => void | Promise<void>

type EventSourcePort = {
  readonly readyState: number
  addEventListener(type: string, listener: EventListenerOrEventListenerObject): void
  close(): void
}

type BroadcastPort = {
  onmessage: ((event: MessageEvent<unknown>) => void) | null
  postMessage(message: unknown): void
  close(): void
}

export type NotificationRealtimeEnvironment = {
  createEventSource(url: string): EventSourcePort
  eventSourceClosed: number
  coordinationAvailable(): boolean
  createBroadcastChannel(name: string): BroadcastPort
  requestExclusiveLock(name: string, signal: AbortSignal, callback: () => Promise<void>): Promise<void>
  isVisible(): boolean
  addVisibilityListener(listener: () => void): void
  removeVisibilityListener(listener: () => void): void
}

export type NotificationRealtimeSubscription = {
  actorUserId: number
  apiBaseUrl: string
  revision: NotificationRevisionState
  refresh: NotificationRevisionRefresh
}

type Subscriber = Pick<NotificationRealtimeSubscription, 'revision' | 'refresh'>
type RealtimeMessage =
  | { type: 'revision', actorUserId: number, revision: number }
  | { type: 'refresh', actorUserId: number }

const RECONNECT_INITIAL_DELAY_MS = 1000
const RECONNECT_MAX_DELAY_MS = 30_000
const FALLBACK_REFRESH_MS = 30_000
const REFRESH_COALESCE_MS = 100
const CHANNEL_PREFIX = 'sforum:notifications:revision:user:'

function defaultEnvironment(): NotificationRealtimeEnvironment {
  return {
    createEventSource: url => new EventSource(url, { withCredentials: true }),
    get eventSourceClosed() { return EventSource.CLOSED },
    coordinationAvailable: () => typeof BroadcastChannel !== 'undefined'
      && typeof navigator !== 'undefined'
      && typeof navigator.locks?.request === 'function',
    createBroadcastChannel: name => new BroadcastChannel(name),
    requestExclusiveLock: async (name, signal, callback) => {
      await navigator.locks.request(name, { mode: 'exclusive', signal }, callback)
    },
    isVisible: () => typeof document === 'undefined' || document.visibilityState === 'visible',
    addVisibilityListener: (listener) => {
      if (typeof document !== 'undefined') document.addEventListener('visibilitychange', listener)
    },
    removeVisibilityListener: (listener) => {
      if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', listener)
    }
  }
}

function normalizeBaseURL(value: string) {
  return value.replace(/\/+$/, '')
}

function runtimeKey(actorUserId: number, apiBaseUrl: string) {
  return `${actorUserId}:${normalizeBaseURL(apiBaseUrl)}`
}

function channelName(actorUserId: number) {
  return `${CHANNEL_PREFIX}${actorUserId}`
}

function isRealtimeMessage(value: unknown, actorUserId: number): value is RealtimeMessage {
  if (!value || typeof value !== 'object') return false
  const message = value as Partial<RealtimeMessage>
  if (message.actorUserId !== actorUserId) return false
  if (message.type === 'refresh') return true
  return message.type === 'revision'
    && Number.isSafeInteger(message.revision)
    && Number(message.revision) >= 0
}

function createRuntime(
  environment: NotificationRealtimeEnvironment,
  subscription: NotificationRealtimeSubscription,
  onEmpty: () => void
) {
  const actorUserId = subscription.actorUserId
  const baseURL = normalizeBaseURL(subscription.apiBaseUrl)
  const subscribers = new Set<Subscriber>()
  let currentRevision = Number.isSafeInteger(subscription.revision.value) && subscription.revision.value >= 0
    ? subscription.revision.value
    : 0
  let source: EventSourcePort | null = null
  let refreshTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let fallbackTimer: ReturnType<typeof setInterval> | null = null
  let reconnectDelayMs = RECONNECT_INITIAL_DELAY_MS
  let broadcast: BroadcastPort | null = null
  let lockAbort: AbortController | null = null
  let releaseLeadership: (() => void) | null = null
  let coordinated = false
  let leader = false
  let started = false
  let stopped = false

  function setRevision(revision: number) {
    currentRevision = revision
    for (const subscriber of subscribers) subscriber.revision.value = revision
  }

  function scheduleRefresh() {
    if (refreshTimer || stopped) return
    refreshTimer = setTimeout(() => {
      refreshTimer = null
      for (const subscriber of subscribers) {
        void Promise.resolve(subscriber.refresh()).catch(() => {})
      }
    }, REFRESH_COALESCE_MS)
  }

  function publish(message: RealtimeMessage) {
    try {
      broadcast?.postMessage(message)
    } catch {
      // 跨标签页广播失败时，本标签页仍由 SSE 与 REST 对账维持正确性。
    }
  }

  function closeSource() {
    source?.close()
    source = null
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = null
    reconnectDelayMs = RECONNECT_INITIAL_DELAY_MS
  }

  function ownsConnection() {
    return !coordinated || leader
  }

  function scheduleReconnect() {
    if (stopped || !ownsConnection() || reconnectTimer || subscribers.size === 0) return
    const delay = reconnectDelayMs
    reconnectDelayMs = Math.min(delay * 2, RECONNECT_MAX_DELAY_MS)
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, delay)
  }

  function connect() {
    if (stopped || !ownsConnection() || subscribers.size === 0 || reconnectTimer) return
    if (source && source.readyState !== environment.eventSourceClosed) return
    source?.close()
    source = null

    let nextSource: EventSourcePort
    try {
      nextSource = environment.createEventSource(`${baseURL}/notifications/stream?revision=${currentRevision}`)
      source = nextSource
    } catch {
      scheduleReconnect()
      return
    }

    nextSource.addEventListener('open', () => {
      if (source !== nextSource) return
      reconnectDelayMs = RECONNECT_INITIAL_DELAY_MS
      scheduleRefresh()
      publish({ type: 'refresh', actorUserId })
    })
    nextSource.addEventListener('error', () => {
      scheduleRefresh()
      publish({ type: 'refresh', actorUserId })
      nextSource.close()
      if (source !== nextSource) return
      source = null
      scheduleReconnect()
    })
    nextSource.addEventListener('revision', (event) => {
      try {
        const payload = JSON.parse((event as MessageEvent<string>).data) as { revision?: unknown }
        const revision = Number(payload.revision)
        if (!Number.isSafeInteger(revision) || revision <= currentRevision) return
        setRevision(revision)
        scheduleRefresh()
        publish({ type: 'revision', actorUserId, revision })
      } catch {
        // 畸形或旧服务端事件不影响可见页的 REST 对账。
      }
    })
  }

  function refreshWhenVisible() {
    if (!environment.isVisible()) return
    scheduleRefresh()
    connect()
  }

  function startFallbacks() {
    fallbackTimer = setInterval(refreshWhenVisible, FALLBACK_REFRESH_MS)
    environment.addVisibilityListener(refreshWhenVisible)
  }

  function startDirectConnection() {
    coordinated = false
    broadcast?.close()
    broadcast = null
    connect()
  }

  function startCoordinatedConnection() {
    coordinated = true
    const name = channelName(actorUserId)
    try {
      broadcast = environment.createBroadcastChannel(name)
      broadcast.onmessage = (event) => {
        if (!isRealtimeMessage(event.data, actorUserId)) return
        if (event.data.type === 'revision' && event.data.revision > currentRevision) {
          setRevision(event.data.revision)
        }
        scheduleRefresh()
      }
    } catch {
      startDirectConnection()
      return
    }

    lockAbort = new AbortController()
    void environment.requestExclusiveLock(name, lockAbort.signal, async () => {
      if (stopped || subscribers.size === 0) return
      leader = true
      connect()
      await new Promise<void>((resolve) => { releaseLeadership = resolve })
      releaseLeadership = null
      leader = false
      closeSource()
    }).catch((error: unknown) => {
      if (stopped || (error instanceof Error && error.name === 'AbortError')) return
      startDirectConnection()
    })
  }

  function stop() {
    if (stopped) return
    stopped = true
    releaseLeadership?.()
    releaseLeadership = null
    lockAbort?.abort()
    lockAbort = null
    closeSource()
    if (refreshTimer) clearTimeout(refreshTimer)
    if (fallbackTimer) clearInterval(fallbackTimer)
    refreshTimer = null
    fallbackTimer = null
    environment.removeVisibilityListener(refreshWhenVisible)
    broadcast?.close()
    broadcast = null
  }

  return {
    subscribe(next: Subscriber) {
      subscribers.add(next)
      if (Number.isSafeInteger(next.revision.value) && next.revision.value > currentRevision) {
        setRevision(next.revision.value)
      } else {
        next.revision.value = currentRevision
      }
      if (!started) {
        started = true
        startFallbacks()
        if (environment.coordinationAvailable()) startCoordinatedConnection()
        else startDirectConnection()
      }
      return () => {
        subscribers.delete(next)
        if (subscribers.size > 0) return
        stop()
        onEmpty()
      }
    },
    stop
  }
}

export function createNotificationRealtimeClient(environment: NotificationRealtimeEnvironment = defaultEnvironment()) {
  const runtimes = new Map<string, ReturnType<typeof createRuntime>>()

  function subscribe(subscription: NotificationRealtimeSubscription) {
    if (!Number.isSafeInteger(subscription.actorUserId) || subscription.actorUserId <= 0) return () => {}
    const key = runtimeKey(subscription.actorUserId, subscription.apiBaseUrl)
    let runtime = runtimes.get(key)
    if (!runtime) {
      runtime = createRuntime(environment, subscription, () => runtimes.delete(key))
      runtimes.set(key, runtime)
    }
    return runtime.subscribe({ revision: subscription.revision, refresh: subscription.refresh })
  }

  function stopAll() {
    for (const runtime of runtimes.values()) runtime.stop()
    runtimes.clear()
  }

  return { subscribe, stopAll }
}
