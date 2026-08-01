import { describe, expect, test } from 'bun:test'

import {
  createNotificationRealtimeClient,
  type NotificationRealtimeEnvironment
} from '../../app/composables/notifications/notificationRealtime'

type LockRequest = {
  signal: AbortSignal
  callback: () => Promise<void>
  resolve: () => void
  reject: (error: Error) => void
}

class FakeLockHub {
  private readonly active = new Set<string>()
  private readonly queues = new Map<string, LockRequest[]>()

  request(name: string, signal: AbortSignal, callback: () => Promise<void>) {
    return new Promise<void>((resolve, reject) => {
      const request = { signal, callback, resolve, reject }
      const queue = this.queues.get(name) || []
      queue.push(request)
      this.queues.set(name, queue)
      signal.addEventListener('abort', () => {
        const pending = this.queues.get(name)
        if (!pending?.includes(request)) return
        this.queues.set(name, pending.filter(item => item !== request))
        reject(new DOMException('aborted', 'AbortError'))
      }, { once: true })
      this.pump(name)
    })
  }

  private pump(name: string) {
    if (this.active.has(name)) return
    const queue = this.queues.get(name)
    const request = queue?.shift()
    if (!request) return
    if (request.signal.aborted) {
      request.reject(new DOMException('aborted', 'AbortError'))
      this.pump(name)
      return
    }
    this.active.add(name)
    void request.callback().then(() => {
      this.active.delete(name)
      request.resolve()
      this.pump(name)
    }, (error: Error) => {
      this.active.delete(name)
      request.reject(error)
      this.pump(name)
    })
  }
}

class FakeBroadcastHub {
  private readonly ports = new Map<string, Set<FakeBroadcastPort>>()

  create(name: string) {
    const port = new FakeBroadcastPort(name, this)
    const group = this.ports.get(name) || new Set()
    group.add(port)
    this.ports.set(name, group)
    return port
  }

  publish(sender: FakeBroadcastPort, message: unknown) {
    for (const port of this.ports.get(sender.name) || []) {
      if (port !== sender) port.onmessage?.({ data: message } as MessageEvent<unknown>)
    }
  }

  close(port: FakeBroadcastPort) {
    this.ports.get(port.name)?.delete(port)
  }
}

class FakeBroadcastPort {
  onmessage: ((event: MessageEvent<unknown>) => void) | null = null

  constructor(readonly name: string, private readonly hub: FakeBroadcastHub) {}

  postMessage(message: unknown) { this.hub.publish(this, message) }
  close() { this.hub.close(this) }
}

class FakeEventSource {
  static readonly CLOSED = 2
  readyState = 1
  readonly listeners = new Map<string, Array<(event: Event) => void>>()

  constructor(readonly tab: string, readonly url: string) {}

  addEventListener(type: string, listener: EventListenerOrEventListenerObject) {
    const callback = typeof listener === 'function'
      ? listener
      : (event: Event) => listener.handleEvent(event)
    this.listeners.set(type, [...(this.listeners.get(type) || []), callback])
  }

  emitRevision(revision: number) {
    const event = { data: JSON.stringify({ revision }) } as MessageEvent<string>
    for (const listener of this.listeners.get('revision') || []) listener(event)
  }

  close() { this.readyState = FakeEventSource.CLOSED }
}

function createHarness() {
  const locks = new FakeLockHub()
  const broadcasts = new FakeBroadcastHub()
  const sources: FakeEventSource[] = []

  function environment(tab: string): NotificationRealtimeEnvironment {
    return {
      createEventSource(url) {
        const source = new FakeEventSource(tab, url)
        sources.push(source)
        return source
      },
      eventSourceClosed: FakeEventSource.CLOSED,
      coordinationAvailable: () => true,
      createBroadcastChannel: name => broadcasts.create(name),
      requestExclusiveLock: (name, signal, callback) => locks.request(name, signal, callback),
      isVisible: () => true,
      addVisibilityListener: () => {},
      removeVisibilityListener: () => {}
    }
  }

  return { environment, sources }
}

const settleLeadership = () => new Promise(resolve => setTimeout(resolve, 0))
const settleRefresh = () => new Promise(resolve => setTimeout(resolve, 130))

describe('notification realtime cross-tab coordination', () => {
  test('uses one EventSource for many tabs and broadcasts recipient revisions', async () => {
    const harness = createHarness()
    const clients = Array.from({ length: 6 }, (_, index) => (
      createNotificationRealtimeClient(harness.environment(`tab-${index + 1}`))
    ))
    const revisions = clients.map(() => ({ value: 0 }))
    const refreshes = clients.map(() => 0)
    const stops = clients.map((client, index) => client.subscribe({
      actorUserId: 42,
      apiBaseUrl: '/api/v1',
      revision: revisions[index]!,
      refresh: () => { refreshes[index]++ }
    }))

    expect(harness.sources).toHaveLength(1)
    harness.sources[0]!.emitRevision(7)
    await settleRefresh()

    expect(revisions.map(item => item.value)).toEqual([7, 7, 7, 7, 7, 7])
    expect(refreshes).toEqual([1, 1, 1, 1, 1, 1])
    stops.forEach(stop => stop())
  })

  test('hands leadership to a waiting tab after the leader closes', async () => {
    const harness = createHarness()
    const first = createNotificationRealtimeClient(harness.environment('first'))
    const second = createNotificationRealtimeClient(harness.environment('second'))
    const firstRevision = { value: 0 }
    const secondRevision = { value: 0 }
    const stopFirst = first.subscribe({ actorUserId: 9, apiBaseUrl: '/api/v1', revision: firstRevision, refresh: () => {} })
    const stopSecond = second.subscribe({ actorUserId: 9, apiBaseUrl: '/api/v1', revision: secondRevision, refresh: () => {} })

    harness.sources[0]!.emitRevision(11)
    expect(secondRevision.value).toBe(11)
    stopFirst()
    await settleLeadership()

    expect(harness.sources).toHaveLength(2)
    expect(harness.sources[0]!.readyState).toBe(FakeEventSource.CLOSED)
    expect(harness.sources[1]!.tab).toBe('second')
    expect(harness.sources[1]!.url).toBe('/api/v1/notifications/stream?revision=11')
    stopSecond()
  })

  test('isolates locks and broadcasts by authenticated user id', async () => {
    const harness = createHarness()
    const first = createNotificationRealtimeClient(harness.environment('user-one'))
    const second = createNotificationRealtimeClient(harness.environment('user-two'))
    const firstRevision = { value: 0 }
    const secondRevision = { value: 0 }
    const stopFirst = first.subscribe({ actorUserId: 1, apiBaseUrl: '/api/v1', revision: firstRevision, refresh: () => {} })
    const stopSecond = second.subscribe({ actorUserId: 2, apiBaseUrl: '/api/v1', revision: secondRevision, refresh: () => {} })

    expect(harness.sources).toHaveLength(2)
    harness.sources.find(source => source.tab === 'user-one')!.emitRevision(5)
    expect(firstRevision.value).toBe(5)
    expect(secondRevision.value).toBe(0)
    stopFirst()
    stopSecond()
  })
})
