<script setup lang="ts">
import type { AdminSlotProps } from '@sforum/admin-sdk'
defineProps<AdminSlotProps<'admin.jobs.detail.sections'>>()
const missingModule = '/__sforum_missing_fixture_module__.js'
await import(/* @vite-ignore */ missingModule)
</script>
<template><div /></template>
