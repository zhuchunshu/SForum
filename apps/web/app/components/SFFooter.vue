<script setup lang="ts">
// 获取 i18n 翻译实例与全局站点配置
const { t } = useI18n()
const { siteName } = useWebOptions()

// 动态计算当前年份，以保证版权年份的正确性
const currentYear = computed(() => new Date().getFullYear())
</script>

<template>
  <footer class="sf-footer">
    <div class="sf-footer__inner">
      <!-- 版权信息 -->
      <div class="sf-footer__copyright">
        {{ t('footer.copyright', { year: currentYear, siteName: siteName }) }}
      </div>
      <!-- 辅助虚拟链接 -->
      <div class="sf-footer__links">
        <a href="#" class="sf-footer__link">{{ t('footer.terms') }}</a>
        <a href="#" class="sf-footer__link">{{ t('footer.privacy') }}</a>
        <a href="#" class="sf-footer__link">{{ t('footer.guidelines') }}</a>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.sf-footer {
  width: 100%;
  border-top: 1px solid var(--border-default);
  background-color: transparent;
  transition: border-color 0.2s;
}

.sf-footer__inner {
  max-width: 1376px;
  margin: 0 auto;
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
}

/* 适配桌面端布局，转为两端对齐 */
@media (min-width: 768px) {
  .sf-footer__inner {
    flex-direction: row;
    justify-content: space-between;
    padding: 1.5rem 2rem;
    gap: 0;
  }
}

.sf-footer__copyright {
  font-size: 0.8125rem;
  color: var(--text-muted);
}

.sf-footer__links {
  display: flex;
  align-items: center;
  gap: 1.25rem;
}

.sf-footer__link {
  font-size: 0.8125rem;
  color: var(--text-muted);
  transition: color 0.15s ease;
}

/* 悬浮颜色变化，浅色模式为深松石绿，暗色模式为亮松石绿 */
.sf-footer__link:hover {
  color: #0f766e;
}

.dark .sf-footer__link:hover {
  color: #2dd4bf;
}
</style>
