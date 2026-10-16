// CI fixture：view=component 公开页的 L2 组件（prebuilt module，无构建步骤）。
// 契约要求**具名导出**（`apiVersion` + `mount`）：公开前端加载器校验的是模块命名空间本身，
// 用 `export default` 会让 apiVersion 落在 default 上而被判为 contract mismatch。
export const apiVersion = 1

export function mount(target, bridge) {
  const doc = target.ownerDocument
  const card = doc.createElement('section')
  card.setAttribute('data-testid', 'page-component-reference-body')
  card.setAttribute('data-component-id', bridge?.componentId || '')

  const title = doc.createElement('h2')
  title.setAttribute('data-testid', 'page-component-reference-title')
  title.textContent = 'page component reference body'
  card.appendChild(title)

  const note = doc.createElement('p')
  note.setAttribute('data-testid', 'page-component-reference-note')
  note.textContent = `extension=${bridge?.extensionId || ''} api=${bridge?.apiVersion ?? ''}`
  card.appendChild(note)

  target.appendChild(card)
  return () => {
    card.remove()
  }
}
