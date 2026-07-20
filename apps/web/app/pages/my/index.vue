<script setup lang="ts">
/** forum.my.home 路由壳：outlet + fail-closed；呈现由主题 L1 + SFMyHomePage 岛拥有。 */
definePageMeta({ requiresAuth: true })
</script>

<template>
  <SFPageOutlet page="forum.my.home">
    <SFMyHomePage />
  </SFPageOutlet>
</template>
