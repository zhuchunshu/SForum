const fs = require('fs');
const path = require('path');

const DEMOS_DIR = path.join(__dirname, '../apps/web/app/assets/demos');
const FILES_TO_VALIDATE = [
  'forum-components.html',
  'forum-components-v2.html',
  'forum-components-swiss.html',
  'forum-components-glass.html',
  'forum-components-neobrutalism.html'
];

const REQUIRED_IDS = [
  'id="nav"',
  'id="avatars"',
  'id="feed"',
  'id="forms"',
  'id="comments"',
  'id="search"',
  'id="profile"',
  'id="interactions"',
  'id="lists"',
  'id="editor"',
  'id="analytics"',
  'id="badges"',
  'id="poll"',
  'id="thread"'
];

const FORBIDDEN_PLACEHOLDERS = ['TODO', 'TBD'];

let overallPassed = true;

console.log('Starting validation of demo files...');
console.log(`Demos directory: ${DEMOS_DIR}\n`);

for (const file of FILES_TO_VALIDATE) {
  const filePath = path.join(DEMOS_DIR, file);
  console.log(`Checking ${file}...`);
  
  if (!fs.existsSync(filePath)) {
    console.error(`  ❌ Error: File does not exist: ${file}`);
    overallPassed = false;
    console.log();
    continue;
  }

  try {
    const content = fs.readFileSync(filePath, 'utf8');
    let filePassed = true;

    // 1. Verify starts with <!DOCTYPE html>
    const trimmedContent = content.trimStart();
    if (!trimmedContent.startsWith('<!DOCTYPE html>')) {
      console.error(`  ❌ Error: File does not start with <!DOCTYPE html>`);
      filePassed = false;
    } else {
      console.log(`  ✓ Starts with <!DOCTYPE html>`);
    }

    // 2. Verify all 14 required section IDs
    const missingIds = [];
    for (const id of REQUIRED_IDS) {
      if (!content.includes(id)) {
        missingIds.push(id);
      }
    }

    if (missingIds.length > 0) {
      console.error(`  ❌ Error: Missing required section IDs: ${missingIds.join(', ')}`);
      filePassed = false;
    } else {
      console.log(`  ✓ Contains all 14 required section IDs`);
    }

    // 3. Verify no TODO or TBD placeholders
    const foundPlaceholders = [];
    for (const placeholder of FORBIDDEN_PLACEHOLDERS) {
      if (content.includes(placeholder)) {
        foundPlaceholders.push(placeholder);
      }
    }

    if (foundPlaceholders.length > 0) {
      console.error(`  ❌ Error: Contains forbidden placeholders: ${foundPlaceholders.join(', ')}`);
      filePassed = false;
    } else {
      console.log(`  ✓ No "TODO" or "TBD" placeholders found`);
    }

    if (filePassed) {
      console.log(`  🎉 ${file} validation PASSED!`);
    } else {
      console.log(`  ❌ ${file} validation FAILED.`);
      overallPassed = false;
    }

  } catch (error) {
    console.error(`  ❌ Error reading or validating file: ${error.message}`);
    overallPassed = false;
  }
  console.log();
}

if (overallPassed) {
  console.log('✅ All demos validated successfully!');
  process.exit(0);
} else {
  console.error('❌ Validation failed.');
  process.exit(1);
}
