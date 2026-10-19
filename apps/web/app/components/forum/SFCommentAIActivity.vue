<script setup lang="ts">
import {
  activityVisible,
  commentCitationsPath,
  createCitationFetcher,
  toolCallsVisible,
  visibleCitations,
  visibleToolCalls,
  type ForumCitation,
  type ForumToolCall
} from '~/components/forum/aiActivity'

/**
 * AI 产出评论下方的「AI 活动」折叠块：工具调用 + 依据。
 *
 * 契约（OpenAPI `schemas/forum.yaml#/CommentCitationList`）：
 *
 *   GET /api/v1/comments/{commentID}/citations
 *   → 200 {"items":[{"topicId":123,"commentId":913,"excerpt":"..."}],
 *          "toolCalls":[{"tool":"web-search","arguments":"{\"query\":\"…\"}",
 *                        "latencyMs":120}]}
 *
 * 六条硬约束，逐条对应实现：
 *
 * 1. **两个分区都为空就不渲染**：只以「取回非空」为渲染条件。**未知状态一律不
 *    渲染**——此前是「未取数时先渲染一个收起入口、取回空再消失」，浏览器 QA 实测
 *    的结果是每条普通评论都挂着一个「依据」标记（一页 5 条就 5 个），既误导读者
 *    （暗示人人都有出处），也与本节第 1 条自相矛盾。
 * 2. **按需取数、有并发上限**：契约是单条评论粒度、没有批量端点，所以每页最多
 *    N 个请求（N = 该页评论数）。它们由 `createCitationFetcher` 的共享 limiter
 *    限制为 4 并发，既不突发也不让第 20 条等第 19 条。取数与渲染都只发生在客户端
 *    （SSR 不发请求、不渲染空壳）。工具调用与依据**共用同一个请求**——它们是同一
 *    个端点的两个分区，这也是不新增 `/tool-calls` 路由的理由。
 * 3. **没有 bot 标记可依赖**：公开 `Comment` / `ForumUser` DTO 都不带 `isBot`
 *    或 `kind`（已核对 `Forum.UserSummary` 与 OpenAPI `ForumUser`）。因此不猜
 *    「这条是不是 AI 产出的」——所有评论都取一次，普通评论得到空结果，块立刻消失。
 * 4. **404 当作「没有活动」**：契约刻意让「评论不存在」与「主题不可见」不可
 *    区分，用来防 commentID 枚举。界面因此不显示任何「无权限」提示。
 * 5. **`excerpt` 是原始正文，不是模型输出**（数据库里的 plain_text）。按纯文本
 *    渲染——**永远不 `v-html`**，否则任意用户的评论正文会变成 HTML 执行面。
 * 6. **`arguments` 是模型输出**（工具入参），同样按纯文本渲染。工具名用等宽徽标
 *    原样显示：它是宿主登记表里的稳定标识，不做本地化（插件工具的名字由插件定义，
 *    前端维护一张翻译表只会永远落后于安装了什么）。
 */
const props = withDefaults(defineProps<{
  /** 评论 id。非正数时不发起任何请求。 */
  commentId?: number
  /** 是否默认展开。公开评论区保持收起。 */
  open?: boolean
}>(), {
  commentId: 0,
  open: false
})

const { t } = useI18n()
const { request } = useApiClient()

const items = ref<ForumCitation[]>([])
const toolCalls = ref<ForumToolCall[]>([])
// 已尝试过取数（成功或失败都算）：失败后不再每次展开都重试同一个请求。
const settled = ref(false)
const pending = ref(false)

const entries = computed(() => visibleCitations(items.value))
const toolEntries = computed(() => visibleToolCalls(toolCalls.value))
const hasCitations = computed(() => entries.value.length > 0)
const hasToolCalls = computed(() => toolEntries.value.length > 0)

// 只在「确认有活动」时渲染：未知不渲染（见文件头第 1 条）。
const showBlock = computed(() => activityVisible(items.value, toolCalls.value))

// 折叠标题就是「这块里有什么」的摘要。三种组合各有自己的文案，而不是拼一句
// 「AI 活动（0 次工具 · 2 条依据）」——零值不该出现在读者眼前。
const summaryLabel = computed(() => {
  if (hasToolCalls.value && hasCitations.value) {
    return t('topicDetail.aiActivity.both', {
      tools: toolEntries.value.length,
      citations: entries.value.length
    })
  }
  if (hasToolCalls.value) {
    return t('topicDetail.aiActivity.toolCallsTitle', { count: toolEntries.value.length })
  }
  return t('topicDetail.citations.title', { count: entries.value.length })
})

