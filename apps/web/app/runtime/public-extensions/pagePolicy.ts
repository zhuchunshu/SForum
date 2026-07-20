export const PUBLIC_FRONTEND_POLICY_SCHEMA_VERSION = 'sforum.public-frontend-policy@1'
export const PUBLIC_FRONTEND_DOCUMENT_POLICY_SCHEMA_VERSION = 'sforum.public-frontend-document-policy@1'
export const PUBLIC_PAGE_POLICY_PATH = '/extensions/runtime/page-policy'
export const PUBLIC_PAGE_POLICY_TIMEOUT_MS = 10_000
export const PUBLIC_PAGE_CSP_HEADER = 'Content-Security-Policy'
export const PUBLIC_L2_WIDGET_COMPONENT_ID = 'core.component.shared.sfextension_widget'

const ID_PATTERN = /^[a-z0-9][a-z0-9._-]{1,120}$/
const DIGEST_PATTERN = /^[0-9a-f]{64}$/
const MAX_COMPONENTS = 256
const MAX_HEADER_BYTES = 8 * 1024

export type PublicFrontendComponentRef = {
  extensionId: string
  componentId: string
}

export type PublicFrontendPolicyDirective = {
  name: string
  sources: string[]
}

export type PublicFrontendPolicyComponent = {
  extensionId: string
  extensionVersion: string
  packageDigest: string
  impactDigest: string
  componentId: string
  contractVersion: string
}

export type PublicFrontendDocumentPolicy = {
  schemaVersion: typeof PUBLIC_FRONTEND_DOCUMENT_POLICY_SCHEMA_VERSION
  digest: string
  directives: PublicFrontendPolicyDirective[]
  headerValue: string
}

export type PublicFrontendPolicy = {
  schemaVersion: typeof PUBLIC_FRONTEND_POLICY_SCHEMA_VERSION
  graphDigest: string
  extensionPolicyDigest: string
  directives: PublicFrontendPolicyDirective[]
  admittedComponents: PublicFrontendPolicyComponent[]
  documentPolicy: PublicFrontendDocumentPolicy
}

export class PublicPagePolicyError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'PublicPagePolicyError'
  }
}

/** 规范化并去重 soft refs；非法 ID 直接失败关闭。 */
export function normalizePublicFrontendComponentRefs(
  input: readonly PublicFrontendComponentRef[]
): PublicFrontendComponentRef[] {
  if (!Array.isArray(input) || input.length > MAX_COMPONENTS) {
    throw new PublicPagePolicyError('public page policy component bound exceeded')
  }
  const seen = new Map<string, PublicFrontendComponentRef>()
  for (const raw of input) {
    const extensionId = String(raw?.extensionId || '').trim()
    const componentId = String(raw?.componentId || '').trim()
    if (!ID_PATTERN.test(extensionId) || !ID_PATTERN.test(componentId)) {
      throw new PublicPagePolicyError('invalid public page policy component ref')
    }
    const key = `${extensionId}\0${componentId}`
    if (!seen.has(key)) {
      seen.set(key, { extensionId, componentId })
    }
  }
  return [...seen.values()].sort((left, right) => {
    if (left.extensionId !== right.extensionId) {
      return left.extensionId < right.extensionId ? -1 : 1
    }
    return left.componentId < right.componentId ? -1 : left.componentId > right.componentId ? 1 : 0
  })
}

export function publicPagePolicyPath(refs: readonly PublicFrontendComponentRef[] = []): string {
  const normalized = normalizePublicFrontendComponentRefs(refs)
  if (!normalized.length) {
    return PUBLIC_PAGE_POLICY_PATH
  }
  const query = normalized
    .map(ref => `component=${encodeURIComponent(`${ref.extensionId}/${ref.componentId}`)}`)
    .join('&')
  return `${PUBLIC_PAGE_POLICY_PATH}?${query}`
}

