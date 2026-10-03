// 通用「修订号流」客户端：SSE 只负责唤醒，REST/本地对账负责正确性。
//
// 抽自通知实时客户端（原 notificationRealtime.ts），两处消费同一套机制：
//   - 每浏览器每协调键只保留一条 EventSource（BroadcastChannel + Web Locks 选举）；
//   - 断线按 1s → 30s 退避重连，open/error/事件帧都会触发合并刷新（100ms 去抖）；
//   - 标签页不可见时暂停连接与兜底轮询，回到前台立刻补一帧；
//   - 兜底轮询保证「流全挂」时页面仍会收敛，只是慢一点。
//
// 该模块不依赖 Nuxt/DOM 全局：宿主环境通过 RevisionStreamEnvironment 注入，便于单测。

export type RevisionStreamEnvironment = {
  createEventSource(url: string): RevisionEventSourcePort
  eventSourceClosed: number
  coordinationAvailable(): boolean
  createBroadcastChannel(name: string): RevisionBroadcastPort
  requestExclusiveLock(name: string, signal: AbortSignal, callback: () => Promise<void>): Promise<void>
  isVisible(): boolean
  addVisibilityListener(listener: () => void): void
  removeVisibilityListener(listener: () => void): void
}

export type RevisionEventSourcePort = {
  readonly readyState: number
  addEventListener(type: string, listener: EventListenerOrEventListenerObject): void
  close(): void
}

export type RevisionBroadcastPort = {
  onmessage: ((event: MessageEvent<unknown>) => void) | null
  postMessage(message: unknown): void
  close(): void
}

export type RevisionState = { value: number }

export type RevisionStreamSubscription = {
  /** 跨标签页协调键：同时用作 BroadcastChannel 名与 Web Lock 名。 */
  coordinationKey: string
  /** 按当前已知修订号构造 SSE 地址。 */
  streamUrl: (revision: number) => string
  /** 服务端事件名。 */
  eventName: string
  /** 本地已知修订号；收到更大的远端修订号时会被回写。 */
  revision: RevisionState
  /** 解析事件负载中的修订号；返回 null 表示忽略该帧。 */
  extractRevision: (data: unknown) => number | null
  /**
   * 变更合并后的刷新回调。
   * SSE 事件帧会把解析后的负载透传进来（调用方可省掉一次对账请求）；
   * 连接建立/断开与兜底轮询不带参数。
   */
  refresh: (payload?: unknown) => void | Promise<void>
  /** 兜底轮询间隔（毫秒）。 */
  fallbackIntervalMs: number
}

type Subscriber = Pick<RevisionStreamSubscription, 'revision' | 'refresh'>
type RealtimeMessage =
  | { type: 'revision', scope: string, revision: number }
  | { type: 'refresh', scope: string }

const RECONNECT_INITIAL_DELAY_MS = 1000
const RECONNECT_MAX_DELAY_MS = 30_000
const REFRESH_COALESCE_MS = 100

export function defaultRevisionStreamEnvironment(): RevisionStreamEnvironment {
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

export function normalizeRevisionValue(value: unknown): number {
  return Number.isSafeInteger(value) && Number(value) >= 0 ? Number(value) : 0
}

function isRealtimeMessage(value: unknown, scope: string): value is RealtimeMessage {
  if (!value || typeof value !== 'object') return false
  const message = value as Partial<RealtimeMessage>
  if (message.scope !== scope) return false
  if (message.type === 'refresh') return true
  return message.type === 'revision'
    && Number.isSafeInteger(message.revision)
    && Number(message.revision) >= 0
}

function createRuntime(
  environment: RevisionStreamEnvironment,
  subscription: RevisionStreamSubscription,
  onEmpty: () => void
) {
  const scope = subscription.coordinationKey
  const subscribers = new Set<Subscriber>()
  let currentRevision = normalizeRevisionValue(subscription.revision.value)
  let source: RevisionEventSourcePort | null = null
  let refreshTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let fallbackTimer: ReturnType<typeof setInterval> | null = null
  let reconnectDelayMs = RECONNECT_INITIAL_DELAY_MS
  let broadcast: RevisionBroadcastPort | null = null
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

  function scheduleRefresh(payload?: unknown) {
    if (refreshTimer || stopped) return
    refreshTimer = setTimeout(() => {
      refreshTimer = null
      for (const subscriber of subscribers) {
        void Promise.resolve(subscriber.refresh(payload)).catch(() => {})
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

    let nextSource: RevisionEventSourcePort
    try {
      nextSource = environment.createEventSource(subscription.streamUrl(currentRevision))
      source = nextSource
    } catch {
      scheduleReconnect()
      return
    }

    nextSource.addEventListener('open', () => {
      if (source !== nextSource) return
      reconnectDelayMs = RECONNECT_INITIAL_DELAY_MS
      scheduleRefresh()
      publish({ type: 'refresh', scope })
    })
    nextSource.addEventListener('error', () => {
      scheduleRefresh()
      publish({ type: 'refresh', scope })
      nextSource.close()
      if (source !== nextSource) return
      source = null
      scheduleReconnect()
    })
    nextSource.addEventListener(subscription.eventName, (event) => {
      let payload: unknown
      try {
        payload = JSON.parse((event as MessageEvent<string>).data)
      } catch {
        // 畸形或旧服务端事件不影响可见页的 REST 对账。
        return
      }
      const revision = subscription.extractRevision(payload)
      if (revision === null || revision <= currentRevision) return
      setRevision(revision)
      scheduleRefresh(payload)
      publish({ type: 'revision', scope, revision })
    })
  }

  function refreshWhenVisible() {
    if (!environment.isVisible()) return
    scheduleRefresh()
    connect()
  }

  function startFallbacks() {
    fallbackTimer = setInterval(refreshWhenVisible, subscription.fallbackIntervalMs)
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
    try {
      broadcast = environment.createBroadcastChannel(scope)
      broadcast.onmessage = (event) => {
        if (!isRealtimeMessage(event.data, scope)) return
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
    void environment.requestExclusiveLock(scope, lockAbort.signal, async () => {
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
      const nextRevision = normalizeRevisionValue(next.revision.value)
      if (nextRevision > currentRevision) {
        setRevision(nextRevision)
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

export function createRevisionStreamClient(environment: RevisionStreamEnvironment = defaultRevisionStreamEnvironment()) {
  const runtimes = new Map<string, ReturnType<typeof createRuntime>>()

  function subscribe(subscription: RevisionStreamSubscription) {
    if (!subscription.coordinationKey) return () => {}
    const key = subscription.coordinationKey
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
