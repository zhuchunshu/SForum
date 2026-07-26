<script setup lang="ts">
import SFAnnouncementBanner from './SFAnnouncementBanner.vue'
import SFFooter from './SFFooter.vue'
import SFNavbar from './SFNavbar.vue'

/**
 * 宿主公开页 chrome：仅用于 Page Registry fail-closed / core 回退路径。
 * 主题 L1 成功路径应自行挂载 sf-navbar / sf-footer 岛，避免双层导航。
 */
const { layoutShowFooter, layoutShowAnnouncements } = useActiveThemeSettings()
</script>

<template>
  <div
    class="sf-host-public-chrome sf-host-public-chrome--fullwidth-3col"
    data-host-chrome="public"
    data-layout="fullwidth-3col"
  >
    <SFNavbar />
    <SFAnnouncementBanner v-if="layoutShowAnnouncements" />
    <div class="sf-host-public-chrome__body">
      <slot />
    </div>
    <SFFooter v-if="layoutShowFooter" />
  </div>
</template>
