const FAILURE_PREFIX = 'sforum:admin-extension-failures:'
const FAILURE_LIMIT = 3

export type ContributionFailureState = {
  count: number
  quarantined: boolean
}

export function contributionFailureKey(releaseId: string, extensionId: string, contributionId: string) {
  return `${FAILURE_PREFIX}${encodeURIComponent(releaseId)}:${encodeURIComponent(extensionId)}:${encodeURIComponent(contributionId)}`
}

export function contributionFailureState(storage: Storage, key: string): ContributionFailureState {
  const stored = Number.parseInt(storage.getItem(key) || '0', 10)
  const count = Number.isFinite(stored) && stored > 0 ? stored : 0
  return { count, quarantined: count >= FAILURE_LIMIT }
}

export function recordContributionFailure(storage: Storage, key: string): ContributionFailureState {
  const count = contributionFailureState(storage, key).count + 1
  storage.setItem(key, String(count))
  return { count, quarantined: count >= FAILURE_LIMIT }
}

export function clearContributionFailures(storage: Storage, key: string) {
  storage.removeItem(key)
}
