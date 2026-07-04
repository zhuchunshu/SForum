# Global Footer Design Spec

This spec outlines the design and integration of the global footer (`SFFooter`) component for the SForum project.

## Purpose

Add a simple, theme-adaptive, non-card global footer to public-facing forum pages to show copyright info and utility dummy links (Terms of Service, Privacy Policy, Guidelines), supporting both Light and Dark modes.

## Proposed Changes

### Component Design

#### [NEW] [SFFooter.vue](file:///Users/inkedus/Code/SForum/apps/web/app/components/SFFooter.vue)
A single-line, flat footer component.
* **Content**:
  * Left: `© {{ currentYear }} {{ siteName }}. All rights reserved.` / `© {{ currentYear }} {{ siteName }}. 保留所有权利。`
  * Right: Dummy links to `Terms`, `Privacy`, and `Guidelines` mapping to `#` for now.
* **Styling**:
  * Flat top border: `border-t border-slate-200 dark:border-zinc-800`
  * Background: Transparent or matched to the page background (no card structure).
  * Spacing: `py-6 px-4 md:px-8 mt-auto`
  * Text Colors: Muted Slate (`slate-500`) in light mode, muted Zinc (`zinc-400`) in dark mode.
  * Responsiveness: Flex layout. Stacks vertically and centers on mobile; horizontal row space-between on desktop.

### Layout Integration

#### [MODIFY] [default.vue](file:///Users/inkedus/Code/SForum/apps/web/app/layouts/default.vue)
Add sticky footer wrapping structure:
* Use `flex flex-col min-h-screen` on the outer wrapper.
* Wrap `<slot />` with class `flex-1` to expand and push the footer to the bottom.
* Render `<SFFooter />` below the slot wrapper.

### Localizations

#### [MODIFY] [zh-CN.json](file:///Users/inkedus/Code/SForum/apps/web/i18n/locales/zh-CN.json)
Add the following translations:
```json
  "footer": {
    "copyright": "© {year} {siteName}。保留所有权利。",
    "terms": "服务条款",
    "privacy": "隐私政策",
    "guidelines": "社区指南"
  }
```

#### [MODIFY] [en-US.json](file:///Users/inkedus/Code/SForum/apps/web/i18n/locales/en-US.json)
Add the following translations:
```json
  "footer": {
    "copyright": "© {year} {siteName}. All rights reserved.",
    "terms": "Terms of Service",
    "privacy": "Privacy Policy",
    "guidelines": "Guidelines"
  }
```

## Verification Plan

* Verify that the footer is rendered at the bottom of public pages.
* Verify responsiveness on mobile screen resolutions.
* Verify text and link colors match the Light and Dark theme modes correctly.
* Verify language switching works correctly.
