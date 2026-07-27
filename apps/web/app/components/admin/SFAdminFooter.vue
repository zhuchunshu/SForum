<script setup lang="ts">
const { t } = useI18n()

// 后台 footer 使用产品版权年，不读取站点运营方自定义 footer 文案。
const currentYear = computed(() => new Date().getFullYear())
</script>

<template>
  <!--
    外层 mt-auto：短页把整块顶到内容区底部。
    footer 自身 mt-6/sm:mt-8：与正文始终保留 margin-top，不贴内容。
  -->
  <div class="mt-auto shrink-0">
    <footer
      data-testid="sforum-admin-footer"
      class="sforum-admin-footer mt-6 -mx-4 -mb-4 border-t border-[var(--border-admin)] bg-[var(--bg-admin-card)] px-4 py-3 sm:mt-8 sm:-mx-6 sm:-mb-6 sm:px-8 sm:py-3.5"
    >
      <div class="flex min-w-0 flex-col gap-1 text-center sm:flex-row sm:items-center sm:justify-between sm:gap-6 sm:text-left">
        <p
          data-testid="sforum-admin-footer-left"
          class="truncate text-xs font-medium text-slate-500 dark:text-zinc-400"
        >
          {{ t('admin.shell.footerCopyright', { year: currentYear }) }}
        </p>
        <p
          data-testid="sforum-admin-footer-right"
          class="truncate text-xs font-medium text-slate-400 dark:text-zinc-500 sm:text-right"
        >
          {{ t('admin.shell.footerProductSummary') }}
        </p>
      </div>
    </footer>
  </div>
</template>
