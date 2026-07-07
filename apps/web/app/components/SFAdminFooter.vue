<script setup lang="ts">
const { t } = useI18n()

// 后台 footer 使用产品版权年，不读取站点运营方自定义 footer 文案。
const currentYear = computed(() => new Date().getFullYear())
</script>

<template>
  <footer
    data-testid="sforum-admin-footer"
    class="mt-auto border-t border-[var(--border-admin)] px-1 pt-4 text-xs font-medium text-slate-500 dark:text-zinc-500"
  >
    <div class="flex flex-col gap-2 text-center sm:flex-row sm:items-center sm:justify-between sm:text-left">
      <p data-testid="sforum-admin-footer-left">
        {{ t('admin.shell.footerCopyright', { year: currentYear }) }}
      </p>
      <p data-testid="sforum-admin-footer-right" class="sm:text-right">
        {{ t('admin.shell.footerProductSummary') }}
      </p>
    </div>
  </footer>
</template>
