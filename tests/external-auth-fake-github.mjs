// 仅供隔离 external-auth 证据使用的 GitHub OAuth 上游；不暴露到非 loopback 网络。
import http from 'node:http'
import { createHash } from 'node:crypto'

const port = Number(process.env.SFORUM_FAKE_GITHUB_PORT || '18080')
const clientID = process.env.SFORUM_FAKE_GITHUB_CLIENT_ID || 't8d-github-client'
const clientSecret = process.env.SFORUM_FAKE_GITHUB_CLIENT_SECRET || 't8d-github-secret'
const issued = new Map()
let sequence = 0

function sendJSON(response, status, value) {
  const body = JSON.stringify(value)
  response.writeHead(status, { 'content-type': 'application/json', 'content-length': Buffer.byteLength(body) })
  response.end(body)
}

async function body(request) {
  const chunks = []
  for await (const chunk of request) chunks.push(chunk)
  return Buffer.concat(chunks).toString('utf8')
}

function challenge(verifier) {
  return createHash('sha256').update(verifier).digest('base64url')
}

const server = http.createServer(async (request, response) => {
  const url = new URL(request.url || '/', `http://127.0.0.1:${port}`)
  if (request.method === 'GET' && url.pathname === '/login/oauth/authorize') {
    const redirectURI = url.searchParams.get('redirect_uri')
    const state = url.searchParams.get('state')
    const codeChallenge = url.searchParams.get('code_challenge')
    if (!redirectURI || !state || !codeChallenge) return sendJSON(response, 400, { error: 'invalid_request' })
    const code = `isolated-github-code-${++sequence}`
    issued.set(code, { codeChallenge, used: false })
    const callback = new URL(redirectURI)
    callback.searchParams.set('code', code)
    callback.searchParams.set('state', state)
    response.writeHead(302, { location: callback.toString() })
    return response.end()
  }
  if (request.method === 'POST' && url.pathname === '/login/oauth/access_token') {
    const form = new URLSearchParams(await body(request))
    const record = issued.get(form.get('code'))
    if (form.get('client_id') !== clientID || form.get('client_secret') !== clientSecret || !record || record.used || challenge(form.get('code_verifier') || '') !== record.codeChallenge) {
      return sendJSON(response, 400, { error: 'invalid_grant' })
    }
    record.used = true
    return sendJSON(response, 200, { access_token: 'isolated-github-access-token', token_type: 'bearer', scope: 'read:user,user:email' })
  }
  if (request.method === 'GET' && url.pathname === '/') return sendJSON(response, 200, { current_user_url: `http://127.0.0.1:${port}/user` })
  if (request.method === 'GET' && url.pathname === '/user') {
    if (request.headers.authorization !== 'Bearer isolated-github-access-token') return sendJSON(response, 401, { message: 'bad credentials' })
    return sendJSON(response, 200, { id: 424242, login: 'octocat-isolated', name: 'Isolated GitHub User', email: 'octocat@example.test' })
  }
  if (request.method === 'GET' && url.pathname === '/user/emails') return sendJSON(response, 200, [{ email: 'octocat@example.test', primary: true, verified: true }])
  return sendJSON(response, 404, { message: 'not found' })
})

server.listen(port, '127.0.0.1', () => console.log(`external-auth fake GitHub listening on ${port}`))
