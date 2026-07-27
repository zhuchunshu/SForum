<script setup lang="ts">
import { useNotFoundPageContext } from '~/composables/errors/useNotFoundPageContext'
import { provideSystemErrorPageContext } from '~/composables/errors/useSystemErrorPageContext'
import SFSystemErrorSidebar from '~/components/errors/SFSystemErrorSidebar.vue'
import SFSystemErrorRecovery from '~/components/errors/SFSystemErrorRecovery.vue'
import SFSystemErrorRail from '~/components/errors/SFSystemErrorRail.vue'
import SFSystemErrorDetails from '~/components/errors/SFSystemErrorDetails.vue'
import SFSystemErrorActions from '~/components/errors/SFSystemErrorActions.vue'
const context = useNotFoundPageContext()
if (context) {
  provideSystemErrorPageContext(context.error)
}
</script>

<template>
  <main class="sforum-system-error sforum-not-found-page" data-not-found-body="host" data-system-error-body="host">
    <div class="sforum-system-error__layout sforum-not-found-page__layout sforum-home__layout sforum-home__layout--with-right">
      <aside class="sforum-system-error__sidebar sforum-not-found-page__sidebar sforum-home__sidebar">
        <SFSystemErrorSidebar />
      </aside>
      <div class="sforum-system-error__main sforum-not-found-page__main sforum-home__main">
        <SFSystemErrorDetails />
        <SFSystemErrorActions />
        <SFSystemErrorRecovery />
      </div>
      <SFSystemErrorRail />
    </div>
  </main>
</template>

<style scoped>
.sforum-system-error__main {
  min-width: 0;
}
</style>
