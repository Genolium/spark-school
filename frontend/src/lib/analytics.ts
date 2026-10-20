export type FunnelStep =
  | "landing_view"
  | "demo_lesson_view"
  | "diagnostic_start"
  | "diagnostic_complete"
  | "bot_handoff"
  | "pricing_select_tier"
  | "order_created"
  | "payment_confirmed";

export interface FunnelEvent {
  step: FunnelStep;
  timestamp: string;
  metadata?: Record<string, any>;
}

const STORAGE_KEY = "spark_funnel_events";

export function trackFunnelStep(step: FunnelStep, metadata?: Record<string, any>) {
  if (typeof window === "undefined") return;

  const event: FunnelEvent = {
    step,
    timestamp: new Date().toISOString(),
    metadata,
  };

  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    const events: FunnelEvent[] = raw ? JSON.parse(raw) : [];
    // Keep last 100 events to avoid unbounded growth
    if (events.length >= 100) {
      events.shift();
    }
    events.push(event);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(events));
  } catch {
    // Ignore storage errors in private browsing
  }

  // Dispatch custom event for reactive UI
  window.dispatchEvent(new CustomEvent("spark_funnel_event", { detail: event }));

  // Transmit event to backend API for end-to-end telemetry
  const rawBase = (process.env.NEXT_PUBLIC_API_URL || "").trim().replace(/\/+$/, "");
  const endpoint = rawBase
    ? (rawBase.endsWith("/api/v1") ? `${rawBase}/analytics/track` : `${rawBase}/api/v1/analytics/track`)
    : "/api/v1/analytics/track";

  fetch(endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(event),
  }).catch(() => {
    // Fire-and-forget; do not interrupt user experience
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
