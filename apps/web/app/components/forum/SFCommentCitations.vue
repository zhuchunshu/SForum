<script setup lang="ts">
import {
  citationsVisible,
  commentCitationsPath,
  createCitationFetcher,
  visibleCitations,
  type ForumCitation
} from '~/components/forum/citations'

/**
 * AI 产出评论下方的「依据」折叠块（B4 渲染面）。
 *
 * 契约（T5 冻结，OpenAPI `schemas/forum.yaml#/CommentCitation`）：
 *
 *   GET /api/v1/comments/{commentID}/citations
 *   → 200 {"items":[{"topicId":123,"commentId":913,"excerpt":"..."}]}
 *
 * 五条硬约束，逐条对应实现：
 *
 * 1. **零引用不渲染**：只以「取回非空」为渲染条件。**未知状态一律不渲染**——
 *    此前是「未取数时先渲染一个收起入口、取回空再消失」，浏览器 QA 实测的结果是
 *    每条普通评论都挂着一个「依据」标记（一页 5 条就 5 个），既误导读者（暗示
 *    人人都有出处），也与本节第 1 条自相矛盾。
 * 2. **按需取数、有并发上限**：契约是单条评论粒度、没有批量端点，所以每页最多
 *    N 个请求（N = 该页评论数）。它们由 `createCitationFetcher` 的共享 limiter
 *    限制为 4 并发，既不突发也不让第 20 条等第 19 条。取数与渲染都只发生在客户端
 *    （SSR 不发请求、不渲染空壳）。
 * 3. **没有 bot 标记可依赖**：公开 `Comment` / `ForumUser` DTO 都不带 `isBot`
 *    或 `kind`（已核对 `Forum.UserSummary` 与 OpenAPI `ForumUser`）。因此不猜
 *    「这条是不是 AI 产出的」——所有评论都给出入口，但收起时零请求；普通评论
 *    展开后得到空列表，块立刻消失。
 * 4. **404 当作「没有依据」**：契约刻意让「评论不存在」与「主题不可见」不可
 *    区分，用来防 commentID 枚举。界面因此不显示任何「无权限」提示。
 * 5. **`excerpt` 是原始正文，不是模型输出**（数据库里的 plain_text）。按纯文本
 *    渲染——**永远不 `v-html`**，否则任意用户的评论正文会变成 HTML 执行面。
 */
const props = withDefaults(defineProps<{
  /** 评论 id。非正数时不发起任何请求。 */
  commentId?: number
  /** 是否默认展开。公开评论区保持收起（收起的块不发请求）。 */
  open?: boolean
}>(), {
  commentId: 0,
  open: false
})

const { t } = useI18n()
const { request } = useApiClient()

const items = ref<ForumCitation[]>([])
// 已尝试过取数（成功或失败都算）：失败后不再每次展开都重试同一个请求。
const settled = ref(false)
const pending = ref(false)

const entries = computed(() => visibleCitations(items.value))
const hasCitations = computed(() => citationsVisible(items.value))

// 取过且为空 → 整块消失（连入口也不留）。取过且有 → 渲染列表。
// 还没取过 → 只渲染一个收起的入口，不发请求。
// 只在「确认有引用」时渲染：未知不渲染（见文件头第 1 条）。
const showBlock = computed(() => hasCitations.value)

async function loadOnce() {
  if (settled.value || pending.value) return
  const id = props.commentId
  if (!Number.isInteger(id) || id <= 0) {
    settled.value = true
    return
  }
  pending.value = true
  try {
    const payload = await request<{ items: ForumCitation[] }>(commentCitationsPath(id))
    items.value = Array.isArray(payload?.items) ? payload.items : []
  } catch {
    // 404（不存在 / 不可见）与网络故障对读者是同一个结论：这里没有依据可看。
    // 折叠块不渲染即可，不弹错误、不打断阅读。
    items.value = []
  } finally {
    settled.value = true
    pending.value = false
  }
}

