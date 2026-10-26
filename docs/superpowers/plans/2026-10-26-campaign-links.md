# Campaign Links (redirects + per-link analytics) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Short links like `https://so-called-spark.ru/r/<slug>` that redirect to a Telegram channel or the bot (or to the landing with UTM), record every click, and show per-link traffic and money in the admin panel.

**Architecture:** A public `GET /r/:slug` handler looks up an active link, records an anonymous click row, and 302-redirects to the configured target. For site targets it appends `utm_*` and a one-time `spark_vid` parameter so the existing client tracker adopts the same visitor id; for bot targets it appends the existing `_a_`/`src_` payload, which the bot already parses. A report query joins clicks → analytics events → payment transactions by `visitor_id` and returns per-link traffic, leads, and revenue.

**Tech Stack:** Go 1.26 + Gin + GORM + goose migrations (PostgreSQL), Next.js 16 admin UI, existing analytics schema (`analytics_events`, `payment_transactions`, `users.visitor_id`).

**Spec:** this document (the request came as a chat brief).

## Global Constraints

- Money figures come only from `payment_transactions` (`status = 'SUCCESS'`), never from receipts or prices.
- Anonymous identifiers only: no phone, email, telegram id, or IP in URLs or link rows. IP is stored as a salted SHA-256 hash or not at all.
- Existing attribution plumbing must be reused: `visitor_id`, `attribution_token`, `utm_*`, bot `_a_`/`src_` parsing.
- Admin endpoints live under `/api/v1/admin/*` and are protected by the existing admin JWT middleware.
- Every new table gets a goose migration; `RequiredSchemaVersion` is bumped in the same commit.
- New analytics must show "unattributed" money instead of pretending it belongs to a link.

## Review Focus

- A click that never becomes a site visit must still be counted as a click; the report must not silently drop it.
- Two clicks from the same browser count as one unique visitor but two clicks.
- An inactive or unknown slug must return 404 without recording a click.
- A link must never redirect to an arbitrary URL supplied by the visitor (open redirect).
- Bot/channel targets must keep working after the Telegram invite link rotates.

---

### Task 1: Schema for links and clicks

**Files:**
- Create: `backend/internal/database/migrations/00026_campaign_links.sql`
- Modify: `backend/internal/models/models.go`
- Modify: `backend/internal/database/db.go` (`RequiredSchemaVersion` → 26, `ValidateSchema` entry)
- Test: `backend/internal/database/migrations_test.go`

**Interfaces:**
- Produces: `models.CampaignLink{ID, Slug, Title, TargetType, TargetValue, UTMSource, UTMMedium, UTMCampaign, UTMContent, IsActive, CreatedAt, CreatedBy}` and `models.CampaignLinkClick{ID, LinkID, VisitorID, AttributionToken, IPHash, UserAgent, Referer, CreatedAt}`.

- [ ] **Step 1: Write the failing migration test**

In `TestPostgresMigrations_EmptyDatabaseAndRerun`, assert both tables and the `slug` unique index exist:

```go
for _, table := range []string{"campaign_links", "campaign_link_clicks"} {
    var exists bool
    sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)", table).Scan(&exists)
    if !exists { t.Errorf("expected table %s", table) }
}
```

- [ ] **Step 2: Run it and confirm it fails** — `TEST_DATABASE_URL=... go test ./internal/database -run TestPostgresMigrations_EmptyDatabaseAndRerun -v` → "expected table campaign_links".

