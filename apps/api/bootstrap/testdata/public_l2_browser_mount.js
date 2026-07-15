async (page) => {
  const origin = await page.evaluate(() => location.origin)
  const verifyPackageRequests = await page.evaluate(() =>
    sessionStorage.getItem('sforum:p9-public-l2-network-verified') !== '1'
  )
  const requests = []
  const responseErrors = []
  const errors = []
  page.on('request', request => requests.push(request.url()))
  page.on('response', response => {
    const status = response.status()
    if (status < 400) return
    const url = response.url()
    const expectedGuestAuth = status === 401 && (
      url === `${origin}/api/v1/auth/session` ||
      url.startsWith(`${origin}/api/v1/auth/session?`)
    )
    if (!expectedGuestAuth) responseErrors.push(`${status} ${url}`)
  })
  page.on('console', message => {
    if (
      message.type() === 'error' &&
      !message.text().startsWith('Failed to load resource:')
    ) errors.push(message.text())
  })
  page.on('pageerror', error => errors.push(error.message))

  await page.goto(`${origin}/`, { waitUntil: 'networkidle' })
  const widget = page.locator(
    '[data-extension-id="sforum.public-l2-e2e-theme"]' +
    '[data-component-id="sforum.public-l2-e2e-theme.component.card"]'
  )
  await widget.waitFor({ state: 'visible', timeout: 30000 })
  await page.waitForFunction(() => {
    const widget = document.querySelector(
      '[data-extension-id="sforum.public-l2-e2e-theme"]' +
      '[data-component-id="sforum.public-l2-e2e-theme.component.card"]'
    )
    return widget?.getAttribute('data-l2-state') === 'mounted'
  }, null, { timeout: 30000 })

  const mounted = widget.locator('[data-public-l2-mounted]')
  await mounted.waitFor({ state: 'visible', timeout: 10000 })
  const marker = await mounted.getAttribute('data-module-marker')
  const moduleURL = await mounted.getAttribute('data-module-url')
  if (marker !== 'relative-chunk-ready') throw new Error(`relative ESM chunk marker: ${marker}`)
  if (!moduleURL?.includes('/api/v1/extensions/runtime/sforum.public-l2-e2e-theme/packages/')) {
    throw new Error(`module URL is not package-local: ${moduleURL}`)
  }
  const cssReady = await mounted.evaluate(element =>
    getComputedStyle(element).getPropertyValue('--sforum-public-l2-relative-css').trim()
  )
  if (cssReady !== 'ready') throw new Error(`relative CSS resource did not apply: ${cssReady}`)
  if (await widget.locator('[data-public-l2-ssr-fallback]').isVisible()) {
    throw new Error('SSR fallback remained visible after L2 mount')
  }

  await mounted.locator('[data-public-l2-action]').click()
  if ((await mounted.locator('[data-public-l2-count]').textContent())?.trim() !== '1') {
    throw new Error('mounted L2 interaction did not update its real DOM state')
  }
  const packageRequests = requests.filter(url => url.includes(
    '/api/v1/extensions/runtime/sforum.public-l2-e2e-theme/packages/'
  ))
  if (verifyPackageRequests) {
    for (const suffix of ['/card.mjs', '/chunk.mjs', '/card.css', '/nested.css']) {
      if (!packageRequests.some(url => url.endsWith(suffix))) {
        throw new Error(`missing package-local browser request ${suffix}: ${packageRequests.join(', ')}`)
      }
    }
    await page.evaluate(() => sessionStorage.setItem('sforum:p9-public-l2-network-verified', '1'))
  }
  if (responseErrors.length) {
    throw new Error(`browser response errors: ${responseErrors.join(' | ')}`)
  }
  if (errors.length) throw new Error(`browser errors: ${errors.join(' | ')}`)
  return {
    state: await widget.getAttribute('data-l2-state'),
    marker,
    moduleURL,
    cssReady,
    packageRequests
  }
}
