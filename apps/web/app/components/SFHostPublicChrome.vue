<script setup lang="ts">
import { useActiveThemeSettings } from '~/composables/themes/useActiveThemeSettings'
import SFAnnouncementBanner from './notifications/SFAnnouncementBanner.vue'
import SFContentColumnFooter from './forum/SFContentColumnFooter.vue'
import SFNavbar from './SFNavbar.vue'

/**
 * 宿主公开页 chrome。
 *
 * Page Registry 插件公开页（主题未声明 L1 外壳）与 core 回退路径共用此处：
 * navbar / 页脚与宿主文件路由页面同一组件族（页脚走 SFContentColumnFooter，
 * 与 settings 页同形），只保证恰好一份 chrome，不做主题级定制。
 */
const { layoutShowFooter, layoutShowAnnouncements } = useActiveThemeSettings()
</script>

<template>
  <div
    class="sf-host-public-chrome sf-host-public-chrome--fullwidth-3col"
    data-host-chrome="public"
    data-layout="fullwidth-3col"
  >
    <SFNavbar layout="fullwidth-3col" />
    <SFAnnouncementBanner v-if="layoutShowAnnouncements" />
    <div class="sf-host-public-chrome__body">
      <slot />
      <SFContentColumnFooter v-if="layoutShowFooter" class="sf-host-public-chrome__footer" />
    </div>
  </div>
</template>

<style scoped>
/* 页面正文已自带内容列页脚（如插件页的 sf-content-column-footer 岛）时，
   chrome 级页脚退位，保证「每个渲染路径恰好一个可见页脚」。 */
.sf-host-public-chrome--fullwidth-3col:has(.sforum-settings .sforum-content-footer)
  .sf-host-public-chrome__footer {
  display: none;
}
</style>
