<script setup lang="ts">
function failInDescendant(): string { throw new Error('trusted fixture descendant failure') }
</script>
<template><span>{{ failInDescendant() }}</span></template>
