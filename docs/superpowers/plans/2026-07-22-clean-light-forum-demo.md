# Clean Light Forum Demo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a complete, standalone, responsive forum UI demo (Homepage, Post Detail, Multi-level Comment section, and interaction JS) under `tmp/demos/clean-light-forum/`.

**Architecture:** A pure static HTML5 + CSS3 + ES6 JavaScript bundle. Uses CSS Custom Properties for design system tokens, 3-column responsive layout grids, inline SVG Tabler icons, and vanilla JS DOM handling for upvoting, modal management, and dynamic nested replies.

**Tech Stack:** HTML5, Vanilla CSS3 (CSS Variables, Flexbox, Grid), Vanilla JavaScript (ES6 Modules/Scripts, DOM manipulation), zero external dependencies.

---

### Task 1: Create Design System and Layout Stylesheet

**Files:**
- Create: `tmp/demos/clean-light-forum/style.css`

- [ ] **Step 1: Write `style.css` with CSS Custom Properties and Reset**

Create `tmp/demos/clean-light-forum/style.css` with color variables (`--bg-body: #f8fafc`, `--bg-surface: #ffffff`, `--border-color: #e2e8f0`, `--text-primary: #0f172a`, `--accent: #2563eb`), base reset, typography settings, button styles, card containers, badge tags, modal overlay styles, and 3-column layout classes.

- [ ] **Step 2: Commit `style.css`**

```bash
git add tmp/demos/clean-light-forum/style.css
git commit -m "feat(demo): add style.css design system for clean light forum"
```

---

### Task 2: Create Vanilla JS Interactive Module

**Files:**
- Create: `tmp/demos/clean-light-forum/app.js`

- [ ] **Step 1: Write `app.js` with DOM handlers**

Create `tmp/demos/clean-light-forum/app.js` with functions for:
1. `setupVoteButtons()`: Handles `.btn-vote` upvote/downvote active toggles & text counter increments.
2. `setupBookmarkButtons()`: Handles `.btn-bookmark` active state toggle.
3. `setupModalManager()`: Modal open/close listeners for the `#newPostModal`.
4. `setupCommentSystem()`: Primary comment submission appending to `#commentList`, nested reply form toggling (`.btn-reply`), and nested comment submission appending to child container.
5. DOMContentLoaded listener initializing all features.

- [ ] **Step 2: Commit `app.js`**

```bash
git add tmp/demos/clean-light-forum/app.js
git commit -m "feat(demo): add app.js interactive module for clean light forum"
```

---

### Task 3: Build Forum Homepage

**Files:**
- Create: `tmp/demos/clean-light-forum/index.html`

- [ ] **Step 1: Write `index.html` with Header, 3-Column Layout, and New Post Modal**

Create `tmp/demos/clean-light-forum/index.html` including:
- Navbar: Brand logo, search input, notification bell, user avatar, "New Post" button.
- Left Sidebar: Main navigation links (Home, Trending, Following), Categories (General, Tech & Dev, Showcase, Feedback, Q&A).
- Central Main Feed: Category pills filter bar, sorting tabs (Latest, Top, Unanswered), 4 sample post cards with realistic tech forum content (Go, Nuxt, Plugin Architecture), tags, and metadata.
- Right Sidebar: Announcements box, Trending hashtags box, Forum statistics box.
- New Post Modal element with title input, category selector, textarea, and publish button.

- [ ] **Step 2: Commit `index.html`**

```bash
git add tmp/demos/clean-light-forum/index.html
git commit -m "feat(demo): add index.html forum homepage for clean light forum"
```

---

### Task 4: Build Post Detail Page with Multi-Level Comment Tree

**Files:**
- Create: `tmp/demos/clean-light-forum/post.html`

- [ ] **Step 1: Write `post.html` with Article Detail and Multi-Level Nested Comments**

Create `tmp/demos/clean-light-forum/post.html` including:
- Breadcrumb navigation (`Home / Tech & Dev / Article`).
- Article Header: Title, Author avatar & badge, timestamp, reading time tag.
- Article Content Body: Formatted headings, prose, styled code blocks with syntax styling preview, quote callouts, and image attachment preview.
- Floating/Bottom Action Bar: Upvote button with count, Downvote button, Bookmark button, Share button, Jump to comments button.
- Comment Section:
  - Total comments count header & sort dropdown.
  - Primary comment editor with formatting bar and submit button.
  - Nested Comment Tree: Level 1 comments with author metadata, level 2 indented replies with `@author` badges, upvote buttons, and interactive reply triggers.

- [ ] **Step 2: Commit `post.html`**

```bash
git add tmp/demos/clean-light-forum/post.html
git commit -m "feat(demo): add post.html detail and comment tree for clean light forum"
```

---

### Task 5: Documentation and Verification

**Files:**
- Create: `tmp/demos/clean-light-forum/README.md`

- [ ] **Step 1: Write `README.md` with usage instructions**

Create `tmp/demos/clean-light-forum/README.md` describing the demo features, structure, browser usage, and design guidelines.

- [ ] **Step 2: Commit `README.md`**

```bash
git add tmp/demos/clean-light-forum/README.md
git commit -m "docs(demo): add README.md for clean light forum demo"
```
