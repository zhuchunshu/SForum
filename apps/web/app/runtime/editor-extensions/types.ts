export const EDITOR_CATALOG_SCHEMA_VERSION = 'sforum.editor-catalog@1'
export const EDITOR_L2_MODULE_API_VERSION = 1

const DIGEST_PATTERN = /^[0-9a-f]{64}$/
const ID_PATTERN = /^[a-z0-9][a-z0-9._-]{1,120}$/

export type EditorCatalogContribution = {
  id: string
  contractVersion: string
  kind: 'node' | 'mark' | 'command' | 'toolbar'
  schema?: string
  extensionName?: string
  l2Module?: string
  l2Digest?: string
  commandKey?: string
  commandId?: string
  label?: string
  icon?: string
  group?: string
  order?: number
  priority?: number
  permission?: string
  artifact: {
    extensionId: string
    extensionVersion: string
    packageDigest: string
  }
}

export type EditorCatalogModule = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  l2Module: string
  l2Digest: string
  assetPath: string
  nodes: EditorCatalogContribution[]
  marks: EditorCatalogContribution[]
  commands: EditorCatalogContribution[]
  toolbars: EditorCatalogContribution[]
}

export type EditorCatalog = {
  schemaVersion: typeof EDITOR_CATALOG_SCHEMA_VERSION
  revision: number
  digest: string
  safeMode?: boolean
  modules: EditorCatalogModule[]
  toolbars: EditorCatalogContribution[]
}

/**
 * Trusted L2 editor module contract. Host verifies package digest bytes before
 * import(); createExtensions must return Tiptap Extension instances only.
 */
export type EditorL2ModuleV1 = {
  apiVersion: typeof EDITOR_L2_MODULE_API_VERSION
  createExtensions: (bridge: EditorL2BridgeV1) => unknown[] | Promise<unknown[]>
}

export type EditorL2BridgeV1 = Readonly<{
  apiVersion: typeof EDITOR_L2_MODULE_API_VERSION
  extensionId: string
  extensionVersion: string
  packageDigest: string
  modulePath: string
  moduleDigest: string
}>

export class EditorL2ContractError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'EditorL2ContractError'
  }
}

export function parseEditorCatalog(input: unknown): EditorCatalog {
  if (!isRecord(input)
    || input.schemaVersion !== EDITOR_CATALOG_SCHEMA_VERSION
    || typeof input.revision !== 'number'
    || !Number.isFinite(input.revision)
    || typeof input.digest !== 'string'
    || !DIGEST_PATTERN.test(input.digest)
    || !Array.isArray(input.modules)
    || !Array.isArray(input.toolbars)) {
    throw new EditorL2ContractError('editor catalog contract mismatch')
  }
  return {
    schemaVersion: EDITOR_CATALOG_SCHEMA_VERSION,
    revision: input.revision,
    digest: input.digest,
    safeMode: input.safeMode === true,
    modules: input.modules.map(parseEditorCatalogModule),
    toolbars: input.toolbars.map(value => parseContribution(value, true))
  }
}

export function parseEditorCatalogModule(input: unknown): EditorCatalogModule {
  if (!isRecord(input)
    || !ID_PATTERN.test(String(input.extensionId || ''))
    || typeof input.extensionVersion !== 'string'
    || !DIGEST_PATTERN.test(String(input.packageDigest || ''))
    || typeof input.l2Module !== 'string'
    || !DIGEST_PATTERN.test(String(input.l2Digest || ''))
    || typeof input.assetPath !== 'string'
    || !Array.isArray(input.nodes)
    || !Array.isArray(input.marks)
    || !Array.isArray(input.commands)
    || !Array.isArray(input.toolbars)) {
    throw new EditorL2ContractError('editor catalog module contract mismatch')
  }
  if (!isExactEditorAssetPath(
    String(input.assetPath), String(input.extensionId), String(input.packageDigest), String(input.l2Module)
  )) {
    throw new EditorL2ContractError('editor catalog module asset path mismatch')
  }
  return {
    extensionId: String(input.extensionId),
    extensionVersion: String(input.extensionVersion),
    packageDigest: String(input.packageDigest),
    l2Module: String(input.l2Module),
    l2Digest: String(input.l2Digest),
    assetPath: String(input.assetPath),
    nodes: input.nodes.map(value => parseContribution(value, false)),
    marks: input.marks.map(value => parseContribution(value, false)),
    commands: input.commands.map(value => parseContribution(value, false)),
    toolbars: input.toolbars.map(value => parseContribution(value, true))
  }
}

export function isExactEditorAssetPath(
  assetPath: string,
  extensionId: string,
  packageDigest: string,
  modulePath: string
) {
  const encodedModulePath = modulePath.split('/').map(segment => encodeURIComponent(segment)).join('/')
  return assetPath === `/_sforum/assets/extensions/${encodeURIComponent(extensionId)}/${packageDigest}/${encodedModulePath}`
}

export function isEditorL2Module(value: unknown): value is EditorL2ModuleV1 {
  return isRecord(value)
    && value.apiVersion === EDITOR_L2_MODULE_API_VERSION
    && typeof value.createExtensions === 'function'
}

function parseContribution(input: unknown, allowToolbar: boolean): EditorCatalogContribution {
  if (!isRecord(input)
    || !ID_PATTERN.test(String(input.id || ''))
    || typeof input.contractVersion !== 'string'
    || typeof input.kind !== 'string'
    || !isRecord(input.artifact)
    || !ID_PATTERN.test(String(input.artifact.extensionId || ''))
    || !DIGEST_PATTERN.test(String(input.artifact.packageDigest || ''))) {
    throw new EditorL2ContractError('editor contribution contract mismatch')
  }
  const kind = String(input.kind)
  if (kind === 'toolbar' && !allowToolbar) {
    throw new EditorL2ContractError('unexpected toolbar contribution')
  }
  if (!['node', 'mark', 'command', 'toolbar'].includes(kind)) {
    throw new EditorL2ContractError('editor contribution kind is invalid')
  }
  return {
    id: String(input.id),
    contractVersion: String(input.contractVersion),
    kind: kind as EditorCatalogContribution['kind'],
    schema: optionalString(input.schema),
    extensionName: optionalString(input.extensionName),
    l2Module: optionalString(input.l2Module),
    l2Digest: optionalString(input.l2Digest),
    commandKey: optionalString(input.commandKey),
    commandId: optionalString(input.commandId),
    label: optionalString(input.label),
    icon: optionalString(input.icon),
    group: optionalString(input.group),
    order: typeof input.order === 'number' ? input.order : undefined,
    priority: typeof input.priority === 'number' ? input.priority : undefined,
    permission: optionalString(input.permission),
    artifact: {
      extensionId: String(input.artifact.extensionId),
      extensionVersion: String(input.artifact.extensionVersion || ''),
      packageDigest: String(input.artifact.packageDigest)
    }
  }
}

function optionalString(value: unknown) {
  return typeof value === 'string' && value ? value : undefined
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
