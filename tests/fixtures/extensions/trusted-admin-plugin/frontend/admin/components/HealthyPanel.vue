<script setup lang="ts">
/// <reference path="./fixture-dependency.d.ts" />
import type { AdminSlotProps } from '@sforum/admin-sdk'
import { fixtureLabel } from 'sforum-fixture-dependency'
defineProps<AdminSlotProps<'admin.jobs.table.columns'>>()
</script>
<template><span class="text-xs font-medium text-[var(--sf-accent)]">{{ fixtureLabel() }}</span></template>
