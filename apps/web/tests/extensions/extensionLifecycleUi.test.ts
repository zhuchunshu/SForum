import { describe, expect, test } from 'bun:test'

import {
  RECOMMENDED_LIFECYCLE_REMOVAL_MODE,
  canRecoverLifecycleOperation,
  canSubmitLifecycleRecovery,
  isLifecycleV2Plugin,
  type AdminExtension,
  type AdminLifecycleOperation,
  type AdminLifecycleRecoveryInput
} from '../../app/utils/admin/adminExtensions'

describe('V3 lifecycle operator controls', () => {
  test('defaults every V2 uninstall to the safe preserve mode', () => {
    expect(RECOMMENDED_LIFECYCLE_REMOVAL_MODE).toBe('preserve')
  })

  test('detects only plugins declaring both protocol V2 and a lifecycle contract', () => {
    expect(isLifecycleV2Plugin(extension())).toBe(true)
    expect(isLifecycleV2Plugin(extension({ backend: { protocolVersion: 1 }, lifecycle: { contractVersion: 'v2' } }))).toBe(false)
    expect(isLifecycleV2Plugin(extension({ backend: { protocolVersion: 2 } }))).toBe(false)
    expect(isLifecycleV2Plugin(extension({ backend: { protocolVersion: 2 }, lifecycle: { contractVersion: '  ' } }))).toBe(false)
    expect(isLifecycleV2Plugin(extension({ backend: { protocolVersion: 2 }, lifecycle: { contractVersion: 'v2' } }, 'theme'))).toBe(false)
  })

  test('offers recovery only for failed or cancelled durable operations', () => {
    expect(canRecoverLifecycleOperation(operation({ terminalResult: 'failed' }))).toBe(true)
    expect(canRecoverLifecycleOperation(operation({ terminalResult: 'cancelled' }))).toBe(true)
    expect(canRecoverLifecycleOperation(operation({ terminalResult: 'succeeded' }))).toBe(false)
    expect(canRecoverLifecycleOperation(operation({ terminalResult: 'skipped' }))).toBe(false)
    expect(canRecoverLifecycleOperation(operation({ terminalResult: undefined }))).toBe(false)
  })

  test('allows an ordinary retry without a reason', () => {
    expect(canSubmitLifecycleRecovery(
      operation({ terminalResult: 'failed' }),
      recovery({ decision: 'retry' }),
      false
    )).toBe(true)
  })

  test('requires an explicit reason before skipping a step', () => {
    const failed = operation({ terminalResult: 'failed' })

    expect(canSubmitLifecycleRecovery(failed, recovery({ decision: 'skip_step' }), true)).toBe(false)
    expect(canSubmitLifecycleRecovery(failed, recovery({ decision: 'skip_step', reason: '  reviewed cleanup impact  ' }), true)).toBe(true)
  })

  test('gates forced recovery to uninstall, super-admin, reason, and residual-risk acknowledgement', () => {
    const forced = recovery({
      decision: 'retry',
      reason: 'External webhook cleanup will be completed manually.',
      escalateForced: true,
      residualRiskAcknowledged: true
    })

    expect(canSubmitLifecycleRecovery(operation({ operation: 'disable' }), forced, true)).toBe(false)
    expect(canSubmitLifecycleRecovery(operation({ operation: 'uninstall' }), forced, false)).toBe(false)
    expect(canSubmitLifecycleRecovery(operation({ operation: 'uninstall' }), { ...forced, reason: ' ' }, true)).toBe(false)
    expect(canSubmitLifecycleRecovery(operation({ operation: 'uninstall' }), { ...forced, residualRiskAcknowledged: false }, true)).toBe(false)
    expect(canSubmitLifecycleRecovery(operation({ operation: 'uninstall' }), forced, true)).toBe(true)
  })
})

function extension(
  manifest: Partial<AdminExtension['manifest']> = {
    backend: { protocolVersion: 2 },
    lifecycle: { contractVersion: 'v2' }
  },
  type: AdminExtension['type'] = 'plugin'
): AdminExtension {
  return {
    id: 'demo.plugin',
    name: 'Demo Plugin',
    version: '1.0.0',
    type,
    status: 'enabled',
    source: 'uploaded',
    isSystem: false,
    isDeletable: true,
    manifest: {
      id: 'demo.plugin',
      name: 'Demo Plugin',
      version: '1.0.0',
      type,
      sforumVersion: '^3.0.0',
      ...manifest
    },
    packageDigest: 'sha256:demo',
    packagePath: 'storage/extensions/demo.plugin',
    installedAt: '2026-07-14T00:00:00Z',
    updatedAt: '2026-07-14T00:00:00Z'
  }
}

function operation(overrides: Partial<AdminLifecycleOperation> = {}): AdminLifecycleOperation {
  return {
    id: 41,
    extensionId: 'demo.plugin',
    extensionVersion: '1.0.0',
    packageDigest: 'sha256:demo',
    operation: 'uninstall',
    state: 'failed',
    planVersion: 'demo.plugin.lifecycle@v2',
    removalMode: 'preserve',
    forced: false,
    attemptCount: 1,
    revision: 1,
    terminalResult: 'failed',
    error: { retryable: true },
    createdAt: '2026-07-14T00:00:00Z',
    updatedAt: '2026-07-14T00:01:00Z',
    ...overrides
  }
}

function recovery(overrides: Partial<AdminLifecycleRecoveryInput> = {}): AdminLifecycleRecoveryInput {
  return {
    decision: 'retry',
    reason: '',
    escalateForced: false,
    residualRiskAcknowledged: false,
    ...overrides
  }
}
