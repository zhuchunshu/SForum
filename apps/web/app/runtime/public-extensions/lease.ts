import type { PublicFrontendComponentDescriptor } from './types'
import {
  parsePublicFrontendDescriptor,
  samePublicFrontendDescriptor
} from './types'

export const PUBLIC_FRONTEND_LEASE_INTERVAL_MS = 5_000

type TimerHandle = ReturnType<typeof setTimeout>

type PublicFrontendLeaseScheduler = {
  set: (callback: () => void, delay: number) => TimerHandle
  clear: (handle: TimerHandle) => void
}

type PublicFrontendLeaseMonitorOptions = {
  intervalMs?: number
  read: (current: PublicFrontendComponentDescriptor) => Promise<unknown>
  onChanged: (next: PublicFrontendComponentDescriptor) => void | Promise<void>
  onUnavailable: () => void | Promise<void>
  scheduler?: PublicFrontendLeaseScheduler
}

export type PublicFrontendLeaseMonitor = {
  start: (descriptor: PublicFrontendComponentDescriptor) => void
  trigger: () => Promise<void>
  stop: () => void
}

/**
 * Browser code is fully trusted, but its Host-issued authority is still leased.
 * A bounded no-store descriptor check restores L1 when trust disappears and
 * remounts only after the exact artifact becomes live again.
 */
export function createPublicFrontendLeaseMonitor(
  options: PublicFrontendLeaseMonitorOptions
): PublicFrontendLeaseMonitor {
  const intervalMs = options.intervalMs ?? PUBLIC_FRONTEND_LEASE_INTERVAL_MS
  if (!Number.isSafeInteger(intervalMs) || intervalMs < 1_000 || intervalMs > 60_000) {
    throw new Error('public frontend lease interval is invalid')
  }
  const scheduler = options.scheduler ?? {
    set: (callback, delay) => setTimeout(callback, delay),
    clear: handle => clearTimeout(handle)
  }
  let current: PublicFrontendComponentDescriptor | undefined
  let timer: TimerHandle | undefined
  let checking: Promise<void> | undefined
  let active = false
  let unavailable = false

  function clearTimer() {
    if (timer === undefined) return
    scheduler.clear(timer)
    timer = undefined
  }

  function schedule() {
    clearTimer()
    if (!active || !current) return
    timer = scheduler.set(() => {
      timer = undefined
      void trigger()
    }, intervalMs)
  }

  async function check() {
    const expected = current
    if (!active || !expected) return
    let next: PublicFrontendComponentDescriptor
    try {
      const raw = await options.read(expected)
      next = parsePublicFrontendDescriptor(raw, expected.extensionId, expected.componentId)
    } catch {
      if (!active || current !== expected) return
      if (!unavailable) {
        unavailable = true
        await options.onUnavailable()
      }
      return
    } finally {
      if (active) schedule()
    }
    if (!active || current !== expected) return
    if (unavailable || !samePublicFrontendDescriptor(expected, next)) {
      unavailable = false
      await options.onChanged(next)
    }
  }

  function trigger() {
    clearTimer()
    if (!active || !current) return Promise.resolve()
    if (!checking) {
      checking = check().finally(() => {
        checking = undefined
      })
    }
    return checking
  }

  return {
    start(descriptor) {
      current = descriptor
      unavailable = false
      active = true
      schedule()
    },
    trigger,
    stop() {
      active = false
      current = undefined
      unavailable = false
      clearTimer()
    }
  }
}
