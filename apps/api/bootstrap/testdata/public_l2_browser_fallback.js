async (page) => {
  const widgetSelector =
    '[data-extension-id="sforum.public-l2-e2e-theme"]' +
    '[data-component-id="sforum.public-l2-e2e-theme.component.card"]'
  await page.bringToFront()
  await page.waitForFunction((selector) => {
    const widget = document.querySelector(selector)
    return widget?.getAttribute('data-l2-state') === 'fallback'
  }, widgetSelector, { timeout: 15000 })

  const widget = page.locator(widgetSelector)
  const fallback = widget.locator('[data-public-l2-ssr-fallback]')
  await fallback.waitFor({ state: 'visible', timeout: 10000 })
  if (!(await fallback.textContent())?.includes('P9 public L2 SSR fallback')) {
    throw new Error('visible fallback lost its primary SSR content')
  }
  if (await widget.locator('[data-public-l2-mounted]').count()) {
    throw new Error('revoked L2 left third-party DOM mounted')
  }
  const styles = await page.locator(
    'link[data-sforum-extension="sforum.public-l2-e2e-theme"]'
  ).count()
  if (styles !== 0) throw new Error(`revoked L2 left ${styles} stylesheet lease(s)`)
  return {
    state: await widget.getAttribute('data-l2-state'),
    fallback: (await fallback.textContent())?.trim(),
    styles
  }
}
