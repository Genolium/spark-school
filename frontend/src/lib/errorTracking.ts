"use client";

import { API_BASE } from "./api";

let sent = 0;
const MAX_PER_SESSION = 10;

export function reportClientError(message: string, stack?: string) {
  if (typeof window === "undefined" || sent >= MAX_PER_SESSION) return;
  sent += 1;
  void fetch(`${API_BASE}/client-errors`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    keepalive: true,
    body: JSON.stringify({
      message: message.slice(0, 500),
      stack: (stack || "").slice(0, 8000),
      url: window.location.href.slice(0, 500),
    }),
  }).catch(() => {
    // Never let error reporting break the page.
  });
}
