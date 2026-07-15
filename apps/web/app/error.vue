<script setup lang="ts">
import type { NuxtError } from '#app'
import { normalizeErrorStatus } from '~/utils/errorPage'

const props = defineProps<{
  error: NuxtError
}>()

const isNotFound = computed(() => normalizeErrorStatus(props.error?.statusCode) === 404)
</script>

<template>
  <UApp>
    <SFPageOutlet v-if="isNotFound" page="system.not_found">
      <SFErrorPageContent :error="error" />
    </SFPageOutlet>
    <SFErrorPageContent v-else :error="error" />
  </UApp>
</template>
