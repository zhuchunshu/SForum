/**
 * 「依据」折叠块的展示逻辑（B4 渲染面）。
 *
 * 契约是冻结的（T5 定稿，OpenAPI `schemas/forum.yaml#/CommentCitation`）：
 *
 *   GET /api/v1/comments/{commentID}/citations
 *   → 200 {"items":[{"topicId":123,"commentId":913,"excerpt":"..."}]}
 *
 * 四条边界：
 * - 无分页、无 limit：写入侧已硬限 20 条（`MaxCitationsPerReply`）。
 * - 零引用 = `{"items":[]}` + 200，不是 404 —— 空数组即不渲染折叠块。
 * - 不可见与不存在都返回 404，**两者刻意不可区分**（commentID 枚举防护）。
 *   界面不得从 404 推断「没权限」，也不要给出可区分的提示。
 * - `excerpt` 是被引用楼层的**原始正文**（数据库里的 `plain_text`），
 *   **不是模型输出**。因此它按纯文本渲染，永远不解析 HTML。
 */

/**
 * ForumCitation 是一条出处的载荷形态，与后端 `aireply.Citation` / OpenAPI
 * `CommentCitation` 逐字一致（T5 已冻结，三处契约测试钉住字段名）。
 *
 * 三个字段全部 `omitempty`，服务端会省略零值：
 * - `topicId`：被引用楼层所属主题。**每条引用都有它**（评论级引用也带）。
 * - `commentId`：被引用的评论 id。**主题级引用省略此字段**（此时链接不带锚点）。
 * - `excerpt`：被引用楼层的正文片段，已截断到 240 rune。取自数据库原文，
 *   可以直接展示，不需要标「AI 转述」，也**永远不解析为 HTML**。
 */
export type ForumCitation = {
  topicId?: number
  commentId?: number
  excerpt?: string
}

/** 引用指向的位置类型。`topic` 指话题主楼（楼层为 0 的那条）。 */
export type CitationAnchorKind = 'comment' | 'topic' | 'none'

export type VisibleCitation = {
  /** 展示序号，1 起。 */
  index: number
  citation: ForumCitation
  kind: CitationAnchorKind
  /** 楼层 id（kind=comment 时）或话题 id（kind=topic 时）；不可解析时为 0。 */
  targetId: number
  /**
   * 跳转地址（路径 + fragment）。空串表示这条引用不可跳转：界面只展示摘录，
   * 不给出一个点了没反应或跳到别处的链接。
   */
  href: string
  excerpt: string
}

// 引文长度上限与后端 `MaxCitationExcerptRunes`（240）保持一致。
//
// 后端已经截过，前端再截一个更短的长度会**静默丢弃服务端给出的真实内容**：那是
// 撒谎，不是保护。这里只是防御异常载荷的兜底，不改变正常数据。
const CITATION_EXCERPT_MAX_RUNES = 240

function positiveID(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 0
  if (!Number.isInteger(value) || value <= 0) return 0
  return value
}

function clipExcerpt(value: unknown): string {
  if (typeof value !== 'string') return ''
  const trimmed = value.trim()
  if (!trimmed) return ''
  if (trimmed.length <= CITATION_EXCERPT_MAX_RUNES) return trimmed
  // 按 code point 截断，避免在多字节字符中间切开。
  return `${Array.from(trimmed).slice(0, CITATION_EXCERPT_MAX_RUNES).join('')}…`
}

/**
 * citationTarget 把一条引用翻译成位置。
 *
 * 优先级：楼层 > 主楼。有 `commentId` 时指向那条回复（它必然属于某个话题），
 * 只有 `topicId` 时指向话题主楼——主楼同样是一条真实出处。
 *
 * 路径用 `/t/<topicId>`：详情页的定位解析接受纯 id 形态（`topicPathLookupCandidates`
 * 的 `id` 候选），通知深链已经在用同一条路径形状。
 */
