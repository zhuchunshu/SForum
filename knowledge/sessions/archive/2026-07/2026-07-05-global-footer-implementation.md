# 2026-07-05 Global Footer Implementation Handoff

## Changed

- Created a new flat, theme-adaptive global footer component **[SFFooter.vue](file:///Users/inkedus/Code/SForum/apps/web/app/components/SFFooter.vue)**. It features dynamic copyright information matching `siteName` and `currentYear`, along with dummy links for Terms of Service, Privacy Policy, and Guidelines.
- Added translation keys under the `"footer"` block in English **[en-US.json](file:///Users/inkedus/Code/SForum/apps/web/i18n/locales/en-US.json)** and Chinese **[zh-CN.json](file:///Users/inkedus/Code/SForum/apps/web/i18n/locales/zh-CN.json)** locales.
- Modified the default layout **[default.vue](file:///Users/inkedus/Code/SForum/apps/web/app/layouts/default.vue)** to integrate `<SFFooter />` using a CSS flexbox sticky footer layout (`flex flex-col min-h-screen` and `flex-1` wrapper).

## Decisions

- Selected **Option A (Single-line Minimalist)** for the global footer layout following design review.
- Kept footer links as dummy links (`#`) as the actual legal and guidelines pages are not yet defined.
- Implemented footer styling inside scoped CSS within the `SFFooter.vue` component, maintaining color adaptiveness for both Light (muted slate text, dark teal hover) and Dark modes (muted zinc text, light teal hover).

## Next

- Create pages and routing for the legal terms, privacy policy, and community guidelines when contents are ready.
- Link the footer links to these new pages.
