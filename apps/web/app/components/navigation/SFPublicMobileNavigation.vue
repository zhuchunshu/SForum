<script setup lang="ts">
import SFPublicNavigationLinks from './SFPublicNavigationLinks.vue'
import type { PublicNavigationItem } from '~/utils/navigation/publicNavigation'

defineProps<{
  open: boolean
  items: PublicNavigationItem[]
}>()

const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
</script>

<template>
  <template v-if="open">
    <button
      type="button"
      class="sforum-mobile-drawer__backdrop sf-public-mobile-navigation__backdrop"
      :aria-label="t('common.close')"
      @click="emit('close')"
    />
    <aside
      class="sforum-mobile-drawer sforum-mobile-drawer--left sf-public-mobile-navigation"
      data-navigation-location="public.mobile.primary"
    >
      <header class="sforum-mobile-drawer__head">
        <strong>{{ t('home.sidebar.drawerTitle') }}</strong>
        <button type="button" :aria-label="t('common.close')" @click="emit('close')">
          <UIcon name="i-lucide-x" class="size-5" aria-hidden="true" />
        </button>
      </header>
      <SFPublicNavigationLinks mode="mobile" :items="items" @navigate="emit('close')" />
    </aside>
  </template>
</template>

<style scoped>
.sf-public-mobile-navigation {
  z-index: 91;
}

.sf-public-mobile-navigation__backdrop {
  z-index: 90;
}

@media (min-width: 981px) {
  .sf-public-mobile-navigation,
  .sf-public-mobile-navigation__backdrop {
    display: none;
  }
}
</style>
