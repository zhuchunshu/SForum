const FAILURE_PREFIX = 'sforum:public-extension-failures:'
const FAILURE_LIMIT = 3

export type PublicContributionFailureState = {
  count: number
  quarantined: boolean
}

export function publicContributionFailureKey(impactDigest: string, extensionId: string, componentId: string) {
  return `${FAILURE_PREFIX}${encodeURIComponent(impactDigest)}:${encodeURIComponent(extensionId)}:${encodeURIComponent(componentId)}`
}

export function publicContributionFailureState(storage: Storage, key: string): PublicContributionFailureState {
  const stored = Number.parseInt(storage.getItem(key) || '0', 10)
  const count = Number.isFinite(stored) && stored > 0 ? stored : 0
  return { count, quarantined: count >= FAILURE_LIMIT }
}

export function recordPublicContributionFailure(storage: Storage, key: string): PublicContributionFailureState {
  const count = publicContributionFailureState(storage, key).count + 1
  storage.setItem(key, String(count))
  return { count, quarantined: count >= FAILURE_LIMIT }
}

export function clearPublicContributionFailures(storage: Storage, key: string) {
  storage.removeItem(key)
}
