export type AdminSystemUpdateState = 'current' | 'update_available' | 'development' | 'unavailable'
export type AdminSystemUpdateSource = 'official' | 'mirror'
export type AdminSystemUpdateErrorCode = 'invalid_source' | 'request_failed' | 'invalid_response' | 'no_release'

export type AdminSystemUpdateStatus = {
  state: AdminSystemUpdateState
  updateAvailable: boolean
  currentVersion: string
  currentCommit?: string
  latestVersion?: string
  latestTag?: string
  releaseName?: string
  releaseUrl?: string
  publishedAt?: string
  checkedAt: string
  source: AdminSystemUpdateSource
  mirrorConfigured: boolean
  errorCode?: AdminSystemUpdateErrorCode
}

type RefreshSystemUpdatesOptions = {
  force?: boolean
  serverInternal?: boolean
}

export function useAdminSystemUpdates() {
  const { request } = useApiClient()
  const status = useState<AdminSystemUpdateStatus | null>('admin-system-update-status', () => null)
  const pending = useState('admin-system-update-pending', () => false)
  const requestFailed = useState('admin-system-update-request-failed', () => false)

  async function refresh(options: RefreshSystemUpdatesOptions = {}) {
    pending.value = true
    requestFailed.value = false
    try {
      status.value = await request<AdminSystemUpdateStatus>(
        options.force ? '/admin/system-updates/check' : '/admin/system-updates',
        {
          method: options.force ? 'POST' : undefined,
          serverInternal: options.serverInternal
        }
      )
      return status.value
    } catch (error) {
      requestFailed.value = true
      throw error
    } finally {
      pending.value = false
    }
  }

  return {
    status,
    pending,
    requestFailed,
    refresh
  }
}
