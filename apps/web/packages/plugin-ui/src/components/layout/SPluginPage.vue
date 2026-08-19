<script setup lang="ts">
withDefaults(defineProps<{
  busy?: boolean
}>(), {
  busy: false
})
</script>

<template>
  <div class="splugin-page" :aria-busy="busy || undefined">
    <slot />
  </div>
</template>
