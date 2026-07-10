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
  'admin.jobs.detail.sections': { point: 'admin.jobs.detail.sections', owner: 'jobs', multiple: true }
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
  const value = key.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') {
      return undefined
    }
    return (current as Record<string, unknown>)[segment]
  }, localeMessages)

  if (typeof value !== 'string') {
    return key
  }

  return value.replace(/\{([^{}]+)\}/g, (match, name: string) => {
    return Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : match
  })
}
