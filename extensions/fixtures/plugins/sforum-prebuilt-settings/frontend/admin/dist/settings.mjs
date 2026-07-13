export const apiVersion = 1

export function mount(target, bridge) {
  const root = document.createElement('section')
  root.className = 'sforum-prebuilt-fixture'

  const heading = document.createElement('h3')
  heading.textContent = bridge.locale.startsWith('zh') ? '预构建设置组件' : 'Prebuilt settings component'

  const input = document.createElement('input')
  input.type = 'text'
  input.value = bridge.settings.values().message || ''
  input.setAttribute('aria-label', 'message')

  const save = document.createElement('button')
  save.type = 'button'
  save.textContent = bridge.locale.startsWith('zh') ? '保存' : 'Save'

  const onInput = () => bridge.settings.updateValue('message', input.value)
  const onSave = async () => {
    await bridge.settings.save()
    bridge.toast({ title: bridge.locale.startsWith('zh') ? '组件已保存设置' : 'Component saved settings' })
  }
  input.addEventListener('input', onInput)
  save.addEventListener('click', onSave)
  root.append(heading, input, save)
  target.append(root)

  return () => {
    input.removeEventListener('input', onInput)
    save.removeEventListener('click', onSave)
    root.remove()
  }
}
