import { inject, type Ref } from 'vue'

import { ADMIN_HOST_INJECTION_KEY } from './internal'

export type SForumAdminToastInput = {
  title: string
  description?: string
  kind?: 'success' | 'error'
}

export type SForumAdminHost = {
  extensionId: string
  locale: Readonly<Ref<string>>
  t: (key: string, params?: Record<string, unknown>) => string
  navigate: (adminPath: string) => Promise<void>
  toast: (input: SForumAdminToastInput) => void
  extensionRequest: <T>(path: string, options?: Record<string, unknown>) => Promise<T>
}

export function useSForumAdminHost(): SForumAdminHost {
  const host = inject(ADMIN_HOST_INJECTION_KEY)
  if (!host) {
    throw new Error('useSForumAdminHost() must be called inside an SForum admin extension contribution')
  }
  return host
}
