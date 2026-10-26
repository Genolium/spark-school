import test from "node:test";
import assert from "node:assert/strict";

import {
  isValidCampaignSlug,
  moscowDate,
  reportWindow,
  slugifyCampaignLink,
  sumLinkReportRows,
} from "../src/lib/campaign-links.mjs";

test("moscowDate: the business day rolls over at MSK midnight", () => {
  assert.equal(moscowDate(new Date("2026-10-26T20:59:59Z")), "2026-10-26");
  assert.equal(moscowDate(new Date("2026-10-26T21:00:00Z")), "2026-10-27");
  assert.equal(moscowDate(new Date("2026-01-01T21:00:00Z")), "2026-01-02");
});

test("reportWindow: includes today in Moscow time for 7/14/30 days", () => {
  // 2026-10-27 01:30 MSK: the UTC date is still the 26th.
  const now = new Date("2026-10-26T22:30:00Z");
  assert.deepEqual(reportWindow(7, now), { from: "2026-10-21", to: "2026-10-27" });
  assert.deepEqual(reportWindow(14, now), { from: "2026-10-14", to: "2026-10-27" });
  assert.deepEqual(reportWindow(30, now), { from: "2026-09-28", to: "2026-10-27" });
});

test("reportWindow: just before MSK midnight keeps the previous day", () => {
  const now = new Date("2026-10-26T20:59:59Z");
  assert.deepEqual(reportWindow(7, now), { from: "2026-10-20", to: "2026-10-26" });
});

test("slugifyCampaignLink: matches the backend latin/dash rules", () => {
  assert.equal(slugifyCampaignLink("Nemtsov October"), "nemtsov-october");
  assert.equal(slugifyCampaignLink("UPPER_case"), "upper-case");
  assert.equal(slugifyCampaignLink("abc!!!def"), "abc-def");
  assert.equal(slugifyCampaignLink("  --abc--  "), "abc");
  assert.equal(slugifyCampaignLink("a".repeat(60)).length, 40);
  assert.equal(
    slugifyCampaignLink(`${"a".repeat(39)}-tail`),
    "a".repeat(39),
    "truncation must not leave a trailing dash"
  );
});

test("slugifyCampaignLink: Cyrillic-only input normalizes to empty", () => {
  assert.equal(slugifyCampaignLink("Привет"), "");
  assert.equal(slugifyCampaignLink("Ссылка на канал"), "");
  assert.equal(slugifyCampaignLink("тест-abc"), "abc");
});

test("isValidCampaignSlug: 3-40 chars, lowercase latin, digits and dash", () => {
  assert.equal(isValidCampaignSlug("ab"), false);
  assert.equal(isValidCampaignSlug("abc"), true);
  assert.equal(isValidCampaignSlug("a".repeat(40)), true);
  assert.equal(isValidCampaignSlug("a".repeat(41)), false);
  assert.equal(isValidCampaignSlug("UPPER"), false);
  assert.equal(isValidCampaignSlug("has space"), false);
  assert.equal(isValidCampaignSlug(""), false);
});

test("sumLinkReportRows: totals rows and never the unattributed revenue", () => {
  const report = {
    rows: [
      { slug: "a", clicks: 10, payments: 1, revenue: 1000 },
      { slug: "b", clicks: 4, payments: 2, revenue: 500.5 },
    ],
    unattributed_revenue: 7777,
  };

  const totals = sumLinkReportRows(report.rows);
  assert.deepEqual(totals, { clicks: 14, payments: 3, revenue: 1500.5 });
  assert.equal("unattributed_revenue" in totals, false);
  assert.notEqual(totals.revenue, 1500.5 + report.unattributed_revenue);
});

test("sumLinkReportRows: empty and missing rows produce zero totals", () => {
  assert.deepEqual(sumLinkReportRows([]), { clicks: 0, payments: 0, revenue: 0 });
  assert.deepEqual(sumLinkReportRows(null), { clicks: 0, payments: 0, revenue: 0 });
});
