import { apiErrorMessage, apiErrorReason } from '../useApiClient'

export type RoleSuggestionApprovalState = 'pending' | 'approved' | 'rejected'
type ApprovalState = RoleSuggestionApprovalState
type ApprovalFilter = ApprovalState | 'all'
type BadgeColor = 'warning' | 'success' | 'error' | 'neutral'

export type RoleSuggestion = {
  id: number
  permissionKey: string
  ownerExtensionId: string
  extensionVersionId: number
  extensionVersion: string
  packageDigest: string
  permissionContractVersion: string
  declarationDigest: string
  roleKey: string
  approvalState: ApprovalState
  applied: boolean
  revision: number
  decidedAt?: string
  appliedAt?: string
  createdAt: string
  updatedAt: string
}

type RoleSuggestionPage = {
  items: RoleSuggestion[]
  nextCursor?: string
}

type PendingDecision = {
  suggestion: RoleSuggestion
  state: 'approved' | 'rejected'
}

export function roleSuggestionHasConsistentEvidence(
  suggestion: Pick<RoleSuggestion, 'approvalState' | 'applied'>
) {
  if (suggestion.applied !== true && suggestion.applied !== false) return false
  switch (suggestion.approvalState) {
    case 'approved':
      return true
    case 'pending':
    case 'rejected':
      return !suggestion.applied
    default:
      return false
  }
}

export function roleSuggestionDecisionHasEvidence(
  requestedState: 'approved' | 'rejected',
  result: Pick<RoleSuggestion, 'approvalState' | 'applied'>
) {
  if (!roleSuggestionHasConsistentEvidence(result)) return false
  if (requestedState === 'approved') {
    return result.approvalState === 'approved' && result.applied
  }
  return result.approvalState === 'rejected' && !result.applied
}

function isAdoptableDecisionConflict(reason: string) {
  return reason === 'identity.role_suggestion.stale' ||
    reason === 'identity.role_suggestion.revision_conflict' ||
    reason === 'identity.role_suggestion.target_unavailable'
}

