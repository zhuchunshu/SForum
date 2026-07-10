import { createHash } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { basename, dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const packageRoot = join(root, 'tests/fixtures/npm/sforum-fixture-dependency')
const temporary = mkdtempSync(join(tmpdir(), 'sforum-local-registry-'))
const packed = JSON.parse(execFileSync('npm', ['pack', '--json', '--pack-destination', temporary], {
  cwd: packageRoot,
  encoding: 'utf8'
}))[0]
const tarballName = basename(packed.filename)
const tarball = readFileSync(join(temporary, tarballName))
const integrity = `sha512-${createHash('sha512').update(tarball).digest('base64')}`
const shasum = createHash('sha1').update(tarball).digest('hex')
const requestedPort = Number(process.env.PORT || process.argv[2] || 4873)

const server = createServer((request, response) => {
  const pathname = new URL(request.url || '/', 'http://127.0.0.1').pathname
  if (pathname === `/sforum-fixture-dependency/-/${tarballName}`) {
    response.writeHead(200, { 'content-type': 'application/octet-stream', 'content-length': tarball.length })
    response.end(tarball)
    return
  }
  if (pathname === '/sforum-fixture-dependency') {
    const port = server.address().port
    const metadata = {
      name: 'sforum-fixture-dependency',
      'dist-tags': { latest: '1.0.0' },
      versions: {
        '1.0.0': {
          name: 'sforum-fixture-dependency',
          version: '1.0.0',
          type: 'module',
          main: 'index.js',
          types: 'index.d.ts',
          exports: { '.': { types: './index.d.ts', default: './index.js' } },
          scripts: { postinstall: 'node postinstall.mjs' },
          dist: {
            tarball: `http://127.0.0.1:${port}/sforum-fixture-dependency/-/${tarballName}`,
            integrity,
            shasum
          }
        }
      }
    }
    response.writeHead(200, { 'content-type': 'application/json' })
    response.end(JSON.stringify(metadata))
    return
  }
  response.writeHead(404, { 'content-type': 'application/json' })
  response.end(JSON.stringify({ error: 'not_found' }))
})

function close() {
  server.close(() => {
    rmSync(temporary, { recursive: true, force: true })
    process.exit(0)
  })
}

process.on('SIGINT', close)
process.on('SIGTERM', close)
server.listen(requestedPort, '127.0.0.1', () => {
  const address = server.address()
  console.log(JSON.stringify({ ready: true, port: address.port, integrity, tarball: tarballName }))
})
