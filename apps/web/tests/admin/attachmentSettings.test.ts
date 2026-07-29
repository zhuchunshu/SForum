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
})
