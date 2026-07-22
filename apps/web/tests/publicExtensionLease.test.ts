import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'bun:test'
import { Window } from 'happy-dom'

import {
  PUBLIC_FRONTEND_LEASE_INTERVAL_MS,
  createPublicFrontendLeaseMonitor
} from '../app/runtime/public-extensions/lease'
import {
  publicContributionFailureKey,
  publicContributionFailureState,
  recordPublicContributionFailure
} from '../app/runtime/public-extensions/quarantine'
import {
  PUBLIC_FRONTEND_API_VERSION,
  PUBLIC_FRONTEND_SCHEMA_VERSION,
  PUBLIC_FRONTEND_TRUST_NOTICE,
  type PublicFrontendComponentDescriptor
} from '../app/runtime/public-extensions/types'

describe('public extension exact descriptor lease', () => {
  it('detects an exact package upgrade within the bounded interval', async () => {
    const current = descriptorFixture('1.0.0', 'a', 'b')
    const upgraded = descriptorFixture('1.1.0', 'c', 'd')
    const scheduler = fakeScheduler()
    const changed: PublicFrontendComponentDescriptor[] = []
    const monitor = createPublicFrontendLeaseMonitor({
      read: async () => upgraded,
      onChanged: next => { changed.push(next) },
      onUnavailable: () => { throw new Error('upgrade must remain available') },
      scheduler
    })

    monitor.start(current)
    expect(scheduler.delays).toEqual([PUBLIC_FRONTEND_LEASE_INTERVAL_MS])
    await monitor.trigger()
    expect(changed).toHaveLength(1)
    expect(changed[0]?.packageDigest).toBe('c'.repeat(64))
    monitor.stop()
  })

  it('unmounts once on revoke and remounts when the same exact lease returns', async () => {
    const descriptor = descriptorFixture('1.0.0', 'a', 'b')
    const scheduler = fakeScheduler()
    let reads = 0
    let unavailable = 0
    let restored = 0
    const monitor = createPublicFrontendLeaseMonitor({
      read: async () => {
        reads++
        if (reads < 3) throw new Error('revoked')
        return descriptor
      },
      onChanged: () => { restored++ },
      onUnavailable: () => { unavailable++ },
      scheduler
    })

    monitor.start(descriptor)
    await monitor.trigger()
    await monitor.trigger()
    expect(unavailable).toBe(1)
    expect(restored).toBe(0)
    await monitor.trigger()
    expect(unavailable).toBe(1)
    expect(restored).toBe(1)
    monitor.stop()
  })

  it('ignores a late failure from a replaced descriptor lease', async () => {
    const current = descriptorFixture('1.0.0', 'a', 'b')
    const upgraded = descriptorFixture('1.1.0', 'c', 'd')
    const scheduler = fakeScheduler()
    let rejectCurrent: ((reason?: unknown) => void) | undefined
    const currentRead = new Promise<unknown>((_resolve, reject) => { rejectCurrent = reject })
    let reads = 0
    let unavailable = 0
    const monitor = createPublicFrontendLeaseMonitor({
      read: async () => {
        reads++
        return reads === 1 ? currentRead : upgraded
      },
      onChanged: () => {},
      onUnavailable: () => { unavailable++ },
      scheduler
    })

    monitor.start(current)
    const staleCheck = monitor.trigger()
    monitor.start(upgraded)
    rejectCurrent?.(new Error('old lease revoked'))
    await staleCheck
    expect(unavailable).toBe(0)
    monitor.stop()
  })

  it('keeps the same artifact quarantined but reloads an upgraded impact digest', async () => {
    const current = descriptorFixture('1.0.0', 'a', 'b')
    const upgraded = descriptorFixture('1.1.0', 'c', 'd')
    const storage = new Window().sessionStorage
    const currentKey = publicContributionFailureKey(current.impactDigest, current.extensionId, current.componentId)
    const upgradedKey = publicContributionFailureKey(upgraded.impactDigest, upgraded.extensionId, upgraded.componentId)
    recordPublicContributionFailure(storage, currentKey)
    recordPublicContributionFailure(storage, currentKey)
    recordPublicContributionFailure(storage, currentKey)
    expect(publicContributionFailureState(storage, currentKey).quarantined).toBe(true)

    const scheduler = fakeScheduler()
    let live = current
    let state: 'quarantined' | 'mounted' = 'quarantined'
    let reloads = 0
    const monitor = createPublicFrontendLeaseMonitor({
      read: async () => live,
      onChanged: next => {
        reloads++
        const key = publicContributionFailureKey(next.impactDigest, next.extensionId, next.componentId)
        state = publicContributionFailureState(storage, key).quarantined ? 'quarantined' : 'mounted'
      },
      onUnavailable: () => { throw new Error('artifact must remain available') },
      scheduler
    })

    monitor.start(current)
    await monitor.trigger()
    expect(state).toBe('quarantined')
    expect(reloads).toBe(0)

    live = upgraded
    await monitor.trigger()
    expect(state).toBe('mounted')
    expect(reloads).toBe(1)
    expect(publicContributionFailureState(storage, upgradedKey)).toEqual({ count: 0, quarantined: false })
    expect(publicContributionFailureState(storage, currentKey).quarantined).toBe(true)
    monitor.stop()
  })

  it('binds an unavailable lease to widget cleanup and SSR fallback restoration', () => {
    const source = readFileSync(new URL('../app/components/SFExtensionWidget.vue', import.meta.url), 'utf8')
    expect(source).toContain('onUnavailable: () => enqueue(async () => {')
    expect(source).toContain('await dispose()')
    expect(source).toContain("state.value = 'fallback'")
    expect(source).toContain('v-show="state !== \'mounted\'"')
    expect(source).toContain('publicFrontendRequestOptions(current, options)')
    expect(source).toContain('timeout: PUBLIC_FRONTEND_DESCRIPTOR_TIMEOUT_MS')
    expect(source).toMatch(/publicContributionFailureState\(storage, key\)\.quarantined[\s\S]*state\.value = 'quarantined'[\s\S]*leaseMonitor\.start\(current\)[\s\S]*return/)
  })
})

