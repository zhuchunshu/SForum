<script setup lang="ts">
import { forumUserProfilePath, type ForumUserSummary } from '~/utils/forum/forumTaxonomy'
import { mentionAnchorFromEvent, mentionUsernameFromEvent } from '~/utils/forum/forumMentions'
import {
  openForumMentionPreview,
  useForumMentionPreview
} from '~/composables/forum/useForumMentionPreview'
import { userDisplayName } from '~/utils/forum/userDisplay'

/**
 * 正文容器：渲染后端已净化的 HTML，并把 @提及 链接化 / 可点。
 * 宿主页面挂载 SFUserMentionPreview 后，点击 @ 会弹出与该用户头像预览同款的资料卡片；
 * 未挂载宿主的页面（组件预览页等）保持普通链接行为。
 */
const props = defineProps<{
  /** 后端已净化并渲染好的正文 HTML。 */
  html: string
  /** 宿主已加载的 username -> UserSummary 映射；命中时卡片直接吃已有数据，避免占位闪变。 */
  userLookup?: Map<string, ForumUserSummary>
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
  const known = props.userLookup?.get(username.toLowerCase())
  openForumMentionPreview({
    username,
    displayName: userDisplayName(known, username),
    avatar: known?.avatar || null,
    profilePath: mentionHref(username)
  }, anchor)
}
</script>

<template>
  <div v-mention="mentionHref" v-html="html" @click="onContentClick" />
</template>
