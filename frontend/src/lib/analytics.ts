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

/** Stable anonymous browser identifier. Contains no personal data. */
export function getVisitorId(): string {
  if (typeof window === "undefined") return "";
  try {
    let id = localStorage.getItem(VISITOR_KEY);
    if (!id) {
      id = typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
        ? crypto.randomUUID()
        : randomId("v", 24);
      localStorage.setItem(VISITOR_KEY, id);
    }
    return id;
  } catch {
    return "";
  }
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
    const next = `${start ? `${start}_` : ""}a_${token}`.slice(0, 64);
    parsed.searchParams.set("start", next);
    return parsed.toString();
  } catch {
    return url;
  }
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
    const result: Record<string, string> = {};
    for (const key of ["utm_source", "utm_medium", "utm_campaign", "utm_content"]) {
      const value = params.get(key);
      if (value) result[key] = value.slice(0, 80);
    }
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
