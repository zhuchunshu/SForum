import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'

const source = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8')

const liveComposable = () => source('../../app/composables/forum/useTopicCommentLive.ts')
const noticeComponent = () => source('../../app/components/forum/SFCommentStreamNotice.vue')
const topicPage = () => source('../../app/components/forum/SFTopicShowPage.vue')
const forumApi = () => source('../../app/composables/forum/useForumApi.ts')
const revisionStream = () => source('../../app/utils/realtime/revisionStream.ts')

describe('topic comment live stream contract', () => {
  test('subscribes to the topic stream with a per-topic coordination key', () => {
    const live = liveComposable()
    // SSE 只做唤醒：URL 携带本地修订号，服务端只在变更后回帧。
    expect(live).toContain('sforum:comments:revision:topic:')
    expect(live).toContain('/topics/${topicId}/comments/stream?revision=${currentRevision}')
    expect(live).toContain("eventName: 'revision'")
    // 兜底轮询周期与通知流一致（30s），断线时也能收敛。
    expect(live).toContain('const RECONCILE_FALLBACK_MS = 30_000')
  })

  test('reconciles through the revision endpoint and the shared decision matrix', () => {
    const live = liveComposable()
    expect(live).toContain('getCommentRevision(')
    expect(live).toContain('decideCommentLive(')
    expect(live).toContain('commentListChanged(')
    expect(live).toContain('commentLiveAppendLimit(')
    // 自己刚发布的评论不该被提示成新评论。
    expect(live).toContain('lastItemIsOwnFreshComment')
    // 追加必须复用页面既有刷新（同一 useAsyncData key），不能另开列表请求。
    expect(live).toContain('options.refresh()')
    expect(forumApi()).toContain('getCommentRevision')
    expect(forumApi()).toContain('/comments/revision')
  })

  test('keeps SSR clean and only runs the stream on the client', () => {
    const live = liveComposable()
    expect(live).toContain('if (import.meta.client)')
    expect(live).toContain('onBeforeUnmount(stopStreamNow)')
    // 服务端渲染不得建立连接：订阅入口自带 client 守卫，且只在 mounted/watch 内被调用。
    expect(live).toContain('if (!import.meta.client || stopStream || !options.enabled.value || options.topicId.value <= 0) return')
    expect(live).toContain('onMounted(() => {\n      trackScroll()\n      startStream()')
  })

  test('keeps the SSE resume cursor separate from the reconciled revision', () => {
    const live = liveComposable()
    // 传输层（SSE 游标）会立刻推进修订号；决策必须用「已对账」版本，
    // 否则每一帧都会被判成 remote <= local 而漏掉新评论。
    expect(live).toContain('const streamRevision = ref(0)')
    expect(live).toContain('const appliedRevision = ref(0)')
    expect(live).toContain('revision: streamRevision')
    expect(live).toContain('revision: appliedRevision.value')
    expect(live).toContain('appliedRevision.value = Math.max(appliedRevision.value, fact.revision)')
  })

  test('measures the reader position against the real scroll container', () => {
    const live = liveComposable()
    // 桌面宽度下中心列是内层滚动容器：window/document 的滚动度量看不到它，
    // 必须用评论流底边与视口比较，并在捕获阶段监听滚动。
    expect(live).toContain("document.querySelector('.sforum-topic-comments__stream')")
    expect(live).toContain('getBoundingClientRect().bottom')
    expect(live).toContain("{ passive: true, capture: true }")
    expect(live).toContain("window.addEventListener('resize', computeNearBottom")
  })

  test('announces new comments accessibly without emoji icons', () => {
    const notice = noticeComponent()
    expect(notice).toContain('role="status"')
    expect(notice).toContain('aria-live="polite"')
    expect(notice).toContain("t('topicDetail.live.newComments'")
    expect(notice).toContain("t('topicDetail.live.jumpToLastPage'")
    expect(notice).toContain("t('topicDetail.live.changed'")
    expect(notice).toContain("t('topicDetail.live.dismiss'")
    expect(notice).toContain('i-lucide-arrow-down')
    // 内层滚动容器里必须 sticky，读者上滑时才看得到提示。
    expect(notice).toContain('position: sticky')
    expect(notice).toContain('bottom: 16px')
    // 移动端底部导航是 fixed 60px 视口底栏，胶囊必须抬到它之上才可见可点。
    expect(notice).toContain('@media (max-width: 980px)')
    expect(notice).toContain('bottom: calc(60px + 12px)')
    expect(notice).toContain('prefers-reduced-motion: reduce')
    // 不使用 emoji 作为图标/状态标记。
    expect(notice).not.toMatch(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}]/u)
  })

  test('mounts the notice in the topic page and leaves streaming outside the route shell', () => {
    const page = topicPage()
    expect(page).toContain('<SFCommentStreamNotice')
    expect(page).toContain(':notice="commentLive.notice"')
    expect(page).toContain(':new-count="commentLive.newCount"')
    expect(page).toContain('@apply="commentLive.apply()"')
    expect(page).toContain('@dismiss="commentLive.dismiss()"')
    expect(page).toContain('useTopicCommentLive({')
    // 页面本身不建 EventSource：SSE/对账归 composable，路由壳只做装配。
    expect(page).not.toContain('EventSource')
    expect(page).not.toContain('/comments/stream')
  })

  test('shared revision stream runtime carries the extracted notification behavior', () => {
    const runtime = revisionStream()
    expect(runtime).toContain('BroadcastChannel')
    expect(runtime).toContain('requestExclusiveLock')
    expect(runtime).toContain('RECONNECT_MAX_DELAY_MS = 30_000')
    expect(runtime).toContain('REFRESH_COALESCE_MS')
    expect(runtime).toContain('visibilitychange')
    // 通知客户端改为消费共享运行时，行为由既有通知测试继续覆盖。
    const notifications = source('../../app/composables/notifications/notificationRealtime.ts')
    expect(notifications).toContain('createRevisionStreamClient')
    expect(notifications).toContain('/notifications/stream?revision=${revision}')
  })
})
