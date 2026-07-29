// Reproducible isolated external-auth HTTP evidence. This is intentionally
// provider-generic: only fixture defaults name the protected reference plugin.
import { createHash } from 'node:crypto'
import { access, mkdir, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'

const api = process.env.SFORUM_API_BASE_URL || 'http://127.0.0.1:8081'
const safeModeAPI = process.env.SFORUM_SAFE_MODE_API_BASE_URL || ''
const artifactDriftAPI = process.env.SFORUM_ARTIFACT_DRIFT_API_BASE_URL || ''
const artifactDriftReadyFile = process.env.SFORUM_ARTIFACT_DRIFT_READY_FILE || ''
const artifactDriftAppliedFile = process.env.SFORUM_ARTIFACT_DRIFT_APPLIED_FILE || ''
const outputPath = process.env.SFORUM_EVIDENCE_OUTPUT || ''
const providerID = process.env.SFORUM_AUTH_PROVIDER_ID || 'sforum.auth-github.auth'
const extensionID = process.env.SFORUM_AUTH_EXTENSION_ID || 'sforum.auth-github'
const adminLogin = process.env.SFORUM_EVIDENCE_ADMIN_LOGIN || 'external-auth-admin'
const adminPassword = process.env.SFORUM_EVIDENCE_ADMIN_PASSWORD || 'ExternalAuthAdminPass123!'
const externalLogin = process.env.SFORUM_EVIDENCE_EXTERNAL_LOGIN || 'external-auth-user'
const externalPassword = process.env.SFORUM_EVIDENCE_EXTERNAL_PASSWORD || 'ExternalAuthUserPass123!'
const clientID = process.env.SFORUM_FAKE_GITHUB_CLIENT_ID || 't8d-github-client'
const clientSecret = process.env.SFORUM_FAKE_GITHUB_CLIENT_SECRET || 't8d-github-secret'

function expect(condition, message, evidence) {
  if (condition) return
  const error = new Error(message)
  error.evidence = evidence
  throw error
}

function splitSetCookie(header) {
  return header ? header.split(/,(?=\s*[^;,=]+=[^;,]+)/g).map(value => value.trim()) : []
}

function safeURL(value) {
  if (!value) return ''
  try {
    const url = new URL(value, api)
    return `${url.origin}${url.pathname}${url.search ? `?${[...url.searchParams.keys()].sort().join('&')}` : ''}`
  } catch {
    return '[unparseable-url]'
  }
}

function redact(value, key = '') {
  const lower = key.toLowerCase()
  if (/(password|secret|token|code|state|verifier|cookie|providersubject|subjectdigest)/.test(lower)) return '[redacted]'
  if (Array.isArray(value)) return value.map(item => redact(item))
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([childKey, childValue]) => [childKey, redact(childValue, childKey)]))
  }
  if (typeof value === 'string' && /^https?:\/\//.test(value)) return safeURL(value)
  return value
}

function summary(response) {
  const data = response.body?.data
  return redact({
    status: response.status,
    location: safeURL(response.location),
    reason: data?.reason || response.body?.reason || '',
    data: data && typeof data === 'object' && !Array.isArray(data)
      ? Object.fromEntries(Object.entries(data).filter(([key]) => [
        'id', 'status', 'providerId', 'operation', 'ready', 'ok', 'reason',
        'packageDigest', 'runtime', 'revision', 'artifactBound', 'enabled',
        'configured', 'probed', 'publiclyActivated', 'safeMode', 'loginEnabled',
        'registrationEnabled', 'linkEnabled', 'ownerExtensionVersion'
      ].includes(key)))
      : data
  })
}

function hasPasswordNamedKey(value) {
  if (Array.isArray(value)) return value.some(item => hasPasswordNamedKey(item))
  if (!value || typeof value !== 'object') return false
  return Object.entries(value).some(([key, childValue]) => /password/i.test(key) || hasPasswordNamedKey(childValue))
}

