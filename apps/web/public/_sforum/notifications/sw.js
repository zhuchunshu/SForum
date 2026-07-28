/* Host-owned Web Push worker. It intentionally imports no extension code. */
self.addEventListener('push', (event) => {
  let payload = {}
  try {
    payload = event.data ? event.data.json() : {}
  } catch (_) {
    payload = {}
  }

  const title = typeof payload.title === 'string' ? payload.title.slice(0, 160) : 'SForum'
  const body = typeof payload.body === 'string' ? payload.body.slice(0, 3072) : ''
  const url = safeSameOriginPath(payload.url)
  event.waitUntil(self.registration.showNotification(title, {
    body,
    data: { url },
    tag: typeof payload.tag === 'string' ? payload.tag.slice(0, 160) : undefined,
  }))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = safeSameOriginPath(event.notification.data && event.notification.data.url)
  event.waitUntil(clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windows) => {
    for (const client of windows) {
      if (client.url === new URL(url, self.location.origin).href && 'focus' in client) return client.focus()
    }
    return clients.openWindow(url)
  }))
})

function safeSameOriginPath(candidate) {
  if (typeof candidate !== 'string') return '/notifications'
  try {
    const url = new URL(candidate, self.location.origin)
    if (url.origin !== self.location.origin || !url.pathname.startsWith('/') || url.pathname.startsWith('/api/')) return '/notifications'
    return url.pathname + url.search + url.hash
  } catch (_) {
    return '/notifications'
  }
}
