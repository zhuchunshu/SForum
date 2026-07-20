<script setup lang="ts">
/**
 * 宿主公开页 chrome：仅用于 Page Registry fail-closed / core 回退路径。
 * 主题 L1 成功路径应自行挂载 sf-navbar / sf-footer 岛，避免双层导航。
 */
const { layoutShowFooter, layoutShowAnnouncements } = useActiveThemeSettings()
</script>

<template>
  <div class="sf-host-public-chrome flex flex-col min-h-screen" data-host-chrome="public">
    <SFNavbar />
    <SFAnnouncementBanner v-if="layoutShowAnnouncements" />
    <div class="flex-1">
      <slot />
    </div>
    <SFFooter v-if="layoutShowFooter" />
  </div>
</template>
