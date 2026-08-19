<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  tone?: 'info' | 'success' | 'warning' | 'danger'
}>(), {
  title: '',
  tone: 'info'
})

</script>

<template>
  <div class="splugin-alert" :class="`splugin-alert--${tone}`" :role="tone === 'danger' ? 'alert' : 'status'">
    <p v-if="title" class="splugin-alert__title">{{ title }}</p>
    <div class="splugin-alert__body"><slot /></div>
  </div>
</template>
