/**
 * Pure helpers for the `spark_vid` query parameter shared by the campaign
 * redirect and the site. Kept as plain ESM (no imports, no side effects) so the
 * frontend bundle and `node --test` can both use exactly the same code.
 */

/** Query parameter carrying a visitor id from a /r/:slug redirect. */
export const VISITOR_PARAM = "spark_vid";

/** Visitor ids minted by the redirect: `v` plus 6–40 lowercase alphanumerics. */
export const VISITOR_PATTERN = /^v[a-z0-9]{6,40}$/;

/**
 * Decode one application/x-www-form-urlencoded component. Malformed escapes are
 * kept verbatim instead of throwing.
 * @param {string} value
 * @returns {string}
 */
function decodeComponent(value) {
  const spaced = value.replace(/\+/g, " ");
  try {
    return decodeURIComponent(spaced);
  } catch {
    return spaced;
  }
}

/**
 * Decoded key of a raw `key=value` query segment.
 * @param {string} part
 * @returns {string}
 */
function keyOf(part) {
  const eq = part.indexOf("=");
  return decodeComponent(eq === -1 ? part : part.slice(0, eq));
}

/**
 * Visitor id carried by a query string, or "" when it is absent, empty or does
 * not match the ids the redirect mints. Only the first `spark_vid` is read.
 * @param {string} search query string, with or without a leading `?`
 * @returns {string}
 */
export function parseVisitorParam(search) {
  const raw = search.startsWith("?") ? search.slice(1) : search;
  if (!raw) return "";
  for (const part of raw.split("&")) {
    if (!part || keyOf(part) !== VISITOR_PARAM) continue;
    const eq = part.indexOf("=");
    const value = decodeComponent(eq === -1 ? "" : part.slice(eq + 1));
    return VISITOR_PATTERN.test(value) ? value : "";
  }
  return "";
}

/**
 * Remove every `spark_vid` parameter while keeping the remaining query string
 * byte-for-byte: order, duplicates and the original `%20`/`+` encoding survive.
 * A leading `?` is kept only when something remains.
 * @param {string} search query string, with or without a leading `?`
 * @returns {string}
 */
export function stripVisitorParam(search) {
  const hasPrefix = search.startsWith("?");
  const raw = hasPrefix ? search.slice(1) : search;
  if (!raw) return "";
  const parts = raw.split("&");
  const kept = parts.filter((part) => !part || keyOf(part) !== VISITOR_PARAM);
  if (kept.length === parts.length) return search;
  const query = kept.join("&");
  if (!query) return "";
  return (hasPrefix ? "?" : "") + query;
}

/**
 * Rewrite a relative URL (path + query + hash) without the visitor parameter.
 * The path, the hash and every other parameter are preserved.
 * @param {string} url
 * @returns {string}
 */
export function stripVisitorFromUrl(url) {
  const hashIndex = url.indexOf("#");
  const hash = hashIndex === -1 ? "" : url.slice(hashIndex);
  const withoutHash = hashIndex === -1 ? url : url.slice(0, hashIndex);
  const queryIndex = withoutHash.indexOf("?");
  const path = queryIndex === -1 ? withoutHash : withoutHash.slice(0, queryIndex);
  const search = queryIndex === -1 ? "" : withoutHash.slice(queryIndex);
  return path + stripVisitorParam(search) + hash;
}
