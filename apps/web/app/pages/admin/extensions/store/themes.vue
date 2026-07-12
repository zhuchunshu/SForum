<script setup lang="ts">
/**
 * 主题商城框架页。安装/检索能力后续再接。
 */
import SFAdminExtensionStoreShelf from '~/components/admin/SFAdminExtensionStoreShelf.vue'

definePageMeta({
  middleware: 'admin',
  layout: 'admin'
})

defineOptions({
  name: 'AdminExtensionStoreThemes'
})
</script>

<template>
  <SFAdminExtensionStoreShelf kind="theme" page-id="/extensions/store/themes" />
</template>
