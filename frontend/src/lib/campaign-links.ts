// UI helpers for admin campaign short links. The pure math lives in
// `campaign-links.mjs` so `node --test` can exercise it directly.

export const SHORT_LINK_BASE = "https://so-called-spark.ru/r/";

// Fired after a link is created so the report can refresh without prop
// drilling between independent admin sections.
export const CAMPAIGN_LINKS_CHANGED_EVENT = "spark:campaign-links-changed";

export function shortLinkUrl(slug: string): string {
  return `${SHORT_LINK_BASE}${slug}`;
}

export function notifyCampaignLinksChanged(): void {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(CAMPAIGN_LINKS_CHANGED_EVENT));
  }
}

export type ClipboardResult = { ok: true } | { ok: false; reason: "unsupported" | "failed" };

/**
 * Copy text only when the async clipboard API is actually available and the
 * write succeeds. Callers must not optimistically show a "copied" state.
 */
export async function copyTextToClipboard(text: string): Promise<ClipboardResult> {
  if (typeof navigator === "undefined" || !navigator.clipboard?.writeText) {
    return { ok: false, reason: "unsupported" };
  }
  try {
    await navigator.clipboard.writeText(text);
    return { ok: true };
  } catch {
    return { ok: false, reason: "failed" };
  }
}

/**
 * Fallback for non-secure contexts where the clipboard API is unavailable:
 * select the visible text so the admin can press Ctrl/Cmd+C.
 */
export function selectElementText(element: HTMLElement | null): void {
  if (typeof window === "undefined" || !element) return;
  const selection = window.getSelection();
  if (!selection) return;
  const range = document.createRange();
  range.selectNodeContents(element);
  selection.removeAllRanges();
  selection.addRange(range);
}
