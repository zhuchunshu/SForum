import { describe, expect, test } from 'bun:test'

import {
  createDefaultAttachmentSettings,
  isRecommendedAttachmentSettings,
  resetAttachmentSettingsToRecommended
} from '../../app/utils/attachments/attachmentSettings'
import {
  attachmentStatusColor,
  buildAttachmentListQuery,
  humanFileSize
} from '../../app/components/admin/settings/attachments/model'
import {
  compressionJPEGQuality,
  estimatedOutputBytes,
  isRecommendedAttachmentCompressionSettings,
  normalizeAttachmentCompressionSettings,
  recommendedAttachmentCompressionSettings
} from '../../app/components/admin/settings/attachments/compressionModel'

const [zhCN, enUS] = await Promise.all([
  Bun.file(new URL('../../i18n/locales/zh-CN.json', import.meta.url)).json(),
  Bun.file(new URL('../../i18n/locales/en-US.json', import.meta.url)).json()
]) as Array<Record<string, any>>

describe('attachment settings defaults', () => {
  test('keeps storage instance translations in the attachment namespace', () => {
    expect(zhCN.admin.attachments.storageInstances.title).toBe('存储服务实例')
    expect(enUS.admin.attachments.storageInstances.title).toBe('Storage service instances')
    expect(zhCN.admin.home.storageInstances).toBeUndefined()
    expect(enUS.admin.home.storageInstances).toBeUndefined()
  })

  test('provides a beginner-friendly local storage default', () => {
    const defaults = createDefaultAttachmentSettings()

    expect(defaults.provider).toBe('local')
    expect(defaults.uploadEnabled).toBe(true)
    expect(defaults.maxFileSizeMb).toBe(20)
    expect(defaults.pathTemplate).toBe('{yyyy}/{mm}/{dd}/{public_id}{ext}')
    expect(defaults.local.root).toBe('storage/app/attachments')
    expect(defaults.allowedExtensions).toContain('.jpg')
    expect(defaults.allowedExtensions).toContain('.pdf')
    expect(defaults.allowedMimeTypes).toContain('image/png')
    expect(defaults.allowedMimeTypes).toContain('application/zip')
    expect(isRecommendedAttachmentSettings(defaults)).toBe(true)
  })

  test('resets custom attachment settings to the recommended defaults', () => {
    const custom = createDefaultAttachmentSettings()
    custom.provider = 'plugin:acme.storage'
    custom.uploadEnabled = false
    custom.maxFileSizeMb = 200
    custom.pathTemplate = 'custom/{public_id}{ext}'

    const restored = resetAttachmentSettingsToRecommended(custom)

    expect(restored.provider).toBe('local')
    expect(restored.uploadEnabled).toBe(true)
    expect(restored.maxFileSizeMb).toBe(20)
    expect(restored.pathTemplate).toBe('{yyyy}/{mm}/{dd}/{public_id}{ext}')
    expect(isRecommendedAttachmentSettings(restored)).toBe(true)
  })

  test('detects when a setting has moved away from the recommended defaults', () => {
    const changed = createDefaultAttachmentSettings()

    changed.cleanupOrphanAfterDays = 7

    expect(isRecommendedAttachmentSettings(changed)).toBe(false)
  })

  test('builds an exact manager filter query without empty fields', () => {
    const query = buildAttachmentListQuery(
      { page: 2, perPage: 50 },
      { query: 'logo', provider: 'plugin:sforum.storage-fs', status: 'active', contentType: '', referenceStatus: 'referenced' }
    )
    const params = new URLSearchParams(query)

    expect(Object.fromEntries(params)).toEqual({
      page: '2',
      perPage: '50',
      query: 'logo',
      provider: 'plugin:sforum.storage-fs',
      status: 'active',
      referenceStatus: 'referenced'
    })
  })

  test('keeps manager presentation helpers deterministic', () => {
    expect(humanFileSize(512)).toBe('512 B')
    expect(humanFileSize(1536)).toBe('1.5 KB')
    expect(attachmentStatusColor('active')).toBe('success')
    expect(attachmentStatusColor('disabled')).toBe('warning')
    expect(attachmentStatusColor('deleted')).toBe('neutral')
  })

  test('keeps compression strength aligned with the backend JPEG quality mapping', () => {
    expect(compressionJPEGQuality(0)).toBe(95)
    expect(compressionJPEGQuality(55)).toBe(81)
    expect(compressionJPEGQuality(100)).toBe(70)
    expect(compressionJPEGQuality(-10)).toBe(95)
    expect(compressionJPEGQuality(110)).toBe(70)
  })

  test('provides a monotonic adjacent size estimate without presenting it as exact output', () => {
    const original = 5 * 1024 * 1024

    expect(estimatedOutputBytes(original, 80)).toBeLessThan(estimatedOutputBytes(original, 20))
    expect(estimatedOutputBytes(0, 55)).toBe(0)
    expect(zhCN.admin.attachments.compression.estimateDisclaimer).toContain('近似值')
    expect(enUS.admin.attachments.compression.estimateDisclaimer).toContain('approximation')
  })

  test('normalizes and recognizes recommended compression settings', () => {
    const recommended = normalizeAttachmentCompressionSettings(recommendedAttachmentCompressionSettings)
    const custom = normalizeAttachmentCompressionSettings({ strength: 90, maxDimension: 1200 })

    expect(recommended.jpegQuality).toBe(81)
    expect(isRecommendedAttachmentCompressionSettings(recommended)).toBe(true)
    expect(isRecommendedAttachmentCompressionSettings(custom)).toBe(false)
    expect(normalizeAttachmentCompressionSettings({ strength: 200 }).strength).toBe(100)
  })
})
