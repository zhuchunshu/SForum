import {
  parsePublicFrontendAssetReference,
  PublicFrontendContractError,
  type PublicFrontendAssetReference
} from './types'

export const PUBLIC_CONTENT_STYLES_SCHEMA_VERSION = 'sforum.public-content-styles@1'
export const PUBLIC_CONTENT_STYLE_SCOPE = 'core.surface.forum-content'

const DIGEST_PATTERN = /^[0-9a-f]{64}$/
const MAX_STYLES = 64

export type PublicContentStyleCatalog = {
  schemaVersion: typeof PUBLIC_CONTENT_STYLES_SCHEMA_VERSION
  graphDigest: string
  styles: PublicFrontendAssetReference[]
}

export function parsePublicContentStyleCatalog(input: unknown): PublicContentStyleCatalog {
  if (!isRecord(input)
    || input.schemaVersion !== PUBLIC_CONTENT_STYLES_SCHEMA_VERSION
    || !DIGEST_PATTERN.test(String(input.graphDigest || ''))
    || !Array.isArray(input.styles)
    || input.styles.length > MAX_STYLES) {
    throw new PublicFrontendContractError('invalid public content style catalog')
  }
  const styles = input.styles.map(parsePublicFrontendAssetReference)
  const handles = new Set<string>()
  for (const style of styles) {
    if (style.type !== 'style'
      || style.module
      || style.loading !== 'blocking'
      || !style.scope.includes(PUBLIC_CONTENT_STYLE_SCOPE)
      || handles.has(style.handle)) {
      throw new PublicFrontendContractError('invalid public content stylesheet')
    }
    handles.add(style.handle)
  }
  return {
    schemaVersion: PUBLIC_CONTENT_STYLES_SCHEMA_VERSION,
    graphDigest: String(input.graphDigest),
    styles
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}
