<script setup lang="ts">
import type { AdminSlotPoint } from '@sforum/admin-sdk'

const props = defineProps<{
  point: AdminSlotPoint | string
  context: unknown
}>()

const { contributionsFor } = useAdminExtensionRegistry()
const contributions = computed(() => contributionsFor(props.point))
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