export function useRoleSuggestions(onPermissionApplied?: (result: RoleSuggestion) => void | Promise<void>) {
  const { t, locale } = useI18n()
  const toast = useToast()
  const { request } = useApiClient()
  const selectedFilter = ref<ApprovalFilter>('pending')
  const suggestions = ref<RoleSuggestion[]>([])
  const nextCursor = ref('')
  const loading = ref(false)
  const loadingMore = ref(false)
  const loadError = ref('')
  const decisionOpen = ref(false)
  const pendingDecision = ref<PendingDecision | null>(null)
  const deciding = ref(false)
  const decisionError = ref('')
  let requestGeneration = 0

  const filterItems = computed(() => [
    { label: t('admin.roles.suggestions.filters.pending'), value: 'pending' },
    { label: t('admin.roles.suggestions.filters.approved'), value: 'approved' },
    { label: t('admin.roles.suggestions.filters.rejected'), value: 'rejected' },
    { label: t('admin.roles.suggestions.filters.all'), value: 'all' }
  ])

  const confirmationTitle = computed(() => {
    if (pendingDecision.value?.state === 'rejected') {
      return t('admin.roles.suggestions.rejectTitle')
    }
    if (pendingDecision.value?.suggestion.approvalState === 'approved') {
      return t('admin.roles.suggestions.applyTitle')
    }
    return t('admin.roles.suggestions.approveTitle')
  })

  const confirmationDescription = computed(() => {
    const item = pendingDecision.value?.suggestion
    if (!item) return ''
    const params = {
      permission: item.permissionKey,
      role: item.roleKey,
      extension: item.ownerExtensionId
    }
    if (pendingDecision.value?.state === 'rejected') {
      return t('admin.roles.suggestions.rejectDescription', params)
    }
    return t('admin.roles.suggestions.approveDescription', params)
  })

  const confirmationAction = computed(() => {
    if (pendingDecision.value?.state === 'rejected') {
      return t('admin.roles.suggestions.reject')
    }
    if (pendingDecision.value?.suggestion.approvalState === 'approved') {
      return t('admin.roles.suggestions.apply')
    }
    return t('admin.roles.suggestions.confirmApprove')
  })

  onMounted(() => {
    void loadSuggestions(true)
  })

  watch(selectedFilter, () => {
    void loadSuggestions(true)
  })

  async function loadSuggestions(reset: boolean) {
    if (reset) {
      requestGeneration++
      suggestions.value = []
      nextCursor.value = ''
      loading.value = true
      // reset 会使旧的 load-more 请求失效，不能再等旧代际的 finally 清理状态。
      loadingMore.value = false
    } else {
      if (!nextCursor.value || loadingMore.value) return
      loadingMore.value = true
    }
    const generation = requestGeneration
    const filter = selectedFilter.value
    loadError.value = ''

    try {
      const query = new URLSearchParams({ limit: '25' })
      if (filter !== 'all') query.set('approvalState', filter)
      if (!reset && nextCursor.value) query.set('cursor', nextCursor.value)
      const page = await request<RoleSuggestionPage>(`/roles/suggestions?${query.toString()}`)
      if (generation !== requestGeneration || filter !== selectedFilter.value) return
      if (!page.items.every(roleSuggestionHasConsistentEvidence)) {
        nextCursor.value = ''
        loadError.value = t('admin.roles.suggestions.evidenceMissing')
        return
      }
      suggestions.value = reset ? page.items : [...suggestions.value, ...page.items]
      nextCursor.value = page.nextCursor || ''
    } catch (error) {
      if (generation !== requestGeneration) return
      loadError.value = roleSuggestionError(error, 'admin.roles.suggestions.loadFailed')
    } finally {
      if (generation === requestGeneration) {
        loading.value = false
        loadingMore.value = false
      }
    }
  }

  function openDecision(suggestion: RoleSuggestion, state: 'approved' | 'rejected') {
    if (deciding.value) return
    pendingDecision.value = { suggestion, state }
    decisionError.value = ''
    decisionOpen.value = true
  }

  function closeDecision() {
    if (deciding.value) return
    decisionOpen.value = false
    decisionError.value = ''
    pendingDecision.value = null
  }

  async function submitDecision() {
    const decision = pendingDecision.value
    if (!decision || deciding.value) return
    deciding.value = true
    decisionError.value = ''
    try {
      const result = await request<RoleSuggestion>(
        `/roles/suggestions/${encodeURIComponent(String(decision.suggestion.id))}/decision`,
        {
          method: 'POST',
          body: {
            expectedRevision: decision.suggestion.revision,
            approvalState: decision.state
          }
        }
      )
      if (!roleSuggestionDecisionHasEvidence(decision.state, result)) {
        decisionError.value = t('admin.roles.suggestions.evidenceMissing')
        return
      }
      decisionOpen.value = false
      pendingDecision.value = null
      const approved = result.approvalState === 'approved' && result.applied
      toast.add({
        title: approved
          ? t('admin.roles.suggestions.approvedToast')
          : t('admin.roles.suggestions.rejectedToast'),
        color: 'success',
        icon: approved ? 'i-lucide-shield-check' : 'i-lucide-circle-x',
        duration: 10000
      })
      if (approved) await onPermissionApplied?.(result)
      await loadSuggestions(true)
    } catch (error) {
      const reason = apiErrorReason(error)
      const message = roleSuggestionError(error, 'admin.roles.suggestions.decisionFailed')
      if (isAdoptableDecisionConflict(reason)) {
        // 冲突意味着旧 revision 已失去提交资格；关闭旧决策并采纳 Host 最新列表。
        decisionOpen.value = false
        pendingDecision.value = null
        decisionError.value = ''
        await loadSuggestions(true)
        if (!loadError.value) loadError.value = message
        return
      }
      decisionError.value = message
    } finally {
      deciding.value = false
    }
  }

  function roleSuggestionError(error: unknown, fallbackKey: string) {
    const reason = apiErrorReason(error)
    switch (reason) {
      case 'identity.role_suggestion.stale':
        return t('admin.roles.suggestions.stale')
      case 'identity.role_suggestion.revision_conflict':
        return t('admin.roles.suggestions.revisionConflict')
      case 'identity.role_suggestion.target_unavailable':
        return t('admin.roles.suggestions.targetUnavailable')
      default:
        // 未映射的协议 reason 不能原样泄漏到操作员界面。
        if (reason.startsWith('identity.role_suggestion.')) return t(fallbackKey)
        const message = apiErrorMessage(error)
        return message.startsWith('identity.role_suggestion.') ? t(fallbackKey) : message || t(fallbackKey)
    }
  }

  function statusColor(suggestion: RoleSuggestion): BadgeColor {
    if (suggestion.approvalState === 'rejected') return 'error'
    if (suggestion.approvalState === 'approved' && suggestion.applied) return 'success'
    if (suggestion.approvalState === 'approved') return 'neutral'
    return 'warning'
  }

  function statusLabel(suggestion: RoleSuggestion) {
    if (suggestion.approvalState === 'approved' && suggestion.applied) {
      return t('admin.roles.suggestions.status.applied')
    }
    if (suggestion.approvalState === 'approved') {
      return t('admin.roles.suggestions.status.reviewOnly')
    }
    return t(`admin.roles.suggestions.status.${suggestion.approvalState}`)
  }

  function shortDigest(value: string) {
    return value.length > 16 ? `${value.slice(0, 12)}...${value.slice(-4)}` : value
  }

  function formatDate(value: string) {
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return new Intl.DateTimeFormat(locale.value, {
      dateStyle: 'medium',
      timeStyle: 'short'
    }).format(date)
  }

  return {
    selectedFilter,
    suggestions,
    nextCursor,
    loading,
    loadingMore,
    loadError,
    decisionOpen,
    pendingDecision,
    deciding,
    decisionError,
    filterItems,
    confirmationTitle,
    confirmationDescription,
    confirmationAction,
    loadSuggestions,
    openDecision,
    closeDecision,
    submitDecision,
    statusColor,
    statusLabel,
    shortDigest,
    formatDate
  }
}
