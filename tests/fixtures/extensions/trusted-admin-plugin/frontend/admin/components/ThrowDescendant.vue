<script setup lang="ts">
import type { AdminSlotProps } from '@sforum/admin-sdk'
import BrokenChild from './BrokenChild.vue'
defineProps<AdminSlotProps<'admin.jobs.row.actions'>>()
</script>
<template><BrokenChild /></template>
