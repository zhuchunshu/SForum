import type { AdminCapabilityGrant, AdminExtensionSource, AdminExtensionType } from '~/utils/admin/adminExtensions'

export type ExecutableTrustAuthority = {
  backendExecution: boolean
  adminFrontendExecution: boolean
  rawRequest: boolean
  rawCoreDatabase: boolean
  outboundNetwork: boolean
  packageFiles: string[]
  secrets: string[]
}

export type ExecutableTrustMigrationDeclaration = {
  id?: string
  contractVersion?: string
  path: string
  digest?: string
  transaction?: 'required' | 'forbidden' | 'auto'
}

export type ExecutableDatabaseGrant = 'own_schema' | 'core_views' | 'host_commands' | 'raw_core' | 'kernel'

type ExecutableTrustDatabaseBase = {
  contractVersion: string
  schema?: string
  role?: string
  coreCompatibility?: string
  backup: {
    required: boolean
    strategy?: string
  }
  retention: {
    onDisable: 'retain'
    onUninstall: 'retain' | 'delete' | 'export'
    days?: number
  }
}

export type ExecutableTrustDatabase = ExecutableTrustDatabaseBase & (
  | { grants: ExecutableDatabaseGrant[], authority?: never }
  | { authority: ExecutableDatabaseGrant, grants?: never }
)

const databaseGrantTierOrder: ExecutableDatabaseGrant[] = [
  'own_schema', 'core_views', 'host_commands', 'raw_core', 'kernel'
]

export function executableDatabaseGrants(database: ExecutableTrustDatabase | null): ExecutableDatabaseGrant[] {
  if (!database) return []
  if (database.grants?.length) {
    return databaseGrantTierOrder.filter(grant => database.grants?.includes(grant))
  }
  if (!database.authority) return []
  const tier = databaseGrantTierOrder.indexOf(database.authority)
  return tier < 0 ? [] : databaseGrantTierOrder.slice(0, tier + 1)
}

export type ExecutableTrustImpact = {
  schemaVersion: 'sforum.trust-impact@2'
  action: 'enable'
  extensionId: string
  extensionVersion: string
  extensionType: AdminExtensionType
  source: AdminExtensionSource
  packageDigest: string
  manifestContract: string
  artifactDigests: Record<string, string>
  binaries: Array<Record<string, unknown>>
  backend: Record<string, unknown>
  routes: Array<Record<string, unknown>>
  guards: Array<Record<string, unknown>>
  guardDeclarations: Array<Record<string, unknown>>
  hooks: Array<Record<string, unknown>>
  events: Array<Record<string, unknown>>
  migrations: Array<Record<string, unknown>>
  migrationDeclarations: ExecutableTrustMigrationDeclaration[]
  providers: Array<Record<string, unknown>>
  jobs: Array<Record<string, unknown>>
  schedules: Array<Record<string, unknown>>
  components: Array<Record<string, unknown>>
  registryComponents: Array<Record<string, unknown>>
  templates: Array<Record<string, unknown>>
  assets: Array<Record<string, unknown>>
  content: Array<Record<string, unknown>>
  database: ExecutableTrustDatabase | null
  cache: Array<Record<string, unknown>>
  services: Array<Record<string, unknown>>
  commands: Array<Record<string, unknown>>
  adminSurfaces: Array<Record<string, unknown>>
  queries: Array<Record<string, unknown>>
  queryResultFilters: Array<Record<string, unknown>>
  identity: Record<string, unknown> | null
  permissionDefinitions: Array<Record<string, unknown>>
  media: Array<Record<string, unknown>>
  navigation: Array<Record<string, unknown>>
  regions: Array<Record<string, unknown>>
  contributions: Array<Record<string, unknown>>
  capabilities: AdminCapabilityGrant[]
  permissions: string[]
  requiredFeatures: string[]
  dependencies: Array<Record<string, unknown>>
  lifecycle: Record<string, unknown> | null
  openapi: Array<Record<string, unknown>>
  packageFiles: Array<Record<string, unknown>>
  requestedAuthority: ExecutableTrustAuthority
  contracts: { hostApi: string, frontendApi?: string }
  digest: string
}

export function executableDatabaseMigrationRisk(impact: Pick<ExecutableTrustImpact, 'database' | 'migrationDeclarations'>) {
  const nonTransactional = impact.migrationDeclarations.filter(migration => migration.transaction === 'forbidden')
  const hasMigrations = impact.migrationDeclarations.length > 0
  const backupRequired = impact.database?.backup.required === true
  return {
    hasDatabaseChanges: Boolean(impact.database) || hasMigrations,
    hasMigrations,
    backupRequired,
    missingRequiredBackup: hasMigrations && !backupRequired,
    nonTransactional
  }
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
