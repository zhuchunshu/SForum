<script setup lang="ts">
import type { SiteFriendLink } from '~/composables/useSiteChromeApi'

const props = withDefaults(defineProps<{
  /** Core 404 应急页只使用本地 footer，不等待已失效的 API。 */
  fetchRemoteChrome?: boolean
}>(), {
  fetchRemoteChrome: true
})

const { locale } = useI18n()
const {
  siteName,
  footerCopyrightTemplate,
  footerLinks,
  footerLinkLabel
} = useWebOptions()
const chromeApi = useSiteChromeApi()

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

// 主题错误树不会等待嵌套异步 setup；footer 先同步挂载，友情链接抵达后再填充。
const emptyFriendLinks = () => [] as SiteFriendLink[]
const friendLinks = props.fetchRemoteChrome
  ? useAsyncData('site-public-friend-links', async () => {
      try {
        return await chromeApi.listPublicFriendLinks()
      } catch {
        return emptyFriendLinks()
      }
    }, { default: emptyFriendLinks }).data
  : shallowRef<SiteFriendLink[]>(emptyFriendLinks())

const visibleFriendLinks = computed(() => friendLinks.value || [])
</script>

<template>
  <footer class="sf-footer">
    <div class="sf-footer__inner">
      <div v-if="copyrightText" class="sf-footer__copyright">
        {{ copyrightText }}
      </div>
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

    <div v-if="visibleFriendLinks.length" class="sf-footer__friends">
      <div class="sf-footer__friends-inner">
        <a
          v-for="item in visibleFriendLinks"
          :key="item.id"
          :href="item.url"
          class="sf-footer__friend"
          target="_blank"
          rel="noopener noreferrer"
          :title="item.description || item.name"
        >
          <img
            v-if="item.logoUrl"
            :src="item.logoUrl"
            alt=""
            class="sf-footer__friend-logo"
          >
          <span>{{ item.name }}</span>
        </a>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.sf-footer {
  width: 100%;
  border-top: 1px solid var(--border-default);
  background: var(--sf-public-bg);
  transition: border-color 0.2s;
}

.sf-footer__inner {
  max-width: var(--sf-public-container);
  margin: 0 auto;
  padding: 1.5rem 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
}

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

.sf-footer__link:hover {
  color: var(--sf-accent);
}

.dark .sf-footer__link:hover {
  color: var(--sf-accent-dark);
}

.sf-footer__friends {
  border-top: 1px solid var(--border-default, #e4e8ef);
  padding: 0.75rem 1rem 1.25rem;
}

.sf-footer__friends-inner {
  max-width: var(--sf-public-container);
  margin: 0 auto;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 0.75rem 1.25rem;
}

.sf-footer__friend {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  color: var(--text-muted);
  text-decoration: none;
}

.sf-footer__friend:hover {
  color: var(--sf-accent);
}

.sf-footer__friend-logo {
  width: 16px;
  height: 16px;
  object-fit: contain;
  border-radius: 3px;
}
</style>
