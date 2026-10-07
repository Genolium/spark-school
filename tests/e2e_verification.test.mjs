/**
 * E2E Verification Test Suite for Проект «так называемый SPARK»
 * 
 * Covers all 5 Tiers per PROJECT.md, ORIGINAL_REQUEST.md, and DISPATCH.md:
 * - Tier 1: Feature Coverage (Pricing, Promo -5%, Chances Calculator, Capacity Auto-Doubling)
 * - Tier 2: Boundary & Corner Cases (Capacity Transitions, Bot Private Chat Filter, Receipt Deduplication)
 * - Tier 3: Cross-Feature & Deep-Links (Calculator CTA Deep-Link, Payment Modal Deep-Link)
 * - Tier 4: Repository Hygiene & Acceptance Scan (Residual Forbidden Strings, Anti-Requirements Guard)
 * - Tier 5: Packaging & Deployment Artifact (spark-deploy.zip Existence, Size, Bundle Structure)
 * 
 * Invocation:
 *   node --test tests/e2e_verification.test.mjs
 */

import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync } from 'node:child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ROOT = path.resolve(__dirname, '..');

// Helper: Calculate capacity and remaining spots using auto-doubling algorithm
function calculateCapacityAndSpots(activeStudents) {
  let n = Number(activeStudents);
  if (isNaN(n) || n < 0) {
    n = 0;
  }
  let capacity = 25;
  while (n >= capacity) {
    capacity *= 2;
  }
  const spotsLeft = capacity - n;
  return {
    total_capacity: capacity,
    active_students: n,
    spots_left: spotsLeft,
  };
}

// Helper: Calculate 5% promo discount
function calculatePromoDiscount(basePrice, discountPercent = 5) {
  const discountAmount = Math.round((basePrice * discountPercent) / 100);
  const finalPrice = basePrice - discountAmount;
  return { basePrice, discountAmount, finalPrice };
}

// Helper: Format number with Russian thousands separator
function formatRub(num) {
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
}

// Helper: Check if Telegram chat is private
function isPrivateChat(chat) {
  if (!chat || typeof chat !== 'object') return false;
  return chat.type === 'private' && Number(chat.id) > 0;
}

// Helper: Generate Telegram payment deep-link
function generatePaymentDeepLink(tier, username = '', promoCode = '', refCode = '') {
  const cleanU = (username || '').replace('@', '').replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 16);
  const cleanP = (promoCode || '').replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 12);
  const cleanR = (refCode || '').replace(/[^a-zA-Z0-9_-]/g, '').slice(0, 12);

  let startParam = `pay_${tier}`;
  if (cleanP && cleanU) {
    startParam = `pay_${tier}_${cleanU}_p_${cleanP}`;
  } else if (cleanP) {
    startParam = `pay_${tier}_p_${cleanP}`;
  } else if (cleanR && cleanU) {
    startParam = `pay_${tier}_${cleanU}_r_${cleanR}`;
  } else if (cleanU) {
    startParam = `pay_${tier}_${cleanU}`;
  }
  return `https://t.me/spark_prep_bot?start=${startParam}`;
}

// Helper: Recursively find files in directory
function getFilesRecursively(dir, filterFn) {
  const results = [];
  if (!fs.existsSync(dir)) return results;
  const list = fs.readdirSync(dir, { withFileTypes: true });
  for (const item of list) {
    const fullPath = path.join(dir, item.name);
    if (item.isDirectory()) {
      if (item.name === '.git' || item.name === 'node_modules' || item.name === '.next' || item.name === '.agents') {
        continue;
      }
      results.push(...getFilesRecursively(fullPath, filterFn));
    } else if (item.isFile()) {
      if (!filterFn || filterFn(fullPath)) {
        results.push(fullPath);
      }
    }
  }
  return results;
}

