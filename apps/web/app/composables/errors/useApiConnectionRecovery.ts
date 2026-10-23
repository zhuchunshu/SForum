import { apiConnectionErrorMessage, apiConnectionErrorReason, apiConnectionErrorStatusCode } from './apiConnectionErrorClassify'

/**
 * API 连接失败的「确认缓冲 + 自动恢复」引擎（模块级单例，仅客户端）。
 *
 * 为什么存在：单次抖动（慢请求、单次 502/503、ERR_NETWORK_CHANGED）不应立即
 * 弹全局模态。失败先进 pending 并探测 /health，只有探测持续失败跨过宽限期才
 * 真正打开模态；模态打开后继续低频探测，恢复即自动关闭，无需用户手动重试。
 *
 * 约束：探测路径不调用任何 Nuxt composable（useRuntimeConfig/useState 在事件
 * 回调等无上下文环境不可用），探测地址由调用方显式传入。定时器状态存于模块级
 * 而非组件级，与 useState 的应用级单例语义对齐；HMR 时通过
 * import.meta.hot.dispose 清理，防泄漏。
 */

export type ApiConnectionRecoveryState = {
  open: boolean
  message: string
  reason: string
  statusCode: number | null
  path: string
  failedAt: number | null
  failureCount: number
  recoveredAt: number | null
}

export type ApiConnectionRecoveryPatch = Partial<ApiConnectionRecoveryState>

type PendingFailure = {
  message: string
  reason: string
  statusCode: number | null
  path: string
  failedAt: number
}

type RecoveryPhase = 'idle' | 'pending' | 'monitoring'

// 退避节奏：1s → 2s → 4s，4s 封顶持续探测。按该节奏探测失败点位于
// 1s/3s/7s/11s/15s，首个跨过宽限的失败点即模态打开时间（探测自身耗时另计，
// 慢响应部署下自然更晚）。面向慢站点放宽：探测给 4s——慢而活的 /health
// 在 4s 内返回即视为存活，不因响应慢被误判为断线。
const PROBE_FIRST_DELAY_MS = 1000
const PROBE_MAX_DELAY_MS = 4000
const PROBE_TIMEOUT_MS = 4000
const MONITOR_INTERVAL_MS = 5000
const GRACE_MS = 15000

let phase: RecoveryPhase = 'idle'
let probeBaseUrl = ''
let pendingFailure: PendingFailure | null = null
let nextDelayMs = PROBE_FIRST_DELAY_MS
let probeTimer: ReturnType<typeof setTimeout> | null = null
let probeInFlight = false
let emitPatch: ((patch: ApiConnectionRecoveryPatch) => void) | null = null
let readSnapshot: (() => Pick<ApiConnectionRecoveryState, 'open' | 'failureCount'> | null) | null = null

export function recordApiConnectionFailure(error: unknown, path: string, baseUrl: string) {
  if (import.meta.server) {
    return
  }

  const snapshot = readSnapshot?.()
  const failureCount = (snapshot?.failureCount ?? 0) + 1

  // pending/monitoring 期间沿用「保留第一个错误」语义：只累加计数，
  // 避免弹窗内容反复跳动；恢复判定统一交给探测循环。
  if (phase !== 'idle') {
    emitPatch?.({ failureCount })
    return
  }

  // 引擎未跟踪但状态仍是打开态（HMR 重载后 useState 存留）：恢复监控而不是重开 pending。
  if (snapshot?.open) {
    resumeMonitoring(baseUrl)
    emitPatch?.({ failureCount })
    return
  }

  probeBaseUrl = baseUrl
  pendingFailure = {
    message: apiConnectionErrorMessage(error),
    reason: apiConnectionErrorReason(error),
    statusCode: apiConnectionErrorStatusCode(error),
    path,
    failedAt: Date.now()
  }
  nextDelayMs = PROBE_FIRST_DELAY_MS
  phase = 'pending'
  emitPatch?.({
    open: false,
    message: pendingFailure.message,
    reason: pendingFailure.reason,
    statusCode: pendingFailure.statusCode,
    path: pendingFailure.path,
    failedAt: pendingFailure.failedAt,
    failureCount,
    recoveredAt: null
  })
  scheduleProbe(PROBE_FIRST_DELAY_MS)
}

function clearProbeTimer() {
  if (probeTimer) {
    clearTimeout(probeTimer)
    probeTimer = null
  }
}

