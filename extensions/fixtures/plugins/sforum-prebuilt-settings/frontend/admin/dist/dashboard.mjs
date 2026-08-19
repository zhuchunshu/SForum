export const apiVersion = 1

export async function mount(target, bridge) {
  const root = document.createElement('section')
  root.className = 'fixture-admin-dashboard'

  const summary = document.createElement('p')
  summary.textContent = `${bridge.extensionId} / ${bridge.page.path}`

  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'fixture-admin-dashboard__action'
  button.textContent = 'Run plugin action'
  button.addEventListener('click', () => {
    bridge.toast({ title: 'Plugin action completed', kind: 'success' })
  })

  root.append(summary, button)
  target.append(root)
  return () => root.remove()
}
