import type {
  AdminComponentLoader,
  AdminComponentMetadata,
  AdminComponentRegistry,
  AdminExtensionLocaleMessages
} from './types'

export type AdminExtensionSlotDefinition = {
  point: string
  owner: string
  multiple: boolean
}

// 插槽位置、上下文和布局始终由核心模块拥有，插件只提供已授权的组件实现。
export const ADMIN_EXTENSION_SLOT_CATALOG = Object.freeze({
  'admin.jobs.table.columns': { point: 'admin.jobs.table.columns', owner: 'jobs', multiple: true },
  'admin.jobs.row.actions': { point: 'admin.jobs.row.actions', owner: 'jobs', multiple: true },
  'admin.jobs.detail.sections': { point: 'admin.jobs.detail.sections', owner: 'jobs', multiple: true },
  // 扩展设置：page 整页替换；header/footer 叠加在宿主通用表单上。
  'admin.extension.settings.page': { point: 'admin.extension.settings.page', owner: 'extensions', multiple: false },
  'admin.extension.settings.header': { point: 'admin.extension.settings.header', owner: 'extensions', multiple: true },
  'admin.extension.settings.footer': { point: 'admin.extension.settings.footer', owner: 'extensions', multiple: true }
}) satisfies Readonly<Record<string, AdminExtensionSlotDefinition>>

export function sortAdminComponentMetadata(items: readonly AdminComponentMetadata[]) {
  return [...items].sort((left, right) => {
    return left.order - right.order
      || left.extensionId.localeCompare(right.extensionId)
      || left.contributionId.localeCompare(right.contributionId)
  })
}

export function loaderKey(extensionId: string, contributionId: string) {
  return `${extensionId}:${contributionId}`
}

export function lookupAdminComponentLoader(
  registry: AdminComponentRegistry,
  extensionId: string,
  contributionId: string
): AdminComponentLoader | undefined {
  return registry[loaderKey(extensionId, contributionId)]
}

export function mapAdminExtensionLocale(locale: string) {
  return locale === 'en' ? 'en-US' : locale
}

export function translateAdminExtensionMessage(
  messages: AdminExtensionLocaleMessages,
  extensionId: string,
  locale: string,
  key: string,
  params: Record<string, unknown> = {}
) {
  const localeMessages = messages[extensionId]?.[mapAdminExtensionLocale(locale)]
  // 设置项 key 常含点号（如 home.notice.zh-CN），locale JSON 会把它写成字面量对象键，
  // 不能简单按 '.' 逐段下钻；优先最长前缀匹配，仍兼容嵌套对象写法。
  const value = resolveAdminExtensionMessagePath(localeMessages, key)

  if (typeof value !== 'string') {
    return key
  }

  return value.replace(/\{([^{}]+)\}/g, (match, name: string) => {
    return Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : match
  })
}

/** 解析扩展 locale 路径：支持嵌套与「键名本身含点」两种 JSON 形态。 */
export function resolveAdminExtensionMessagePath(root: unknown, key: string): unknown {
  if (!key) {
    return undefined
  }
  return walkMessagePath(root, key.split('.'))
}

function walkMessagePath(node: unknown, parts: string[]): unknown {
  if (parts.length === 0) {
    return node
  }
  if (!node || typeof node !== 'object' || Array.isArray(node)) {
    return undefined
  }
  const record = node as Record<string, unknown>
  // 从最长前缀试到最短，避免 home.notice.zh-CN 被拆成 home / notice / zh-CN。
  for (let take = parts.length; take >= 1; take -= 1) {
    const candidate = parts.slice(0, take).join('.')
    if (!Object.prototype.hasOwnProperty.call(record, candidate)) {
      continue
    }
    const found = walkMessagePath(record[candidate], parts.slice(take))
    if (found !== undefined) {
      return found
    }
  }
  return undefined
}