export function citationTarget(
  citation: ForumCitation
): { kind: CitationAnchorKind, targetId: number, href: string } {
  const commentId = positiveID(citation.commentId)
  const topicId = positiveID(citation.topicId)

  if (commentId > 0) {
    // 契约保证评论级引用同时带 topicId（T5 在写入侧保证，且三处测试钉住）。
    // 缺 topicId 时退回「不跳转」而不是猜一个路径。
    return {
      kind: 'comment',
      targetId: commentId,
      href: topicId > 0 ? `/t/${topicId}#comment-${commentId}` : ''
    }
  }
  if (topicId > 0) {
    // 主题级引用省略 commentId，链接因此不带锚点（契约原话）。
    return { kind: 'topic', targetId: topicId, href: `/t/${topicId}` }
  }
  // 两个 id 都不可用：这条引用指不回任何真实位置，展示它只会是装饰。
  return { kind: 'none', targetId: 0, href: '' }
}

/**
 * visibleCitations 过滤掉不可解析的条目并编号。
 *
 * 服务端本应在校验阶段丢弃这类记录（B4 易错点 3），但前端不能因此假定载荷一定
 * 干净：一条点不开的「依据」比不渲染更糟。返回空数组时调用方不渲染整块。
 */
export function visibleCitations(citations: readonly ForumCitation[] | undefined | null): VisibleCitation[] {
  if (!Array.isArray(citations) || citations.length === 0) return []

  const visible: VisibleCitation[] = []
  for (const citation of citations) {
    if (!citation || typeof citation !== 'object') continue
    const target = citationTarget(citation)
    if (target.kind === 'none') continue
    visible.push({
      index: visible.length + 1,
      citation,
      kind: target.kind,
      targetId: target.targetId,
      href: target.href,
      excerpt: clipExcerpt(citation.excerpt)
    })
  }
  return visible
}

/** citationsVisible 是「零引用不渲染」这一条的单一判据。 */
export function citationsVisible(citations: readonly ForumCitation[] | undefined | null): boolean {
  return visibleCitations(citations).length > 0
}

/**
 * commentCitationsPath 构造取引用地址。
 *
 * 契约：`GET /api/v1/comments/{commentID}/citations` —— **没有分页、没有 limit**，
 * 一条回复的引用已在写入侧硬限 20 条。传 perPage/page 会被服务端忽略，
 * 所以这里也不构造它们。
 */
export function commentCitationsPath(commentId: number): string {
  return `/comments/${encodeURIComponent(String(commentId))}/citations`
}

/**
 * citationsConcurrency 限制同一页内「按评论取依据」的并发数。
 *
 * 契约只有单评论粒度（没有批量端点），而一页可能有 20 条评论。无条件并发请求会
 * 在首屏打出一串往返；串行又会让靠后的评论迟迟没有结果。4 是浏览器对同源连接
 * 的常见并发上界的保守取值：既不让 API 突发，也不让第 20 条等第 19 条。
 */
export const citationsConcurrency = 4

/**
 * createCitationFetcher 生成一个带并发上限的取数函数。
 *
 * 每个评论组件各持一个 `load()` 闭包，但共享同一个 limiter 实例——限制必须跨
 * 组件生效，否则「每条评论各自并发 1 个」加起来仍是一页 20 个并发。
 */
export function createCitationFetcher(): {
  run: <T>(task: () => Promise<T>) => Promise<T>
} {
  let active = 0
  const queue: Array<() => void> = []
  const acquire = (): Promise<void> => {
    if (active < citationsConcurrency) {
      active += 1
      return Promise.resolve()
    }
    return new Promise<void>((resolve) => queue.push(resolve))
  }
  const release = () => {
    const next = queue.shift()
    if (next) {
      // 名额直接转交给队首，active 不变——避免「释放后再抢」造成的队尾饥饿。
      next()
      return
    }
    active -= 1
  }
  return {
    async run<T>(task: () => Promise<T>): Promise<T> {
      await acquire()
      try {
        return await task()
      } finally {
        release()
      }
    }
  }
}
