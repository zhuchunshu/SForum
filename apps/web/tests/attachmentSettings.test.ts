import { describe, expect, test } from 'bun:test'

import {
  createDefaultAttachmentSettings,
  isRecommendedAttachmentSettings,
  resetAttachmentSettingsToRecommended
} from '../app/utils/attachmentSettings'

describe('attachment settings defaults', () => {
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
    custom.provider = 'aliyun_oss'
    custom.uploadEnabled = false
    custom.maxFileSizeMb = 200
    custom.pathTemplate = 'custom/{public_id}{ext}'
    custom.aliyunOss.accessKeySecret = 'typed-but-unsaved'
    custom.aliyunOss.accessKeySecretSet = true

    const restored = resetAttachmentSettingsToRecommended(custom)

    expect(restored.provider).toBe('local')
    expect(restored.uploadEnabled).toBe(true)
    expect(restored.maxFileSizeMb).toBe(20)
    expect(restored.pathTemplate).toBe('{yyyy}/{mm}/{dd}/{public_id}{ext}')
    expect(restored.aliyunOss.accessKeySecret).toBe('')
    expect(restored.aliyunOss.accessKeySecretSet).toBe(true)
    expect(isRecommendedAttachmentSettings(restored)).toBe(true)
  })

  test('detects when a setting has moved away from the recommended defaults', () => {
    const changed = createDefaultAttachmentSettings()

    changed.cleanupOrphanAfterDays = 7

    expect(isRecommendedAttachmentSettings(changed)).toBe(false)
  })
})
