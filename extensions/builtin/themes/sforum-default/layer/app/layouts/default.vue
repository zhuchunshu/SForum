<script setup lang="ts">
// 页脚 / 公告条由主题 extension settings 控制
const { layoutShowFooter, layoutShowAnnouncements } = useActiveThemeSettings()
</script>

<template>
  <div class="flex flex-col min-h-screen">
    <SFNavbar />
    <SFAnnouncementBanner v-if="layoutShowAnnouncements" />
    <div class="flex-1">
      <slot />
    </div>
    <SFFooter v-if="layoutShowFooter" />
  </div>
</template>
