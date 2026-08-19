<script setup lang="ts">
import { ref } from 'vue'
import type { AdminPageBridgeV1 } from '@sforum/admin-sdk'
import {
  SPluginAlert,
  SPluginButton,
  SPluginField,
  SPluginInput,
  SPluginPage,
  SPluginSection,
  SPluginTable,
  type SPluginTableColumn
} from '@sforum/plugin-ui'

const props = defineProps<{ bridge: AdminPageBridgeV1 }>()
const message = ref('Ready from the Vue reference plugin')
const columns: SPluginTableColumn[] = [
  { key: 'surface', label: 'Surface' },
  { key: 'owner', label: 'Owner' }
]
const rows = [
  { id: 1, surface: 'Sidebar, topbar, tabs and heading', owner: 'SForum Host' },
  { id: 2, surface: 'Dashboard body', owner: 'Plugin Vue component' }
]

function completeAction() {
  props.bridge.toast({ title: 'Plugin action completed', description: message.value, kind: 'success' })
}
</script>

<template>
  <SPluginPage>
    <SPluginAlert tone="info" title="Host shell inherited">
      This Vue SFC is compiled into immutable ESM/CSS before the plugin is installed.
    </SPluginAlert>

    <SPluginSection title="Plugin action" description="Plugin UI SDK provides the layout and controls.">
      <SPluginField label="Message" for="fixture-dashboard-message">
        <SPluginInput id="fixture-dashboard-message" v-model="message" />
      </SPluginField>
      <template #actions>
        <SPluginButton @click="completeAction">Run action</SPluginButton>
      </template>
    </SPluginSection>

    <SPluginSection title="Surface ownership">
      <SPluginTable :columns="columns" :rows="rows" row-key="id" />
    </SPluginSection>
  </SPluginPage>
</template>
