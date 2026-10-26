import {
  parseVisitorParam,
  stripVisitorFromUrl,
  VISITOR_PARAM,
} from "./visitor-param.mjs";

export type FunnelStep =
  | "landing_view"
  | "demo_lesson_view"
  | "diagnostic_start"
  | "diagnostic_complete"
  | "bot_handoff"
  | "bot_lead"
  | "pricing_select_tier"
  | "order_created"
  | "payment_confirmed"
  | "access_delivered"
  | "first_task_completed";

export interface FunnelEvent {
  step: FunnelStep;
  timestamp: string;
  visitor_id: string;
  attribution_token: string;
  experiments?: Record<string, string>;
  metadata?: Record<string, any>;
}

const STORAGE_KEY = "spark_funnel_events";
const VISITOR_KEY = "spark_visitor_id";
const ATTRIBUTION_KEY = "spark_attribution_token";
const CAMPAIGN_STORAGE_KEY = "spark_utm_context";
/** Cookie the /r/:slug redirect and the site share, so repeat clicks reuse one id. */
const VISITOR_COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 31 536 000 seconds
const VISITOR_ID_LENGTH = 20;
/** Resolved id for this page view: survives a throwing localStorage. */
let cachedVisitorId: string | null = null;

interface StoredCampaign {
  utm_source?: string;
  utm_medium?: string;
  utm_campaign?: string;
  utm_content?: string;
  initial_referrer?: string;
  saved_at?: number;
}

function randomId(prefix: string, length: number): string {
  const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789";
  let result = prefix;
  if (typeof crypto !== "undefined" && typeof crypto.getRandomValues === "function") {
    const bytes = new Uint8Array(length);
    crypto.getRandomValues(bytes);
    for (let i = 0; i < length; i++) result += alphabet[bytes[i] % alphabet.length];
    return result;
  }
  for (let i = 0; i < length; i++) result += alphabet[Math.floor(Math.random() * alphabet.length)];
  return result;
}

function visitorCookieAttributes(): string {
  const secure =
    typeof window !== "undefined" && window.location.protocol === "https:" ? "; Secure" : "";
  return `; Path=/; Max-Age=${VISITOR_COOKIE_MAX_AGE}; SameSite=Lax${secure}`;
}

function readVisitorCookie(): string {
  if (typeof document === "undefined") return "";
  try {
    for (const entry of document.cookie ? document.cookie.split(";") : []) {
      const eq = entry.indexOf("=");
      const name = (eq === -1 ? entry : entry.slice(0, eq)).trim();
      if (name !== VISITOR_PARAM) continue;
      const value = (eq === -1 ? "" : entry.slice(eq + 1)).trim();
      if (!value) return "";
      try {
        return decodeURIComponent(value);
      } catch {
        return value;
      }
    }
  } catch {
    // Cookies can be blocked; fall through to minting a fresh id.
  }
  return "";
}

function writeVisitorCookie(id: string): void {
  if (typeof document === "undefined") return;
  try {
    document.cookie = `${VISITOR_PARAM}=${encodeURIComponent(id)}${visitorCookieAttributes()}`;
  } catch {
    // Cookies can be blocked; the other stores still carry the id.
  }
}

/** Keep localStorage and the shared cookie in sync; neither is required to work. */
function persistVisitorId(id: string): void {
  try {
    localStorage.setItem(VISITOR_KEY, id);
  } catch {
    // Private mode: the module cache and the cookie still carry the id.
  }
  writeVisitorCookie(id);
}

/**
 * Remove `spark_vid` from the current address bar, keeping the path, the hash
 * and every other parameter. Safe to call when the parameter is absent.
 */
export function stripVisitorParamFromUrl(): void {
  if (typeof window === "undefined") return;
  try {
    const { pathname, search, hash } = window.location;
    const original = `${pathname}${search}${hash}`;
    const target = stripVisitorFromUrl(original);
    if (target === original) return;
    window.history.replaceState(window.history.state, "", target);
  } catch {
    // The id is already stored; a failed URL rewrite is not worth breaking on.
  }
}

