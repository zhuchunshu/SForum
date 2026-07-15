export function markRelativeModule(element, moduleURL) {
  element.dataset.moduleMarker = 'relative-chunk-ready'
  element.dataset.moduleUrl = moduleURL
}

export function createFixtureBadge(document, label) {
  const badge = document.createElement('span')
  badge.className = 'public-l2-e2e-card__badge'
  badge.dataset.publicL2RelativeImport = 'ready'
  badge.textContent = label
  return badge
}
