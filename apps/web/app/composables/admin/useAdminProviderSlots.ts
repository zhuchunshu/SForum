export type ProviderSlotArtifact = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  runtimeInstanceId?: string
}

export type ProviderSlotContract = {
  id: string
  slot: string
  contractVersion: string
  requestSchema: string
  responseSchema: string
  requestSchemaDigest?: string
  responseSchemaDigest?: string
  fallback: 'next' | 'closed'
  timeoutMs: number
  artifact: ProviderSlotArtifact
  contractRuntimeAvailable: boolean
}

export type ProviderSlotCandidate = {
  id: string
  targetId: string
  label: string
  handler: string
  priority: number
  rank: number
  artifact: ProviderSlotArtifact
  availability: 'available' | 'unavailable'
}

export type ProviderSlotSelection = {
  contractId: string
  contractVersion: string
  slot: string
  contractArtifact: ProviderSlotArtifact
  candidateId: string
  providerArtifact: ProviderSlotArtifact
  selectedByUserId: number
  selectionAuditEventId: number
  revision: number
  selectedAt: string
  updatedAt: string
}

export type ProviderSlotConflict = {
  kind: 'priority_tie'
  priority: number
  candidateIds: string[]
}

export type ProviderSlotItem = {
  contract: ProviderSlotContract
  candidates: ProviderSlotCandidate[]
  conflicts: ProviderSlotConflict[]
  selectionStatus: 'default' | 'selected' | 'stale'
  selection?: ProviderSlotSelection
  availability: 'available' | 'unavailable'
  unavailabilityReason?: 'no_candidates' | 'runtime_unavailable'
}

export type ProviderSlotInspection = {
  revision: number
  slots: ProviderSlotItem[]
}

export type ProviderSlotProbe = {
  contractId: string
  candidateId: string
  artifact: ProviderSlotArtifact
  ok: boolean
  reason: string
  message: string
  details?: Record<string, string>
  suggestions?: string[]
  durationMs: number
}

export function useAdminProviderSlots() {
  const { request } = useApiClient()
  const basePath = '/admin/extensions/provider-slots'

  const inspect = () => request<ProviderSlotInspection>(basePath)
  const select = (contractId: string, candidateId: string, expectedRevision: number) =>
    request<ProviderSlotSelection>(`${basePath}/selection`, {
      method: 'POST',
      body: { contractId, candidateId, expectedRevision }
    })
  const reset = (contractId: string, expectedRevision: number) =>
    request<{ reset: true, contractId: string }>(`${basePath}/selection/reset`, {
      method: 'POST',
      body: { contractId, expectedRevision, reasonCode: 'operator_reset' }
    })
  const probe = (contractId: string, candidateId: string) =>
    request<ProviderSlotProbe>(`${basePath}/probe`, {
      method: 'POST',
      body: { contractId, candidateId }
    })

  return { inspect, select, reset, probe }
}
