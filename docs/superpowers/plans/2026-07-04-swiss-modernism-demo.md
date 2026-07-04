# Swiss Modernism Demo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a mathematically rigorous Swiss Modernism 2.0 styled UI components demo file `forum-components-swiss.html` based on `forum-components.html`.

**Architecture:** Copy `forum-components.html` to `forum-components-swiss.html`, then systematically transform its configuration, custom styles, HTML classes, layouts, SVGs, and scripts to match the Swiss design constraints (no shadows, no gradients, no rounded corners, high-contrast black/white colors, square corners).

**Tech Stack:** Tailwind CSS, HTML5, Vanilla JavaScript.

---

### Task 1: Create File and Setup Theme Configuration

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-swiss.html`

- [ ] **Step 1: Copy base file and update metadata**
  Copy the contents of `apps/web/app/assets/demos/forum-components.html` into `apps/web/app/assets/demos/forum-components-swiss.html`, and update the `<title>` to "SForum UI 组件 Demo — Swiss Modernism".

- [ ] **Step 2: Update Tailwind Color Palette**
  Modify the tailwind configuration object inside the script block to define high-contrast Swiss Modernism colors:
  ```javascript
  tailwind.config = { theme: { extend: { colors: {
    accent: '#000000', 'accent-light': '#1A1A1A', 'accent-soft': '#F5F5F7',
    surface: '#FFFFFF', card: '#FFFFFF', muted: '#F5F5F7',
    fg: '#000000', 'fg-secondary': '#1A1A1A', 'fg-tertiary': '#7F7F7F',
    border: '#000000', 'border-light': '#E5E5E7'
  }}}}
  ```

- [ ] **Step 3: Update CSS Styles in `<style>` block**
  Replace standard style rules for cards, badges, buttons, and prose elements with Swiss Modernism rules:
  - `.section-badge` -> `border-radius: 0; background: #000000; color: #FFFFFF;`
  - `.card` -> `border-radius: 0; border: 1px solid #000000; box-shadow: none;`
  - `.card:hover` -> `box-shadow: none; transform: none;`
  - `.gradient-btn` -> `background: #000000; color: #ffffff; border: 1px solid #000000; border-radius: 0;`
  - `.gradient-btn:hover` -> `background: #1A1A1A;`
  - `.ghost-btn` -> `background: #ffffff; color: #000000; border: 1px solid #000000; border-radius: 0;`
  - `.ghost-btn:hover` -> `background: #F5F5F7;`
  - `.pill-tag` -> `border-radius: 0; border: 1px solid #000000; background: #ffffff; color: #000000;`
  - `.glass-nav` -> `background: #ffffff; border-bottom: 1px solid #000000; backdrop-filter: none;`
  - Prose heading colors -> change text to `#000000`
  - Prose code/pre -> change background to `#F5F5F7`, text to `#000000`, border to `1px solid #000000`, remove rounded corners.
  - Prose blockquote -> change border to `4px solid #000000`, background to `#F5F5F7`, remove rounded corners.

---

### Task 2: Transform Components and Remove Rounded Corners / Shadows / Gradients

**Files:**
- Modify: `apps/web/app/assets/demos/forum-components-swiss.html`

- [ ] **Step 1: Replace rounded corners and shadows**
  Perform global regex/string replacements to convert rounded classes and shadows:
  - Remove all shadows (`shadow-sm`, `shadow-md`, `shadow-lg`, `shadow-xl`, `shadow-2xl`, `shadow`, `shadow-accent/25`)
  - Replace all `rounded-lg`, `rounded-xl`, `rounded-2xl`, `rounded-3xl`, `rounded-full` with `rounded-none`

- [ ] **Step 2: Replace gradients and colors in HTML body**
  - Convert avatar gradients (`bg-gradient-to-br ...`) to solid black (`bg-black text-white border border-black`) or solid white/gray with borders.
  - Convert cover horizontal banner background (`bg-gradient-to-r ...`) to solid black (`bg-black`).
  - Convert any colored tags (e.g. `bg-emerald-50 text-emerald-600`, `bg-purple-50 text-purple-600`) to Swiss-style tags (`bg-neutral-200 text-black border-black` or similar).

- [ ] **Step 3: Transform SVGs in Dashboard Analytics**
  - Convert Card 1 sparkline path stroke to `#000000` and fill to `#F5F5F7`.
  - Convert Card 2 bar chart columns to square `rx="0"`, solid black `fill="#000000"` and gray `fill="#E5E5E7"`.
  - Convert Card 3 line chart path stroke to `#000000` and circles to `#000000`.

- [ ] **Step 4: Update Interactive Scripts Classes**
  - Adjust JavaScript event handlers to toggle `bg-black`, `text-white`, `border-black` instead of rose/amber gradients for liked, bookmarked, followed, and voted states.

---

### Task 3: Validation and Integration

**Files:**
- Modify: `apps/web/app/assets/demos/forum-components-swiss.html`

- [ ] **Step 1: Run validation tests**
  Run `node tests/validate-demos.js` to ensure the file is valid and contains all 14 required sections.

- [ ] **Step 2: Commit changes to Git**
  Add the new files and commit with a clean git commit message.
