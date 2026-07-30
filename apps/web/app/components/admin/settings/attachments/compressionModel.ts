export type AttachmentCompressionSettings = {
  enabled: boolean
  strength: number
  maxDimension: number
  minSizeKb: number
  minSavingsPercent: number
  jpegQuality: number
  policyDigest: string
}

export type AttachmentCompressionStats = {
  pending: number
  running: number
  failed: number
  readyVariants: number
  originalBytes: number
  variantBytes: number
  savedBytes: number
}

export const recommendedAttachmentCompressionSettings: AttachmentCompressionSettings = {
  enabled: true,
  strength: 55,
  maxDimension: 2560,
  minSizeKb: 256,
  minSavingsPercent: 8,
  jpegQuality: 81,
  policyDigest: ''
}

export function compressionJPEGQuality(strength: number) {
  const normalized = clampInteger(strength, 0, 100)
  return 95 - Math.floor((normalized * 25 + 50) / 100)
}

export function estimatedCompressionRatio(strength: number) {
  const normalized = clampInteger(strength, 0, 100)
  return 0.78 - normalized * 0.0032
}

export function estimatedOutputBytes(originalBytes: number, strength: number) {
  if (!Number.isFinite(originalBytes) || originalBytes <= 0) return 0
  return Math.max(1, Math.round(originalBytes * estimatedCompressionRatio(strength)))
}

export function normalizeAttachmentCompressionSettings(value?: Partial<AttachmentCompressionSettings>): AttachmentCompressionSettings {
  const strength = clampInteger(value?.strength ?? recommendedAttachmentCompressionSettings.strength, 0, 100)
  return {
    enabled: value?.enabled ?? recommendedAttachmentCompressionSettings.enabled,
    strength,
    maxDimension: clampInteger(value?.maxDimension ?? recommendedAttachmentCompressionSettings.maxDimension, 320, 8192),
    minSizeKb: Math.max(1, clampInteger(value?.minSizeKb ?? recommendedAttachmentCompressionSettings.minSizeKb, 1, 1024 * 1024)),
    minSavingsPercent: clampInteger(value?.minSavingsPercent ?? recommendedAttachmentCompressionSettings.minSavingsPercent, 0, 90),
    jpegQuality: compressionJPEGQuality(strength),
    policyDigest: typeof value?.policyDigest === 'string' ? value.policyDigest : ''
  }
}

export function isRecommendedAttachmentCompressionSettings(value: AttachmentCompressionSettings) {
  const recommended = recommendedAttachmentCompressionSettings
  return value.enabled === recommended.enabled
    && value.strength === recommended.strength
    && value.maxDimension === recommended.maxDimension
    && value.minSizeKb === recommended.minSizeKb
    && value.minSavingsPercent === recommended.minSavingsPercent
}

function clampInteger(value: number, min: number, max: number) {
  const normalized = Number.isFinite(value) ? Math.round(value) : min
  return Math.min(max, Math.max(min, normalized))
}
