# Clean Modern Light Forum UI Demo Design Specification

## Overview

This specification details the design for a clean, modern, standalone HTML/CSS/JS frontend demo for a forum application (homepage, post detail page, and multi-level comment section). The demo is built for fast browser visualization without any build dependencies or external CDNs, stored in `tmp/demos/clean-light-forum/`.

## Architectural Goals

1. **Standalone & Zero Build Step**: Raw HTML5, CSS3 (using CSS custom properties), and vanilla JavaScript (ES6+). Can be double-clicked directly in any browser offline.
2. **Design Language**: Clean Modern Light Theme with slate-tinted background (`#F8FAFC`), crisp white cards (`#FFFFFF`), subtle slate borders (`#E2E8F0`), high-contrast text (`#0F172A`), and vibrant brand accent (`#2563EB`).
3. **App Layout**: 3-Column App-Like Layout (Left Sidebar: App Navigation & Categories; Center: Feed/Article & Comments; Right Sidebar: Announcements, Trending Topics, Community Stats).
4. **Interactive Fidelity**: Real DOM manipulation for upvote/downvote counter toggling, bookmarking, new post modal, and nested comment replies.

## Directory & File Structure

Target Location: `tmp/demos/clean-light-forum/`

```
tmp/demos/clean-light-forum/
├── index.html         # Forum Homepage (3-column layout, post feed, filters, modal)
├── post.html          # Post Detail Page (3-column layout, rich body, nested comment tree)
├── style.css          # Design tokens, CSS grid/flexbox layout, typography, components, animations
├── app.js             # Vanilla JS for interactive features (upvotes, modal, nested replies)
└── README.md          # Project documentation and guide on how to open and view the demo
```

## Page Component Specifications

### 1. Unified CSS Design System (`style.css`)
- **CSS Custom Properties (Variables)**:
  - Colors: `--bg-body: #f8fafc`, `--bg-surface: #ffffff`, `--bg-subtle: #f1f5f9`, `--border-color: #e2e8f0`, `--text-primary: #0f172a`, `--text-secondary: #475569`, `--text-muted: #94a3b8`, `--accent: #2563EB`, `--accent-hover: #1d4ed8`, `--accent-light: #eff6ff`, `--danger: #ef4444`, `--success: #10b981`.
  - Shadows: `--shadow-sm: 0 1px 2px 0 rgba(0,0,0,0.05)`, `--shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -2px rgba(0,0,0,0.1)`, `--shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1)`.
  - Border Radius: `--radius-sm: 6px`, `--radius-md: 10px`, `--radius-lg: 16px`, `--radius-full: 9999px`.
- **Layout System**: CSS Grid with `grid-template-columns: 240px 1fr 300px` for desktop screens (> 1024px); responsive stack for tablet/mobile.
- **Micro-Animations**: Hover elevations, button scale transitions (`transform 0.15s ease`), smooth modal backdrop blur (`backdrop-filter: blur(4px)`).

### 2. Forum Homepage (`index.html`)
- **Top Header**: Logo + Brand Name ("SForum Showcase"), Search bar with shortcut indicator (`/`), Notifications icon, User Avatar & Profile menu button, "New Post" primary CTA button.
- **Left Navigation Rail**:
  - Main Navigation: Home, Trending, Following, Bookmarks.
  - Categories: General, Tech & Dev, Showcase, Feedback, Q&A.
  - Footer Links: Help, Dark Mode Toggle mockup, Version tag.
- **Central Feed**:
  - Filter Bar: Category Pills (All, Tech, Discussion, Q&A, Show & Tell) + Sort Tabs (Latest, Top, Unanswered).
  - Feed Cards: Card header with Author Avatar, Badge, Timestamp, Post Title, Body snippet, Category Pill, Action stats (Upvotes, Comments count, View count).
- **Right Sidebar**:
  - Community Announcement Widget.
  - Hot Topics / Trending Tags Widget (e.g. `#Go1.25`, `#Nuxt4`, `#PluginArchitecture`).
  - Active Community Stats Widget (Total Threads, Members online, Daily Posts).
- **New Post Modal**:
  - Interactive overlay with Title input, Category dropdown, Body textarea, Tags input, Publish button.

### 3. Post Detail & Comment Section (`post.html`)
- **Breadcrumb Nav**: `Home / Tech & Dev / Post Detail`
- **Post Header**: Large Post Title, Author info row (Avatar, Username, Role badge "Core Maintainer", Post time, Reading time estimate).
- **Post Body**: Rich text rendering with styled H2/H3 headings, paragraphs, formatted inline code & highlighted code blocks, blockquotes, callout alert box, list items, and image preview container.
- **Action Bar**: Upvote pill counter, Downvote button, Bookmark/Favorite, Share link, Jump to Comments.
- **Multi-Level Comment Tree**:
  - Comment Count Header & Sort selector (Newest, Most Upvoted).
  - Main Comment Composer: Avatar, Textarea, Action toolbar (Formatting icons, Submit Comment button).
  - Nested Comments Structure:
    - Level 1 Comment: Avatar, Username, Badge, Time, Comment Content, Upvote count, Reply button.
    - Level 2 Nested Comment (Indented with left accent border): Target user tag ("@Author"), Reply Content, Upvote, Reply.
    - Interactive Reply Box: Triggering "Reply" inserts an inline textarea under the target comment with "Submit Reply" button.

### 4. Client-side Interaction (`app.js`)
- **Stateful Counter Controls**: Event listeners on `.btn-vote` to toggle active upvote/downvote states and update count text dynamically.
- **Bookmark Toggle**: Toggles icon fill and active state.
- **Modal Manager**: Open modal on "New Post" click, close modal on overlay or close-icon click, ESC key listener.
- **Comment Manager**:
  - Handles primary comment submission (prepends a new comment DOM element to the comment list).
  - Handles nested reply trigger (toggles inline reply form underneath specific comment node).
  - Handles nested reply submission (appends nested child comment DOM node).

## Verification Plan

### Automated Checks
- Validate HTML structure with standard HTML validator tool/script if needed.
- Inspect CSS syntax and verify JS error-free execution via browser console log checks.

### Manual Verification
- Open `index.html` and `post.html` in browser.
- Test responsive layout scaling.
- Click Upvote/Downvote, New Post modal, and Nested Reply buttons to verify interactive feedback.
