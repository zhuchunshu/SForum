# Task 5: Neobrutalism Theme Demo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create `forum-components-neobrutalism.html` by adapting `forum-components.html` to a Neobrutalist aesthetic, ensuring all 14 sections are present and pass validation tests.

**Architecture:** Use custom CSS styling within the `<style>` tag for base components (cards, buttons, tags) and Tailwind CSS classes for layout and structural composition. The background will use a retro paper tone, cards will have thick black borders and flat black shadows, and accent actions will use high-contrast bright yellow and pink colors.

**Tech Stack:** HTML5, Tailwind CSS CDN, Native JavaScript.

---

### Task 1: Create and Configure Base File

**Files:**
- Create: `apps/web/app/assets/demos/forum-components-neobrutalism.html`

- [ ] **Step 1: Copy base file content and update metadata**
  Copy contents of `forum-components.html` to `forum-components-neobrutalism.html`, change document title to "SForum UI 组件 Demo — Neobrutalism".

- [ ] **Step 2: Update Tailwind configuration**
  Modify `tailwind.config` to define Neobrutalist accents:
  - `accent`: `#FFDE47` (Bright Yellow)
  - `accent-light`: `#22C55E` (Neon Green)
  - `accent-soft`: `#FAF8F5` (Saturated Paper Backing)
  - `surface`: `#FAF8F5`
  - `card`: `#FFFFFF`
  - `muted`: `#EAE6DF`
  - `fg`: `#000000` (Pure Black)
  - `fg-secondary`: `#1C1917` (Stone Black)
  - `fg-tertiary`: `#4B5563`
  - `border`: `#000000` (Thick Black Border)
  - `border-light`: `#000000`

- [ ] **Step 3: Define Custom CSS classes in `<style>` tag**
  Set body background to `#FAF8F5`.
  Define:
  - `.card`: `background: #FFFFFF; border: 4px solid #000000; border-radius: 8px; box-shadow: 4px 4px 0px 0px #000000; padding: 1.5rem; transition: all 0.2s ease-in-out;`
  - `.card:hover`: `transform: translate(-2px, -2px); box-shadow: 6px 6px 0px 0px #000000;`
  - `.gradient-btn`: `background: #FFDE47; color: #000000; border: 2px solid #000000; font-weight: 800; border-radius: 8px; padding: 0.5rem 1rem; transition: all 0.15s ease-in-out;`
  - `.gradient-btn:hover`: `transform: translate(-2px, -2px); box-shadow: 4px 4px 0px 0px #000000;`
  - `.gradient-btn:active`: `transform: translate(0px, 0px); box-shadow: none;`
  - `.ghost-btn`: `background: #FFFFFF; color: #000000; border: 2px solid #000000; font-weight: 700; border-radius: 8px; padding: 0.5rem 1rem; transition: all 0.15s ease-in-out;`
  - `.ghost-btn:hover`: `background: #F3F4F6; transform: translate(-2px, -2px); box-shadow: 4px 4px 0px 0px #000000;`
  - `.ghost-btn:active`: `transform: translate(0px, 0px); box-shadow: none;`
  - `.pill-tag`: `display: inline-flex; align-items: center; border: 2px solid #000000; font-size: 0.75rem; font-weight: 900; color: #000000; text-transform: uppercase; letter-spacing: 0.05em; border-radius: 6px; background: #FFDE47; padding: 0.125rem 0.5rem;`
  - `.section-badge`: `display: inline-flex; align-items: center; height: 24px; padding: 0 10px; border-radius: 6px; border: 2px solid #000000; background: #EC4899; color: #000000; font-size: 0.7rem; font-weight: 800; letter-spacing: 0.05em; text-transform: uppercase;`
  - `.prose blockquote`: `border: 2px solid #000000; border-left-width: 6px; padding: 0.75rem 1rem; color: #1C1917; font-style: italic; background: #EAE6DF; border-radius: 6px; box-shadow: 2px 2px 0px 0px #000000;`
  - `.prose pre`: `background: #FAF8F5; color: #000000; border: 2px solid #000000; border-radius: 6px; box-shadow: 2px 2px 0px 0px #000000; padding: 1rem;`
  - `.prose code`: `background: #EAE6DF; color: #000000; padding: 0.125rem 0.25rem; border: 1px solid #000000; border-radius: 4px;`

