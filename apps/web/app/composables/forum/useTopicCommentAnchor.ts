import type { Ref } from 'vue'
import { useForumApi } from '~/composables/forum/useForumApi'
import type { ForumComment, ForumCommentList } from '~/utils/forum/forumTaxonomy'

type TopicCommentAnchorOptions = {
  /** 当前 URL 锚点目标评论 id（0 = 无锚点）。 */
  targetCommentId: Ref<number>
  /** 当前评论页码（服务端分页口径）。 */
  commentPage: Ref<number>
  /** 已加载主题 id（未加载时为 0）。 */
  topicId: Ref<number>
  /** 评论列表数据；对象身份变化代表一次刷新完成。 */
  commentList: Ref<ForumCommentList>
  commentsPending: Ref<boolean>
  /** 页码 → 规范路径（含 /page/N 段与 locale 前缀）。 */
  pageTo: (page: number) => string
}

// 评论锚点定位与「回复后落到新评论」：
// - SSR 首屏含目标评论时浏览器原生定位已够；客户端导航或翻页后需兜底滚动到 #comment-{id} 并短暂高亮。
// - flat 评论流是时间流（服务端 created_at ASC, id ASC），新回复落在列表末尾而不是被回复评论下方，
//   因此发布成功后必须把视口带到新评论；已翻过页时先跳到该评论所在页。
export function useTopicCommentAnchor(options: TopicCommentAnchorOptions) {
  const forumApi = useForumApi()

  // 深链定位后的短暂强调高亮 id；与 CSS .sf-comment--flash / :target 动画时长对齐（约 3.2s）。
  const flashCommentId = ref(0)
  const COMMENT_FLASH_MS = 3200
  let flashCommentTimer: ReturnType<typeof setTimeout> | null = null

  function clearCommentFlashTimer() {
    if (flashCommentTimer != null) {
      clearTimeout(flashCommentTimer)
      flashCommentTimer = null
    }
  }

  function flashTargetComment(commentId: number) {
    if (commentId <= 0) {
      return
    }
    flashCommentId.value = commentId
    clearCommentFlashTimer()
    flashCommentTimer = setTimeout(() => {
      if (flashCommentId.value === commentId) {
        flashCommentId.value = 0
      }
      flashCommentTimer = null
    }, COMMENT_FLASH_MS)
  }

  onBeforeUnmount(() => {
    clearCommentFlashTimer()
  })

  // 每个锚点目标只定位一次：hash 整个访问期间留在 URL 里，不去重的话发回复/编辑/删除
  // 触发的 refreshComments 都会把视口重新拽回锚点评论并再次闪烁。
  const scrolledCommentId = ref(0)
  const topicPageMounted = ref(false)
  onMounted(() => { topicPageMounted.value = true })
  // 当前页找不到目标评论时的一次性兜底反查：显式页码深链（如个人主页动态）
  // 可能因软删占位、钳页或评论被删而指错页，向后端按当前 viewer 重新反查并跳转。
  const anchorFallbackTriedId = ref(0)

  async function resolveAnchorPageFallback() {
    const commentId = options.targetCommentId.value
    if (commentId <= 0 || anchorFallbackTriedId.value === commentId) {
      return
    }
    // 列表加载中或还没有任何数据说明目标可能尚未到位，等后续 watch 再判断。
    if (options.commentsPending.value || options.commentList.value.items.length === 0) {
      return
    }
    const id = options.topicId.value
    if (id <= 0) {
      return
    }
    anchorFallbackTriedId.value = commentId
    try {
      const resolved = await forumApi.resolveCommentPage(id, commentId)
      if (resolved.page > 0 && resolved.page !== options.commentPage.value) {
        await navigateTo({ path: options.pageTo(resolved.page), hash: `#comment-${commentId}` }, { replace: true })
      }
    } catch {
      // 评论不存在/对当前用户不可见：保持当前页，锚点静默失效。
    }
  }

  watch(
    [() => options.commentList.value, options.targetCommentId, topicPageMounted],
    async () => {
      if (import.meta.server || !topicPageMounted.value || options.targetCommentId.value <= 0) {
        return
      }
      if (scrolledCommentId.value === options.targetCommentId.value) {
        return
      }
      await nextTick()
      const el = document.getElementById(`comment-${options.targetCommentId.value}`)
      if (!el) {
        await resolveAnchorPageFallback()
        return
      }
      scrolledCommentId.value = options.targetCommentId.value
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
      flashTargetComment(options.targetCommentId.value)
    },
    { flush: 'post', immediate: true }
  )

  // 回复成功后把视口带到新评论；页码复用服务端反查（与分页同口径），失败时退回当前页。
  async function focusCreatedComment(comment: ForumComment) {
    if (import.meta.server || comment.id <= 0) {
      return
    }
    const id = options.topicId.value
    let page = options.commentPage.value
    if (id > 0) {
      try {
        const resolved = await forumApi.resolveCommentPage(id, comment.id)
        if (resolved.page > 0) {
          page = resolved.page
        }
      } catch {
        // 反查失败不阻断：当前页存在该评论时仍会正常滚动高亮。
      }
    }
    await navigateTo({ path: options.pageTo(page), hash: `#comment-${comment.id}` })
  }

  return {
    flashCommentId,
    focusCreatedComment
  }
}
