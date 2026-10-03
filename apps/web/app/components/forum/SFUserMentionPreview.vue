<script setup lang="ts">
import SFCommentUserPreview from '~/components/forum/SFCommentUserPreview.vue'
import { useForumMentionPreviewHost } from '~/composables/forum/useForumMentionPreview'

// 提及预览卡片宿主：整页单例，读取 composable 里的共享状态，只渲染当前被点开的那个 @。
// 用 Teleport 挂到 body，避免被正文容器的 overflow/transform 裁剪，定位由锚点矩形计算。
const { state, layerRef } = useForumMentionPreviewHost()
</script>

<template>
  <Teleport to="body">
    <div
      v-if="state.open && state.target"
      ref="layerRef"
      class="sf-mention-preview-layer"
      data-testid="mention-user-preview"
      :style="state.style"
    >
      <SFCommentUserPreview
        :author="state.target.displayName"
        :username="state.target.username"
        :avatar="state.target.avatar"
        :profile-path="state.target.profilePath"
      />
    </div>
  </Teleport>
</template>
