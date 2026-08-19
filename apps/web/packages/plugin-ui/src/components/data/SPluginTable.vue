<script setup lang="ts">
import type { SPluginTableColumn, SPluginTableRow } from '../../types'

withDefaults(defineProps<{
  columns: readonly SPluginTableColumn[]
  rows: readonly SPluginTableRow[]
  rowKey?: string
  busy?: boolean
  emptyText?: string
}>(), {
  rowKey: 'id',
  busy: false,
  emptyText: 'No data'
})
</script>

<template>
  <div class="splugin-table-wrap">
    <table class="splugin-table" :aria-busy="busy || undefined">
      <thead>
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            scope="col"
            :class="`splugin-table__cell--${column.align || 'start'}`"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="busy">
          <td :colspan="columns.length" class="splugin-table__message">Loading...</td>
        </tr>
        <template v-else-if="rows.length">
          <tr v-for="(row, rowIndex) in rows" :key="String(row[rowKey] ?? rowIndex)">
            <td
              v-for="column in columns"
              :key="column.key"
              :class="`splugin-table__cell--${column.align || 'start'}`"
            >
              <slot
                :name="`cell-${column.key}`"
                :row="row"
                :value="row[column.key]"
                :column="column"
              >
                {{ row[column.key] ?? '' }}
              </slot>
            </td>
          </tr>
        </template>
        <tr v-else>
          <td :colspan="columns.length" class="splugin-table__message">
            <slot name="empty">{{ emptyText }}</slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
