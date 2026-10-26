/**
 * Pure helpers for admin campaign short links. Kept as plain ESM (no imports,
 * no side effects) so the frontend bundle and `node --test` use exactly the
 * same code — the same pattern as `visitor-param.mjs`.
 */

/** Report windows use the same fixed Moscow business day as the backend. */
export const MSK_OFFSET_MS = 3 * 60 * 60 * 1000;

const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * Calendar date in Moscow time as `YYYY-MM-DD`. Shifting the instant and then
 * reading the UTC date avoids depending on the browser's local timezone.
 * @param {Date} date
 * @returns {string}
 */
export function moscowDate(date) {
  return new Date(date.getTime() + MSK_OFFSET_MS).toISOString().slice(0, 10);
}

/**
 * Half-open report window that includes today in Moscow time. The backend
 * interprets a date-only `to` as the whole day, so 7 days means today plus the
 * previous six.
 * @param {number} days
 * @param {Date} [now]
 * @returns {{from: string, to: string}}
 */
export function reportWindow(days, now = new Date()) {
  const start = new Date(now.getTime() - (days - 1) * DAY_MS);
  return { from: moscowDate(start), to: moscowDate(now) };
}

/**
 * Mirror of the backend `slugify`: lowercase latin letters, digits and single
 * dashes, at most 40 characters. Anything outside the allowed set acts as a
 * separator, so a fully non-latin slug normalizes to "".
 * @param {string} raw
 * @returns {string}
 */
export function slugifyCampaignLink(raw) {
  return String(raw ?? "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 40)
    .replace(/-+$/g, "");
}

/**
 * Whether a normalized slug satisfies `validateCampaignLink` in the backend.
 * @param {string} slug
 * @returns {boolean}
 */
export function isValidCampaignSlug(slug) {
  return (
    typeof slug === "string" &&
    slug.length >= 3 &&
    slug.length <= 40 &&
    /^[a-z0-9-]+$/.test(slug)
  );
}

/**
 * Sum the per-link report rows. Deliberately accepts rows only, never the
 * report envelope: the global `unattributed_revenue` must never leak into a
 * row total or be divided between links.
 * @param {Array<{clicks?: number, payments?: number, revenue?: number}>} rows
 * @returns {{clicks: number, payments: number, revenue: number}}
 */
export function sumLinkReportRows(rows) {
  const totals = { clicks: 0, payments: 0, revenue: 0 };
  if (!Array.isArray(rows)) return totals;
  for (const row of rows) {
    totals.clicks += Number(row?.clicks) || 0;
    totals.payments += Number(row?.payments) || 0;
    totals.revenue += Number(row?.revenue) || 0;
  }
  return totals;
}
