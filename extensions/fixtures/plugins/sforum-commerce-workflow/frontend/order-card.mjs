// Prebuilt L2 order card component for the commerce-workflow reference plugin.
// 公开 L2 模块契约要求**具名导出** `apiVersion` + `mount(target, bridge)`；旧版本这里是
// 一个 Vue 风格的 `export default { name, props }`，既没有 mount，apiVersion 也落在
// default 上，任何公开前端加载都会判为 contract mismatch。
export const apiVersion = 1

export function mount(target, bridge) {
  const doc = target.ownerDocument
  const card = doc.createElement('section')
  card.setAttribute('data-testid', 'commerce-order-card')

  const title = doc.createElement('h3')
  title.textContent = 'order card'
  card.appendChild(title)

  const orderId = doc.createElement('p')
  orderId.setAttribute('data-testid', 'commerce-order-card-order-id')
  orderId.textContent = `order=${bridge?.props?.orderId || ''}`
  card.appendChild(orderId)

  const status = doc.createElement('p')
  status.setAttribute('data-testid', 'commerce-order-card-status')
  status.textContent = `status=${bridge?.props?.status || ''}`
  card.appendChild(status)

  target.appendChild(card)
  return () => {
    card.remove()
  }
}
