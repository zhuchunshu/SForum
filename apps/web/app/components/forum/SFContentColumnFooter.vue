<script setup lang="ts">
import SFFooter from '../SFFooter.vue'
</script>

<template>
  <div class="sforum-content-footer" data-sforum-content-footer>
    <SFFooter />
  </div>
</template>

<style scoped>
.sforum-content-footer {
  flex: 0 0 auto;
  margin-top: auto;
  padding-top: 20px;
}

.sforum-content-footer :deep(.sf-footer) {
  border-top-color: var(--sf-public-border);
  background: transparent;
}

.sforum-content-footer :deep(.sf-footer__inner),
.sforum-content-footer :deep(.sf-footer__friends-inner) {
  max-width: none;
  padding-top: 14px;
  padding-right: 0;
  padding-bottom: 14px;
  padding-left: 0;
}

@media (min-width: 981px) {
  .sforum-content-footer {
    margin-right: -24px;
    margin-left: -24px;
  }

  .sforum-content-footer :deep(.sf-footer__inner),
  .sforum-content-footer :deep(.sf-footer__friends-inner) {
    padding-right: 24px;
    padding-left: 24px;
  }
}
</style>