class CookieJar {
  constructor(base = api) {
    this.base = base
    this.cookies = new Map()
  }

  set(response) {
    const values = typeof response.headers.getSetCookie === 'function'
      ? response.headers.getSetCookie()
      : splitSetCookie(response.headers.get('set-cookie'))
    for (const value of values) {
      const [pair] = value.split(';', 1)
      const index = pair.indexOf('=')
      if (index > 0) this.cookies.set(pair.slice(0, index), pair.slice(index + 1))
    }
  }

  async request(path, { method = 'GET', json, headers = {}, redirect = 'manual' } = {}) {
    const requestHeaders = { ...headers }
    const cookie = [...this.cookies.entries()].map(([name, value]) => `${name}=${value}`).join('; ')
    if (cookie) requestHeaders.Cookie = cookie
    if (json !== undefined) {
      requestHeaders['Content-Type'] = 'application/json'
      if (method !== 'GET' && method !== 'HEAD') requestHeaders['X-CSRF-Token'] = this.cookies.get('csrf_') || ''
    }
    const response = await fetch(path.startsWith('http') ? path : `${this.base}${path}`, {
      method, headers: requestHeaders, redirect, body: json === undefined ? undefined : JSON.stringify(json)
    })
    this.set(response)
    const text = await response.text()
    let body = null
    try { body = text ? JSON.parse(text) : null } catch { body = text }
    return { status: response.status, location: response.headers.get('location') || '', body }
  }
}

function compactProvider(item) {
  return redact({
    id: item.id,
    enabled: item.enabled,
    configured: item.configured,
    probed: item.probed,
    artifactBound: item.artifactBound,
    publiclyActivated: item.publiclyActivated,
    safeMode: item.safeMode,
    revision: item.revision,
    ownerExtensionVersion: item.ownerExtensionVersion,
    packageDigest: item.ownerPackageDigest
  })
}

async function provider(jar) {
  const response = await jar.request('/api/v1/admin/identity/providers')
  const item = response.body?.data?.find(candidate => candidate.id === providerID)
  expect(response.status === 200 && item, 'admin provider is unavailable', summary(response))
  return item
}

async function assertLocalLogin(jar, login, credential) {
  const catalog = await jar.request('/api/v1/auth/providers')
  expect(catalog.status === 200, 'password login could not obtain CSRF catalog', summary(catalog))
  const response = await jar.request('/api/v1/auth/login', {
    method: 'POST',
    json: { login, password: credential, humanVerification: { provider: 'disabled', token: '' } }
  })
  expect(response.status === 200, 'password login failed', summary(response))
  const session = await jar.request('/api/v1/auth/session')
  const sessionUserMatches = session.body?.data?.username === login
  expect(session.status === 200 && sessionUserMatches, 'password session was not issued', summary(session))
}

async function patchActivation(jar, item, operations) {
  const response = await jar.request(`/api/v1/admin/identity/providers/${providerID}`, {
    method: 'PATCH',
    json: { expectedRevision: item.revision, priority: item.priority || 100, ...operations }
  })
  expect(response.status === 200, 'provider activation update failed', summary(response))
  return summary(response)
}

async function probe(jar) {
  const response = await jar.request(`/api/v1/admin/identity/providers/${providerID}/probe`, { method: 'POST', json: {} })
  expect(response.status === 200 && response.body?.data?.ok === true, 'provider probe failed', summary(response))
  return summary(response)
}

async function oauthRoundTrip(jar, operation, suffix) {
  const catalog = await jar.request('/api/v1/auth/providers')
  expect(catalog.status === 200, `${operation} start could not obtain CSRF catalog`, summary(catalog))
  const started = await jar.request(`/api/v1/auth/providers/${providerID}/${operation}/start`, {
    method: 'POST', json: { correlationId: `evidence-${operation}-${suffix}`, redirectHint: '/after-external-auth' }
  })
  expect(started.status === 200 && started.body?.data?.redirectUrl, `${operation} start failed`, summary(started))
  const authorization = await fetch(started.body.data.redirectUrl, { redirect: 'manual' })
  const callbackURL = authorization.headers.get('location') || ''
  expect(authorization.status >= 300 && authorization.status < 400 && callbackURL, `${operation} authorization did not redirect`, {
    status: authorization.status, location: safeURL(callbackURL)
  })
  const callback = await jar.request(callbackURL, { redirect: 'manual' })
  return { start: summary(started), authorize: { status: authorization.status, location: safeURL(callbackURL) }, callback, callbackURL }
}