- [ ] **Step 3: Write migration 00026**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS campaign_links (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(40) NOT NULL UNIQUE,
    title VARCHAR(120) NOT NULL,
    target_type VARCHAR(16) NOT NULL CHECK (target_type IN ('channel','bot','landing')),
    target_value VARCHAR(255) NOT NULL DEFAULT '',
    utm_source VARCHAR(64),
    utm_medium VARCHAR(64),
    utm_campaign VARCHAR(80),
    utm_content VARCHAR(80),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS campaign_link_clicks (
    id BIGSERIAL PRIMARY KEY,
    link_id BIGINT NOT NULL REFERENCES campaign_links(id) ON DELETE CASCADE,
    visitor_id VARCHAR(64),
    attribution_token VARCHAR(64),
    ip_hash VARCHAR(64),
    user_agent VARCHAR(255),
    referer VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_link_created ON campaign_link_clicks(link_id, created_at);
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_visitor ON campaign_link_clicks(visitor_id);

-- +goose Down
DROP TABLE IF EXISTS campaign_link_clicks;
DROP TABLE IF EXISTS campaign_links;
```

- [ ] **Step 4: Add the models, bump `RequiredSchemaVersion` to 26, add the `ValidateSchema` entry, run the migration tests** — expected PASS.

- [ ] **Step 5: Commit** — `feat(links): add campaign links and click tables`.

---

### Task 2: Public redirect endpoint with click recording

**Files:**
- Create: `backend/internal/handlers/links.go`
- Modify: `backend/cmd/server/main.go` (route `r.GET("/r/:slug", handlers.RedirectCampaignLink(db))`)
- Test: `backend/internal/handlers/links_pg_test.go`

**Interfaces:**
- Consumes: `models.CampaignLink`, `models.CampaignLinkClick`.
- Produces: `func RedirectCampaignLink(db *gorm.DB) gin.HandlerFunc`; `func campaignTargetURL(link models.CampaignLink, visitorID, attributionToken string) string`.

- [ ] **Step 1: Write the failing tests**

```go
func TestRedirectCampaignLink_RecordsClickAndRedirects(t *testing.T) {
    // seed link{Slug:"yt", TargetType:"landing", UTMSource:"youtube", UTMCampaign:"oct"}
    // GET /r/yt -> 302, Location contains "utm_source=youtube" and "spark_vid="
    // expect exactly one campaign_link_clicks row with that visitor id
}
func TestRedirectCampaignLink_UnknownSlugIs404(t *testing.T) {
    // GET /r/missing -> 404 and zero click rows
}
func TestRedirectCampaignLink_BotTargetCarriesAttribution(t *testing.T) {
    // link{TargetType:"bot", TargetValue:"src_youtube"} ->
    // Location contains "t.me/spark_prep_bot?start=", "src_youtube" and "_a_"
}
```

- [ ] **Step 2: Run and confirm failure** — handler not defined.

- [ ] **Step 3: Implement the handler**

Rules: look up `slug` with `is_active = true` (404 otherwise); generate `visitor_id` (`v` + 12 random alnum) and `attribution_token` (`a` + 12 random alnum) when absent; insert the click row best-effort (a click-recording failure must not break the redirect); build the target:
- `landing` → `FRONTEND_URL + "/?spark_vid=<visitor_id>&utm_source=...&utm_campaign=..."`
- `bot` → `https://t.me/spark_prep_bot?start=<target_value>_a_<attribution_token>` truncated to 64
- `channel` → `target_value` (must start with `https://t.me/`; otherwise fall back to the configured `TelegramInviteLink`)

- [ ] **Step 4: Tests pass; also assert no open redirect** — a link whose target starts with `http` but not `https://t.me/` must not be used.

- [ ] **Step 5: Commit** — `feat(links): add /r/:slug redirect with click tracking`.

---

### Task 3: Visitor-id adoption on the site

**Files:**
- Modify: `frontend/src/lib/analytics.ts` (`getVisitorId`)
- Test: manual browser check (localStorage + URL parameter).

**Interfaces:**
- Consumes: `spark_vid` query parameter from Task 2.
- Produces: the site uses the redirect's visitor id for `landing_view` and every later event.

- [ ] **Step 1: Write the failing check** — open `/?spark_vid=vtest123` in the browser with empty localStorage and assert (before the fix) that `localStorage.spark_visitor_id` is not `vtest123`.
- [ ] **Step 2: Implement** — in `getVisitorId`, if the URL has `spark_vid` matching `^v[a-z0-9]{6,40}$`, store it and remove the parameter with `history.replaceState` (so it does not leak into shared URLs).
- [ ] **Step 3: Verify in the browser** — the same check now returns `vtest123`; `landing_view` in the network tab carries it.
- [ ] **Step 4: Commit** — `feat(analytics): adopt the visitor id from a campaign redirect`.

---

### Task 4: Link CRUD + report API

**Files:**
- Modify: `backend/internal/handlers/links.go`
- Modify: `backend/cmd/server/main.go` (admin routes)
- Test: `backend/internal/handlers/links_pg_test.go`

**Interfaces:**
- Produces:
  - `GET /api/v1/admin/links` → `[]CampaignLink` + click counts.
  - `POST /api/v1/admin/links` `{slug,title,target_type,target_value,utm_*}`.
  - `PUT /api/v1/admin/links/:id`, `DELETE /api/v1/admin/links/:id`.
  - `GET /api/v1/admin/links/report?from&to` → `[]LinkReportRow{Slug,Title,Clicks,UniqueVisitors,SiteVisits,Leads,Payments,Revenue,RPV,EPC}`.
  - `func slugify(raw string) string` (lowercase, `[a-z0-9-]`, 3–40 chars).

- [ ] **Step 1: Write failing tests** — create a link; record two clicks for one visitor and one for another; create a `landing_view`, a `bot_lead`, and a `payment_transactions` row for the first visitor; assert `Clicks=3`, `UniqueVisitors=2`, `SiteVisits=1`, `Leads=1`, `Payments=1`, `Revenue=...`, `EPC = revenue / clicks`.
- [ ] **Step 2: Run and confirm failure.**
- [ ] **Step 3: Implement the queries**

```sql
WITH clicks AS (
  SELECT link_id, COUNT(*) AS clicks, COUNT(DISTINCT visitor_id) AS visitors
  FROM campaign_link_clicks
  WHERE created_at >= ? AND created_at < ?
  GROUP BY link_id
),
visits AS (
  SELECT c.link_id, COUNT(DISTINCT ae.visitor_id) AS site_visits
  FROM campaign_link_clicks c
  JOIN analytics_events ae ON ae.visitor_id = c.visitor_id AND ae.step = 'landing_view'
  WHERE c.created_at >= ? AND c.created_at < ?
  GROUP BY c.link_id
),
money AS (
  SELECT c.link_id, COUNT(DISTINCT pt.payment_id) AS payments, COALESCE(SUM(pt.amount),0) AS revenue
  FROM campaign_link_clicks c
  JOIN payment_transactions pt ON pt.visitor_id = c.visitor_id AND pt.status = 'SUCCESS'
  WHERE c.created_at >= ? AND c.created_at < ?
  GROUP BY c.link_id
)
SELECT l.slug, l.title, COALESCE(cl.clicks,0), COALESCE(cl.visitors,0), ...
FROM campaign_links l
LEFT JOIN clicks cl ON cl.link_id = l.id ...
ORDER BY COALESCE(cl.clicks,0) DESC
```

- [ ] **Step 4: Tests pass; validate slug collisions return 409.**
- [ ] **Step 5: Commit** — `feat(links): admin CRUD and per-link traffic report`.

---

### Task 5: Admin UI — create short links and read the report

**Files:**
- Create: `frontend/src/components/Admin/LinksReport.tsx`
- Modify: `frontend/src/components/Admin/CampaignLinkGenerator.tsx` (add a "create short link" action that POSTs and shows `/r/<slug>`)
- Modify: `frontend/src/app/admin/page.tsx`
- Modify: `frontend/src/lib/api-admin.ts`, `frontend/src/lib/api-types.ts`

**Interfaces:**
- Consumes: Task 4 endpoints; existing `getAdminToken()` helper.
- Produces: `LinksReport` table with period filter (7/14/30 days), columns: link, clicks, unique visitors, site visits, leads, payments, revenue, EPC, and a "copy short link" button.

- [ ] **Step 1: Add types and API methods** (`getLinks`, `createLink`, `getLinksReport`).
- [ ] **Step 2: Build the report component** — empty state "пока нет переходов", error state, refresh button.
- [ ] **Step 3: Wire the generator** — after creating, show the ready `https://so-called-spark.ru/r/<slug>` with a copy button.
- [ ] **Step 4: `npx tsc --noEmit` and `npm run build` pass.**
- [ ] **Step 5: Commit** — `feat(admin): short links and per-link report`.

---

### Task 6: Privacy, retention, and rollout

**Files:**
- Modify: `backend/internal/privacy/cleaner.go` (or the existing retention worker)
- Modify: `README.md` (short section "Campaign links")
- Test: `backend/internal/privacy/retention_test.go`

**Interfaces:**
- Consumes: `campaign_link_clicks.created_at`.
- Produces: clicks older than `ANALYTICS_RETENTION_DAYS` are deleted in the same daily job as analytics events.

- [ ] **Step 1: Write the failing retention test** — seed a click older than the retention window, run the cleaner, assert it is gone.
- [ ] **Step 2: Implement the delete** in the retention transaction.
- [ ] **Step 3: Verify the full suite** — `go test ./...` and node tests.
- [ ] **Step 4: Deploy and smoke-check** — create a link, open `/r/<slug>` twice from one browser and once from another, confirm `clicks=3`, `unique=2`, and the redirect lands on the right target.
- [ ] **Step 5: Commit** — `feat(links): retention for click rows and docs`.

---

## Self-review notes

- Coverage: click tracking (Task 2), site visit attribution (Tasks 2–3), per-link money (Task 4), UI (Task 5), privacy/rollout (Task 6). Link CRUD is in Task 4 because the report needs links to exist.
- Deliberate non-goals: per-click geo/device breakdown, UTM auto-tagging for paid ads, and a link-level A/B test. Add later if the report proves useful.
- Known limitation to surface in the UI: revenue is attributed only when the payer has a `visitor_id`; payments without one are shown as "без атрибуции" (global number, not per link).