- [ ] **Step 4: Commit intermediate configurations**
  Add the initial file and commit.

---

### Task 2: Systematically Adapt HTML Component Structures

**Files:**
- Modify: `apps/web/app/assets/demos/forum-components-neobrutalism.html`

- [ ] **Step 1: Adapt Top Navbar & Main Structure**
  Replace soft gradients, rings, and rounded corners with thick borders (`border-2 border-black` or `border-4`), flat background colors, and simple flat shapes. Make sure the navbar style has a solid bottom border.
  Update the user avatar icon (`U`) to have `bg-[#FFDE47] text-black border-2 border-black rounded-lg`.

- [ ] **Step 2: Adapt Sections 1-9 (Core Components)**
  Go through sections:
  1. `nav`: 1.1 Header nav styles, 1.2 Sidebar styles, 1.3 Breadcrumb, 1.4 Pagination. Change active items to have `border-2 border-black bg-[#FFDE47]` instead of smooth gradients.
  2. `avatars`: Update sizes and styling. Make avatars have `border-2 border-black`.
  3. `feed`: Update list items, cards, badges.
  4. `forms`: Update input fields, select menus, toggle switches to have `border-2 border-black rounded-md bg-white text-black shadow-[2px_2px_0px_0px_#000000] focus:ring-0 focus:border-black`.
  5. `comments`: Update comments thread layout, hierarchy markers, reply inputs.
  6. `search`: Update search bars, tag pills.
  7. `profile`: Update profile headers, stats blocks.
  8. `interactions`: Update tab menus, modals, tooltips.
  9. `lists`: Update member list cards, thread ranking badges.

- [ ] **Step 3: Adapt Sections 10-14 (New Components)**
  Go through sections:
  10. `editor`: Markdown editor panel. Update editor controls, input textarea (`border-2 border-black`), and live preview container (`border-2 border-black bg-white`).
  11. `analytics`: Statistics blocks. Use different high-contrast colors (e.g., green for positive indicators, pink for warning, yellow for standard stats) with bold black fonts.
  12. `badges`: Achievement grids. Set badges with thick borders and flat icons.
  13. `poll`: Interactive Poll. Update progress bars (`border-2 border-black bg-white`, inside filled with `#FFDE47`).
  14. `thread`: Thread detail page layout. Match prose styles to Neobrutalist custom prose.

- [ ] **Step 4: Update SVGs across all components**
  Convert all SVG path elements to have `stroke="#000000"` or `fill="#000000"` (or color accent fills) and `stroke-width="2.5"` or `stroke-width="3"`.

- [ ] **Step 5: Commit intermediate styling**
  Add modified styles to Git.

---

### Task 3: Verify and Adapt Interactions in JavaScript

**Files:**
- Modify: `apps/web/app/assets/demos/forum-components-neobrutalism.html`

- [ ] **Step 1: Check and adapt Interactive Poll JS state classes**
  Update Javascript code for Poll Logic so the active classes added dynamically represent Neobrutalist elements (e.g., `border-black`, `shadow-[2px_2px_0px_0px_#000000]`, `bg-[#FFDE47]`).
  Verify that class manipulation like `btn.classList.add(...)` matches.

- [ ] **Step 2: Check and adapt Thread Like/Bookmark JS state classes**
  Ensure the toggled classes for "Like" (`bg-[#EC4899] text-black border-2 border-black`) and "Bookmark" (`bg-[#FFDE47] text-black border-2 border-black`) use solid Neobrutalist theme colors rather than soft tailwind defaults.

---

### Task 4: Validation and Verification

**Files:**
- None

- [ ] **Step 1: Run Validate Script**
  Run `node tests/validate-demos.js` and verify it produces a successful exit status.

- [ ] **Step 2: Commit and Report**
  Commit the final changes to Git, capture the commit hash, and send the final report back to the caller.