async function checkPublicCatalog(jar, present) {
  const response = await jar.request('/api/v1/auth/providers')
  const item = response.body?.data?.find(candidate => candidate.id === providerID)
  expect(response.status === 200 && Boolean(item) === present, 'public provider catalog did not match expected state', summary(response))
  if (item) expect(!Object.hasOwn(item, 'ownerPackageDigest') && !Object.hasOwn(item, 'subjectDigest'), 'public catalog leaked private identity data', item)
  return { response: summary(response), count: response.body?.data?.length || 0, present: Boolean(item) }
}

async function checkRateLimit() {
  const jar = new CookieJar()
  const catalog = await jar.request('/api/v1/auth/providers')
  expect(catalog.status === 200, 'rate limit fixture could not obtain CSRF catalog', summary(catalog))
  let limited = null
  for (let index = 0; index < 25; index += 1) {
    const response = await jar.request(`/api/v1/auth/providers/${providerID}/login/start`, {
      method: 'POST', json: { correlationId: `rate-evidence-${index}`, redirectHint: '/' }
    })
    if (response.status === 429) {
      limited = response
      break
    }
  }
  expect(limited?.status === 429, 'external auth start never returned HTTP 429', limited && summary(limited))
  return summary(limited)
}

async function checkSafeMode(admin) {
  expect(safeModeAPI, 'SFORUM_SAFE_MODE_API_BASE_URL is required for R7 Safe Mode evidence')
  const jar = new CookieJar(safeModeAPI)
  jar.cookies = new Map(admin.cookies)
  const ready = await jar.request('/api/v1/ready')
  expect(ready.status === 200 && ready.body?.data?.ready === true, 'Safe Mode API is not ready', summary(ready))
  const item = await provider(jar)
  expect(item.safeMode === true && item.publiclyActivated === false, 'Safe Mode did not suppress public provider activation', compactProvider(item))
  const catalog = await checkPublicCatalog(jar, false)
  const startJar = new CookieJar(safeModeAPI)
  await startJar.request('/api/v1/auth/providers')
  const start = await startJar.request(`/api/v1/auth/providers/${providerID}/login/start`, {
    method: 'POST', json: { correlationId: 'safe-mode-start', redirectHint: '/' }
  })
  expect(start.status === 503, 'Safe Mode login start did not fail closed', summary(start))
  return { ready: summary(ready), provider: compactProvider(item), catalog, start: summary(start) }
}

async function checkArtifactDrift(admin) {
  expect(artifactDriftAPI, 'SFORUM_ARTIFACT_DRIFT_API_BASE_URL is required for R7 artifact-drift evidence')
  const jar = new CookieJar(artifactDriftAPI)
  jar.cookies = new Map(admin.cookies)
  let ready
  let lastError
  for (let attempt = 0; attempt < 60; attempt += 1) {
    try {
      ready = await jar.request('/api/v1/ready')
      if (ready.status === 200) break
    } catch (error) {
      lastError = error
    }
    await new Promise(resolveDelay => setTimeout(resolveDelay, 250))
  }
  if (!ready) throw lastError || new Error('artifact-drift API did not start')
  expect(ready.status === 200 && ready.body?.data?.ready === true, 'artifact-drift API is not ready', summary(ready))
  const item = await provider(jar)
  expect(item.artifactBound === false && item.publiclyActivated === false, 'artifact drift retained executable public authority', compactProvider(item))
  const catalog = await checkPublicCatalog(jar, false)
  const startJar = new CookieJar(artifactDriftAPI)
  await startJar.request('/api/v1/auth/providers')
  const start = await startJar.request(`/api/v1/auth/providers/${providerID}/login/start`, {
    method: 'POST', json: { correlationId: 'artifact-drift-start', redirectHint: '/' }
  })
  expect(start.status === 503, 'artifact-drift login start did not fail closed', summary(start))
  return { ready: summary(ready), provider: compactProvider(item), catalog, start: summary(start) }
}

