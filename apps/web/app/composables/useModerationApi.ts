export type ModerationTargetType = 'topic' | 'comment'
export type ModerationReasonCode = 'spam' | 'abuse' | 'illegal' | 'off_topic' | 'other'
export type ModerationReportStatus = 'open' | 'reviewing' | 'resolved' | 'rejected'

export type ModerationReport = {
  id: number
  reporterUserId: number
  reporterName?: string
  targetType: ModerationTargetType
  targetId: number
  reasonCode: ModerationReasonCode
  body: string
  status: ModerationReportStatus
  reviewerUserId?: number | null
  reviewerName?: string
  reviewNote: string
  createdAt: string
  updatedAt: string
  resolvedAt?: string | null
}

export type ModerationReportList = {
  items: ModerationReport[]
  total: number
  page: number
  perPage: number
}

export type ModerationReportFilters = {
  status?: ModerationReportStatus
  targetType?: ModerationTargetType
  reporterId?: number
  page?: number
  perPage?: number
}

export function useModerationApi() {
  const { request } = useApiClient()

  function createReport(input: {
    targetType: ModerationTargetType
    targetId: number
    reasonCode: ModerationReasonCode
    body?: string
  }) {
    return request<ModerationReport>('/moderation/reports', {
      method: 'POST',
      body: input
    })
  }

  function listReports(filters: ModerationReportFilters = {}) {
    const query: Record<string, string> = {}
    if (filters.status) query.status = filters.status
    if (filters.targetType) query.targetType = filters.targetType
    if (filters.reporterId) query.reporterId = String(filters.reporterId)
    if (filters.page) query.page = String(filters.page)
    if (filters.perPage) query.perPage = String(filters.perPage)
    const qs = new URLSearchParams(query).toString()
    return request<ModerationReportList>(`/admin/moderation/reports${qs ? '?' + qs : ''}`)
  }

  function updateReport(reportId: number, input: { status: ModerationReportStatus; reviewNote?: string }) {
    return request<ModerationReport>(`/admin/moderation/reports/${reportId}`, {
      method: 'PATCH',
      body: input
    })
  }

  return { createReport, listReports, updateReport }
}
