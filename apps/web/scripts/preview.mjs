import { existsSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { printSForumPreviewBanner } from './sforum-preview-banner.mjs'

const serverEntry = new URL('../.output/server/index.mjs', import.meta.url)

if (!existsSync(fileURLToPath(serverEntry))) {
  console.error('SForum preview build not found. Run `bun run build` before `bun run preview`.')
  process.exit(1)
}

printSForumPreviewBanner()
await import(serverEntry.href)
