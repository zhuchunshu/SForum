// 扩展事件/投递的合并与分页面：纯函数，输入数组、输出排序或分页结果，
// 不依赖任何运行时状态。分页尺寸常量随这些函数一起搬来（它只服务于它们）。
//
// 这里只 `import type`：adminExtensions.ts 会 re-export 本文件，值级依赖会形成
// 运行期循环，而类型在编译后会被抹掉，方向保持单向。
import type {
  AdminEffectiveContribution,
  AdminExtensionEvent,
  AdminExtensionEventDelivery,
  AdminExtensionEventDefinition
} from './adminExtensions'

// EXTENSION_EVENT_PAGE_SIZE 是事件/投递/贡献列表的默认页大小。
// 它被 `extensionEventPage` 等函数的默认参数引用，因此与它们同处一个模块；
// adminExtensions.ts 仍然 re-export，既有 import 路径不变。
export const EXTENSION_EVENT_PAGE_SIZE = 8

export function mergeExtensionEvents(eventsByExtension: Record<string, AdminExtensionEvent[]>) {
  return Object.values(eventsByExtension)
    .flat()
    .slice()
    .sort((left, right) => {
      const timeDiff = Date.parse(right.createdAt) - Date.parse(left.createdAt)
      return timeDiff || right.id - left.id
    })
}

export function mergeExtensionDeliveries(items: AdminExtensionEventDelivery[]) {
  return items.slice().sort((left, right) => {
    const timeDiff = Date.parse(right.createdAt) - Date.parse(left.createdAt)
    return timeDiff || right.id - left.id
  })
}

export function extensionEventPage(items: AdminExtensionEvent[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionDeliveryPage(items: AdminExtensionEventDelivery[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionDefinitionPage(items: AdminExtensionEventDefinition[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionContributionPage(items: AdminEffectiveContribution[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

/** 通用列表分页（页面注册表等管理列表复用）。 */
export function paginateItems<T>(items: T[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  return extensionItemsPage(items, page, pageSize)
}

export function extensionContributionLabel(item: AdminEffectiveContribution, locale = 'zh-CN') {
  const labels = item.label || {}
  return labels[locale] || labels['zh-CN'] || labels['en-US'] || Object.values(labels).find(Boolean) || item.id
}

export function extensionContributionPayloadSummary(item: AdminEffectiveContribution) {
  const payload = item.payload || {}
  const parts = [payload.type, payload.method, payload.path]
    .map(value => `${value || ''}`.trim())
    .filter(Boolean)
  return parts.length ? parts.join(' ') : '-'
}

function extensionItemsPage<T>(items: T[], page: number, pageSize = EXTENSION_EVENT_PAGE_SIZE) {
  const safePageSize = Math.max(1, Math.floor(pageSize))
  const totalPages = Math.max(1, Math.ceil(items.length / safePageSize))
  const currentPage = Math.min(totalPages, Math.max(1, Math.floor(page) || 1))
  const start = items.length === 0 ? 0 : (currentPage - 1) * safePageSize + 1
  const end = items.length === 0 ? 0 : Math.min(items.length, currentPage * safePageSize)

  return {
    items: items.slice(start === 0 ? 0 : start - 1, end),
    page: currentPage,
    pageSize: safePageSize,
    totalPages,
    start,
    end,
    total: items.length
  }
}
