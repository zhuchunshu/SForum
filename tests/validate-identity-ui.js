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

console.log('Identity UI validation passed.');
