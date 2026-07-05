const fs = require('fs');
const path = require('path');

const root = process.cwd();
const indexPagePath = path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/pages/index.vue');
const themeLayerConfigPath = path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/nuxt.config.ts');
const nuxtConfigPath = path.resolve(root, 'apps/web/nuxt.config.ts');
const themeCssPath = path.resolve(root, 'extensions/builtin/themes/sforum-default/layer/app/assets/css/sforum-theme.css');
const zhLocalePath = path.resolve(root, 'apps/web/i18n/locales/zh-CN.json');
const enLocalePath = path.resolve(root, 'apps/web/i18n/locales/en-US.json');

console.log('Validating SForum homepage implementation...\n');

// 1. Verify files exist
if (!fs.existsSync(indexPagePath)) {
  throw new Error('default theme index.vue is missing');
}
if (!fs.existsSync(themeCssPath)) {
  throw new Error('default theme CSS is missing');
}
if (!fs.existsSync(themeLayerConfigPath)) {
  throw new Error('default theme layer nuxt.config.ts is missing');
}
if (!fs.existsSync(nuxtConfigPath)) {
  throw new Error('nuxt.config.ts is missing');
}
console.log('✓ default theme homepage and CSS files exist.');

// 2. Read contents
const indexContent = fs.readFileSync(indexPagePath, 'utf8');
const themeLayerConfig = fs.readFileSync(themeLayerConfigPath, 'utf8');
const nuxtConfig = fs.readFileSync(nuxtConfigPath, 'utf8');
const themeCss = fs.readFileSync(themeCssPath, 'utf8');
const zh = JSON.parse(fs.readFileSync(zhLocalePath, 'utf8'));
const en = JSON.parse(fs.readFileSync(enLocalePath, 'utf8'));

if (!nuxtConfig.includes('../../extensions/builtin/themes/sforum-default/layer')) {
  throw new Error('apps/web/nuxt.config.ts should statically extend the built-in default theme layer');
}
if (!nuxtConfig.includes('extends')) {
  throw new Error('apps/web/nuxt.config.ts should use Nuxt layers through extends');
}
console.log('✓ Nuxt config extends the built-in default theme layer.');

if (!themeCss.includes('--sf-surface') || !themeCss.includes('.auth-shell') || !themeCss.includes('.navbar')) {
  throw new Error('default theme CSS should own public surface, auth, and navbar styles');
}
console.log('✓ default theme CSS owns public surface, auth, and navbar styles.');

if (themeLayerConfig.includes('~/assets/css/sforum-theme.css')) {
  throw new Error('default theme layer CSS must resolve from the layer directory, not the host app ~/ alias');
}
if (!themeLayerConfig.includes('import.meta.url') || !themeLayerConfig.includes('sforum-theme.css')) {
  throw new Error('default theme layer nuxt.config.ts should register sforum-theme.css with a layer-relative path');
}
console.log('✓ default theme layer CSS path is layer-relative.');

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
if (!indexContent.includes('useSForumSeo') || !indexContent.includes('home.metaTitle')) {
  throw new Error('index.vue should configure metadata with useSForumSeo and i18n keys');
}
console.log('✓ index.vue contains useSForumSeo configuration.');

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
if (!indexContent.includes('max-w-[1376px]') || !indexContent.includes('lg:grid-cols-[270px_1fr_290px]') || !indexContent.includes('md:grid-cols-[1fr_290px]')) {
  throw new Error('index.vue layout grid configuration is missing or incorrect');
}
console.log('✓ 3-column responsive grid classes found in index.vue template.');

console.log('\nSForum homepage validation PASSED!');