/**
 * A campaign redirect arrives as `?spark_vid=<id>&utm_*`. Adopt that visitor id
 * so the landing event joins the click row, then drop the parameter from the
 * address bar so it never ends up in a copied or shared URL. Values that do not
 * look like a redirect id are ignored and storage is left untouched.
 */
function adoptVisitorIdFromUrl(): string {
  if (typeof window === "undefined") return "";
  try {
    const candidate = parseVisitorParam(window.location.search);
    if (!candidate) return "";

    // Persist before rewriting the URL so a failure cannot lose the id.
    persistVisitorId(candidate);
    stripVisitorParamFromUrl();
    return candidate;
  } catch {
    return "";
  }
}

/** Stable anonymous browser identifier. Contains no personal data. */
export function getVisitorId(): string {
  if (typeof window === "undefined") return "";
  if (cachedVisitorId) return cachedVisitorId;

  const adopted = adoptVisitorIdFromUrl();
  if (adopted) {
    cachedVisitorId = adopted;
    return adopted;
  }

  let id = "";
  try {
    id = localStorage.getItem(VISITOR_KEY) || "";
  } catch {
    id = "";
  }
  if (!id) id = readVisitorCookie();
  if (!id) id = randomId("v", VISITOR_ID_LENGTH);

  cachedVisitorId = id;
  persistVisitorId(id);
  return id;
}

/**
 * Short token that travels through Telegram's start parameter. It links a
 * landing visitor to a bot lead; it is opaque and contains no user identity.
 */
export const BOT_URL = "https://t.me/spark_prep_bot";

/**
 * Append the anonymous attribution token to a Telegram bot link. The bot
 * copies the visitor id from the landing event with the same token, so every
 * entry point (lead magnet, calculator, pricing) must carry it.
 */
export function withBotAttribution(url: string): string {
  if (typeof window === "undefined") return url;
  const token = getAttributionToken();
  if (!token) return url;
  try {
    const parsed = new URL(url);
    if (!/(^|\.)t\.me$/i.test(parsed.hostname)) return url;
    if (!parsed.pathname.startsWith("/spark_prep_bot")) return url;
    const start = parsed.searchParams.get("start") || "";
    if (start.includes("_a_") || start.startsWith("a_")) {
      return url;
    }
    const next = compactStartParam(start, token);
    parsed.searchParams.set("start", next);
    return parsed.toString();
  } catch {
    return url;
  }
}

/** Telegram start payloads are capped at 64 characters. Keep action + attribution, drop flags. */
export function compactStartParam(action: string, attributionToken: string, ref = ""): string {
  const cleanAction = action.replace(/[^a-zA-Z0-9_]/g, "").slice(0, 24);
  const token = attributionToken.replace(/[^a-zA-Z0-9]/g, "").slice(0, 16);
  const cleanRef = ref.replace(/[^a-zA-Z0-9_-]/g, "").slice(0, 12);
  let param = cleanAction || "start";
  if (token) param += `_a_${token}`;
  if (cleanRef) {
    const withRef = `${param}_ref_${cleanRef}`;
    if (withRef.length <= 64) param = withRef;
  }
  return param.slice(0, 64);
}

export function buildCheckoutStartParam(tier: "basic" | "accelerator" | "vip", ref = ""): string {
  return compactStartParam(`pay_${tier}`, getAttributionToken(), ref);
}

/** Build a bot link with an optional start payload and the attribution token. */
export function botStartUrl(params: string = ""): string {
  const base = params ? `${BOT_URL}?start=${params}` : BOT_URL;
  return withBotAttribution(base);
}

export function getAttributionToken(): string {
  if (typeof window === "undefined") return "";
  try {
    let token = localStorage.getItem(ATTRIBUTION_KEY);
    if (!token || !/^a[a-z0-9]{8,15}$/.test(token)) {
      token = randomId("a", 12);
      localStorage.setItem(ATTRIBUTION_KEY, token);
    }
    return token;
  } catch {
    return "";
  }
}

