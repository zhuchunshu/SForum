import type { AdminJob, AdminJobsOverview } from '~/utils/adminJobs'

export function useAdminJobs() {
  const { request } = useApiClient()
  const toast = useToast()
  const { t } = useI18n()
  const filters = reactive({ queue: '', state: '', kind: '' })
  const selected = ref<AdminJob | null>(null)
  const busy = ref('')
  const overview = useAsyncData<AdminJobsOverview>('admin-jobs-overview', () => request('/admin/jobs/overview'), { default: () => ({ counts: {}, queues: [] }) })
  const jobs = useAsyncData<AdminJob[]>('admin-jobs-list', () => request(`/admin/jobs?limit=100&queue=${encodeURIComponent(filters.queue)}&state=${encodeURIComponent(filters.state)}&kind=${encodeURIComponent(filters.kind)}`), { default: () => [] })
  async function refresh() { await Promise.all([overview.refresh(), jobs.refresh()]) }
  async function detail(id: number) { selected.value = await request(`/admin/jobs/${id}`) }
  async function jobAction(id: number, action: 'retry' | 'cancel') {
    busy.value = `${id}:${action}`
    try {
      await request(`/admin/jobs/${id}/${action}`, { method: 'POST', body: {} })
      toast.add({ color: 'success', icon: action === 'retry' ? 'i-lucide-refresh-cw' : 'i-lucide-ban', title: t(`admin.jobs.${action}Success`), duration: 10000 })
      await refresh()
    } catch (error) {
      toast.add({ color: 'error', icon: 'i-lucide-triangle-alert', title: `${error}`, duration: 0 })
    } finally { busy.value = '' }
  }
  async function queueAction(name: string, paused: boolean) {
    busy.value = `queue:${name}`
    try {
      await request(`/admin/jobs/queues/${encodeURIComponent(name)}/${paused ? 'pause' : 'resume'}`, { method: 'POST', body: {} })
      toast.add({ color: 'success', icon: paused ? 'i-lucide-pause' : 'i-lucide-play', title: t(paused ? 'admin.jobs.pauseSuccess' : 'admin.jobs.resumeSuccess'), duration: 10000 })
      await refresh()
    } finally { busy.value = '' }
  }
  watch(filters, () => { void jobs.refresh() })
  return { overview, jobs, filters, selected, busy, refresh, detail, jobAction, queueAction }
}
