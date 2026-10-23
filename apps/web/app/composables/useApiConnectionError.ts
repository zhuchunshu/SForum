import {
  apiConnectionErrorStatusCode,
  isApiConnectionError
} from './errors/apiConnectionErrorClassify'
import {
  bindApiConnectionRecovery,
  recordApiConnectionFailure,
  stopApiConnectionRecovery,
  type ApiConnectionRecoveryPatch,
  type ApiConnectionRecoveryState
} from './errors/useApiConnectionRecovery'

export type ApiConnectionErrorState = ApiConnectionRecoveryState

export { apiConnectionErrorStatusCode, isApiConnectionError }

const API_CONNECTION_ERROR_STATE_KEY = 'api:connection-error'

export function useApiConnectionError() {
  const state = useState<ApiConnectionErrorState>(API_CONNECTION_ERROR_STATE_KEY, () => ({
    open: false,
    message: '',
    reason: '',
    statusCode: null,
    path: '',
    failedAt: null,
    failureCount: 0,
    recoveredAt: null
  }))

  // 引擎是模块级单例；useState 对同一 key 返回同一 ref，重复绑定幂等。
  bindApiConnectionRecovery({
    emit: (patch: ApiConnectionRecoveryPatch) => {
      Object.assign(state.value, patch)
    },
    snapshot: () => ({ open: state.value.open, failureCount: state.value.failureCount })
  })

  function show(error: unknown, path = '', apiBaseUrl = '') {
    if (!isApiConnectionError(error)) {
      return
    }

    // 不再立即弹窗：交给探测引擎进入 pending 缓冲，持续失败跨过宽限期才 open。
    recordApiConnectionFailure(error, path, apiBaseUrl)
  }

  function close() {
    stopApiConnectionRecovery()
    state.value.open = false
  }

  function reload() {
    if (import.meta.server) {
      return
    }
    window.location.reload()
  }

  return { state, show, close, reload }
}

export function notifyApiConnectionError(error: unknown, path = '', apiBaseUrl = '') {
  if (import.meta.server || !isApiConnectionError(error)) {
    return
  }

  const { show } = useApiConnectionError()
  show(error, path, apiBaseUrl)
}