function experimentAssignments(): Record<string, string> {
  if (typeof window === "undefined") return {};
  const result: Record<string, string> = {};
  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (!key || !key.startsWith("spark_exp_")) continue;
      const variant = localStorage.getItem(key);
      if (variant) result[key.slice("spark_exp_".length)] = variant;
    }
  } catch {
    // Ignore storage errors.
  }
  return result;
}

function campaignContext(): Record<string, string> {
  if (typeof window === "undefined") return {};
  try {
    const params = new URLSearchParams(window.location.search);
    const fromUrl: Record<string, string> = {};
    for (const key of ["utm_source", "utm_medium", "utm_campaign", "utm_content"]) {
      const value = params.get(key);
      if (value) fromUrl[key] = value.slice(0, 80);
    }

    // 30 days retention window for stored campaign attribution
    const THIRTY_DAYS_MS = 30 * 24 * 60 * 60 * 1000;
    let stored: StoredCampaign = {};
    const rawStored = localStorage.getItem(CAMPAIGN_STORAGE_KEY);
    if (rawStored) {
      try {
        const parsed = JSON.parse(rawStored);
        if (parsed && typeof parsed.saved_at === "number" && Date.now() - parsed.saved_at < THIRTY_DAYS_MS) {
          stored = parsed;
        }
      } catch {
        // Safe fallback
      }
    }

    // If URL has new UTMs, persist them (Last-touch update)
    if (Object.keys(fromUrl).length > 0) {
      stored = {
        ...stored,
        ...fromUrl,
        initial_referrer: stored.initial_referrer || (typeof document !== "undefined" ? document.referrer.slice(0, 200) : ""),
        saved_at: Date.now(),
      };
      try {
        localStorage.setItem(CAMPAIGN_STORAGE_KEY, JSON.stringify(stored));
      } catch {
        // Ignore storage errors in private mode
      }
    } else if (typeof document !== "undefined" && document.referrer && !stored.initial_referrer) {
      // Record organic or external referrer if no UTMs
      stored.initial_referrer = document.referrer.slice(0, 200);
      stored.saved_at = stored.saved_at || Date.now();
      try {
        localStorage.setItem(CAMPAIGN_STORAGE_KEY, JSON.stringify(stored));
      } catch {
        // Ignore
      }
    }

    const result: Record<string, string> = {};
    if (stored.utm_source) result.utm_source = stored.utm_source;
    if (stored.utm_medium) result.utm_medium = stored.utm_medium;
    if (stored.utm_campaign) result.utm_campaign = stored.utm_campaign;
    if (stored.utm_content) result.utm_content = stored.utm_content;
    if (stored.initial_referrer) result.initial_referrer = stored.initial_referrer;

    return result;
  } catch {
    return {};
  }
}

export function trackFunnelStep(step: FunnelStep, metadata?: Record<string, any>) {
  if (typeof window === "undefined") return;

  const visitorId = getVisitorId();
  const attributionToken = getAttributionToken();
  const event: FunnelEvent = {
    step,
    timestamp: new Date().toISOString(),
    visitor_id: visitorId,
    attribution_token: attributionToken,
    experiments: experimentAssignments(),
    metadata: { ...campaignContext(), ...(metadata || {}) },
  };

  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const events: FunnelEvent[] = raw ? JSON.parse(raw) : [];
    if (events.length >= 100) events.shift();
    events.push(event);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(events));
  } catch {
    // Ignore storage errors in private browsing.
  }

  window.dispatchEvent(new CustomEvent("spark_funnel_event", { detail: event }));

  const rawBase = (process.env.NEXT_PUBLIC_API_URL || "").trim().replace(/\/+$/, "");
  const endpoint = rawBase
    ? rawBase.endsWith("/api/v1")
      ? `${rawBase}/analytics/track`
      : `${rawBase}/api/v1/analytics/track`
    : "/api/v1/analytics/track";

  fetch(endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    keepalive: true,
    body: JSON.stringify(event),
  })
    .then((res) => {
      if (!res.ok) console.warn(`[analytics] ${step} was not recorded: HTTP ${res.status}`);
    })
    .catch(() => {
      // Offline: events stay in localStorage for a later retry/manual export.
    });
}

export function getFunnelEvents(): FunnelEvent[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}
