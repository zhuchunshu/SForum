<script setup lang="ts">
import SFErrorPageContent from '~/components/errors/SFErrorPageContent.vue'
import type { NuxtError } from '#app'

defineProps<{
  error: NuxtError
}>()
</script>

<template>
  <SFErrorPageContent :error="error" emergency />
</template>