export function parsePublicFrontendPolicy(raw: unknown): PublicFrontendPolicy {
  if (!isObject(raw)
    || raw.schemaVersion !== PUBLIC_FRONTEND_POLICY_SCHEMA_VERSION
    || !DIGEST_PATTERN.test(String(raw.graphDigest || ''))
    || !DIGEST_PATTERN.test(String(raw.extensionPolicyDigest || ''))
    || !Array.isArray(raw.directives)
    || !Array.isArray(raw.admittedComponents)
    || !isObject(raw.documentPolicy)
  ) {
    throw new PublicPagePolicyError('invalid public page policy')
  }
  const documentPolicy = parsePublicFrontendDocumentPolicy(raw.documentPolicy)
  return {
    schemaVersion: PUBLIC_FRONTEND_POLICY_SCHEMA_VERSION,
    graphDigest: String(raw.graphDigest),
    extensionPolicyDigest: String(raw.extensionPolicyDigest),
    directives: raw.directives.map(parseDirective),
    admittedComponents: raw.admittedComponents.map(parseAdmittedComponent),
    documentPolicy
  }
}

export function parsePublicFrontendDocumentPolicy(raw: unknown): PublicFrontendDocumentPolicy {
  if (!isObject(raw)
    || raw.schemaVersion !== PUBLIC_FRONTEND_DOCUMENT_POLICY_SCHEMA_VERSION
    || !DIGEST_PATTERN.test(String(raw.digest || ''))
    || !Array.isArray(raw.directives)
    || typeof raw.headerValue !== 'string'
    || !raw.headerValue.trim()
    || raw.headerValue.length > MAX_HEADER_BYTES
    || raw.headerValue.includes('\n')
    || raw.headerValue.includes('\r')
  ) {
    throw new PublicPagePolicyError('invalid public document policy')
  }
  return {
    schemaVersion: PUBLIC_FRONTEND_DOCUMENT_POLICY_SCHEMA_VERSION,
    digest: String(raw.digest),
    directives: raw.directives.map(parseDirective),
    headerValue: raw.headerValue
  }
}

/**
 * 从主题渲染树中收集即将挂载的公开 L2 soft refs。
 * 兼容 typed island props 与 legacy `extension-id`/`component-id` 属性。
 */
export function collectPublicL2ComponentRefsFromRenderNodes(
  nodes: readonly {
    kind: string
    descriptor?: { componentId?: string }
    props?: Record<string, string | boolean | number>
    children?: readonly unknown[]
  }[]
): PublicFrontendComponentRef[] {
  const refs: PublicFrontendComponentRef[] = []
  const walk = (items: readonly typeof nodes[number][]) => {
    for (const node of items) {
      if (node.kind === 'island'
        && node.descriptor?.componentId === PUBLIC_L2_WIDGET_COMPONENT_ID
        && node.props
      ) {
        const extensionId = stringProp(node.props, 'extensionId', 'extension-id')
        const componentId = stringProp(node.props, 'componentId', 'component-id')
        if (extensionId && componentId) {
          refs.push({ extensionId, componentId })
        }
      }
      if (Array.isArray(node.children) && node.children.length) {
        walk(node.children as typeof nodes)
      }
    }
  }
  walk(nodes)
  return normalizePublicFrontendComponentRefs(refs)
}

function parseDirective(raw: unknown): PublicFrontendPolicyDirective {
  if (!isObject(raw) || typeof raw.name !== 'string' || !Array.isArray(raw.sources) || !raw.sources.length) {
    throw new PublicPagePolicyError('invalid public page policy directive')
  }
  if (raw.sources.some(source => typeof source !== 'string' || !source)) {
    throw new PublicPagePolicyError('invalid public page policy directive source')
  }
  return { name: raw.name, sources: raw.sources.map(String) }
}

function parseAdmittedComponent(raw: unknown): PublicFrontendPolicyComponent {
  if (!isObject(raw)
    || !ID_PATTERN.test(String(raw.extensionId || ''))
    || !ID_PATTERN.test(String(raw.componentId || ''))
    || !DIGEST_PATTERN.test(String(raw.packageDigest || ''))
    || !DIGEST_PATTERN.test(String(raw.impactDigest || ''))
    || typeof raw.extensionVersion !== 'string'
    || typeof raw.contractVersion !== 'string'
  ) {
    throw new PublicPagePolicyError('invalid admitted public component')
  }
  return {
    extensionId: String(raw.extensionId),
    extensionVersion: String(raw.extensionVersion),
    packageDigest: String(raw.packageDigest),
    impactDigest: String(raw.impactDigest),
    componentId: String(raw.componentId),
    contractVersion: String(raw.contractVersion)
  }
}

function stringProp(
  props: Record<string, string | boolean | number>,
  camel: string,
  kebab: string
): string {
  const value = props[camel] ?? props[kebab]
  return typeof value === 'string' ? value.trim() : ''
}

function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}