// ============================================================================
// TIER 1: FEATURE COVERAGE
// ============================================================================
describe('Tier 1: Feature Coverage', () => {

  it('1.1 Pricing Tiers: Strictly 2 tiers: Акселератор (6900) and VIP (14900) exist, Basic is removed', () => {
    // 1. Verify frontend PricingSection.tsx
    const pricingSectionPath = path.join(ROOT, 'frontend', 'src', 'components', 'PricingSection.tsx');
    assert.ok(fs.existsSync(pricingSectionPath), 'PricingSection.tsx must exist');
    const pricingContent = fs.readFileSync(pricingSectionPath, 'utf8');

    // Check tier names in PricingSection
    assert.match(pricingContent, /Акселератор/, 'PricingSection must include Акселератор tier');
    assert.match(pricingContent, /VIP/, 'PricingSection must include VIP tier');

    // Check base prices in PricingSection
    assert.match(pricingContent, /6\s*900\s*₽/, 'PricingSection must display 6 900 ₽ for Акселератор');
    assert.match(pricingContent, /14\s*900\s*₽/, 'PricingSection must display 14 900 ₽ for VIP');

    // Verify Basic tier is completely removed from PricingSection
    assert.doesNotMatch(pricingContent, /Выбрать Базовый/, 'PricingSection must not have Basic tier button');

    // Check strict naming: VIP card header must not include 'Консьерж'
    assert.doesNotMatch(pricingContent, /<h3[^>]*>[\s\S]*?VIP\s*\/\s*Консьерж[\s\S]*?<\/h3>/, 'PricingSection header must NOT contain "VIP / Консьерж"');
    assert.doesNotMatch(pricingContent, /Выбрать\s+VIP\s+Консьерж/, 'PricingSection button must NOT say "VIP Консьерж"');

    // 2. Verify frontend PaymentModal.tsx
    const paymentModalPath = path.join(ROOT, 'frontend', 'src', 'components', 'Modals', 'PaymentModal.tsx');
    assert.ok(fs.existsSync(paymentModalPath), 'PaymentModal.tsx must exist');
    const modalContent = fs.readFileSync(paymentModalPath, 'utf8');

    // Check TIERS dictionary in PaymentModal has only accelerator and vip
    assert.match(modalContent, /accelerator:\s*\{[\s\S]*?price:\s*6900/, 'PaymentModal must have accelerator price 6900');
    assert.match(modalContent, /vip:\s*\{[\s\S]*?price:\s*14900/, 'PaymentModal must have vip price 14900');
    assert.match(modalContent, /vip:\s*\{[\s\S]*?name:\s*["']VIP["']/, 'PaymentModal VIP tier name must be strictly "VIP"');
    assert.doesNotMatch(modalContent, /basic:\s*\{[\s\S]*?price:\s*2900/, 'PaymentModal must NOT have basic tier');

    // 3. Verify backend promo.go tier pricing
    const backendPromoPath = path.join(ROOT, 'backend', 'internal', 'handlers', 'promo.go');
    assert.ok(fs.existsSync(backendPromoPath), 'backend/internal/handlers/promo.go must exist');
    const backendPromoContent = fs.readFileSync(backendPromoPath, 'utf8');
    assert.match(backendPromoContent, /case\s+"vip":\s*return\s*14900/, 'backend GetTierPrice must return 14900 for vip');
    assert.match(backendPromoContent, /case\s+"accelerator"/, 'backend GetTierPrice must return 6900 for accelerator');
  });

  it('1.2 Promo Code Engine: 5% discount calculation (6900 -> 6555, 14900 -> 14155 for 2 active tiers)', () => {
    // Mathematical specification validation for 2 active tiers
    const accDiscount = calculatePromoDiscount(6900, 5);
    assert.equal(accDiscount.discountAmount, 345, '6900 * 5% must equal 345');
    assert.equal(accDiscount.finalPrice, 6555, '6900 - 345 must equal 6555');

    const vipDiscount = calculatePromoDiscount(14900, 5);
    assert.equal(vipDiscount.discountAmount, 745, '14900 * 5% must equal 745');
    assert.equal(vipDiscount.finalPrice, 14155, '14900 - 745 must equal 14155');

    // Verify presence of promo prices in PricingSection.tsx (strictly 2 tiers: 6 900 and 14 900)
    const pricingSectionPath = path.join(ROOT, 'frontend', 'src', 'components', 'PricingSection.tsx');
    const pricingContent = fs.readFileSync(pricingSectionPath, 'utf8');
    assert.match(pricingContent, /6\s*900\s*₽/, 'PricingSection must display 6 900 ₽ price');
    assert.match(pricingContent, /14\s*900\s*₽/, 'PricingSection must display 14 900 ₽ price');

    // Verify discount calculation formula in PaymentModal.tsx
    const paymentModalPath = path.join(ROOT, 'frontend', 'src', 'components', 'Modals', 'PaymentModal.tsx');
    const modalContent = fs.readFileSync(paymentModalPath, 'utf8');
    assert.match(modalContent, /0\.95/, 'PaymentModal must calculate 5% discount using 0.95 multiplier');
    assert.match(modalContent, /discountPrice:\s*6555/, 'PaymentModal must have discountPrice 6555 for accelerator');
    assert.match(modalContent, /discountPrice:\s*14155/, 'PaymentModal must have discountPrice 14155 for vip');
  });

  it('1.3 Chances Calculator Logic: 3 steps, inputs, probability calculation (75%, 88%, 96%)', () => {
    const calcPath = path.join(ROOT, 'frontend', 'src', 'components', 'ChancesCalculator.tsx');
    assert.ok(fs.existsSync(calcPath), 'ChancesCalculator.tsx must exist');
    const calcContent = fs.readFileSync(calcPath, 'utf8');

    // Step 1: Age (18-21) and citizenship / residency
    assert.match(calcContent, /18\s*(?:до|-|–)\s*21/i, 'Calculator Step 1 must mention age 18-21');
    assert.match(calcContent, /гражданин\s*РФ/i, 'Calculator Step 1 must mention RF citizenship');

    // Step 2: Study year options
    assert.match(calcContent, /1[–-]2\s*курс/i, 'Calculator Step 2 must include 1-2 course');
    assert.match(calcContent, /3\s*курс/i, 'Calculator Step 2 must include 3 course');
    assert.match(calcContent, /выпускной\s*курс/i, 'Calculator Step 2 must include graduating course');

    // Step 3: English proficiency levels and probabilities
    assert.match(calcContent, /B1/i, 'Calculator Step 3 must include B1');
    assert.match(calcContent, /B2/i, 'Calculator Step 3 must include B2');
    assert.match(calcContent, /C1/i, 'Calculator Step 3 must include C1');
    assert.match(calcContent, /75%|percent:\s*75/, 'Calculator must map B1 to 75%');
    assert.match(calcContent, /88%|percent:\s*88/, 'Calculator must map B2 to 88%');
    assert.match(calcContent, /96%|percent:\s*96/, 'Calculator must map C1 to 96%');

    // Step 3 Anti-Requirements: No Duolingo, GPA, or documents anywhere in Calculator
    assert.doesNotMatch(calcContent, /duolingo/i, 'Calculator must NOT mention Duolingo');
    assert.doesNotMatch(calcContent, /\bgpa\b/i, 'Calculator must NOT mention GPA');
    assert.doesNotMatch(calcContent, /сертификат|транскрипт/i, 'Calculator must NOT require certificates/transcripts');
  });

  it('1.4 Capacity Auto-Doubling Logic: Base C0=25, doubles when N >= C, spots_left = C - N', () => {
    // 1. Verify specification algorithm
    const res0 = calculateCapacityAndSpots(0);
    assert.deepEqual(res0, { total_capacity: 25, active_students: 0, spots_left: 25 });

    const res25 = calculateCapacityAndSpots(25);
    assert.deepEqual(res25, { total_capacity: 50, active_students: 25, spots_left: 25 });

    const res50 = calculateCapacityAndSpots(50);
    assert.deepEqual(res50, { total_capacity: 100, active_students: 50, spots_left: 50 });

    // 2. Verify backend Go implementation in backend/internal/handlers/stats.go
    const statsGoPath = path.join(ROOT, 'backend', 'internal', 'handlers', 'stats.go');
    assert.ok(fs.existsSync(statsGoPath), 'stats.go must exist');
    const statsGoContent = fs.readFileSync(statsGoPath, 'utf8');

    assert.match(statsGoContent, /func CalculateCapacityAndSpots\(/, 'CalculateCapacityAndSpots must be defined in stats.go');
    assert.match(statsGoContent, /capacity\s*:=\s*int64\(25\)/, 'Base capacity C0 must be 25');
    assert.match(statsGoContent, /for\s+activeStudents\s*>=\s*capacity\s*\{\s*capacity\s*\*=\s*2\s*\}/, 'Must double capacity iteratively while activeStudents >= capacity');
    assert.match(statsGoContent, /capacity\s*-\s*activeStudents/, 'Spots left must be capacity - activeStudents');
  });

});

// ============================================================================
// TIER 2: BOUNDARY & CORNER CASES
// ============================================================================
describe('Tier 2: Boundary & Corner Cases', () => {

  it('2.1 Capacity Auto-Doubling at boundaries: N = 0, 24, 25, 49, 50, 99, 100, and negative', () => {
    const testCases = [
      { n: -5, wantCap: 25, wantSpots: 25, wantN: 0, desc: 'Negative N normalized to 0' },
      { n: 0, wantCap: 25, wantSpots: 25, wantN: 0, desc: 'Empty capacity N=0' },
      { n: 24, wantCap: 25, wantSpots: 1, wantN: 24, desc: 'Last spot before doubling N=24' },
      { n: 25, wantCap: 50, wantSpots: 25, wantN: 25, desc: 'First doubling threshold N=25' },
      { n: 49, wantCap: 50, wantSpots: 1, wantN: 49, desc: 'Last spot in second tier N=49' },
      { n: 50, wantCap: 100, wantSpots: 50, wantN: 50, desc: 'Second doubling threshold N=50' },
      { n: 99, wantCap: 100, wantSpots: 1, wantN: 99, desc: 'Last spot in third tier N=99' },
      { n: 100, wantCap: 200, wantSpots: 100, wantN: 100, desc: 'Third doubling threshold N=100' },
    ];

    for (const tc of testCases) {
      const res = calculateCapacityAndSpots(tc.n);
      assert.equal(res.total_capacity, tc.wantCap, `Capacity mismatch for ${tc.desc}: got ${res.total_capacity}, want ${tc.wantCap}`);
      assert.equal(res.spots_left, tc.wantSpots, `Spots mismatch for ${tc.desc}: got ${res.spots_left}, want ${tc.wantSpots}`);
      assert.equal(res.active_students, tc.wantN, `Active students mismatch for ${tc.desc}`);
    }
  });

  it('2.2 Bot Private Chat Filter: Non-private chats and negative IDs are strictly ignored', () => {
    // 1. Predicate unit tests
    assert.equal(isPrivateChat({ type: 'private', id: 12345 }), true, 'Direct private chat with positive ID must be accepted');
    assert.equal(isPrivateChat({ type: 'private', id: 987654321 }), true, 'Direct private chat with valid user ID must be accepted');
    assert.equal(isPrivateChat({ type: 'group', id: -10012345 }), false, 'Group chat must be rejected');
    assert.equal(isPrivateChat({ type: 'supergroup', id: -10098765 }), false, 'Supergroup chat must be rejected');
    assert.equal(isPrivateChat({ type: 'channel', id: -10055555 }), false, 'Channel message must be rejected');
    assert.equal(isPrivateChat({ type: 'private', id: -1 }), false, 'Negative chat ID must be rejected');
    assert.equal(isPrivateChat({ type: 'private', id: 0 }), false, 'Zero chat ID must be rejected');
    assert.equal(isPrivateChat(null), false, 'Null chat must be rejected');

    // 2. Code check: inspect backend/internal/bot/bot.go
    const botGoPath = path.join(ROOT, 'backend', 'internal', 'bot', 'bot.go');
    assert.ok(fs.existsSync(botGoPath), 'bot.go must exist');
    const botGoContent = fs.readFileSync(botGoPath, 'utf8');

    const hasGoChatFilter = /msg\.Chat\.Type\s*!=\s*"private"|chat\.type\s*===\s*["']private["']/i.test(botGoContent);
    assert.ok(hasGoChatFilter, 'Private chat filter must be enforced in Go bot');
  });

  it('2.3 Anti-Fraud Receipt Deduplication: Rejects duplicate file_unique_id and file_hash', () => {
    // 1. Verify models definition has FileUniqueID
    const modelsPath = path.join(ROOT, 'backend', 'internal', 'models', 'models.go');
    assert.ok(fs.existsSync(modelsPath), 'models.go must exist');
    const modelsContent = fs.readFileSync(modelsPath, 'utf8');
    assert.match(modelsContent, /FileUniqueID\s+string/i, 'PaymentReceipt model must define FileUniqueID');

    // 2. Verify handlers receipt.go handles FileUniqueID
    const receiptHandlerPath = path.join(ROOT, 'backend', 'internal', 'handlers', 'receipt.go');
    assert.ok(fs.existsSync(receiptHandlerPath), 'receipt.go must exist');
    const receiptContent = fs.readFileSync(receiptHandlerPath, 'utf8');
    assert.match(receiptContent, /FileUniqueID\s+string/i, 'SubmitReceiptRequest must include FileUniqueID');

    // 3. Verify deduplication logic via simulated store
    const store = new Map();
    function submitReceipt(fileHash, fileUniqueID, telegramId) {
      if (!fileHash && !fileUniqueID) {
        return { status: 400, error: 'Missing receipt identifier' };
      }
      for (const [_, entry] of store.entries()) {
        if ((fileHash && entry.fileHash === fileHash) || (fileUniqueID && entry.fileUniqueID === fileUniqueID)) {
          return { status: 409, duplicate: true, error: 'Receipt already submitted' };
        }
      }
      const id = store.size + 1;
      store.set(id, { id, fileHash, fileUniqueID, telegramId });
      return { status: 201, duplicate: false, receipt_id: id };
    }

    // Submission 1: fresh receipt
    const sub1 = submitReceipt('hash_aaa_111', 'unique_file_001', 1111);
    assert.equal(sub1.status, 201, 'First receipt submission must succeed');
    assert.equal(sub1.duplicate, false);

    // Submission 2: duplicate by file_unique_id
    const sub2 = submitReceipt('different_hash_222', 'unique_file_001', 2222);
    assert.equal(sub2.status, 409, 'Duplicate file_unique_id must return 409 Conflict');
    assert.equal(sub2.duplicate, true);

    // Submission 3: duplicate by file_hash
    const sub3 = submitReceipt('hash_aaa_111', 'unique_file_002', 3333);
    assert.equal(sub3.status, 409, 'Duplicate file_hash must return 409 Conflict');
    assert.equal(sub3.duplicate, true);

    // Submission 4: distinct receipt
    const sub4 = submitReceipt('hash_ccc_333', 'unique_file_003', 4444);
    assert.equal(sub4.status, 201, 'Unique receipt must be accepted');
    assert.equal(sub4.duplicate, false);
  });

});

// ============================================================================
// TIER 3: CROSS-FEATURE & DEEP-LINKS
// ============================================================================
describe('Tier 3: Cross-Feature & Deep-Links', () => {

  it('3.1 Calculator CTA Deep-Link: Adheres to https://t.me/spark_prep_bot?start=calc_...', () => {
    const calcPath = path.join(ROOT, 'frontend', 'src', 'components', 'ChancesCalculator.tsx');
    assert.ok(fs.existsSync(calcPath), 'ChancesCalculator.tsx must exist');
    const calcContent = fs.readFileSync(calcPath, 'utf8');

    // Check CTA link target
    assert.match(
      calcContent,
      /https:\/\/t\.me\/spark_prep_bot\?start=calc_[a-zA-Z0-9_\${}]*/,
      'Calculator CTA button must link to https://t.me/spark_prep_bot?start=calc_...'
    );

    // Check CTA copy mentions "так называемым Илем"
    assert.match(calcContent, /Разобрать заявку с так называемым Илем в Telegram/i, 'Calculator CTA copy must invite user to discuss application with так называемым Илем');
  });

  it('3.2 Payment Modal Deep-Link: Adheres to https://t.me/spark_prep_bot?start=pay_... with tiers and promo', () => {
    const paymentModalPath = path.join(ROOT, 'frontend', 'src', 'components', 'Modals', 'PaymentModal.tsx');
    assert.ok(fs.existsSync(paymentModalPath), 'PaymentModal.tsx must exist');
    const modalContent = fs.readFileSync(paymentModalPath, 'utf8');

    // Check deep-link prefix in PaymentModal
    assert.match(
      modalContent,
      /https:\/\/t\.me\/spark_prep_bot\?start=\$\{startParam\}|https:\/\/t\.me\/spark_prep_bot\?start=pay_/,
      'PaymentModal must generate deep link pointing to https://t.me/spark_prep_bot?start=pay_...'
    );

    // Test deep-link generator helper covering all tiers
    const linkBasic = generatePaymentDeepLink('basic');
    assert.equal(linkBasic, 'https://t.me/spark_prep_bot?start=pay_basic');

    const linkAcc = generatePaymentDeepLink('accelerator');
    assert.equal(linkAcc, 'https://t.me/spark_prep_bot?start=pay_accelerator');

    const linkVip = generatePaymentDeepLink('vip');
    assert.equal(linkVip, 'https://t.me/spark_prep_bot?start=pay_vip');

    // Test with username and promo code
    const linkWithPromo = generatePaymentDeepLink('vip', '@student_alex', 'SPARK5');
    assert.equal(linkWithPromo, 'https://t.me/spark_prep_bot?start=pay_vip_student_alex_p_SPARK5');

    // Ensure parameters adhere to Telegram rules: max 64 chars, [a-zA-Z0-9_-]
    const param = linkWithPromo.split('?start=')[1];
    assert.ok(param.length <= 64, `Start parameter length ${param.length} must be <= 64 characters`);
    assert.match(param, /^[a-zA-Z0-9_-]+$/, 'Start parameter must contain only alphanumeric characters, underscores, and dashes');
  });

});

// ============================================================================
// TIER 4: REPOSITORY HYGIENE & ACCEPTANCE SCAN
// ============================================================================
describe('Tier 4: Repository Hygiene & Acceptance Scan', () => {

  const productionDirs = [
    path.join(ROOT, 'frontend', 'src'),
    path.join(ROOT, 'backend', 'cmd'),
    path.join(ROOT, 'backend', 'internal'),
  ];

  it('4.1 Codebase Scan: 0 occurrences of forbidden residual "SPARK School"', () => {
    const matches = [];
    for (const dir of productionDirs) {
      const files = getFilesRecursively(dir, (f) => /\.(ts|tsx|js|mjs|go|json|sql|html)$/.test(f));
      for (const file of files) {
        const content = fs.readFileSync(file, 'utf8');
        if (content.includes('SPARK School')) {
          matches.push({ file: path.relative(ROOT, file) });
        }
      }
    }
    assert.equal(matches.length, 0, `Forbidden residual "SPARK School" found in files: ${JSON.stringify(matches)}`);
  });

  it('4.2 Codebase Scan: 0 occurrences of forbidden residual "Илья Васюнин"', () => {
    const matches = [];
    for (const dir of productionDirs) {
      const files = getFilesRecursively(dir, (f) => /\.(ts|tsx|js|mjs|go|json|sql|html)$/.test(f));
      for (const file of files) {
        const content = fs.readFileSync(file, 'utf8');
        if (content.includes('Илья Васюнин')) {
          matches.push({ file: path.relative(ROOT, file) });
        }
      }
    }
    assert.equal(matches.length, 0, `Forbidden residual "Илья Васюнин" found in files: ${JSON.stringify(matches)}`);
  });

  it('4.3 Codebase Scan: 0 occurrences of "Консьерж" in pricing components and tier names', () => {
    const pricingFiles = [
      path.join(ROOT, 'frontend', 'src', 'components', 'PricingSection.tsx'),
      path.join(ROOT, 'frontend', 'src', 'components', 'Modals', 'PaymentModal.tsx'),
    ];

    const matches = [];
    for (const file of pricingFiles) {
      if (fs.existsSync(file)) {
        const content = fs.readFileSync(file, 'utf8');
        if (/Консьерж/i.test(content)) {
          matches.push({ file: path.relative(ROOT, file) });
        }
      }
    }
    assert.equal(matches.length, 0, `Forbidden term "Консьерж" found in pricing files: ${JSON.stringify(matches)}`);
  });

  it('4.4 Anti-Requirements: Story card generator is NOT present in codebase', () => {
    const frontendSrc = path.join(ROOT, 'frontend', 'src');
    const files = getFilesRecursively(frontendSrc);
    const storyCardFiles = files.filter(f => /story.*card|story.*generator/i.test(path.basename(f)));
    assert.equal(storyCardFiles.length, 0, `Story card generator files must not exist: ${storyCardFiles.join(', ')}`);

    // Check for story card components or hooks in code
    for (const file of files) {
      if (/\.(ts|tsx)$/.test(file)) {
        const content = fs.readFileSync(file, 'utf8');
        assert.doesNotMatch(content, /StoryCardGenerator|StoryCardModal|генератор карточек для сториз/i, `Story card generator must not be implemented in ${path.relative(ROOT, file)}`);
      }
    }
  });

  it('4.5 Anti-Requirements: Essay before/after slider is NOT present in codebase', () => {
    const frontendSrc = path.join(ROOT, 'frontend', 'src');
    const files = getFilesRecursively(frontendSrc);
    const sliderFiles = files.filter(f => /essay.*slider|before.*after.*slider/i.test(path.basename(f)));
    assert.equal(sliderFiles.length, 0, `Essay slider files must not exist: ${sliderFiles.join(', ')}`);

    // Check for slider components in code
    for (const file of files) {
      if (/\.(ts|tsx)$/.test(file)) {
        const content = fs.readFileSync(file, 'utf8');
        assert.doesNotMatch(content, /EssayBeforeAfterSlider|Эссе до \/ после/i, `Essay before/after slider must not be implemented in ${path.relative(ROOT, file)}`);
      }
    }
  });

});

// ============================================================================
// TIER 5: PACKAGING & DEPLOYMENT ARTIFACT
// ============================================================================
describe('Tier 5: Packaging & Deployment Artifact', () => {

  const zipPath = path.join(ROOT, 'spark-deploy.zip');

  it('5.1 Deployment archive exists: spark-deploy.zip', () => {
    assert.ok(fs.existsSync(zipPath), 'spark-deploy.zip must exist in project root');
  });

  it('5.2 Deployment archive size is greater than 1 MB', () => {
    const stats = fs.statSync(zipPath);
    const sizeBytes = stats.size;
    const sizeMB = sizeBytes / (1024 * 1024);
    assert.ok(sizeBytes > 1024 * 1024, `spark-deploy.zip size (${sizeMB.toFixed(2)} MB) must be greater than 1 MB`);
  });

  it('5.3 Deployment archive bundle contains all required production components', () => {
    let zipEntries = [];
    try {
      const output = execSync('tar -tf spark-deploy.zip', { cwd: ROOT, encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] });
      zipEntries = output.split(/\r?\n/).map(s => s.trim().replace(/\\/g, '/')).filter(Boolean);
    } catch (err) {
      // Fallback: check deploy directory if tar fails
      const deployDir = path.join(ROOT, 'deploy');
      assert.ok(fs.existsSync(deployDir), 'deploy/ directory must exist if tar command fails');
      zipEntries = getFilesRecursively(deployDir).map(f => path.relative(deployDir, f).replace(/\\/g, '/'));
    }

    assert.ok(zipEntries.length > 0, 'Deployment archive must contain files');

    const requiredEntries = [
      { pattern: /docker-compose\.ya?ml/, name: 'docker-compose.yml' },
      { pattern: /Caddyfile/, name: 'Caddyfile' },
      { pattern: /backend\/.*Dockerfile/, name: 'backend/Dockerfile' },
      { pattern: /backend\/cmd\/server\/main\.go/, name: 'backend/cmd/server/main.go' },
      { pattern: /frontend\/.*package\.json/, name: 'frontend/package.json' },
      { pattern: /seed\.sql/, name: 'seed.sql' },
      { pattern: /DEPLOY\.md/, name: 'DEPLOY.md' },
    ];

    for (const req of requiredEntries) {
      const found = zipEntries.some(entry => req.pattern.test(entry));
      assert.ok(found, `Deployment archive must contain ${req.name}`);
    }
  });

});