async function loadOnce() {
  if (settled.value || pending.value) return
  const id = props.commentId
  if (!Number.isInteger(id) || id <= 0) {
    settled.value = true
    return
  }
  pending.value = true
  try {
    const payload = await request<{ items?: ForumCitation[], toolCalls?: ForumToolCall[] }>(
      commentCitationsPath(id)
    )
    items.value = Array.isArray(payload?.items) ? payload.items : []
    toolCalls.value = Array.isArray(payload?.toolCalls) ? payload.toolCalls : []
  } catch {
    // 404（不存在 / 不可见）与网络故障对读者是同一个结论：这里没有活动可看。
    // 折叠块不渲染即可，不弹错误、不打断阅读。
    items.value = []
    toolCalls.value = []
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
  <details v-if="showBlock" class="sf-comment-ai-activity" :open="open" @toggle="onToggle">
    <summary
      class="sf-comment-ai-activity__summary"
      :aria-label="summaryLabel"
      :title="summaryLabel"
    >
      <UIcon
        :name="hasToolCalls ? 'i-lucide-wrench' : 'i-lucide-book-marked'"
        class="sf-comment-ai-activity__icon"
        aria-hidden="true"
      />
      <span>{{ summaryLabel }}</span>
    </summary>

    <!-- 分区一：工具调用。顺序即调用顺序（seq）。 -->
    <section v-if="hasToolCalls" class="sf-comment-ai-activity__section">
      <!-- 只有一个分区时，分区标题与折叠标题重复，省掉它。 -->
      <p v-if="hasCitations" class="sf-comment-ai-activity__heading">
        {{ t('topicDetail.aiActivity.toolCallsTitle', { count: toolEntries.length }) }}
      </p>
      <ol class="sf-comment-ai-activity__list">
        <li v-for="entry in toolEntries" :key="entry.index" class="sf-comment-ai-activity__item">
          <code class="sf-comment-ai-activity__tool">{{ entry.tool }}</code>
          <!-- 纯文本插值：arguments 是模型输出，绝不解析为 HTML。 -->
          <span v-if="entry.argumentSummary" class="sf-comment-ai-activity__argument">
            {{ entry.argumentSummary }}
          </span>
          <span v-if="entry.isError" class="sf-comment-ai-activity__failed">
            {{ t('topicDetail.aiActivity.failed') }}
          </span>
          <span v-if="entry.latencyMs > 0" class="sf-comment-ai-activity__latency">
            {{ t('topicDetail.aiActivity.latency', { ms: entry.latencyMs }) }}
          </span>
        </li>
      </ol>
    </section>

    <!-- 分区二：依据。 -->
    <section v-if="hasCitations" class="sf-comment-ai-activity__section">
      <p v-if="hasToolCalls" class="sf-comment-ai-activity__heading">
        {{ t('topicDetail.citations.title', { count: entries.length }) }}
      </p>
      <ol class="sf-comment-ai-activity__list">
        <li v-for="entry in entries" :key="entry.index" class="sf-comment-ai-activity__item">
          <NuxtLink
            v-if="entry.href"
            class="sf-comment-ai-activity__target"
            :to="entry.href"
            :title="entry.kind === 'comment'
              ? t('topicDetail.citations.jumpToComment', { id: entry.targetId })
              : t('topicDetail.citations.jumpToTopic', { id: entry.targetId })"
          >
            {{ entry.kind === 'comment'
              ? t('topicDetail.citations.commentLabel', { id: entry.targetId })
              : t('topicDetail.citations.topicLabel', { id: entry.targetId }) }}
          </NuxtLink>
          <span v-else class="sf-comment-ai-activity__target sf-comment-ai-activity__target--plain">
            {{ entry.kind === 'comment'
              ? t('topicDetail.citations.commentLabel', { id: entry.targetId })
              : t('topicDetail.citations.topicLabel', { id: entry.targetId }) }}
          </span>
          <!-- 纯文本插值：excerpt 是任意用户写下的正文，绝不解析为 HTML。 -->
          <span v-if="entry.excerpt" class="sf-comment-ai-activity__excerpt">{{ entry.excerpt }}</span>
        </li>
      </ol>
    </section>
  </details>
</template>

<style scoped>
.sf-comment-ai-activity {
  margin-top: 0.5rem;
  border: 1px solid var(--sf-border-light, #e2e8f0);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  background: var(--sf-surface-subtle, #f8fafc);
  font-size: 0.8125rem;
  min-width: 0;
}

.sf-comment-ai-activity__summary {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  cursor: pointer;
  color: var(--sf-fg-muted, #64748b);
  list-style: none;
}

.sf-comment-ai-activity__summary::-webkit-details-marker {
  display: none;
}

.sf-comment-ai-activity__icon {
  width: 0.875rem;
  height: 0.875rem;
  flex: none;
}

.sf-comment-ai-activity__section {
  margin-top: 0.5rem;
  min-width: 0;
}

.sf-comment-ai-activity__section + .sf-comment-ai-activity__section {
  margin-top: 0.625rem;
  padding-top: 0.5rem;
  border-top: 1px dashed var(--sf-border-light, #e2e8f0);
}

.sf-comment-ai-activity__heading {
  margin: 0 0 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--sf-fg-muted, #64748b);
}

.sf-comment-ai-activity__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  min-width: 0;
}

.sf-comment-ai-activity__item {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.375rem;
  min-width: 0;
}

.sf-comment-ai-activity__tool {
  flex: none;
  font-family: var(--sf-font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  font-size: 0.75rem;
  padding: 0 0.25rem;
  border-radius: 4px;
  background: var(--sf-surface-raised, #eef2f7);
  color: var(--sf-fg, #0f172a);
}

.sf-comment-ai-activity__argument {
  min-width: 0;
  color: var(--sf-fg-muted, #64748b);
  overflow-wrap: anywhere;
  /* 参数是模型给的原文：保留换行，但不解析为 HTML。 */
  white-space: pre-wrap;
}

.sf-comment-ai-activity__failed {
  flex: none;
  font-size: 0.75rem;
  color: var(--sf-danger, #b91c1c);
}

.sf-comment-ai-activity__latency {
  flex: none;
  font-size: 0.75rem;
  color: var(--sf-fg-subtle, #94a3b8);
  font-variant-numeric: tabular-nums;
}

.sf-comment-ai-activity__target {
  flex: none;
  color: var(--sf-accent, #2563eb);
  text-decoration: none;
}

.sf-comment-ai-activity__target:hover {
  text-decoration: underline;
}

.sf-comment-ai-activity__target--plain {
  color: var(--sf-fg-muted, #64748b);
}

.sf-comment-ai-activity__excerpt {
  min-width: 0;
  color: var(--sf-fg-muted, #64748b);
  overflow-wrap: anywhere;
  /* 引文是楼层正文：保留换行，但不解析为 HTML。 */
  white-space: pre-wrap;
}
</style>
