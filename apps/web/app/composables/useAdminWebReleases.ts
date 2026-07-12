import type { AdminWebReleaseDetail, AdminWebReleasePage } from '~/utils/adminWebReleases'
import { webReleaseIsFinal } from '~/utils/adminWebReleases'

const TYPECHECK_FAIL_OPTION = 'web_release.typecheck_fail'

type AdminWebOption = {
  name: string
  value: string
  public?: boolean
}

function isEnabledOption(value: string | undefined) {
  const raw = `${value ?? ''}`.trim().toLowerCase()
  return raw === 'enabled' || raw === 'true' || raw === '1' || raw === 'yes' || raw === 'on'
}

export function useAdminWebReleases() {
  const { request } = useApiClient()
  const toast = useToast()
  const { t } = useI18n()
  const page = ref(1)
  const perPage = 20
  const selected = ref<AdminWebReleaseDetail | null>(null)
  const commandId = ref(0)
  const rebuilding = ref(false)
  const typecheckFail = ref(false)
  const typecheckSaving = ref(false)
  const typecheckLoading = ref(false)
  let timer: ReturnType<typeof setTimeout> | null = null
  let loading = false

  const state = useAsyncData<AdminWebReleasePage>('admin-web-releases', () => request(`/admin/web-releases?page=${page.value}&perPage=${perPage}`), {
    default: () => ({ items: [], total: 0, page: 1, perPage })
  })

  async function load() {
    if (loading) return
    loading = true
    try {
      await state.refresh()
    } finally {
      loading = false
      schedule()
    }
  }

  async function loadTypecheckPolicy() {
    typecheckLoading.value = true
    try {
      const items = await request<AdminWebOption[]>('/admin/web-options')
      const hit = (items || []).find(item => item.name === TYPECHECK_FAIL_OPTION)
      typecheckFail.value = isEnabledOption(hit?.value)
    } catch {
      // 无权限或失败时保持默认非阻断，不打断发布列表。
      typecheckFail.value = false
    } finally {
      typecheckLoading.value = false
    }
  }

  async function setTypecheckFail(enabled: boolean) {
    if (typecheckSaving.value) return
    typecheckSaving.value = true
    const previous = typecheckFail.value
    typecheckFail.value = enabled
    try {
      await request('/admin/web-options', {
        method: 'PUT',
        body: {
          options: [{
            name: TYPECHECK_FAIL_OPTION,
            value: enabled ? 'enabled' : 'disabled'
          }]
        }
      })
      toast.add({
        color: 'success',
        icon: 'i-lucide-shield-check',
        title: enabled
          ? t('admin.extensions.releases.typecheckFailEnabled')
          : t('admin.extensions.releases.typecheckFailDisabled'),
        duration: 10000
      })
    } catch (error) {
      typecheckFail.value = previous
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
    } finally {
      typecheckSaving.value = false
    }
  }

  async function select(id: number) {
    selected.value = await request<AdminWebReleaseDetail>(`/admin/web-releases/${id}`)
  }

  async function rebuild() {
    if (rebuilding.value) return
    rebuilding.value = true
    try {
      const operation = await request<{ webRelease?: { id?: number }, queued?: boolean }>('/admin/web-releases/rebuild', {
        method: 'POST',
        body: {}
      })
      const releaseId = operation?.webRelease?.id
      toast.add({
        color: 'success',
        icon: 'i-lucide-hammer',
        title: t('admin.extensions.releases.rebuildQueued'),
        description: releaseId
          ? t('admin.extensions.webReleaseQueuedHint', { id: releaseId })
          : t('admin.extensions.webReleaseQueuedHintNoId'),
        duration: 10000
      })
      await load()
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
    } finally {
      rebuilding.value = false
    }
  }

  async function command(id: number, action: 'retry' | 'rollback') {
    commandId.value = id
    try {
      await request(`/admin/web-releases/${id}/${action}`, { method: 'POST', body: {} })
      toast.add({ color: 'success', icon: action === 'retry' ? 'i-lucide-refresh-cw' : 'i-lucide-undo-2', title: t(`admin.extensions.releases.${action}Queued`), duration: 10000 })
      await load()
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
    } finally {
      commandId.value = 0
    }
  }

  function schedule() {
    if (timer) clearTimeout(timer)
    if (state.data.value.items.some(item => !webReleaseIsFinal(item.status))) timer = setTimeout(load, 2500)
  }

  watch(page, load)
  onMounted(() => {
    schedule()
    void loadTypecheckPolicy()
  })
  onUnmounted(() => { if (timer) clearTimeout(timer) })
  return {
    ...state,
    page,
    perPage,
    selected,
    commandId,
    rebuilding,
    typecheckFail,
    typecheckSaving,
    typecheckLoading,
    load,
    select,
    rebuild,
    command,
    setTypecheckFail,
    loadTypecheckPolicy
  }
}
