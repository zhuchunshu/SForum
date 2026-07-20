<script setup lang="ts">
/** forum.profile.show 路由壳：outlet + fail-closed；呈现由主题 L1 + SFProfileShowPage 岛拥有。 */
definePageMeta({ public: true })
</script>

<template>
  <SFPageOutlet page="forum.profile.show">
    <SFProfileShowPage />
  </SFPageOutlet>
</template>
