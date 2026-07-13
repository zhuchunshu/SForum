import type { AdminCapabilityGrant, AdminExtensionSource, AdminExtensionType } from '~/utils/adminExtensions'

export type ExecutableTrustAuthority = {
  backendExecution: boolean
  adminFrontendExecution: boolean
  rawRequest: boolean
  rawCoreDatabase: boolean
  outboundNetwork: boolean
  packageFiles: string[]
  secrets: string[]
}

export type ExecutableTrustImpact = {
  schemaVersion: string
  action: 'enable'
  extensionId: string
  extensionVersion: string
  extensionType: AdminExtensionType
  source: AdminExtensionSource
  packageDigest: string
  artifactDigests: Record<string, string>
  binaries: Array<Record<string, unknown>>
  routes: Array<Record<string, unknown>>
  guards: Array<Record<string, unknown>>
  hooks: Array<Record<string, unknown>>
  events: Array<Record<string, unknown>>
  migrations: Array<Record<string, unknown>>
  providers: Array<Record<string, unknown>>
  jobs: Array<Record<string, unknown>>
  schedules: string[]
  components: Array<Record<string, unknown>>
  contributions: Array<Record<string, unknown>>
  capabilities: AdminCapabilityGrant[]
  permissions: string[]
  requiredFeatures: string[]
  dependencies: Array<Record<string, unknown>>
  requestedAuthority: ExecutableTrustAuthority
  contracts: { hostApi: string, frontendApi?: string }
  digest: string
}

export type ExecutableTrustStatus = {
  impact: ExecutableTrustImpact
  trustRequired: boolean
  trusted: boolean
}

export type ExecutableTrustChallenge = {
  token: string
  impact: ExecutableTrustImpact
  expiresAt: string
}

export type ExtensionEnableTrustMode = 'legacy' | 'exact'
