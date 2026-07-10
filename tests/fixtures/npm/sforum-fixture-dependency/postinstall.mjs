import { writeFile } from 'node:fs/promises'
await writeFile(new URL('postinstall-ran', import.meta.url), 'dependency lifecycle script executed\n')
