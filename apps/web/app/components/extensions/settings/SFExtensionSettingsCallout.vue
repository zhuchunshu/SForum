<script setup lang="ts">
import type { AdminExtensionSettingsCallout } from '~/utils/admin/adminExtensions'
import { safeUrl } from '~/utils/sfUrl'

const props = defineProps<{ callout: AdminExtensionSettingsCallout }>()
const color = computed(() => {
  switch (props.callout.tone) {
    case 'warning': return 'warning'
    case 'error': return 'error'
    case 'success': return 'success'
    default: return 'info'
  }
})
const linkHref = computed(() => safeUrl(props.callout.linkUrl))
</script>

<template>
  <UAlert
    :color="color"
    variant="subtle"
    :icon="callout.tone === 'warning' ? 'i-lucide-triangle-alert' : 'i-lucide-info'"
    :title="callout.title"
  >
    <template #description>
      <div class="space-y-3">
        <p v-if="callout.body" class="whitespace-pre-line">
          {{ callout.body }}
        </p>
        <a
          v-if="linkHref && callout.linkLabel"
          :href="linkHref"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 font-medium text-[var(--sf-accent)] hover:underline"
        >
          {{ callout.linkLabel }}
          <UIcon name="i-lucide-external-link" class="size-4" aria-hidden="true" />
        </a>
      </div>
    </template>
  </UAlert>
</template>
