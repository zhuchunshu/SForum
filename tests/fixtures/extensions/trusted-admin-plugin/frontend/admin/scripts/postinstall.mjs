import { writeFile } from 'node:fs/promises'
await writeFile(new URL('../postinstall-ran', import.meta.url), 'lifecycle script executed\n')
