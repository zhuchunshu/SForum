<script setup lang="ts">
const { t } = useI18n()
const { changed, reload } = useAdminReleaseMonitor()
</script>

<template>
  <UAlert v-if="changed" color="warning" variant="solid" icon="i-lucide-refresh-cw" :title="t('admin.extensions.releaseNotice.title')" :description="t('admin.extensions.releaseNotice.description')" :actions="[{ label: t('admin.extensions.releaseNotice.reload'), onClick: reload }]" />
</template>
