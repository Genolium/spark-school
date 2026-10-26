// Shared helpers for admin campaign short links.

export const SHORT_LINK_BASE = "https://so-called-spark.ru/r/";

// Fired after a link is created so the report can refresh without prop
// drilling between independent admin sections.
export const CAMPAIGN_LINKS_CHANGED_EVENT = "spark:campaign-links-changed";

export function shortLinkUrl(slug: string): string {
  return `${SHORT_LINK_BASE}${slug}`;
}

// Mirrors the backend slugify: lowercase latin letters, digits and single
// dashes, at most 40 characters. The backend still normalizes on save; this
// keeps the preview and the client-side validation honest.
export function slugifyCampaignLink(raw: string): string {
  return raw
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 40)
    .replace(/-+$/g, "");
}

export function notifyCampaignLinksChanged(): void {
  if (typeof window !== "undefined") {
    window.dispatchEvent(new CustomEvent(CAMPAIGN_LINKS_CHANGED_EVENT));
  }
}
