import type { AdminWebReleaseDetail, AdminWebReleasePage } from '~/utils/adminWebReleases'
import { webReleaseIsFinal } from '~/utils/adminWebReleases'

export function useAdminWebReleases() {
  const { request } = useApiClient()
  const toast = useToast()
  const { t } = useI18n()
  const page = ref(1)
  const perPage = 20
  const selected = ref<AdminWebReleaseDetail | null>(null)
  const commandId = ref(0)
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

  async function select(id: number) {
    selected.value = await request<AdminWebReleaseDetail>(`/admin/web-releases/${id}`)
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
  onMounted(schedule)
  onUnmounted(() => { if (timer) clearTimeout(timer) })
  return { ...state, page, perPage, selected, commandId, load, select, command }
}
