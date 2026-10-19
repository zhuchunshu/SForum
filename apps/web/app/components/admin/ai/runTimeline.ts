/**
 * 观测页签 run 时间线的纯展示逻辑。
 *
 * 这里刻意不依赖 Nuxt 运行时：这些函数是「运营者能看到什么」的权威口径
 * （哪条记录能展开、哪条是缓存命中的整次 run、哪条没有出站），必须能被单元测试
 * 直接覆盖，而不是靠读源码字符串来断言。
 */

/** RunTimelineRecord 是渲染需要的最小结构化面。AdminAIExecution 结构上满足它。 */
export type RunTimelineRecord = {
  status: string
  cacheHit: boolean
  gateReason?: string
  gateScope?: string
  model?: string
  latencyMs?: number
  inputTokens?: number
  outputTokens?: number
  errorSummary?: string
  resourceId?: string
  createdAt?: string
}

export type RunTimelineEntry = {
  /** 1 起的展示序号。 */
  index: number
  record: RunTimelineRecord
}

/**
 * timelineEntries 给每条记录分配展示序号。
 *
 * 不重新排序：后端已按 `created_at ASC, id ASC` 返回。前端再排一次会引入第二个
 * 真相——同毫秒内的顺序只有 id 知道。
 */
export function timelineEntries(records: readonly RunTimelineRecord[]): RunTimelineEntry[] {
  return records.map((record, position) => ({ index: position + 1, record }))
}

/**
 * executionExpandable 决定是否给出展开入口。
 *
 * A3 之前的历史记录没有 run 归属（`resourceId` 为空串）：展开它们只会得到空列表。
 * 入口本身不该存在——这不是「点了才说没有」，而是「本来就没有」。
 */
export function executionExpandable(record: { resourceId?: string }): boolean {
  return Boolean(record.resourceId && record.resourceId.trim())
}

/**
 * cacheHitOnlyRun 识别「整次 run 命中结果缓存」：只有一条记录且它 `cacheHit=true`。
 *
 * 这是**正确行为**，不是「没调用工具」。两者在界面上必须能被区分，否则运营者会把
 * 缓存命中误读成工具循环失效。记录为 0 条时不算：什么都还没读到。
 */
export function cacheHitOnlyRun(records: readonly RunTimelineRecord[]): boolean {
  if (records.length !== 1) return false
  return records[0]?.cacheHit === true
}

/**
 * deniedWithoutOutbound 识别被闸门拒绝的记录。
 *
 * 它没有出站：既没有 token 消耗，也没有供应商侧延迟。界面必须显式写「未出站」，
 * 不能让 `0 ms / 0 token` 被读成「调用了但用量为 0」。
 */
export function deniedWithoutOutbound(record: RunTimelineRecord): boolean {
  return record.status === 'denied'
}

/** executionStatusColor 是状态徽标的语义色。表格与时间线共用，避免两套口径。 */
export function executionStatusColor(status: string): 'success' | 'warning' | 'error' {
  if (status === 'succeeded') return 'success'
  if (status === 'denied') return 'warning'
  return 'error'
}
