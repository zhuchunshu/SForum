// Minimal prebuilt L2 widget for the region-demo fixture (forum.page.regions l2Widget).
// 公开 L2 模块契约要求**具名导出**：加载器校验模块命名空间本身
// （`module.apiVersion === 1 && typeof module.mount === 'function'`，见
// apps/web/app/runtime/public-extensions/assets.ts 的 isPublicFrontendModule），
// `export default` 会让 apiVersion 落在 default 上而被判为 contract mismatch。
export const apiVersion = 1

export function mount(target) {
  const card = target.ownerDocument.createElement('p')
  card.setAttribute('data-testid', 'region-demo-widget')
  card.textContent = 'region demo widget'
  target.appendChild(card)
  return () => {
    card.remove()
  }
}
