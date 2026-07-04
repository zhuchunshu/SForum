<script setup lang="ts">
definePageMeta({ middleware: 'admin' })

const { t } = useI18n()
const localePath = useLocalePath()

useSeoMeta({
  title: t('admin.home.metaTitle')
})
</script>

<template>
  <main class="page-shell">
    <section class="forum-board" aria-labelledby="admin-home-title">
      <p class="eyebrow">
        {{ t('admin.home.eyebrow') }}
      </p>
      <h1 id="admin-home-title">
        {{ t('admin.home.title') }}
      </h1>
      <p class="intro">
        {{ t('admin.home.intro') }}
      </p>

      <nav class="admin-actions" :aria-label="t('admin.home.navLabel')">
        <NuxtLink class="admin-link" :to="localePath('/admin/roles')">
          {{ t('admin.home.rolesLink') }}
        </NuxtLink>
      </nav>
    </section>
  </main>
</template>

<style scoped>
.admin-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 28px;
}

.admin-link {
  display: inline-flex;
  align-items: center;
  min-height: 40px;
  padding: 0 16px;
  border: 1px solid var(--sf-accent);
  border-radius: 8px;
  background: var(--sf-accent);
  color: #ffffff;
  font-size: 14px;
  font-weight: 700;
}
</style>
