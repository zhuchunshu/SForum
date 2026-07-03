# Product Notes

## Goal

Define the forum product clearly before implementation starts.

## Draft Areas

- User roles and permissions.
- Topic/category model.
- Posting, replying, editing, and moderation workflows.
- Search and discovery.
- Notifications.
- Administration.

## Localization

- Product features must support multiple languages from the first
  implementation.
- The default language is Simplified Chinese (`zh-CN`).
- English (`en-US`) is the first secondary language.
- Public Simplified Chinese pages should use default unprefixed URLs; English
  pages should use `/en/*`.
- User interface text, validation messages, emails, notifications, moderation
  labels, and seed/admin labels must be localizable.
- User-generated forum posts are stored as authored and are not automatically
  translated by default.
