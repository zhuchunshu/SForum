# Global Footer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a simple, theme-adaptive, non-card global footer (Option A) to public-facing forum pages.

**Architecture:** Create an auto-imported `SFFooter.vue` component, add corresponding localization keys in `zh-CN.json` and `en-US.json`, and integrate it into `layouts/default.vue` using a flexbox sticky footer pattern.

**Tech Stack:** Nuxt 4, Vue 3, Tailwind CSS v4, @nuxtjs/i18n.

---

### Task 1: Add i18n Translations

**Files:**
- Modify: `apps/web/i18n/locales/zh-CN.json`
- Modify: `apps/web/i18n/locales/en-US.json`

- [ ] **Step 1: Add translation keys in `zh-CN.json`**
  Modify `apps/web/i18n/locales/zh-CN.json` to append the `footer` keys.
  
  Existing code around line 206-208:
  ```json
      "rateLimited": "操作过于频繁，请稍后再试。"
    }
  }
  ```
  
  New code:
  ```json
      "rateLimited": "操作过于频繁，请稍后再试。"
    },
    "footer": {
      "copyright": "© {year} {siteName}。保留所有权利。",
      "terms": "服务条款",
      "privacy": "隐私政策",
      "guidelines": "社区指南"
    }
  }
  ```

- [ ] **Step 2: Add translation keys in `en-US.json`**
  Modify `apps/web/i18n/locales/en-US.json` to append the `footer` keys.
  
  Existing code around line 206-208:
  ```json
      "rateLimited": "Too many attempts. Please try again later."
    }
  }
  ```
  
  New code:
  ```json
      "rateLimited": "Too many attempts. Please try again later."
    },
    "footer": {
      "copyright": "© {year} {siteName}. All rights reserved.",
      "terms": "Terms of Service",
      "privacy": "Privacy Policy",
      "guidelines": "Guidelines"
    }
  }
  ```

---

### Task 2: Create SFFooter.vue Component

**Files:**
- Create: `apps/web/app/components/SFFooter.vue`

- [ ] **Step 1: Write `SFFooter.vue`**
  Create the component file at `apps/web/app/components/SFFooter.vue` with the following implementation:
  
  ```vue
  <script setup lang="ts">
  const { t } = useI18n()
  const { siteName } = useWebOptions()
  const currentYear = computed(() => new Date().getFullYear())
  </script>
  
  <template>
    <footer class="sf-footer">
      <div class="sf-footer__inner">
        <div class="sf-footer__copyright">
          {{ t('footer.copyright', { year: currentYear, siteName: siteName }) }}
        </div>
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
    color: #0f766e;
  }
  
  .dark .sf-footer__link:hover {
    color: #2dd4bf;
  }
  </style>
  ```

---

### Task 3: Integrate Footer into Default Layout

**Files:**
- Modify: `apps/web/app/layouts/default.vue`

- [ ] **Step 1: Modify default layout**
  Update the template of `apps/web/app/layouts/default.vue` to wrap inside a full-height flex column and append `<SFFooter />`.
  
  Existing code:
  ```vue
  <template>
    <div>
      <SFNavbar />
      <slot />
    </div>
  </template>
  ```
  
  New code:
  ```vue
  <template>
    <div class="flex flex-col min-h-screen">
      <SFNavbar />
      <div class="flex-1">
        <slot />
      </div>
      <SFFooter />
    </div>
  </template>
  ```

---

### Task 4: Verification and Run Tests

**Files:**
- None

- [ ] **Step 1: Run typechecks and test script**
  Run: `scripts/test.sh`
  Expected output:
  - Running Go tests... PASS
  - Running Nuxt typecheck... PASS
  - Running admin framework validation... PASS
  - Running identity UI validation... PASS
  - Running SF component library validation... PASS
