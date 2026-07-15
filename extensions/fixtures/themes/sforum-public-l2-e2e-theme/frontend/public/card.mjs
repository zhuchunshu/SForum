import { createFixtureBadge, markRelativeModule } from './chunk.mjs'

export const apiVersion = 1

export function mount(target, bridge) {
  const document = target.ownerDocument
  const card = document.createElement('section')
  card.className = 'public-l2-e2e-card'
  card.dataset.publicL2Mounted = 'ready'
  markRelativeModule(card, import.meta.url)

  const title = document.createElement('h2')
  title.textContent = 'Trusted public component mounted'
  const identity = document.createElement('p')
  identity.dataset.publicL2Identity = 'ready'
  identity.textContent = `${bridge.extensionId}@${bridge.extensionVersion}`
  const status = document.createElement('output')
  status.dataset.publicL2Count = 'ready'
  status.textContent = '0'
  const button = document.createElement('button')
  button.type = 'button'
  button.dataset.publicL2Action = 'increment'
  button.textContent = 'Increment'
  let clicks = 0
  button.addEventListener('click', () => {
    clicks++
    status.textContent = String(clicks)
  })

  card.append(
    createFixtureBadge(document, 'Relative ESM import ready'),
    title,
    identity,
    status,
    button
  )
  target.dataset.publicL2PackageDigest = bridge.packageDigest
  target.replaceChildren(card)

  return () => {
    target.removeAttribute('data-public-l2-package-digest')
    target.replaceChildren()
  }
}
