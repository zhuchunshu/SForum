import { describe, expect, it } from 'bun:test'
import { Window } from 'happy-dom'

import { mountPublicFrontendModule } from '../../app/runtime/public-extensions/mount'
import {
  PUBLIC_FRONTEND_API_VERSION,
  PUBLIC_FRONTEND_TRUST_NOTICE,
  PublicFrontendContractError,
  type PublicFrontendBridgeV1
} from '../../app/runtime/public-extensions/types'

describe('public extension mount boundary', () => {
  it('uses returned cleanup exactly once and clears third-party DOM', async () => {
    const { target, bridge } = mountFixture()
    let cleanups = 0
    let unmounts = 0
    const release = await mountPublicFrontendModule({
      apiVersion: 1,
      mount(element) {
        element.append(element.ownerDocument.createElement('button'))
        return () => { cleanups++ }
      },
      unmount() { unmounts++ }
    }, target, bridge)

    expect(target.childElementCount).toBe(1)
    await Promise.all([release(), release()])
    expect(cleanups).toBe(1)
    expect(unmounts).toBe(0)
    expect(target.childElementCount).toBe(0)
  })

  it('calls module unmount after a partial mount throws', async () => {
    const { target, bridge } = mountFixture()
    let unmounts = 0
    const failure = new Error('mount crashed')
    await expect(mountPublicFrontendModule({
      apiVersion: 1,
      mount(element) {
        element.append(element.ownerDocument.createElement('button'))
        throw failure
      },
      unmount() { unmounts++ }
    }, target, bridge)).rejects.toBe(failure)
    expect(unmounts).toBe(1)
    expect(target.childElementCount).toBe(0)
  })

  it('rejects an invalid cleanup contract and still restores Host DOM', async () => {
    const { target, bridge } = mountFixture()
    let unmounts = 0
    await expect(mountPublicFrontendModule({
      apiVersion: 1,
      mount(element) {
        element.append(element.ownerDocument.createElement('button'))
        return 'invalid' as never
      },
      unmount() { unmounts++ }
    }, target, bridge)).rejects.toBeInstanceOf(PublicFrontendContractError)
    expect(unmounts).toBe(1)
    expect(target.childElementCount).toBe(0)
  })

  it('times out a hanging mount, attempts unmount, and restores Host DOM', async () => {
    const { target, bridge } = mountFixture()
    let unmounts = 0
    await expect(mountPublicFrontendModule({
      apiVersion: 1,
      mount(element) {
        element.append(element.ownerDocument.createElement('button'))
        return new Promise<void>(() => {})
      },
      unmount() { unmounts++ }
    }, target, bridge, 1)).rejects.toThrow('public component mount timed out')
    expect(unmounts).toBe(1)
    expect(target.childElementCount).toBe(0)
  })

  it('bounds a hanging cleanup and still restores Host DOM', async () => {
    const { target, bridge } = mountFixture()
    const release = await mountPublicFrontendModule({
      apiVersion: 1,
      mount(element) {
        element.append(element.ownerDocument.createElement('button'))
        return () => new Promise<void>(() => {})
      }
    }, target, bridge, 1)

    await expect(release()).rejects.toThrow('public component cleanup timed out')
    expect(target.childElementCount).toBe(0)
  })
})

function mountFixture() {
  const browser = new Window({ url: 'https://forum.example/' })
  const target = browser.document.createElement('div')
  const fallback = browser.document.createElement('article')
  const bridge: PublicFrontendBridgeV1 = Object.freeze({
    apiVersion: PUBLIC_FRONTEND_API_VERSION,
    trust: PUBLIC_FRONTEND_TRUST_NOTICE,
    extensionId: 'demo.public',
    extensionVersion: '1.0.0',
    packageDigest: 'a'.repeat(64),
    impactDigest: 'b'.repeat(64),
    componentId: 'demo.public.component.card',
    locale: 'en-US',
    appearance: Object.freeze({ colorMode: 'light', accent: '#0057d9', accentContrast: '#ffffff' }),
    props: Object.freeze({}),
    ssrRoot: fallback,
    request: async <T>() => ({}) as T,
    navigate: async () => {}
  })
  return { target, bridge }
}