// 只在第一次「从关到开」时取数。展开态由 <details> 自己维护，
// 这里读事件目标的状态而不是另建一份副本，避免两个真相。
function onToggle(event: Event) {
  const target = event.target as HTMLDetailsElement | null
  if (target?.open) void loadOnce()
}

// 共享 limiter：限制必须跨组件生效，否则「每条评论各自并发 1 个」加起来仍是
// 一页 N 个并发。
const fetcher = createCitationFetcher()

onMounted(() => {
  // 挂载即取一次：块是否出现只能由真实数据决定（未知状态不渲染）。收起的
  // <details> 在拿到数据之前根本不存在，所以这里没有 @toggle 可依赖。
  void fetcher.run(loadOnce)
})
</script>

<template>
  <details v-if="showBlock" class="sf-comment-citations" :open="open" @toggle="onToggle">
    <summary
      class="sf-comment-citations__summary"
      :aria-label="t('topicDetail.citations.summary')"
      :title="t('topicDetail.citations.summary')"
    >
      <UIcon name="i-lucide-book-marked" class="sf-comment-citations__icon" aria-hidden="true" />
      <span v-if="hasCitations">{{ t('topicDetail.citations.title', { count: entries.length }) }}</span>
      <span v-else>{{ pending ? t('topicDetail.citations.loading') : t('topicDetail.citations.summary') }}</span>
    </summary>
    <ol v-if="entries.length" class="sf-comment-citations__list">
      <li v-for="entry in entries" :key="entry.index" class="sf-comment-citations__item">
        <NuxtLink
          v-if="entry.href"
          class="sf-comment-citations__target"
          :to="entry.href"
          :title="entry.kind === 'comment'
            ? t('topicDetail.citations.jumpToComment', { id: entry.targetId })
            : t('topicDetail.citations.jumpToTopic', { id: entry.targetId })"
        >
          {{ entry.kind === 'comment'
            ? t('topicDetail.citations.commentLabel', { id: entry.targetId })
            : t('topicDetail.citations.topicLabel', { id: entry.targetId }) }}
        </NuxtLink>
        <span v-else class="sf-comment-citations__target sf-comment-citations__target--plain">
          {{ entry.kind === 'comment'
            ? t('topicDetail.citations.commentLabel', { id: entry.targetId })
            : t('topicDetail.citations.topicLabel', { id: entry.targetId }) }}
        </span>
        <!-- 纯文本插值：excerpt 是任意用户写下的正文，绝不解析为 HTML。 -->
        <span v-if="entry.excerpt" class="sf-comment-citations__excerpt">{{ entry.excerpt }}</span>
      </li>
    </ol>
  </details>
</template>

<style scoped>
.sf-comment-citations {
  margin-top: 0.5rem;
  border: 1px solid var(--sf-border-light, #e2e8f0);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  background: var(--sf-surface-subtle, #f8fafc);
  font-size: 0.8125rem;
  min-width: 0;
}

.sf-comment-citations__summary {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  cursor: pointer;
  color: var(--sf-fg-muted, #64748b);
  list-style: none;
}

.sf-comment-citations__summary::-webkit-details-marker {
  display: none;
}

.sf-comment-citations__icon {
  width: 0.875rem;
  height: 0.875rem;
  flex: none;
}

.sf-comment-citations__list {
  margin: 0.5rem 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  min-width: 0;
}

.sf-comment-citations__item {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.375rem;
  min-width: 0;
}

.sf-comment-citations__target {
  flex: none;
  color: var(--sf-accent, #2563eb);
  text-decoration: none;
}

.sf-comment-citations__target:hover {
  text-decoration: underline;
}

.sf-comment-citations__target--plain {
  color: var(--sf-fg-muted, #64748b);
}

/* 摘录独占一行：长引文不得把楼层链接挤成换行碎片，也不得撑破容器。
   pre-wrap 保留原文换行，因此不需要任何 HTML 解析即可保持可读性。 */
.sf-comment-citations__excerpt {
  flex: 1 1 100%;
  min-width: 0;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  color: var(--sf-fg-muted, #64748b);
}
</style>
