const fs = require('fs');
const path = require('path');

const root = process.cwd();
const registerPagePath = 'extensions/builtin/themes/sforum-default/layer/app/pages/register.vue';
const loginPagePath = 'extensions/builtin/themes/sforum-default/layer/app/pages/login.vue';
const requiredFiles = [
  'apps/web/app/composables/useAuthSession.ts',
  'apps/web/app/middleware/admin.ts',
  registerPagePath,
  loginPagePath,
  'apps/web/app/pages/admin/index.vue',
  'apps/web/app/pages/admin/roles.vue',
  'apps/web/app/pages/admin/users.vue',
  'apps/web/app/pages/admin/permissions.vue'
];

for (const file of requiredFiles) {
  if (!fs.existsSync(path.resolve(root, file))) {
    throw new Error(`Missing required identity UI file: ${file}`);
  }
}

const zh = JSON.parse(fs.readFileSync(path.resolve(root, 'apps/web/i18n/locales/zh-CN.json'), 'utf8'));
const en = JSON.parse(fs.readFileSync(path.resolve(root, 'apps/web/i18n/locales/en-US.json'), 'utf8'));
const registerPage = fs.readFileSync(path.resolve(root, registerPagePath), 'utf8');
const loginPage = fs.readFileSync(path.resolve(root, loginPagePath), 'utf8');

const requiredKeys = [
  ['auth', 'registerTitle'],
  ['auth', 'loginTitle'],
  ['admin', 'home', 'title'],
  ['admin', 'roles', 'title'],
  ['admin', 'roles', 'permissionEditor'],
  ['admin', 'users', 'permissionSection'],
  ['admin', 'users', 'overrideMode', 'deny'],
  ['admin', 'permissions', 'matrix'],
  ['admin', 'permissions', 'comparisonScope'],
  ['admin', 'permissions', 'roleSearchPlaceholder'],
  ['admin', 'permissions', 'onlyDifferences'],
  ['admin', 'permissions', 'clearFilters'],
  ['admin', 'permissions', 'noDifferences'],
  ['admin', 'permissionModules', 'identity'],
  ['admin', 'permissionCatalog', 'admin', 'access', 'label'],
  ['admin', 'permissionCatalog', 'role', 'manage', 'description'],
  ['admin', 'permissionCatalog', 'post', 'delete_any', 'label'],
  ['errors', 'permissionDenied']
];

function valueAt(object, keyPath) {
  return keyPath.reduce((current, key) => current?.[key], object);
}

const adminUsersPage = fs.readFileSync(path.resolve(root, 'apps/web/app/pages/admin/users.vue'), 'utf8');
const adminRolesPage = fs.readFileSync(path.resolve(root, 'apps/web/app/pages/admin/roles.vue'), 'utf8');
const adminPermissionsPage = fs.readFileSync(path.resolve(root, 'apps/web/app/pages/admin/permissions.vue'), 'utf8');

for (const [name, content] of [
  ['admin/users.vue', adminUsersPage],
  ['admin/roles.vue', adminRolesPage],
  ['admin/permissions.vue', adminPermissionsPage]
]) {
  if (!content.includes("layout: 'admin'")) {
    throw new Error(`${name} should use the admin layout`);
  }
  if (!content.includes('UDashboardToolbar')) {
    throw new Error(`${name} should render a dashboard toolbar`);
  }
  if (!content.includes('i-lucide-')) {
    throw new Error(`${name} should use Nuxt Icon lucide icons`);
  }
}

for (const [name, content] of [
  ['admin/users.vue', adminUsersPage],
  ['admin/roles.vue', adminRolesPage],
  ['admin/permissions.vue', adminPermissionsPage]
]) {
  if (!content.includes('usePermissionText')) {
    throw new Error(`${name} should localize permission labels and descriptions through usePermissionText`);
  }
  if (content.includes('permission.description ||')) {
    throw new Error(`${name} should not render raw backend permission descriptions directly`);
  }
}

