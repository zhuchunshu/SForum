const fs = require('fs');
const path = require('path');

const root = process.cwd();
const indexPagePath = path.resolve(root, 'apps/web/app/pages/index.vue');
const zhLocalePath = path.resolve(root, 'apps/web/i18n/locales/zh-CN.json');
const enLocalePath = path.resolve(root, 'apps/web/i18n/locales/en-US.json');

console.log('Validating SForum homepage implementation...\n');

// 1. Verify files exist
if (!fs.existsSync(indexPagePath)) {
  throw new Error('index.vue is missing');
}
console.log('✓ index.vue file exists.');

// 2. Read contents
const indexContent = fs.readFileSync(indexPagePath, 'utf8');
const zh = JSON.parse(fs.readFileSync(zhLocalePath, 'utf8'));
const en = JSON.parse(fs.readFileSync(enLocalePath, 'utf8'));

// 3. Verify component usages in index.vue
const requiredComponents = [
  'SFCard',
  'SFSearch',
  'SFTabs',
  'SFFeedRow',
  'SFPagination',
  'SFAvatar',
  'SFBadge',
  'SFButton',
  'SFEmptyState',
  'SFSkeleton'
];

for (const comp of requiredComponents) {
  if (!indexContent.includes(comp)) {
    throw new Error(`index.vue should utilize ${comp} component`);
  }
}
console.log('✓ index.vue uses all 10 required SF component library tags.');

// 4. Verify SEO configurations
if (!indexContent.includes('useSeoMeta') || !indexContent.includes('home.metaTitle')) {
  throw new Error('index.vue should configure metadata with useSeoMeta and i18n keys');
}
console.log('✓ index.vue contains useSeoMeta configuration.');

// 5. Verify translation keys in both languages
const keyPaths = [
  ['home', 'metaTitle'],
  ['home', 'searchPlaceholder'],
  ['home', 'filter', 'latest'],
  ['home', 'sidebar', 'navHome'],
  ['home', 'sidebar', 'checkIn'],
  ['home', 'sidebar', 'forumStats']
];

function getKeyValue(obj, pathArr) {
  return pathArr.reduce((prev, curr) => prev?.[curr], obj);
}

for (const pathArr of keyPaths) {
  if (!getKeyValue(zh, pathArr)) {
    throw new Error(`zh-CN.json is missing key path: ${pathArr.join('.')}`);
  }
  if (!getKeyValue(en, pathArr)) {
    throw new Error(`en-US.json is missing key path: ${pathArr.join('.')}`);
  }
}
console.log('✓ All 6 required homepage locale key paths validated in both zh-CN and en-US bundles.');

// 6. Verify layout grids
if (!indexContent.includes('grid-cols-12') || !indexContent.includes('col-span-12') || !indexContent.includes('lg:col-span-6')) {
  throw new Error('index.vue layout grid configuration is missing or incorrect');
}
console.log('✓ 3-column responsive grid classes found in index.vue template.');

console.log('\n🎉 SForum homepage validation PASSED!');
