<script setup lang="ts">
import type { AdminSlotPoint } from '@sforum/admin-sdk'

const props = defineProps<{
  point: AdminSlotPoint | string
  context: unknown
  /** 仅渲染指定扩展的贡献（例如设置页只接受当前扩展自己的组件）。 */
  extensionId?: string
}>()

const { contributionsFor } = useAdminExtensionRegistry()
const contributions = computed(() => {
  const items = contributionsFor(props.point)
  if (!props.extensionId) {
    return items
  }
  return items.filter(item => item.extensionId === props.extensionId)
})
</script>

<template>
  <div v-if="contributions.length" class="sf-admin-extension-slot" :data-admin-extension-point="point">
    <SFAdminExtensionContribution
      v-for="contribution in contributions"
      :key="`${contribution.extensionId}:${contribution.contributionId}`"
      :metadata="contribution"
      :context="context"
    />
  </div>
</template>
