import { describe, expect, it } from 'bun:test'
import { Window } from 'happy-dom'

import {
  clearPublicContributionFailures,
  publicContributionFailureKey,
  publicContributionFailureState,
  recordPublicContributionFailure
} from '../../app/runtime/public-extensions/quarantine'

describe('public extension browser-session quarantine', () => {
  it('quarantines the exact impact/component after three failures and can be reset', () => {
    const storage = new Window().sessionStorage
    const key = publicContributionFailureKey('a'.repeat(64), 'demo.public', 'demo.public.component.card')
    expect(recordPublicContributionFailure(storage, key)).toEqual({ count: 1, quarantined: false })
    expect(recordPublicContributionFailure(storage, key)).toEqual({ count: 2, quarantined: false })
    expect(recordPublicContributionFailure(storage, key)).toEqual({ count: 3, quarantined: true })
    expect(publicContributionFailureState(storage, key).quarantined).toBe(true)
    clearPublicContributionFailures(storage, key)
    expect(publicContributionFailureState(storage, key)).toEqual({ count: 0, quarantined: false })
  })
})