async function signalArtifactDriftFixture() {
  if (!artifactDriftReadyFile) return
  await mkdir(dirname(resolve(artifactDriftReadyFile)), { recursive: true })
  await writeFile(resolve(artifactDriftReadyFile), 'ready\n', 'utf8')
  if (!artifactDriftAppliedFile) return
  for (let attempt = 0; attempt < 120; attempt += 1) {
    try {
      await access(resolve(artifactDriftAppliedFile))
      return
    } catch {
      await new Promise(resolveDelay => setTimeout(resolveDelay, 100))
    }
  }
  throw new Error('artifact-drift fixture did not confirm its durable update')
}

async function writeEvidenceArtifact(document) {
  if (!outputPath) return ''
  // This is a reproducibility checksum for validated public evidence, not credential derivation or storage.
  const digest = createHash('sha256').update(document).digest('hex')
  const target = resolve(outputPath)
  await mkdir(dirname(target), { recursive: true })
  await writeFile(target, document, 'utf8')
  await writeFile(`${target}.sha256`, `${digest}  ${target.split('/').pop()}\n`, 'utf8')
  return digest
}

async function main() {
  const evidence = { version: 1, api, providerID, extensionID, checks: {}, artifacts: {}, final: {} }
  const admin = new CookieJar()

  const ready = await admin.request('/api/v1/ready')
  expect(ready.status === 200 && ready.body?.data?.ready === true, 'isolated API is not ready', summary(ready))
  evidence.checks.readiness = summary(ready)

  const register = await admin.request('/api/v1/auth/register', {
    method: 'POST', json: {
      username: adminLogin, email: `${adminLogin}@example.test`, password: adminPassword,
      displayName: 'External Auth Evidence Admin', humanVerification: { provider: 'disabled', token: '' }
    }
  })
  expect(register.status === 201 || register.status === 409 || register.status === 422, 'isolated first-admin bootstrap failed', summary(register))
  evidence.checks.bootstrapAdmin = register.status === 201 ? summary(register) : { status: register.status, resumed: true }
  await assertLocalLogin(admin, adminLogin, adminPassword)
  evidence.checks.adminCredentialFallback = { verified: true }

  const extension = await admin.request(`/api/v1/admin/extensions?id=${encodeURIComponent(extensionID)}`)
  const packageDigest = extension.body?.data?.[0]?.packageDigest
  const extensionVersion = extension.body?.data?.[0]?.version
  expect(extension.status === 200 && packageDigest && extensionVersion, 'protected built-in package was not discovered', summary(extension))
  evidence.artifacts = { extensionVersion, packageDigest }

  let settings = await admin.request(`/api/v1/admin/extensions/${extensionID}/settings`)
  let secret = settings.body?.data?.items?.find(item => item.key === 'client_secret')
  if (settings.status === 200 && secret?.secretSet !== true) {
    settings = await admin.request(`/api/v1/admin/extensions/${extensionID}/settings`, {
      method: 'PUT', json: { values: { client_id: clientID, client_secret: clientSecret } }
    })
    secret = settings.body?.data?.items?.find(item => item.key === 'client_secret')
  }
  expect(settings.status === 200 && secret?.value === '' && secret?.secretSet === true, 'secret setting was not redacted', summary(settings))
  evidence.checks.configureAndSecretRedaction = { status: settings.status, clientSecretRedacted: true }

  const enable = await admin.request(`/api/v1/admin/extensions/${extensionID}/enable`, {
    method: 'POST', headers: { 'Idempotency-Key': 'external-auth-evidence-enable-1' }, json: { confirmCapabilities: true }
  })
  expect(enable.status === 200 && enable.body?.data?.status === 'enabled', 'extension enable failed', summary(enable))
  evidence.checks.extensionEnable = summary(enable)

  let item = await provider(admin)
  expect(item.enabled && item.configured, 'enabled provider is not configured and executable', compactProvider(item))
  evidence.checks.probe = await probe(admin)
  evidence.checks.activate = await patchActivation(admin, item, { loginEnabled: true, registrationEnabled: true, linkEnabled: true })
  item = await provider(admin)
  expect(item.publiclyActivated, 'provider did not become publicly activated', compactProvider(item))
  evidence.artifacts.runtime = compactProvider(item)
  evidence.checks.publicCatalog = await checkPublicCatalog(admin, true)

  const unlinked = new CookieJar()
  const unlinkedLogin = await oauthRoundTrip(unlinked, 'login', 'unlinked')
  expect(unlinkedLogin.callback.status === 302 && unlinkedLogin.callback.location.includes('auth.external_identity_unlinked'), 'unlinked external login did not fail safely', summary(unlinkedLogin.callback))
  const replay = await unlinked.request(unlinkedLogin.callbackURL, { redirect: 'manual' })
  expect(replay.status === 302 && replay.location.includes('auth.provider_callback_replayed'), 'callback replay was not rejected', summary(replay))
  evidence.checks.callbackCleanupAndReplay = { callback: summary(unlinkedLogin.callback), replay: summary(replay) }

  const external = new CookieJar()
  const registration = await oauthRoundTrip(external, 'registration', 'registration')
  const registrationLocation = new URL(registration.callback.location, api)
  const ticket = registrationLocation.searchParams.get('ticket') || ''
  expect(registration.callback.status === 302 && registrationLocation.pathname === '/register' && ticket, 'external registration callback did not create an opaque continuation', summary(registration.callback))
  const created = await external.request('/api/v1/auth/external-registration', {
    method: 'POST', json: {
      ticket, username: externalLogin, email: `${externalLogin}@example.test`, displayName: 'External Auth Evidence User', locale: 'en-US',
      humanVerification: { provider: 'disabled', token: '' }
    }
  })
  expect(created.status === 201, 'explicit external registration failed', summary(created))
  evidence.checks.explicitExternalRegistration = summary(created)

  let identities = await external.request('/api/v1/auth/external-identities')
  const identity = identities.body?.data?.[0]
  expect(identities.status === 200 && identity?.linkId && !Object.hasOwn(identity, 'subjectDigest') && !Object.hasOwn(identity, 'providerSubject'), 'external identities were missing or leaked raw subject data', redact(identities.body))
  const unlinkBlocked = await external.request(`/api/v1/auth/external-identities/${identity.linkId}`, { method: 'DELETE', json: { requestId: 'unlink-before-password' } })
  expect(unlinkBlocked.status === 422 && unlinkBlocked.body?.data?.reason === 'auth.last_login_method_required', 'last login method protection was not enforced', summary(unlinkBlocked))
  const credentialSetup = await external.request('/api/v1/auth/password', { method: 'POST', json: { password: externalPassword } })
  const credentialSetupResponseEmpty = credentialSetup.status === 204 || (credentialSetup.status === 200 && credentialSetup.body?.data === null)
  expect(credentialSetupResponseEmpty, 'external-only password setup failed', summary(credentialSetup))
  const unlink = await external.request(`/api/v1/auth/external-identities/${identity.linkId}`, { method: 'DELETE', json: { requestId: 'unlink-after-password' } })
  expect(unlink.status === 204 || (unlink.status === 200 && unlink.body?.data === null), 'unlink after password setup failed', summary(unlink))
  const link = await oauthRoundTrip(external, 'link', 'relink')
  expect(link.callback.status === 302 && link.callback.location.includes('auth.external_link_ok'), 'external account relink failed', summary(link.callback))
  identities = await external.request('/api/v1/auth/external-identities')
  expect(identities.status === 200 && identities.body?.data?.filter(row => row.status === 'active').length === 1, 'relink did not restore one active identity', redact(identities.body))
  evidence.checks.linkUnlinkAndCredentialSetup = {
    unlinkBlocked: summary(unlinkBlocked),
    credentialSetup: { status: credentialSetup.status, responseEmpty: credentialSetupResponseEmpty },
    unlink: summary(unlink),
    link: summary(link.callback)
  }
  await assertLocalLogin(new CookieJar(), externalLogin, externalPassword)
  evidence.checks.externalCredentialFallback = { verified: true }

  evidence.checks.safeMode = await checkSafeMode(admin)
  await signalArtifactDriftFixture()
  evidence.checks.artifactDrift = await checkArtifactDrift(admin)

  const disable = await admin.request(`/api/v1/admin/extensions/${extensionID}/disable`, {
    method: 'POST', headers: { 'Idempotency-Key': 'external-auth-evidence-disable-1' }, json: {}
  })
  expect(disable.status === 200 && disable.body?.data?.status === 'disabled', 'real extension disable failed', summary(disable))
  const disabled = await provider(admin)
  expect(!disabled.enabled && !disabled.publiclyActivated, 'disabled extension retained live identity authority', compactProvider(disabled))
  const disabledCatalog = await checkPublicCatalog(admin, false)
  const disabledStartJar = new CookieJar()
  await disabledStartJar.request('/api/v1/auth/providers')
  const disabledStart = await disabledStartJar.request(`/api/v1/auth/providers/${providerID}/login/start`, { method: 'POST', json: { correlationId: 'disabled-start', redirectHint: '/' } })
  expect(disabledStart.status === 503, 'disabled extension start did not fail closed', summary(disabledStart))
  evidence.checks.realExtensionDisable = { disable: summary(disable), provider: compactProvider(disabled), catalog: disabledCatalog, start: summary(disabledStart) }

  const restoredEnable = await admin.request(`/api/v1/admin/extensions/${extensionID}/enable`, {
    method: 'POST', headers: { 'Idempotency-Key': 'external-auth-evidence-enable-2' }, json: { confirmCapabilities: true }
  })
  expect(restoredEnable.status === 200 && restoredEnable.body?.data?.status === 'enabled', 'extension re-enable failed', summary(restoredEnable))
  item = await provider(admin)
  evidence.checks.restoreProbe = await probe(admin)
  evidence.checks.restoreActivation = await patchActivation(admin, item, { loginEnabled: true, registrationEnabled: true, linkEnabled: true })
  evidence.final.catalog = await checkPublicCatalog(admin, true)
  const finalProvider = await provider(admin)
  expect(finalProvider.publiclyActivated && finalProvider.artifactBound, 'final provider state was not restored', compactProvider(finalProvider))
  evidence.final.provider = compactProvider(finalProvider)
  evidence.checks.rateLimit = await checkRateLimit()

  expect(!hasPasswordNamedKey(evidence), 'evidence JSON contains a password-named output field')
  const evidenceDocument = `${JSON.stringify(evidence, null, 2)}\n`
  const submittedSecrets = [adminPassword, externalPassword, clientSecret]
  expect(!submittedSecrets.some(value => value && evidenceDocument.includes(value)), 'evidence JSON contains a submitted credential')
  expect(!/(isolated-github-code|access-token|subjectDigest|providerSubject)/i.test(evidenceDocument), 'evidence JSON contains prohibited sensitive material')
  evidence.outputSHA256 = await writeEvidenceArtifact(evidenceDocument)
  console.log(JSON.stringify(evidence, null, 2))
}

main().catch(error => {
  console.error(JSON.stringify({ error: error.message, evidence: redact(error.evidence) }, null, 2))
  process.exit(1)
})
