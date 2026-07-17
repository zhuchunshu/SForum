import { afterEach, describe, expect, test } from 'bun:test'
import { computed, ref } from 'vue'

import {
  roleSuggestionDecisionHasEvidence,
  roleSuggestionHasConsistentEvidence,
  type RoleSuggestion,
  useRoleSuggestions
} from '../app/composables/useRoleSuggestions'

const globals = globalThis as typeof globalThis & Record<string, unknown>
const originalGlobals = new Map<string, unknown>()

for (const key of ['computed', 'onMounted', 'ref', 'useApiClient', 'useI18n', 'useToast', 'watch']) {
  originalGlobals.set(key, globals[key])
}

afterEach(() => {
  for (const [key, value] of originalGlobals) globals[key] = value
})

function suggestion(overrides: Partial<RoleSuggestion> = {}): RoleSuggestion {
  return {
    id: 41,
    permissionKey: 'demo.publish',
    ownerExtensionId: 'demo.identity',
    extensionVersionId: 9,
    extensionVersion: '1.0.0',
    packageDigest: 'a'.repeat(64),
    permissionContractVersion: 'demo.permission.publish@1',
    declarationDigest: 'b'.repeat(64),
    roleKey: 'member',
    approvalState: 'pending',
    applied: false,
    revision: 1,
    createdAt: '2026-07-17T00:00:00Z',
    updatedAt: '2026-07-17T00:00:00Z',
    ...overrides
  }
}

type RequestOptions = {
  method?: string
  body?: unknown
}

type RequestCall = {
  path: string
  options?: RequestOptions
}

function installRuntime(
  request: (path: string, options?: RequestOptions) => unknown | Promise<unknown>,
  toastCalls: unknown[] = []
) {
  globals.computed = computed
  globals.ref = ref
  globals.onMounted = () => undefined
  globals.watch = () => undefined
  globals.useI18n = () => ({ locale: ref('en-US'), t: (key: string) => key })
  globals.useToast = () => ({ add: (input: unknown) => toastCalls.push(input) })
  globals.useApiClient = () => ({ request })
  return toastCalls
}

