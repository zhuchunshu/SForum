<script setup lang="ts">
import type { AdminSlotProps } from '@sforum/admin-sdk'
defineProps<AdminSlotProps<'admin.jobs.row.actions'>>()
function failDuringRender(): string { throw new Error('trusted fixture render failure') }
</script>
<template><span>{{ failDuringRender() }}</span></template>
