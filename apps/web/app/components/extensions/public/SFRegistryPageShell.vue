<script setup lang="ts">
import { computed } from 'vue'
import type { RegistryPageBody } from '~/composables/pages/useRegistryPageBody'
import { provideRegistryPageBody } from '~/composables/pages/useRegistryPageBody'
import SFRegistryPageBody from '~/components/extensions/public/SFRegistryPageBody.vue'

/**
 * Page Registry 动态公开页的外壳。
 *
 * 外壳决策的唯一位置（两个 catch-all 共用），三个互斥分支：
 * - 主题声明了 L1 外壳模板 → 用主题外壳，正文在 `sf-plugin-page-body` 岛位渲染。
 *   「恰好一个 navbar/footer」由主题模板自身契约保证（一个顶栏 + 一个页脚 + 一个正文岛）。
 * - 否则 layout=site（含未声明）→ 宿主公共外壳，结构性保证恰好一个 chrome。
 * - 否则显式 bare → 无 chrome 裸容器，作者自持 chrome。
 *
 * 只区分「有没有显式 bare」：未知值已在安装/启用预检处 fail closed，不会到达渲染层。
 * 正文以数据形式传入（不再是 slot），因为主题外壳里的岛是独立组件，拿不到 slot。
 */
const props = defineProps<{
  layout?: string | null
  shellHtml?: string | null
  shellProvider?: string | null
  body: RegistryPageBody
}>()

const siteShell = computed(() => props.layout !== 'bare')
const themeShellHtml = computed(() => (props.shellHtml || '').trim())

provideRegistryPageBody(() => props.body)
</script>

<template>
  <SFThemeTemplate
    v-if="themeShellHtml"
    :html="themeShellHtml"
    :extension-id="shellProvider || ''"
    data-registry-shell="site"
    data-registry-shell-source="theme"
  />
  <SFHostPublicChrome
    v-else-if="siteShell"
    data-registry-shell="site"
    data-registry-shell-source="host"
  >
    <div class="sf-registry-add sf-registry-add--site">
      <SFRegistryPageBody v-bind="body" />
    </div>
  </SFHostPublicChrome>
  <main
    v-else
    class="sf-public-page sf-registry-add mx-auto w-full max-w-4xl px-4 py-8"
    data-registry-shell="bare"
    data-registry-shell-source="none"
  >
    <SFRegistryPageBody v-bind="body" />
  </main>
</template>