function apiError(reason: string, message = reason, code = 409) {
  return {
    data: {
      code,
      message,
      data: { reason }
    }
  }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe('role suggestion evidence', () => {
  test('rejects applied evidence for pending and rejected states', () => {
    expect(roleSuggestionHasConsistentEvidence({ approvalState: 'pending', applied: true })).toBe(false)
    expect(roleSuggestionHasConsistentEvidence({ approvalState: 'rejected', applied: true })).toBe(false)
    expect(roleSuggestionHasConsistentEvidence({ approvalState: 'approved', applied: true })).toBe(true)
    expect(roleSuggestionHasConsistentEvidence({ approvalState: 'approved', applied: false })).toBe(true)
    expect(roleSuggestionHasConsistentEvidence({ approvalState: 'unknown' as never, applied: false })).toBe(false)
    expect(roleSuggestionHasConsistentEvidence({ approvalState: 'approved', applied: 'yes' as never })).toBe(false)
  })

  test('requires exact positive or negative decision evidence', () => {
    expect(roleSuggestionDecisionHasEvidence('approved', { approvalState: 'approved', applied: true })).toBe(true)
    expect(roleSuggestionDecisionHasEvidence('approved', { approvalState: 'approved', applied: false })).toBe(false)
    expect(roleSuggestionDecisionHasEvidence('rejected', { approvalState: 'rejected', applied: false })).toBe(true)
    expect(roleSuggestionDecisionHasEvidence('rejected', { approvalState: 'rejected', applied: true })).toBe(false)
  })

  test('keeps contradictory rejection visible and emits no success signal', async () => {
    const toastCalls: unknown[] = []
    let appliedCalls = 0
    globals.computed = computed
    globals.ref = ref
    globals.onMounted = () => undefined
    globals.watch = () => undefined
    globals.useI18n = () => ({ locale: ref('en-US'), t: (key: string) => key })
    globals.useToast = () => ({ add: (input: unknown) => toastCalls.push(input) })
    globals.useApiClient = () => ({
      request: async () => suggestion({ approvalState: 'rejected', applied: true, revision: 2 })
    })

    const state = useRoleSuggestions(() => { appliedCalls++ })
    state.openDecision(suggestion(), 'rejected')
    await state.submitDecision()

    expect(state.decisionOpen.value).toBe(true)
    expect(state.decisionError.value).toBe('admin.roles.suggestions.evidenceMissing')
    expect(toastCalls).toHaveLength(0)
    expect(appliedCalls).toBe(0)
  })

  test('rejects a contradictory list page without retaining its cursor', async () => {
    globals.computed = computed
    globals.ref = ref
    globals.onMounted = () => undefined
    globals.watch = () => undefined
    globals.useI18n = () => ({ locale: ref('en-US'), t: (key: string) => key })
    globals.useToast = () => ({ add: () => undefined })
    globals.useApiClient = () => ({
      request: async () => ({
        items: [suggestion({ approvalState: 'rejected', applied: true, revision: 2 })],
        nextCursor: 'untrusted-cursor'
      })
    })

    const state = useRoleSuggestions()
    await state.loadSuggestions(true)

    expect(state.suggestions.value).toHaveLength(0)
    expect(state.nextCursor.value).toBe('')
    expect(state.loadError.value).toBe('admin.roles.suggestions.evidenceMissing')
  })
})

describe('role suggestion decisions', () => {
  test('submits the exact approval CAS body, awaits the apply callback, and refreshes', async () => {
    const calls: RequestCall[] = []
    const events: string[] = []
    const toastCalls: unknown[] = []
    let appliedResult: RoleSuggestion | undefined
    installRuntime(async (path, options) => {
      calls.push({ path, options })
      if (options?.method === 'POST') {
        events.push('decision')
        return suggestion({ approvalState: 'approved', applied: true, revision: 8 })
      }
      events.push('list')
      return { items: [], nextCursor: '' }
    }, toastCalls)

    const state = useRoleSuggestions(async (result) => {
      appliedResult = result
      events.push('callback')
    })
    state.openDecision(suggestion({ revision: 7 }), 'approved')
    await state.submitDecision()

    expect(calls[0]).toEqual({
      path: '/roles/suggestions/41/decision',
      options: {
        method: 'POST',
        body: { expectedRevision: 7, approvalState: 'approved' }
      }
    })
    expect(events).toEqual(['decision', 'callback', 'list'])
    expect(appliedResult?.roleKey).toBe('member')
    expect(appliedResult?.permissionKey).toBe('demo.publish')
    expect(state.decisionOpen.value).toBe(false)
    expect(state.pendingDecision.value).toBeNull()
    expect(toastCalls).toHaveLength(1)
  })

  test('rejects without invoking the permission-applied callback', async () => {
    const calls: RequestCall[] = []
    let appliedCalls = 0
    installRuntime(async (path, options) => {
      calls.push({ path, options })
      if (options?.method === 'POST') {
        return suggestion({ approvalState: 'rejected', applied: false, revision: 2 })
      }
      return { items: [], nextCursor: '' }
    })

    const state = useRoleSuggestions(() => { appliedCalls++ })
    state.openDecision(suggestion(), 'rejected')
    await state.submitDecision()

    expect(calls[0]?.options?.body).toEqual({ expectedRevision: 1, approvalState: 'rejected' })
    expect(appliedCalls).toBe(0)
    expect(state.decisionOpen.value).toBe(false)
    expect(state.pendingDecision.value).toBeNull()
  })

  test('keeps a denied decision open and emits no success signal', async () => {
    const toastCalls: unknown[] = []
    let appliedCalls = 0
    let calls = 0
    installRuntime(async () => {
      calls++
      throw apiError('permission.denied', 'You do not have permission to perform this action.', 403)
    }, toastCalls)

    const state = useRoleSuggestions(() => { appliedCalls++ })
    state.openDecision(suggestion(), 'approved')
    await state.submitDecision()

    expect(calls).toBe(1)
    expect(state.decisionOpen.value).toBe(true)
    expect(state.pendingDecision.value?.suggestion.revision).toBe(1)
    expect(state.decisionError.value).toBe('You do not have permission to perform this action.')
    expect(appliedCalls).toBe(0)
    expect(toastCalls).toHaveLength(0)
  })

  for (const [reason, expectedMessage] of [
    ['identity.role_suggestion.stale', 'admin.roles.suggestions.stale'],
    ['identity.role_suggestion.revision_conflict', 'admin.roles.suggestions.revisionConflict'],
    ['identity.role_suggestion.target_unavailable', 'admin.roles.suggestions.targetUnavailable']
  ] as const) {
    test(`adopts refreshed Host state after ${reason}`, async () => {
      const calls: RequestCall[] = []
      installRuntime(async (path, options) => {
        calls.push({ path, options })
        if (options?.method === 'POST') throw apiError(reason)
        return { items: [suggestion({ revision: 2 })], nextCursor: '' }
      })

      const state = useRoleSuggestions()
      state.openDecision(suggestion(), 'approved')
      await state.submitDecision()

      expect(calls).toHaveLength(2)
      expect(calls[1]?.path).toBe('/roles/suggestions?limit=25&approvalState=pending')
      expect(state.decisionOpen.value).toBe(false)
      expect(state.pendingDecision.value).toBeNull()
      expect(state.suggestions.value[0]?.revision).toBe(2)
      expect(state.loadError.value).toBe(expectedMessage)

      await state.submitDecision()
      expect(calls).toHaveLength(2)
    })
  }

  test('does not let another row replace a decision while its request is in flight', async () => {
    const decision = deferred<RoleSuggestion>()
    installRuntime(async (_path, options) => {
      if (options?.method === 'POST') return decision.promise
      return { items: [], nextCursor: '' }
    })

    const first = suggestion({ id: 41, revision: 3 })
    const second = suggestion({ id: 42, revision: 9 })
    const state = useRoleSuggestions()
    state.openDecision(first, 'approved')
    const submitting = state.submitDecision()

    expect(state.deciding.value).toBe(true)
    state.openDecision(second, 'rejected')
    expect(state.pendingDecision.value?.suggestion.id).toBe(41)
    expect(state.pendingDecision.value?.state).toBe('approved')

    decision.resolve(suggestion({ id: 41, approvalState: 'approved', applied: true, revision: 4 }))
    await submitting
    expect(state.deciding.value).toBe(false)
  })

  test('uses friendly fallbacks for unmapped raw role-suggestion reasons', async () => {
    installRuntime(async (_path, options) => {
      if (options?.method === 'POST') {
        throw apiError('identity.role_suggestion.invalid')
      }
      throw apiError('identity.role_suggestion.cookie_required', undefined, 403)
    })

    const state = useRoleSuggestions()
    await state.loadSuggestions(true)
    expect(state.loadError.value).toBe('admin.roles.suggestions.loadFailed')

    state.openDecision(suggestion(), 'approved')
    await state.submitDecision()
    expect(state.decisionOpen.value).toBe(true)
    expect(state.decisionError.value).toBe('admin.roles.suggestions.decisionFailed')
  })
})

describe('role suggestion list generations', () => {
  test('uses non-empty select values and omits approvalState for the all filter', async () => {
    const paths: string[] = []
    installRuntime(async (path) => {
      paths.push(path)
      return { items: [], nextCursor: '' }
    })

    const state = useRoleSuggestions()
    expect(state.filterItems.value.map(item => item.value)).toEqual([
      'pending',
      'approved',
      'rejected',
      'all'
    ])

    state.selectedFilter.value = 'all'
    await state.loadSuggestions(true)
    expect(paths).toEqual(['/roles/suggestions?limit=25'])
  })

  test('a reset clears loading-more while the superseded page is still pending', async () => {
    const oldPage = deferred<{ items: RoleSuggestion[], nextCursor: string }>()
    let calls = 0
    installRuntime(async () => {
      calls++
      if (calls === 1) {
        return { items: [suggestion()], nextCursor: 'page-2' }
      }
      if (calls === 2) return oldPage.promise
      return { items: [suggestion({ id: 99, revision: 4 })], nextCursor: '' }
    })

    const state = useRoleSuggestions()
    await state.loadSuggestions(true)
    const loadingOldPage = state.loadSuggestions(false)
    expect(state.loadingMore.value).toBe(true)

    await state.loadSuggestions(true)
    expect(state.loadingMore.value).toBe(false)
    expect(state.suggestions.value.map(item => item.id)).toEqual([99])

    oldPage.resolve({ items: [suggestion({ id: 42 })], nextCursor: '' })
    await loadingOldPage
    expect(state.loadingMore.value).toBe(false)
    expect(state.suggestions.value.map(item => item.id)).toEqual([99])
  })
})
