<script setup lang="ts">
import { useRegistryPageBody } from '~/composables/pages/useRegistryPageBody'
import SFRegistryPageBody from '~/components/extensions/public/SFRegistryPageBody.vue'

/**
 * 主题外壳模板里的插件正文岛。
 *
 * 主题用 `<sf-plugin-page-body></sf-plugin-page-body>` 声明「插件正文插在这里」，
 * 本组件从外壳注入的上下文取正文。正文仍旧只由 SFRegistryPageBody 这一处渲染，
 * 因此主题外壳与宿主外壳不会各自长出分支。
 */
const body = useRegistryPageBody()
</script>

<template>
  <SFRegistryPageBody v-if="body" v-bind="body" />
</template>
