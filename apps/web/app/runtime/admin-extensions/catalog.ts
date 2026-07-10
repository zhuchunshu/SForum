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

// 生产插槽由各核心模块显式注册；任务监控插槽属于后续 jobs 模块。
export const ADMIN_EXTENSION_SLOT_CATALOG = Object.freeze({}) as Readonly<Record<string, AdminExtensionSlotDefinition>>

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