function fakeScheduler() {
  const delays: number[] = []
  return {
    delays,
    set(_callback: () => void, delay: number) {
      delays.push(delay)
      return 1 as unknown as ReturnType<typeof setTimeout>
    },
    clear(_handle: ReturnType<typeof setTimeout>) {}
  }
}

function descriptorFixture(
  version: string,
  packageCharacter: string,
  impactCharacter: string
): PublicFrontendComponentDescriptor {
  const body = new TextEncoder().encode(`export const version = ${JSON.stringify(version)}`)
  const digest = createHash('sha256').update(body).digest('hex')
  const packageDigest = packageCharacter.repeat(64)
  const impactDigest = impactCharacter.repeat(64)
  const componentId = 'demo.public.component.card'
  const handle = `${componentId}.l2.entry`
  return {
    schemaVersion: PUBLIC_FRONTEND_SCHEMA_VERSION,
    apiVersion: PUBLIC_FRONTEND_API_VERSION,
    trustNotice: PUBLIC_FRONTEND_TRUST_NOTICE,
    extensionId: 'demo.public',
    extensionVersion: version,
    packageDigest,
    impactDigest,
    componentId,
    contractVersion: `${componentId}@1`,
    action: 'add',
    entry: {
      handle,
      contractVersion: `${handle}@1`,
      extensionId: 'demo.public',
      packageDigest,
      impactDigest,
      type: 'script',
      digest,
      integrity: `sha256-${Buffer.from(digest, 'hex').toString('base64')}`,
      dependencies: [],
      scope: [componentId],
      module: true,
      loading: 'lazy',
      csp: [],
      assetPath: `/_sforum/assets/extensions/demo.public/${packageDigest}/frontend/public/card.mjs`
    },
    assets: [],
    csp: []
  }
}
