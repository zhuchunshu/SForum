// Minimal prebuilt L2 widget for the region-demo fixture (forum.page.regions l2Widget).
export default {
  apiVersion: 1,
  mount(target) {
    const card = target.ownerDocument.createElement('p')
    card.setAttribute('data-testid', 'region-demo-widget')
    card.textContent = 'region demo widget'
    target.appendChild(card)
    return () => {
      card.remove()
    }
  }
}
