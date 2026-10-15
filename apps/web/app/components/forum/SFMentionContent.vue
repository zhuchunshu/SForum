<script setup lang="ts">
import { forumUserProfilePath } from '~/utils/forum/forumTaxonomy'
import { mentionAnchorFromEvent, mentionUsernameFromEvent } from '~/utils/forum/forumMentions'
import {
  openForumMentionPreview,
  useForumMentionPreview
} from '~/composables/forum/useForumMentionPreview'

/**
 * 正文容器：渲染后端已净化的 HTML，并把 @提及 链接化 / 可点，
 * 同时由 v-outbound 保证容器内链接一律新标签页、站外走中转页。
 * 宿主页面挂载 SFUserMentionPreview 后，点击 @ 会弹出与该用户头像预览同款的资料卡片；
 * 未挂载宿主的页面（组件预览页等）保持普通链接行为。
 *
 * 提及渲染契约：锚点永远显示真实用户名 @username（与通知/存储语义一致），不做昵称替换。
 */
defineProps<{
  /** 后端已净化并渲染好的正文 HTML。 */
  html: string
}>()

const localePath = useLocalePath()
const { previewAvailable } = useForumMentionPreview()

// v-mention 的地址解析器：与站点其它资料入口一致，带语言前缀。
function mentionHref(username: string) {
  return localePath(forumUserProfilePath(username))
}

function onContentClick(event: MouseEvent) {
  if (!previewAvailable.value) {
    return
  }
  // 只有裸左键才拦截；修饰键/中键保留浏览器默认的「新标签打开资料页」行为。
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
    return
  }
  const username = mentionUsernameFromEvent(event.target)
  if (!username) {
    return
  }
  const anchor = mentionAnchorFromEvent(event.target)
  event.preventDefault()
  // 占位名就是 username；卡片内的资料请求会把昵称补齐。
  openForumMentionPreview({
    username,
    displayName: username,
    avatar: null,
    profilePath: mentionHref(username)
  }, anchor)
}
</script>

<template>
  <!-- v-mention 先补齐提及锚点，v-outbound 再统一链接形态（提及锚点会被跳过）。 -->
  <div v-mention="mentionHref" v-outbound v-html="html" @click="onContentClick" />
</template>
