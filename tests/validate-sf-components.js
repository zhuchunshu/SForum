const fs = require('fs');
const path = require('path');

const ROOT = path.join(__dirname, '..');
const COMPONENT_DIR = path.join(ROOT, 'apps/web/app/components');
const CSS_FILE = path.join(ROOT, 'apps/web/app/assets/css/sforum-components.css');
const DOCS_PAGE = path.join(ROOT, 'apps/web/app/pages/components.vue');

const REQUIRED_COMPONENTS = [
  'SFAlert',
  'SFAvatar',
  'SFBadge',
  'SFButton',
  'SFCard',
  'SFComment',
  'SFEditor',
  'SFEmptyState',
  'SFFeedRow',
  'SFIconPicker',
  'SFInput',
  'SFPagination',
  'SFProgress',
  'SFSearch',
  'SFSkeleton',
  'SFTabs',
  'SFToast',
  'SFToggle'
];

const REQUIRED_CSS_SELECTORS = [
  '.sf-alert',
  '.sf-avatar',
  '.sf-badge',
  '.sf-button',
  '.sf-card',
  '.sf-comment',
  '.sf-editor',
  '.sf-empty-state',
  '.sf-feed-row',
  '.sf-icon-picker',
  '.sf-input',
  '.sf-pagination',
  '.sf-progress',
  '.sf-search',
  '.sf-skeleton',
  '.sf-tabs',
  '.sf-toast',
  '.sf-toggle'
];

const REQUIRED_AUTOFILL_RULES = [
  '.sf-input__control:-webkit-autofill',
  '.sf-search__input:-webkit-autofill',
  '-webkit-box-shadow',
  '-webkit-text-fill-color'
];

const REQUIRED_DOC_ANCHORS = [
  '#foundations',
  '#icons',
  '#feedback',
  '#forum',
  '#composer',
  '#moderation',
  '#profile',
  '#states'
];

const REQUIRED_DOC_COPY = [
  '发布工作流',
  'Icons 选择器',
  '审核与管理',
  '成员资料',
  '隐私设置',
  '加载与空状态',
  '富编辑器状态'
];

let passed = true;

function fail(message) {
  passed = false;
  console.error(`  x ${message}`);
}

function pass(message) {
  console.log(`  ✓ ${message}`);
}

console.log('Validating SForum SF component library...\n');

if (!fs.existsSync(COMPONENT_DIR)) {
  fail('components directory does not exist');
} else {
  pass('components directory exists');
}

for (const componentName of REQUIRED_COMPONENTS) {
  const file = path.join(COMPONENT_DIR, `${componentName}.vue`);

  if (!fs.existsSync(file)) {
    fail(`${componentName}.vue is missing`);
    continue;
  }

  const content = fs.readFileSync(file, 'utf8');

  if (!content.includes('<script setup lang="ts">')) {
    fail(`${componentName}.vue is missing a typed script setup block`);
  }

  if (!content.includes('defineProps<')) {
    fail(`${componentName}.vue is missing typed props`);
  }

  if (content.includes('Sf')) {
    fail(`${componentName}.vue contains the old Sf prefix`);
  }

  pass(`${componentName}.vue exists with typed props`);
}

if (!fs.existsSync(CSS_FILE)) {
  fail('sforum-components.css is missing');
} else {
  const css = fs.readFileSync(CSS_FILE, 'utf8');

  for (const selector of REQUIRED_CSS_SELECTORS) {
    if (!css.includes(selector)) {
      fail(`CSS selector ${selector} is missing`);
    }
  }

  for (const rule of REQUIRED_AUTOFILL_RULES) {
    if (!css.includes(rule)) {
      fail(`Component autofill rule ${rule} is missing`);
    }
  }

  pass('component CSS entrypoint exists');
}

if (!fs.existsSync(DOCS_PAGE)) {
  fail('components.vue documentation page is missing');
} else {
  const docs = fs.readFileSync(DOCS_PAGE, 'utf8');

  for (const componentName of ['SFAlert', 'SFBadge', 'SFToast', 'SFTabs']) {
    if (!docs.includes(componentName)) {
      fail(`components.vue does not preview ${componentName}`);
    }
  }

  for (const anchor of REQUIRED_DOC_ANCHORS) {
    if (!docs.includes(anchor)) {
      fail(`components.vue is missing navigation anchor ${anchor}`);
    }
  }

  for (const text of REQUIRED_DOC_COPY) {
    if (!docs.includes(text)) {
      fail(`components.vue is missing expanded scenario copy "${text}"`);
    }
  }

  if (!docs.includes('showError')) {
    fail('components.vue is missing dev-only route gating');
  }

  pass('components.vue previews key feedback components');
}

if (!passed) {
  console.error('\nSForum component library validation failed.');
  process.exit(1);
}

console.log('\nSForum component library validation passed.');
