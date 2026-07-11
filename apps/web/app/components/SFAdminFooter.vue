<script setup lang="ts">
const { t } = useI18n()

// 后台 footer 使用产品版权年，不读取站点运营方自定义 footer 文案。
const currentYear = computed(() => new Date().getFullYear())
</script>

<template>
  <!-- 与 topbar 同属后台壳层：卡片底 + 细边框，贴合主内容区底部 -->
  <footer
    data-testid="sforum-admin-footer"
    class="sforum-admin-footer flex shrink-0 items-center justify-between gap-3 border-t border-[var(--border-admin)] bg-[var(--bg-admin-card)] px-4 py-3 sm:px-8 sm:py-3.5 z-20"
  >
    <div class="flex min-w-0 flex-1 flex-col gap-1 text-center sm:flex-row sm:items-center sm:justify-between sm:gap-4 sm:text-left">
      <p
        data-testid="sforum-admin-footer-left"
        class="truncate text-xs font-medium text-slate-500 dark:text-zinc-400"
      >
        {{ t('admin.shell.footerCopyright', { year: currentYear }) }}
      </p>
      <p
        data-testid="sforum-admin-footer-right"
        class="inline-flex items-center justify-center gap-1.5 truncate text-xs font-medium text-slate-400 dark:text-zinc-500 sm:justify-end sm:text-right"
      >
        <UIcon name="i-lucide-boxes" class="size-3.5 shrink-0 text-[var(--sf-accent)] opacity-80" />
        <span class="truncate">{{ t('admin.shell.footerProductSummary') }}</span>
      </p>
    </div>
  </footer>
</template>
