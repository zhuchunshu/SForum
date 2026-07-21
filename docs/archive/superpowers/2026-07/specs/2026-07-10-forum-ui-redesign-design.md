# SForum UI Redesign Spec (Option A - Linux.do Compact Style)

This document outlines the design and implementation details for redesigning the Home Page, Topic Details Page, and Comment Display elements of SForum to match a compact, data-dense, elegant layout similar to Linux.do and Discourse.

## Goals
1. **Compact & Clean Home Feed**: Shift the home page discussion list from simple rows to structured table rows containing participant avatar stacks, a dedicated activity column, and clean borders.
2. **Balanced Topic Layout**: Introduce a structured topic page layout with a right-hand summary stats sidebar (`side-panel`) and breadcrumbs.
3. **Refined Flat Comments**: Maintain a flat timeline layout for comments, but introduce inline quoting blockquotes representing replied-to comments, and style actions cleanly with subtle icons.

---

## Detailed Specifications

### 1. Home Page & Feed Rows
- **Target Files**:
  - `extensions/builtin/themes/sforum-default/layer/app/pages/index.vue`
  - `extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css`
- **Layout Details**:
  - Grid columns: `minmax(0, 1fr) 120px 80px`.
  - Left column: Thread title (bold, Inter font), followed by a category badge and tag chips. Creator name and simple metadata are rendered below or inline in a smaller muted font.
  - Middle column: A participant stack containing overlapping avatars of users who replied to the thread, plus a compact reply badge.
  - Right column: Clean relative last-activity string (e.g. `1m`, `2h`, `3d`).
- **Styling**:
  - Card borders are simplified to single-line bottom borders.
  - Soft grey background for the body canvas (`#f5f7fa` or dark theme equivalent).
  - Hover effects use soft background transitions.

### 2. Topic Details Page
- **Target Files**:
  - `extensions/builtin/themes/sforum-default/layer/app/pages/t/[...path].vue`
- **Layout Details**:
  - Three-section grid on large screens: Main prose panel, right-side status/sidebar panel, and timeline/navigation links.
  - Sidebar summary: A grid showing reply counts, view counts, participant counts, and latest activity time.

### 3. Comment Cards
- **Target Files**:
  - `apps/web/app/components/SFComment.vue`
- **Layout Details**:
  - Flat layout with a reply quote block. The quote block has a left accent border and displays the author name and the excerpt of the parent comment.
  - Compact action buttons with gray hover effects transitioning to theme color accents.

---

## Verification Plan
- Verify home page rendering under light/dark modes.
- Verify participant avatar stacks and reply badge styling.
- Verify comments and nested reply quotes.
- Verify side stats panel on the topic page.