if (!adminUsersPage.includes('/permission-overrides')) {
  throw new Error('Admin users page should manage per-user permission overrides');
}
if (!adminRolesPage.includes('/permissions')) {
  throw new Error('Admin roles page should manage role permissions');
}
if (!adminRolesPage.includes(':label="t(\'admin.roles.key\')"') || !adminRolesPage.includes('name="role-key"')) {
  throw new Error('Admin roles page should show a visible label for the role key field');
}
if (!adminRolesPage.includes(':label="t(\'admin.roles.alias\')"') || !adminRolesPage.includes('name="role-alias"')) {
  throw new Error('Admin roles page should show a visible label for the role alias field');
}
if (!adminRolesPage.includes('validateRoleForm')) {
  throw new Error('Admin roles page should validate required role fields before saving');
}
if (!adminPermissionsPage.includes('/permissions/matrix')) {
  throw new Error('Admin permissions page should load the permission matrix');
}
if (!adminPermissionsPage.includes('const ROLE_COMPARE_LIMIT = 5')) {
  throw new Error('Admin permissions page should cap the default role comparison width');
}
if (!adminPermissionsPage.includes('const roleSearch = ref')) {
  throw new Error('Admin permissions page should let admins search user groups before comparing permissions');
}
if (!adminPermissionsPage.includes('selectedRoleKeys')) {
  throw new Error('Admin permissions page should support an explicit user-group comparison selection');
}
if (!adminPermissionsPage.includes('showOnlyDifferences')) {
  throw new Error('Admin permissions page should support a differences-only permission audit mode');
}
if (!adminPermissionsPage.includes('visibleRoles')) {
  throw new Error('Admin permissions page should render the matrix against the filtered comparison role set');
}
if (!adminPermissionsPage.includes('filteredPermissionGroups')) {
  throw new Error('Admin permissions page should filter permission rows when differences-only mode is enabled');
}
if (adminPermissionsPage.includes('v-for="role in roles"')) {
  throw new Error('Admin permissions page should not render every user group as a matrix column by default');
}

for (const keyPath of requiredKeys) {
  if (!valueAt(zh, keyPath)) {
    throw new Error(`Missing zh-CN locale key: ${keyPath.join('.')}`);
  }
  if (!valueAt(en, keyPath)) {
    throw new Error(`Missing en-US locale key: ${keyPath.join('.')}`);
  }
}

if (!registerPage.includes(':configuration=')) {
  throw new Error('Registration ALTCHA widget should pass a configuration object');
}
if (!registerPage.includes('hideLogo: altchaWidgetSettings.value.hideLogo')) {
  throw new Error('Registration ALTCHA widget should read logo visibility from runtime settings');
}
if (!registerPage.includes('hideFooter: altchaWidgetSettings.value.hideFooter')) {
  throw new Error('Registration ALTCHA widget should read footer visibility from runtime settings');
}
if (!registerPage.includes('minDuration: altchaWidgetSettings.value.minDuration')) {
  throw new Error('Registration ALTCHA widget should read minimum duration from runtime settings');
}
if (!registerPage.includes("import type { AltchaWidgetElement } from 'altcha'")) {
  throw new Error('Registration ALTCHA widget should use the official AltchaWidgetElement type');
}
if (!registerPage.includes('const altchaWidget = ref<AltchaWidgetElement | null>(null)')) {
  throw new Error('Registration ALTCHA widget should keep a typed template ref');
}
if (!registerPage.includes('humanVerificationEnabled')) {
  throw new Error('Registration ALTCHA widget should be guarded by the configured provider');
}
if (!registerPage.includes("humanVerificationEnabledFor('register')")) {
  throw new Error('Registration ALTCHA widget should read register scenario availability from runtime web options');
}
if (registerPage.includes('public.humanVerificationProvider')) {
  throw new Error('Registration ALTCHA widget should not read provider from Nuxt public runtime config');
}
if (!registerPage.includes('ref="altchaWidget"')) {
  throw new Error('Registration ALTCHA widget should bind the template ref');
}
if (!registerPage.includes('altchaWidget.value?.reset()')) {
  throw new Error('Registration ALTCHA widget should reset the widget instance, not only the token');
}
if (!registerPage.includes("const submittedHumanVerificationToken = humanVerificationEnabled.value ? humanVerificationToken.value : ''")) {
  throw new Error('Registration should only submit ALTCHA tokens when human verification is enabled');
}
if (!registerPage.includes('body.humanVerification')) {
  throw new Error('Registration should omit the human verification payload when ALTCHA is disabled');
}
if (!registerPage.includes("humanVerificationEnabled.value && (submittedHumanVerificationToken || fieldError('humanVerification'))")) {
  throw new Error('Registration should reset ALTCHA after any failed submission that included a token');
}

const altchaWidgetTag = registerPage.match(/<altcha-widget[\s\S]*?\/>/)?.[0] || '';
if (altchaWidgetTag.includes(':key=') || altchaWidgetTag.includes('v-if=')) {
  throw new Error('Registration ALTCHA widget should reset in place instead of remounting');
}

for (const [name, content] of [
  ['login.vue', loginPage],
  ['register.vue', registerPage]
]) {
  if (!content.includes('.auth-input:-webkit-autofill')) {
    throw new Error(`${name} should override browser autofill input background`);
  }
  if (!content.includes('-webkit-box-shadow')) {
    throw new Error(`${name} autofill override should preserve the white input surface`);
  }
  if (!content.includes('-webkit-text-fill-color')) {
    throw new Error(`${name} autofill override should preserve text color`);
  }
}

console.log('Identity UI validation passed.');
