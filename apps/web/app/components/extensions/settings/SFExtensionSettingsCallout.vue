<script setup lang="ts">
import type { AdminExtensionSettingsCallout } from '~/utils/adminExtensions'

const props = defineProps<{ callout: AdminExtensionSettingsCallout }>()
const color = computed(() => {
  switch (props.callout.tone) {
    case 'warning': return 'warning'
    case 'error': return 'error'
    case 'success': return 'success'
    default: return 'info'
  }
})
</script>

<template>
  <UAlert
    :color="color"
    variant="subtle"
    :icon="callout.tone === 'warning' ? 'i-lucide-triangle-alert' : 'i-lucide-info'"
    :title="callout.title"
    :description="callout.body"
  />
</template>