function resetToIdle() {
  clearProbeTimer()
  pendingFailure = null
  probeBaseUrl = ''
  nextDelayMs = PROBE_FIRST_DELAY_MS
  phase = 'idle'
}

function resumeMonitoring(baseUrl = probeBaseUrl) {
  clearProbeTimer()
  probeBaseUrl = baseUrl
  phase = 'monitoring'
  scheduleProbe(MONITOR_INTERVAL_MS)
}

function phaseNow(): RecoveryPhase {
  // 探测 await 期间 phase 可能被 close/reset 改写；TS 的模块变量窄化不跨
  // await 感知这一点，经函数读取强制重新取值。
  return phase
}

async function runProbe() {
  if (probeInFlight || phaseNow() === 'idle') {
    return
  }
  probeInFlight = true

  const target = pendingFailure
  const ok = await probeHealth()
  probeInFlight = false

  // 探测期间可能被 close/reset 改变状态，过期结果直接丢弃。
  if (phaseNow() === 'idle') {
    return
  }

  if (ok) {
    if (phaseNow() === 'monitoring') {
      const recoveredAt = Date.now()
      resetToIdle()
      emitPatch?.({ open: false, recoveredAt })
    } else {
      // pending 期一次探测成功：抖动已恢复，静默丢弃，不弹窗。
      resetToIdle()
    }
    return
  }

  if (phaseNow() === 'monitoring') {
    // 仍不可用：保持模态打开，等待下一次监控探测。
    scheduleProbe(MONITOR_INTERVAL_MS)
    return
  }

  if (!target) {
    resetToIdle()
    return
  }

  // 宽限期判定：pending 持续超过 GRACE_MS 且探测仍失败，才真正弹窗。
  if (Date.now() - target.failedAt >= GRACE_MS) {
    openModal(target)
    return
  }

  nextDelayMs = Math.min(nextDelayMs * 2, PROBE_MAX_DELAY_MS)
  scheduleProbe(nextDelayMs)
}

function openModal(target: PendingFailure) {
  phase = 'monitoring'
  nextDelayMs = PROBE_FIRST_DELAY_MS
  emitPatch?.({
    open: true,
    message: target.message,
    reason: target.reason,
    statusCode: target.statusCode,
    path: target.path,
    failedAt: target.failedAt
  })
  scheduleProbe(MONITOR_INTERVAL_MS)
}

async function probeHealth() {
  try {
    const base = (probeBaseUrl || resolveFallbackBaseUrl()).replace(/\/+$/, '')
    await $fetch(`${base}/health`, {
      credentials: 'include',
      timeout: PROBE_TIMEOUT_MS
    })
    return true
  } catch {
    return false
  }
}

function resolveFallbackBaseUrl() {
  // 探测期刻意不持有 Nuxt 上下文；仅当调用方未传入探测地址时尝试读取
  // runtime config（无上下文则退化为相对 /health，指向同源 API）。
  try {
    return (useRuntimeConfig().public.apiBaseUrl as string) || ''
  } catch {
    return ''
  }
}

function scheduleProbe(delayMs: number) {
  clearProbeTimer()
  probeTimer = setTimeout(() => {
    probeTimer = null
    void runProbe()
  }, delayMs)
}

/**
 * 门面（useApiConnectionError）绑定：注入状态回写与读取回调。
 * useState 在同一应用内对同一 key 返回同一 ref，重复绑定幂等。
 */
export function bindApiConnectionRecovery(options: {
  emit: (patch: ApiConnectionRecoveryPatch) => void
  snapshot: () => Pick<ApiConnectionRecoveryState, 'open' | 'failureCount'> | null
}) {
  emitPatch = options.emit
  readSnapshot = options.snapshot

  // 绑定时若状态已是打开态（HMR 后 useState 仍在而引擎模块已重载），
  // 恢复监控循环，避免模态停留在无人看管的状态。
  const snapshot = options.snapshot()
  if (snapshot?.open && phase === 'idle') {
    resumeMonitoring()
  }
}

export function stopApiConnectionRecovery() {
  resetToIdle()
}

if (import.meta.hot) {
  import.meta.hot.dispose(() => {
    resetToIdle()
    probeInFlight = false
    emitPatch = null
    readSnapshot = null
  })
}

/** 测试隔离：bun test 模块状态跨用例存活，用例间调用以复位引擎。 */
export function __resetApiConnectionRecoveryForTests() {
  resetToIdle()
  probeInFlight = false
  emitPatch = null
  readSnapshot = null
}
