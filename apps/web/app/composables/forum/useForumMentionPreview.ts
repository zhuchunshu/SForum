// 显式导入 Vue 原语（与 composables/admin 下的既有文件一致）：模块级单例状态
// 需要能在 Nuxt 之外（happy-dom 单测）直接加载。
import { computed, onBeforeUnmount, onMounted, reactive, shallowRef } from 'vue'
import type { AvatarView } from '~/composables/profile/useProfileApi'
import { forumMentionPreviewPosition } from '~/utils/forum/forumMentions'

export type ForumMentionPreviewTarget = {
  username: string
  /** 资料加载完成前的占位显示名，通常就是用户名。 */
  displayName: string
  avatar?: AvatarView | null
  profilePath: string
}

export type ForumMentionPreviewStyle = {
  top: string
  left: string
  width: string
  transform?: string
}

type ForumMentionPreviewState = {
  open: boolean
  target: ForumMentionPreviewTarget | null
  style: ForumMentionPreviewStyle | null
  /** 当前挂载的卡片宿主数量；为 0 时提及保持普通链接行为。 */
  hostCount: number
}

// 提及预览是整页单例：同一时刻只允许一张卡片，任何容器打开新卡片都会替换旧卡片。
// 这些字段只在浏览器交互里写入（open 由点击触发），SSR 渲染期间恒为关闭状态，
// 因此模块级状态不会跨请求泄漏。
const state = reactive<ForumMentionPreviewState>({
  open: false,
  target: null,
  style: null,
  hostCount: 0
})

let activeAnchor: HTMLElement | null = null
let listening = false
const layerRef = shallowRef<HTMLElement | null>(null)

function reposition() {
  if (!state.open || !activeAnchor || typeof window === 'undefined') {
    return
  }
  const rect = activeAnchor.getBoundingClientRect()
  const placement = forumMentionPreviewPosition(
    { top: rect.top, bottom: rect.bottom, left: rect.left, width: rect.width },
    { width: window.innerWidth, height: window.innerHeight }
  )
  state.style = {
    top: `${Math.round(placement.top)}px`,
    left: `${Math.round(placement.left)}px`,
    width: `${Math.round(placement.width)}px`,
    transform: placement.placement === 'above' ? 'translateY(-100%)' : undefined
  }
}

function onDocumentPointerDown(event: PointerEvent) {
  if (!state.open || !(event.target instanceof Node)) {
    return
  }
  const target = event.target
  if (layerRef.value?.contains(target) || activeAnchor?.contains(target)) {
    return
  }
  closeForumMentionPreview()
}

function onDocumentKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    closeForumMentionPreview(true)
  }
}

// 宿主/宿主事件的绑定条件用运行时检测而不是 import.meta.client：
// 语义完全一致（SSR 没有 window），同时让 happy-dom 单测能直接覆盖交互分支。
function bindListeners() {
  if (listening || typeof document === 'undefined') {
    return
  }
  listening = true
  document.addEventListener('pointerdown', onDocumentPointerDown)
  document.addEventListener('keydown', onDocumentKeyDown)
  window.addEventListener('scroll', reposition, { passive: true, capture: true })
  window.addEventListener('resize', reposition)
}

function unbindListeners() {
  if (!listening || typeof document === 'undefined') {
    return
  }
  listening = false
  document.removeEventListener('pointerdown', onDocumentPointerDown)
  document.removeEventListener('keydown', onDocumentKeyDown)
  window.removeEventListener('scroll', reposition, { capture: true })
  window.removeEventListener('resize', reposition)
}

/** 打开提及预览卡片；页面没有挂载卡片宿主时保持普通链接行为（不拦截）。 */
export function openForumMentionPreview(target: ForumMentionPreviewTarget, anchor: HTMLElement | null) {
  if (!state.hostCount) {
    return
  }
  if (activeAnchor && activeAnchor !== anchor) {
    activeAnchor.setAttribute('aria-expanded', 'false')
  }
  state.open = true
  state.target = target
  activeAnchor = anchor
  anchor?.setAttribute('aria-expanded', 'true')
  reposition()
  bindListeners()
}

/** 关闭提及预览卡片；restoreFocus 用于 Escape 关闭后把焦点还给被点的提及。 */
export function closeForumMentionPreview(restoreFocus = false) {
  if (!state.open) {
    return
  }
  const anchor = activeAnchor
  state.open = false
  state.target = null
  state.style = null
  activeAnchor = null
  unbindListeners()
  if (anchor) {
    anchor.setAttribute('aria-expanded', 'false')
    if (restoreFocus && anchor.isConnected) {
      anchor.focus()
    }
  }
}

export function useForumMentionPreview() {
  return {
    state: state as Readonly<ForumMentionPreviewState>,
    layerRef,
    previewAvailable: computed(() => state.hostCount > 0),
    openForumMentionPreview,
    closeForumMentionPreview
  }
}

/** 卡片宿主的挂载计数：只有存在宿主时点击提及才会拦截默认跳转。 */
export function useForumMentionPreviewHost() {
  const preview = useForumMentionPreview()
  onMounted(() => {
    state.hostCount += 1
  })
  onBeforeUnmount(() => {
    state.hostCount = Math.max(0, state.hostCount - 1)
    closeForumMentionPreview()
  })
  return preview
}
