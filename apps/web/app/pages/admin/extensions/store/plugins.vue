<script setup lang="ts">
/**
 * 插件商城框架页。安装/检索能力后续再接。
 */
import SFAdminExtensionStoreShelf from '~/components/admin/SFAdminExtensionStoreShelf.vue'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionStorePlugins'
})
</script>

<template>
  <SFAdminExtensionStoreShelf kind="plugin" page-id="/extensions/store/plugins" />
</template>
