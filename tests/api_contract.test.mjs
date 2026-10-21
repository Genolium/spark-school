import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

const ROOT = path.resolve(import.meta.dirname, '..');
const api = fs.readdirSync(path.join(ROOT, 'frontend/src/lib'))
  .filter((f) => f.endsWith('.ts'))
  .map((f) => fs.readFileSync(path.join(ROOT, 'frontend/src/lib', f), 'utf8'))
  .join('\n');
const main = fs.readFileSync(path.join(ROOT, 'backend/cmd/server/main.go'), 'utf8');

// A curated contract: every critical frontend call must have a backend route.
const pairs = [
  ['/stats/places', '/stats/places'],
  ['/pricing', '/pricing'],
  ['/affiliate/stats', '/affiliate/stats'],
  ['/affiliate/payout', '/affiliate/payout'],
  ['/promo/validate', '/promo/validate'],
  ['/analytics/track', '/analytics/track'],
  ['/diagnostic', '/diagnostic'],
  ['/analytics/experiments', '/analytics/experiments'],
  ['/client-errors', '/client-errors'],
  ['/payments/yookassa/create', '/payments/yookassa/create'],
  ['/admin/login', '/admin/login'],
  ['/admin/logout', '/admin/logout'],
  ['/admin/stats', '/admin/stats'],
  ['/admin/users', '/admin/users'],
  ['/admin/payouts', '/admin/payouts'],
  ['/admin/promos', '/admin/promos'],
  ['/admin/pricing', '/admin/pricing'],
  ['/admin/analytics/funnel', '/admin/analytics/funnel'],
  ['/admin/analytics/experiments', '/admin/analytics/experiments'],
  ['/admin/outbox/dead', '/admin/outbox/dead'],
];

test('frontend API paths have backend routes', () => {
  for (const [frontendPath, backendPath] of pairs) {
    assert.ok(api.includes(frontendPath), `frontend API is missing ${frontendPath}`);
    const lastSegment = backendPath.split('/').pop();
    assert.ok(main.includes(lastSegment), `backend is missing route segment ${lastSegment} (${backendPath})`);
  }
});

test('server does not expose mock/demo auth', () => {
  assert.ok(!api.includes('DEMO_USERS'));
  assert.ok(!api.includes('generateMockDailyStats'));
  assert.ok(!api.includes('mock fallback'));
});
