import { computed } from 'vue'
import type { SForumAdminHost } from '@sforum/admin-sdk'
import {
  contributions,
  locales,
  releaseId
} from '#sforum/admin-extension-metadata'

import {
  lookupAdminComponentLoader,
  mapAdminExtensionLocale,
  sortAdminComponentMetadata,
  translateAdminExtensionMessage
} from '~/runtime/admin-extensions/catalog'
import {
  assertAdminExtensionRelativePath,
  extensionRequestPath,
  type AdminComponentRegistry
} from '~/runtime/admin-extensions/types'

let registryPromise: Promise<AdminComponentRegistry> | undefined

async function loadClientRegistry() {
  if (!import.meta.client) {
    return {} as AdminComponentRegistry
  }
  registryPromise ||= import('#sforum/admin-extension-registry').then(module => module.registry)
  return registryPromise
}

export function useAdminExtensionRegistry() {
  const { locale } = useI18n()
  const adminRoutes = useAdminRoutes()
  const toast = useToast()
  const { request } = useApiClient()
  const orderedContributions = sortAdminComponentMetadata(contributions)
  // empty-metadata 固定 releaseId=core 且 contributions 为空；完整 dev supervisor 会注入 Web Release 的 registry。
  const adminFrontendInjected = releaseId !== 'core' || orderedContributions.length > 0

  function contributionsFor(point: string) {
    return orderedContributions.filter(contribution => contribution.point === point)
  }

  async function loaderFor(extensionId: string, contributionId: string) {
    return lookupAdminComponentLoader(await loadClientRegistry(), extensionId, contributionId)
  }

  function hostFor(extensionId: string): SForumAdminHost {
    const hostLocale = computed(() => mapAdminExtensionLocale(locale.value))
    return {
      extensionId,
      locale: hostLocale,
      t: (key, params) => translateAdminExtensionMessage(locales, extensionId, locale.value, key, params),
      navigate: async (adminPath) => {
        await navigateTo(adminRoutes.path(assertAdminExtensionRelativePath(adminPath)))
      },
      toast: (input) => {
        const error = input.kind === 'error'
        toast.add({
          title: input.title,
          description: input.description,
          color: error ? 'error' : 'success',
          icon: error ? 'i-lucide-triangle-alert' : 'i-lucide-check',
          duration: error ? Number.POSITIVE_INFINITY : 10000
        })
      },
      extensionRequest: <T>(path: string, options: Record<string, unknown> = {}) => {
        return request<T>(extensionRequestPath(extensionId, path), options as Parameters<typeof request>[1])
      }
    }
  }

  return {
    releaseId,
    adminFrontendInjected,
    contributionsFor,
    loaderFor,
    hostFor
  }
}
