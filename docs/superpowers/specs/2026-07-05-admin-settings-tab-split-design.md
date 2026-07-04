# Admin Settings Tabs Split Design Spec

This specification outlines the refactoring of the backend site settings page by splitting the two settings tabs (Basic Settings and Verification Settings) into separate Vue component files under a dedicated directory.

## Goal

Currently, `apps/web/app/pages/admin/settings/index.vue` holds all the state, validation, API request logic, and layout markup for both the Basic Settings and Human Verification tabs. This makes the file large (460+ lines) and hard to maintain. We will extract the tab forms into dedicated component files under a new `admin/settings` component folder, while keeping the parent `index.vue` page as the state container and controller.

## Proposed Changes

### Component Location

A new folder `apps/web/app/components/admin/settings/` will be created to house the sub-components.

### 1. `BasicSettingsForm.vue`
[BasicSettingsForm.vue](file:///Users/inkedus/Code/SForum/apps/web/app/components/admin/settings/BasicSettingsForm.vue)

- **Props**:
  - `siteName`: `string`
  - `siteUrl`: `string`
  - `defaultLocale`: `string`
  - `supportedLocales`: `string[]`
  - `saving`: `boolean`
  - `hasChanges`: `boolean`
- **Emits**:
  - `update:siteName`
  - `update:siteUrl`
  - `update:defaultLocale`
  - `update:supportedLocales`
  - `submit`
  - `reset`

- **Responsibilities**:
  - Render the Basic Settings form fields (Site Name, Site URL, Default Locale, Supported Locales).
  - Handle locale checkbox changes and calculate the updated `supportedLocales` list and auto-adjust `defaultLocale` if needed, emitting updates to the parent.
  - Render the form actions via `<SFAdminFormFooter>` and emit `submit` and `reset` events.

### 2. `VerificationSettingsForm.vue`
[VerificationSettingsForm.vue](file:///Users/inkedus/Code/SForum/apps/web/app/components/admin/settings/VerificationSettingsForm.vue)

- **Props**:
  - `humanVerificationProvider`: `string`
  - `altchaSecret`: `string`
  - `altchaSecretSet`: `boolean`
  - `altchaChallengeTTL`: `string`
  - `altchaCost`: `number`
  - `saving`: `boolean`
  - `hasChanges`: `boolean`
- **Emits**:
  - `update:humanVerificationProvider`
  - `update:altchaSecret`
  - `update:altchaChallengeTTL`
  - `update:altchaCost`
  - `submit`
  - `reset`

- **Responsibilities**:
  - Render the Human Verification form fields (Verification Provider selection, ALTCHA secret key, ALTCHA Challenge TTL, ALTCHA Cost).
  - Render the configured/missing status badge based on `altchaSecretSet` prop.
  - Render the form actions via `<SFAdminFormFooter>` and emit `submit` and `reset` events.

### 3. `index.vue` (Parent Container)
[index.vue](file:///Users/inkedus/Code/SForum/apps/web/app/pages/admin/settings/index.vue)

- **Responsibilities**:
  - Keep the original `useAsyncData`, `useWebOptions`, toast alerts, and routing configuration.
  - Retain the centralized reactive `form` state, original settings tracking state (`adminOptionsMap`), and modification checks (`hasBasicChanges`, `hasVerificationChanges`).
  - Retain the saving methods (`saveBasicSettings`, `saveVerificationSettings`) and form reset methods (`resetBasicForm`, `resetVerificationForm`).
  - Import the new sub-components and bind their properties.

## Verification Plan

### Manual Verification
- Access the admin settings panel at `/admin/settings` (or the configured admin prefix).
- Verify both the "Basic Settings" (基础配置) and "Human Verification" (人机验证) tabs render correctly.
- Verify changing the fields activates the unsaved changes warning on the `<SFAdminFormFooter>`.
- Verify clicking "Reset" restores fields to their initial states.
- Verify saving modifications calls the API successfully, updates the page state, and triggers a success toast.
