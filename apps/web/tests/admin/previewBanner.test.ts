import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

import { sforumPreviewBanner } from '../../scripts/sforum-preview-banner.mjs'

describe('preview startup banner', () => {
  test('uses the SForum web preview brand', () => {
    expect(sforumPreviewBanner).toContain('SForum Web Preview')
    expect(sforumPreviewBanner).not.toContain('Fiber')
  })

  test('bun preview routes through the banner wrapper before Nitro starts', () => {
    const packageJson = JSON.parse(readFileSync(new URL('../../package.json', import.meta.url), 'utf8'))
    const previewScript = readFileSync(new URL('../../scripts/preview.mjs', import.meta.url), 'utf8')

    expect(packageJson.scripts.preview).toContain('scripts/preview.mjs')
    expect(previewScript).toContain('printSForumPreviewBanner()')
    expect(previewScript).toContain("../.output/server/index.mjs")
  })
})
