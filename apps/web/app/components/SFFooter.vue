<script setup lang="ts">
const { locale } = useI18n()
const {
  siteName,
  footerCopyrightTemplate,
  footerLinks,
  footerLinkLabel
} = useWebOptions()

// 动态计算当前年份，以保证版权年份的正确性
const currentYear = computed(() => new Date().getFullYear())
const copyrightText = computed(() => {
  return footerCopyrightTemplate(locale.value)
    .replace(/\{year\}/g, String(currentYear.value))
    .replace(/\{siteName\}/g, siteName.value)
})
const visibleLinks = computed(() => {
  return footerLinks.value
    .filter((link) => link.url.trim() !== '')
    .map((link) => ({
      key: link.key,
      label: footerLinkLabel(link, locale.value),
      url: link.url
    }))
})
</script>

<template>
  <footer class="sf-footer">
    <div class="sf-footer__inner">
      <!-- 版权信息 -->
      <div v-if="copyrightText" class="sf-footer__copyright">
        {{ copyrightText }}
      </div>
      <!-- 辅助虚拟链接 -->
      <div v-if="visibleLinks.length" class="sf-footer__links">
        <a
          v-for="link in visibleLinks"
          :key="link.key"
          :href="link.url"
          class="sf-footer__link"
        >
          {{ link.label }}
        </a>
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
  color: var(--sf-accent);
}

.dark .sf-footer__link:hover {
  color: var(--sf-accent-dark);
}
</style>
