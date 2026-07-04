const fs = require('fs');
const path = require('path');

const root = process.cwd();
const requiredFiles = [
  'apps/web/app/composables/useAuthSession.ts',
  'apps/web/app/middleware/admin.ts',
  'apps/web/app/pages/register.vue',
  'apps/web/app/pages/login.vue',
  'apps/web/app/pages/admin/index.vue',
  'apps/web/app/pages/admin/roles.vue'
];

for (const file of requiredFiles) {
  if (!fs.existsSync(path.resolve(root, file))) {
    throw new Error(`Missing required identity UI file: ${file}`);
  }
}

const zh = JSON.parse(fs.readFileSync(path.resolve(root, 'apps/web/i18n/locales/zh-CN.json'), 'utf8'));
const en = JSON.parse(fs.readFileSync(path.resolve(root, 'apps/web/i18n/locales/en-US.json'), 'utf8'));
const registerPage = fs.readFileSync(path.resolve(root, 'apps/web/app/pages/register.vue'), 'utf8');
const loginPage = fs.readFileSync(path.resolve(root, 'apps/web/app/pages/login.vue'), 'utf8');

const requiredKeys = [
  ['auth', 'registerTitle'],
  ['auth', 'loginTitle'],
  ['admin', 'home', 'title'],
  ['admin', 'roles', 'title'],
  ['errors', 'permissionDenied']
];

function valueAt(object, keyPath) {
  return keyPath.reduce((current, key) => current?.[key], object);
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
if (!registerPage.includes('hideLogo: true')) {
  throw new Error('Registration ALTCHA widget should hide the ALTCHA logo icon');
}
if (!registerPage.includes('hideFooter: true')) {
  throw new Error('Registration ALTCHA widget should hide the ALTCHA attribution footer');
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
